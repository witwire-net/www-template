package postgres

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"

	accountsapplication "www-template/packages/backend/internal/application/accounts"
)

var _ accountsapplication.AccountRepository = (*AccountRepository)(nil)
var _ accountsapplication.AccountSearchRepository = (*AccountRepository)(nil)

// AccountRepository は Admin account creation と account search 用に Product Account root を扱う PostgreSQL adapter である。
//
// 役割:
//   - public.accounts / public.account_settings と Admin-owned schema の admin.audit_events だけを扱う。
//   - accountsapplication.AccountRepository と AccountSearchRepository port を実装し、application 層へ GORM record や DB error を公開しない。
//   - Account root の email 正規化、status 初期値、locale 初期値は domain.Account から受け取り、repository 内で業務 rule を再実装しない。
//   - account search では application 検証済み query だけを受け取り、GORM parameter binding で unsafe SQL construction を避ける。
//
// 引数:
//   - NewAccountRepository の db: Admin runtime role として public.accounts と admin.audit_events を更新できる GORM handle。
//   - CreateAccountWithAuditTarget の record: 作成済み Account root と pending audit event ID を含む application DTO。
//
// 戻り値:
//   - accountsapplication.AccountRecord: 永続化済み Account root snapshot。
//   - error: duplicate email、audit not found、または永続化層利用不能を表す application error。
//
// 使用例:
//
//	repo := admin.NewAccountRepository(db)
//	created, err := repo.CreateAccountWithAuditTarget(ctx, record)
//	_ = created
//	_ = err
type AccountRepository struct {
	db *gorm.DB
}

type accountRecord struct {
	ID                  string     `gorm:"column:id;primaryKey"`
	Email               string     `gorm:"column:email"`
	Status              string     `gorm:"column:status"`
	SessionRevokedAfter *time.Time `gorm:"column:session_revoked_after"`
	CreatedAt           time.Time  `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt           time.Time  `gorm:"column:updated_at;autoUpdateTime"`
}

type accountSummaryRecord struct {
	ID           string    `gorm:"column:id;primaryKey"`
	Email        string    `gorm:"column:email"`
	Status       string    `gorm:"column:status"`
	CreatedAt    time.Time `gorm:"column:created_at"`
	PasskeyCount int64     `gorm:"column:passkey_count"`
}

type accountSettingRecord struct {
	AccountID string    `gorm:"column:account_id;primaryKey"`
	Locale    string    `gorm:"column:locale"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

type auditTargetRecord struct {
	ID                 string    `gorm:"column:id;primaryKey"`
	TargetAccountID    string    `gorm:"column:target_account_id"`
	TargetAccountEmail string    `gorm:"column:target_account_email"`
	Outcome            string    `gorm:"column:outcome"`
	StableErrorCode    string    `gorm:"column:stable_error_code"`
	CompletedAt        time.Time `gorm:"column:completed_at"`
}

func (accountRecord) TableName() string {
	// Step 1: Admin adapter から Product Account root を触る経路は public schema を明示し、search_path に依存しない。
	return "public.accounts"
}

func (accountSummaryRecord) TableName() string {
	// Step 1: Admin read route は最小権限 role に SELECT 許可済みの view だけを読み、public.accounts 直 SELECT を避ける。
	return "admin_view.account_summaries"
}

func (accountSettingRecord) TableName() string {
	// Step 1: AccountSetting も Product Account root の child として public schema を明示し、Admin schema へ複製しない。
	return "public.account_settings"
}

func (auditTargetRecord) TableName() string {
	// Step 1: audit target correlation は Admin-owned schema に閉じ、Product table 側へ監査列を追加しない。
	return "admin.audit_events"
}

// NewAccountRepository は Admin account repository を構築する。
//
// db は runtime composition で接続・ping・role validation 済みの GORM handle を渡す。
// nil handle の検出は CreateAccountWithAuditTarget でも fail-closed に行い、構築だけでは外部 I/O を発生させない。
func NewAccountRepository(db *gorm.DB) *AccountRepository {
	// Step 1: DB handle を保持し、接続検証や migration 適用は runtime / deployment 境界に分離する。
	return &AccountRepository{db: db}
}

// CreateAccountWithAuditTarget は Product Account root 作成と Admin audit target / success outcome 関連付けを 1 transaction で実行する。
//
// ctx は transaction 全体に deadline/cancellation を伝播する。
// record.Account は application が domain constructor を通して構築した Account root であり、repository はその snapshot を保存するだけである。
// record.AuditID は mutation 前に作成済みの pending audit event ID で、存在しない場合は account 作成全体を rollback する。
// record.AuditCompletion は AuditService が domain.AdminAuditEvent で作った success outcome で、account 作成 commit と同じ transaction で保存する。
func (r *AccountRepository) CreateAccountWithAuditTarget(ctx context.Context, record accountsapplication.AccountCreationRecord) (accountsapplication.AccountRecord, error) {
	// Step 1: repository または DB handle が欠けている場合、transaction を開始せず application error に畳む。
	if r == nil || r.db == nil {
		return accountsapplication.AccountRecord{}, accountsapplication.ErrAccountRepositoryUnavailable
	}

	// Step 2: 戻り値は transaction closure 内でだけ設定し、rollback 時に未保存 snapshot が返らないようにする。
	var created accountsapplication.AccountRecord

	// Step 3: Product Account root と Admin audit target/outcome を同じ commit/rollback 境界に置き、監査相関なしの account 作成を防ぐ。
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		stored, err := createAccountRoot(ctx, tx, record)
		if err != nil {
			return err
		}

		if err := bindAuditTarget(ctx, tx, record); err != nil {
			return err
		}

		created = stored
		return nil
	})
	if err != nil {
		return accountsapplication.AccountRecord{}, err
	}

	// Step 4: transaction commit 後の snapshot だけを application へ返す。
	return created, nil
}

// SearchAccounts は Admin account search 用の検証済み query を GORM parameter binding で実行する。
//
// ctx は query 全体に deadline/cancellation を伝播する。
// query は application use case が検証済みの email/cursor/limit だけを含み、repository では範囲外 limit の補正を行わない。
// 成功時は Product Account 要約 read model と次ページ cursor を返し、DB failure は ErrAccountRepositoryUnavailable に畳む。
func (r *AccountRepository) SearchAccounts(ctx context.Context, query accountsapplication.AccountSearchQuery) (accountsapplication.AccountSearchRepositoryResult, error) {
	// Step 1: repository または DB handle が欠けている場合、検索 query を開始せず application error に畳む。
	if r == nil || r.db == nil {
		return accountsapplication.AccountSearchRepositoryResult{}, accountsapplication.ErrAccountRepositoryUnavailable
	}

	// Step 2: Admin read model view だけを対象にし、最小権限 DB role で public.accounts 全列 SELECT を要求しない。
	queryBuilder := r.db.WithContext(ctx).Model(&accountSummaryRecord{})

	// Step 3: email 検索は SQL 文字列へ連結せず、LIKE pattern を bound parameter として渡す。
	if query.Email != "" {
		queryBuilder = queryBuilder.Where("email ILIKE ?", accountEmailSearchPattern(query.Email))
	}

	// Step 4: cursor も opaque value のまま bound parameter として渡し、ID 降順の keyset boundary として扱う。
	if query.Cursor != "" {
		queryBuilder = queryBuilder.Where("id < ?", query.Cursor)
	}

	// Step 5: 次ページ有無を判定するため limit+1 件だけ読み、ORDER 句は cursor 比較と同じ ID 降順に固定する。
	var records []accountSummaryRecord
	if err := queryBuilder.Order("id DESC").Limit(int(query.Limit + 1)).Find(&records).Error; err != nil {
		return accountsapplication.AccountSearchRepositoryResult{}, accountsapplication.ErrAccountRepositoryUnavailable
	}

	// Step 6: query limit を超過した 1 件は next cursor へ変換し、response の account 一覧には含めない。
	return accountSearchRecordsToApplicationResult(records, query.Limit), nil
}

// FindAccountByID は Admin account detail 用の read model を 1 件取得する。
//
// ctx は query 全体に deadline/cancellation を伝播する。
// accountID は generated route binding から渡された Product Account ID で、repository では SQL parameter としてだけ扱う。
// 成功時は admin_view.account_summaries の snapshot を返し、対象不在は ErrAccountSearchNotFound に畳む。
func (r *AccountRepository) FindAccountByID(ctx context.Context, accountID string) (accountsapplication.AccountSummaryRecord, error) {
	// Step 1: repository または DB handle が欠けている場合、detail query を開始せず application error に畳む。
	if r == nil || r.db == nil {
		return accountsapplication.AccountSummaryRecord{}, accountsapplication.ErrAccountRepositoryUnavailable
	}

	// Step 2: 許可済み Admin view を ID で引き、Product base table の全列権限に依存しない。
	var record accountSummaryRecord
	if err := r.db.WithContext(ctx).Where("id = ?", accountID).First(&record).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return accountsapplication.AccountSummaryRecord{}, accountsapplication.ErrAccountSearchNotFound
		}
		return accountsapplication.AccountSummaryRecord{}, accountsapplication.ErrAccountRepositoryUnavailable
	}

	// Step 3: GORM record を application DTO へ変換し、adapter 型や column tag を外へ漏らさない。
	return accountSummaryRecordToApplicationRecord(record), nil
}

func createAccountRoot(ctx context.Context, tx *gorm.DB, record accountsapplication.AccountCreationRecord) (accountsapplication.AccountRecord, error) {
	// Step 1: canonical email の重複を先に確認し、通常の duplicate path を application が 409 に写像できる error にする。
	duplicate, err := accountEmailExists(ctx, tx, record.Account.Email().String())
	if err != nil {
		return accountsapplication.AccountRecord{}, err
	}
	if duplicate {
		return accountsapplication.AccountRecord{}, accountsapplication.ErrAccountDuplicateEmail
	}

	// Step 2: domain.Account の snapshot から DB record を作り、email/status/locale の業務判断は repository 内で行わない。
	account := accountRecordFromDomain(record)
	if err := tx.WithContext(ctx).Create(&account).Error; err != nil {
		return accountsapplication.AccountRecord{}, mapAccountMutationError(err)
	}

	// Step 3: migration trigger が作成した AccountSetting を domain snapshot の locale に揃え、root child も同じ transaction に含める。
	if err := updateAccountSetting(ctx, tx, record); err != nil {
		return accountsapplication.AccountRecord{}, err
	}

	// Step 4: 保存済み account record と domain snapshot から application DTO を作る。
	return accountRecordToApplicationRecord(account, record), nil
}

func accountEmailExists(ctx context.Context, tx *gorm.DB, email string) (bool, error) {
	// Step 1: public.accounts の canonical email unique constraint と同じ列で存在確認し、Product repository へ依存しない。
	var count int64
	if err := tx.WithContext(ctx).Model(&accountRecord{}).Where("email = ?", email).Count(&count).Error; err != nil {
		return false, accountsapplication.ErrAccountRepositoryUnavailable
	}

	// Step 2: 1 件以上あれば duplicate とし、race は insert 時の unique violation mapping で再度畳む。
	return count > 0, nil
}

func accountRecordFromDomain(record accountsapplication.AccountCreationRecord) accountRecord {
	// Step 1: Account root の各値は domain object から取り出し、repository 側で trim/lowercase/status 初期値を作らない。
	return accountRecord{
		ID:                  record.Account.ID().String(),
		Email:               record.Account.Email().String(),
		Status:              record.Account.Status().String(),
		SessionRevokedAfter: record.Account.SessionRevokedAfter(),
	}
}

func updateAccountSetting(ctx context.Context, tx *gorm.DB, record accountsapplication.AccountCreationRecord) error {
	// Step 1: 000002 migration の trigger が作成した child row を、domain AccountSetting の locale snapshot へ更新する。
	result := tx.WithContext(ctx).Model(&accountSettingRecord{}).
		Where("account_id = ?", record.Account.ID().String()).
		Update("locale", record.Account.Setting().Locale().String())
	if result.Error != nil {
		return accountsapplication.ErrAccountRepositoryUnavailable
	}

	// Step 2: child row が存在しない場合は migration/trigger 不整合として rollback し、部分的な Account root を残さない。
	if result.RowsAffected == 0 {
		return accountsapplication.ErrAccountRepositoryUnavailable
	}

	// Step 3: AccountSetting の保存境界が満たされたことを nil で返す。
	return nil
}

func bindAuditTarget(ctx context.Context, tx *gorm.DB, record accountsapplication.AccountCreationRecord) error {
	// Step 1: audit ID は空白だけの入力を拒否し、監査 intent なしの account mutation を rollback する。
	auditID := strings.TrimSpace(record.AuditID)
	if auditID == "" {
		return accountsapplication.ErrAccountAuditNotFound
	}

	// Step 2: completion 側の audit ID と intent ID が一致しない場合は、監査相関が壊れているため rollback する。
	completionAuditID := strings.TrimSpace(record.AuditCompletion.AuditID)
	if completionAuditID != auditID {
		return accountsapplication.ErrAccountAuditNotFound
	}

	// Step 3: admin.audit_events の target 欄と success outcome を同時に更新し、account 作成成功と監査完了の atomicity を保つ。
	result := tx.WithContext(ctx).Model(&auditTargetRecord{}).
		Where("id = ? AND outcome = ?", auditID, "pending").
		Updates(map[string]any{
			"target_account_id":    record.Account.ID().String(),
			"target_account_email": record.Account.Email().String(),
			"outcome":              record.AuditCompletion.Outcome,
			"stable_error_code":    record.AuditCompletion.StableErrorCode,
			"completed_at":         record.AuditCompletion.CompletedAt,
		})
	if result.Error != nil {
		return accountsapplication.ErrAccountRepositoryUnavailable
	}

	// Step 4: pending audit intent が存在しない場合は Product Account 作成も rollback し、監査なし mutation を禁止する。
	if result.RowsAffected == 0 {
		return accountsapplication.ErrAccountAuditNotFound
	}

	// Step 5: Admin schema 側の target correlation と success outcome が同じ transaction 内で完了したことを nil で返す。
	return nil
}

func accountRecordToApplicationRecord(account accountRecord, record accountsapplication.AccountCreationRecord) accountsapplication.AccountRecord {
	// Step 1: application DTO は DB column tag を持たない primitive snapshot として組み立てる。
	return accountsapplication.AccountRecord{
		AccountID:           account.ID,
		Email:               account.Email,
		Status:              account.Status,
		Locale:              record.Account.Setting().Locale().String(),
		SessionRevokedAfter: account.SessionRevokedAfter,
		CreatedAt:           account.CreatedAt,
	}
}

func mapAccountMutationError(err error) error {
	// Step 1: GORM が duplicate key を抽象 error として返す構成では、Admin API の 409 用 error に畳む。
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return accountsapplication.ErrAccountDuplicateEmail
	}

	// Step 2: それ以外の DB error は permission denied や接続断を含めて永続化層利用不能として扱う。
	return accountsapplication.ErrAccountRepositoryUnavailable
}

func accountEmailSearchPattern(email string) string {
	// Step 1: LIKE wildcard と escape 記号を literal として扱い、operator 入力が検索範囲を意図せず広げないようにする。
	escaped := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(email)

	// Step 2: SQL fragment ではなく bound parameter の値として部分一致 pattern を組み立てる。
	return "%" + escaped + "%"
}

func accountSearchRecordsToApplicationResult(records []accountSummaryRecord, limit int32) accountsapplication.AccountSearchRepositoryResult {
	// Step 1: limit+1 件を受け取った場合は最後の 1 件を next cursor 用に退避し、返却件数を limit 以下に保つ。
	visibleRecords := records
	nextCursor := ""
	limitIndex := int(limit)
	if len(records) > limitIndex {
		nextCursor = records[limitIndex-1].ID
		visibleRecords = records[:limitIndex]
	}

	// Step 2: GORM record から application read model へ値コピーし、DB tag や adapter 型を application 境界へ漏らさない。
	accounts := make([]accountsapplication.AccountSummaryRecord, 0, len(visibleRecords))
	for _, record := range visibleRecords {
		accounts = append(accounts, accountSummaryRecordToApplicationRecord(record))
	}

	// Step 3: opaque cursor と account 要約だけを返し、HTTP request correlation は application service に残す。
	return accountsapplication.AccountSearchRepositoryResult{Accounts: accounts, NextCursor: nextCursor}
}

func accountSummaryRecordToApplicationRecord(record accountSummaryRecord) accountsapplication.AccountSummaryRecord {
	// Step 1: view の bigint passkey_count を API contract の int32 へ収め、負値は view 不整合として 0 に丸める。
	const maxInt32 = int64(1<<31 - 1)
	passkeyCount := int32(0)
	if record.PasskeyCount > 0 {
		if record.PasskeyCount > maxInt32 {
			passkeyCount = int32(maxInt32)
		} else {
			passkeyCount = int32(record.PasskeyCount)
		}
	}

	// Step 2: application DTO は primitive snapshot だけにし、GORM record を application 境界へ漏らさない。
	return accountsapplication.AccountSummaryRecord{AccountID: record.ID, Email: record.Email, Status: record.Status, PasskeyCount: passkeyCount, CreatedAt: record.CreatedAt}
}
