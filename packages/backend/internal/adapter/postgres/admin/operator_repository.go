package admin

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"gorm.io/gorm"

	adminauth "www-template/packages/backend/internal/application/auth"
	operatorsapplication "www-template/packages/backend/internal/application/operators"
	domain "www-template/packages/backend/internal/domain"
)

var _ operatorsapplication.OperatorRepository = (*OperatorRepository)(nil)

// OperatorRepository は Admin operator auth の Operator snapshot を PostgreSQL から復元する adapter である。
//
// 役割:
//   - admin.operators と admin.operator_passkeys だけを読み、Product account repository へ依存しない。
//   - adminauth.OperatorRepository port を実装し、Admin application DTO だけを返す。
//   - GORM record 型を package 内に閉じ込め、application/domain へ adapter 型を公開しない。
//
// 引数:
//   - NewOperatorRepository の db: Admin schema へ接続可能な GORM DB handle。
//   - FindOperatorByCredential の credentialHandle: WebAuthn 検証済み Admin operator credential handle。
//   - FindOperatorByID の operatorID: Admin OperatorID の文字列表現。
//
// 戻り値:
//   - adminauth.OperatorSnapshot: Admin auth use case が domain.Operator を復元するための snapshot。
//   - error: レコード不在または DB 失敗時の error。
//
// 使用例:
//
//	repo := admin.NewOperatorRepository(db)
//	snapshot, err := repo.FindOperatorByID(ctx, operatorID)
type OperatorRepository struct {
	db *gorm.DB
}

type operatorRecord struct {
	ID                       string     `gorm:"column:id;primaryKey"`
	Email                    string     `gorm:"column:email"`
	Role                     string     `gorm:"column:role"`
	Active                   bool       `gorm:"column:active"`
	PasskeyRegistrationState string     `gorm:"column:passkey_registration_state"`
	SetupTokenHash           *string    `gorm:"column:setup_token_hash"`
	SetupTokenExpiresAt      *time.Time `gorm:"column:setup_token_expires_at"`
	SetupTokenConsumedAt     *time.Time `gorm:"column:setup_token_consumed_at"`
	CreatedAt                time.Time  `gorm:"column:created_at"`
	UpdatedAt                time.Time  `gorm:"column:updated_at"`
}

type operatorPasskeyRecord struct {
	ID               string    `gorm:"column:id;primaryKey"`
	OperatorID       string    `gorm:"column:operator_id"`
	CredentialHandle string    `gorm:"column:credential_handle"`
	PublicKey        []byte    `gorm:"column:public_key"`
	SignCount        int64     `gorm:"column:sign_count"`
	AAGUID           []byte    `gorm:"column:aaguid"`
	BackupEligible   bool      `gorm:"column:backup_eligible"`
	BackupState      bool      `gorm:"column:backup_state"`
	Transports       string    `gorm:"column:transports;type:jsonb"`
	CreatedAt        time.Time `gorm:"column:created_at"`
	UpdatedAt        time.Time `gorm:"column:updated_at"`
}

func (operatorRecord) TableName() string {
	return "admin.operators"
}

func (operatorPasskeyRecord) TableName() string {
	return "admin.operator_passkeys"
}

// NewOperatorRepository は Admin operator repository を構築する。
func NewOperatorRepository(db *gorm.DB) *OperatorRepository {
	// Step 1: DB handle を保持し、接続検証は runtime composition の責務として分離する。
	return &OperatorRepository{db: db}
}

// FindOperatorByCredential は credential handle に紐づく Admin operator snapshot を返す。
func (r *OperatorRepository) FindOperatorByCredential(ctx context.Context, credentialHandle string) (adminauth.OperatorSnapshot, error) {
	// Step 1: Admin operator passkey table から credential owner を解決し、Product passkey table を参照しない境界を保つ。
	var passkey operatorPasskeyRecord
	if err := r.db.WithContext(ctx).Where("credential_handle = ?", credentialHandle).First(&passkey).Error; err != nil {
		return adminauth.OperatorSnapshot{}, mapOperatorRepositoryError(err)
	}

	// Step 2: operator ID 経由の復元へ委譲し、snapshot mapping を一箇所に集約する。
	return r.FindOperatorByID(ctx, passkey.OperatorID)
}

// FindOperatorByID は operator ID に対応する Admin operator snapshot を返す。
func (r *OperatorRepository) FindOperatorByID(ctx context.Context, operatorID string) (adminauth.OperatorSnapshot, error) {
	// Step 1: admin.operators だけを検索し、Product account schema への逆流を避ける。
	var operator operatorRecord
	if err := r.db.WithContext(ctx).Where("id = ?", operatorID).First(&operator).Error; err != nil {
		return adminauth.OperatorSnapshot{}, mapOperatorRepositoryError(err)
	}

	// Step 2: application DTO へ写像し、GORM record を application 境界へ出さない。
	return operator.toSnapshot(), nil
}

// CountOperators は Admin schema に存在する operator 件数を返す。
func (r *OperatorRepository) CountOperators(ctx context.Context) (int64, error) {
	// Step 1: admin.operators だけを数え、Product account 数を初回 Admin setup の判定に使わない。
	var count int64
	if err := r.db.WithContext(ctx).Model(&operatorRecord{}).Count(&count).Error; err != nil {
		return 0, operatorsapplication.ErrOperatorInternal
	}
	return count, nil
}

// CreateInitialAdminOperatorWithPasskey は初回 admin operator と passkey credential を同一 transaction で作成する。
func (r *OperatorRepository) CreateInitialAdminOperatorWithPasskey(ctx context.Context, record operatorsapplication.InitialOperatorRecord) (operatorsapplication.OperatorRecord, error) {
	// Step 1: SERIALIZABLE transaction で operator 0 件確認、operator 作成、passkey 保存を一括し、並行 bootstrap を拒否する。
	var created operatorRecord
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var count int64
		if err := tx.WithContext(ctx).Model(&operatorRecord{}).Count(&count).Error; err != nil {
			return operatorsapplication.ErrOperatorInternal
		}
		if count != 0 {
			return operatorsapplication.ErrOperatorConflict
		}

		created = operatorRecord{ID: record.OperatorID, Email: record.Email, Role: string(domain.OperatorRoleAdmin), Active: true, PasskeyRegistrationState: string(domain.OperatorPasskeyRegistrationRegistered), CreatedAt: record.CompletedAt.UTC(), UpdatedAt: record.CompletedAt.UTC()}
		if err := tx.WithContext(ctx).Create(&created).Error; err != nil {
			return operatorsapplication.ErrOperatorInternal
		}
		return createOperatorPasskeyRecord(ctx, tx, record.OperatorID, record.Passkey, record.CompletedAt)
	}, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return operatorsapplication.OperatorRecord{}, err
	}
	return adminOperatorRecordFromGORM(created), nil
}

// CreateOperatorWithSetupToken は追加 operator と setup token hash を作成し、setup token の配送前は audit outcome を pending に保つ。
func (r *OperatorRepository) CreateOperatorWithSetupToken(ctx context.Context, record operatorsapplication.OperatorCreationRecord) (operatorsapplication.OperatorRecord, error) {
	// Step 1: canonical email の重複を先に確認し、通常の duplicate path を application が 409 に写像できる error にする。
	duplicate, err := r.operatorEmailExists(ctx, record.Email)
	if err != nil {
		return operatorsapplication.OperatorRecord{}, err
	}
	if duplicate {
		return operatorsapplication.OperatorRecord{}, operatorsapplication.ErrOperatorConflict
	}

	// Step 2: setup token hash だけを保存し、平文 token は repository 境界に入れない。
	now := record.CreatedAt.UTC()
	created := operatorRecord{ID: record.OperatorID, Email: record.Email, Role: record.Role, Active: true, PasskeyRegistrationState: string(domain.OperatorPasskeyRegistrationPending), SetupTokenHash: &record.SetupTokenHash, SetupTokenExpiresAt: &record.SetupTokenExpiresAt, CreatedAt: now, UpdatedAt: now}
	if err := r.db.WithContext(ctx).Create(&created).Error; err != nil {
		return operatorsapplication.OperatorRecord{}, mapOperatorMutationError(err)
	}
	return adminOperatorRecordFromGORM(created), nil
}

func (r *OperatorRepository) operatorEmailExists(ctx context.Context, email string) (bool, error) {
	// Step 1: admin.operators の canonical email unique key だけを参照し、Product account email と混同しない。
	var count int64
	if err := r.db.WithContext(ctx).Model(&operatorRecord{}).Where("email = ?", email).Count(&count).Error; err != nil {
		return false, operatorsapplication.ErrOperatorInternal
	}

	// Step 2: 1 件以上あれば duplicate とし、race は insert 時の unique violation mapping で再度畳む。
	return count > 0, nil
}

// DeletePendingOperatorSetup は delivery/audit completion 失敗時に pending operator を削除し、配送済みまたは未配送 token の後続利用を防ぐ。
func (r *OperatorRepository) DeletePendingOperatorSetup(ctx context.Context, operatorID string) error {
	// Step 1: setup 未完了かつ passkey 未登録の operator だけを対象にし、登録済み operator を誤って削除しない。
	result := r.db.WithContext(ctx).Where("id = ? AND passkey_registration_state = ?", operatorID, string(domain.OperatorPasskeyRegistrationPending)).Delete(&operatorRecord{})
	if result.Error != nil || result.RowsAffected == 0 {
		return operatorsapplication.ErrOperatorInternal
	}
	return nil
}

// FindOperatorBySetupToken は有効な pending setup token を opaque hash callback で照合し、token 状態を外部へ区別させない。
func (r *OperatorRepository) FindOperatorBySetupToken(ctx context.Context, now time.Time, match func(hash string) bool) (operatorsapplication.SetupRecord, error) {
	// Step 1: token 未消費・未期限切れ・passkey pending の候補だけを取得し、opaque hash は Go 側の constant-time 実装へ渡す。
	var records []operatorRecord
	if err := r.db.WithContext(ctx).
		Where("active = ? AND passkey_registration_state = ? AND setup_token_hash IS NOT NULL AND setup_token_expires_at > ? AND setup_token_consumed_at IS NULL", true, string(domain.OperatorPasskeyRegistrationPending), now.UTC()).
		Order("created_at ASC, id ASC").
		Find(&records).Error; err != nil {
		return operatorsapplication.SetupRecord{}, operatorsapplication.ErrOperatorInternal
	}
	for _, record := range records {
		if record.SetupTokenHash != nil && match(*record.SetupTokenHash) {
			return operatorsapplication.SetupRecord{OperatorID: record.ID, Email: record.Email, Role: record.Role, Active: record.Active}, nil
		}
	}
	return operatorsapplication.SetupRecord{}, operatorsapplication.ErrOperatorForbidden
}

// CompleteOperatorSetupWithPasskey は setup token を消費し、初回 passkey を保存して operator を登録済みにする。
func (r *OperatorRepository) CompleteOperatorSetupWithPasskey(ctx context.Context, record operatorsapplication.SetupCompletionRecord) (operatorsapplication.OperatorRecord, error) {
	// Step 1: SERIALIZABLE transaction で token 未消費確認、passkey 件数確認、token 消費、credential 保存を一括する。
	var updated operatorRecord
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.WithContext(ctx).Where("id = ? AND active = ? AND passkey_registration_state = ? AND setup_token_hash IS NOT NULL AND setup_token_expires_at > ? AND setup_token_consumed_at IS NULL", record.OperatorID, true, string(domain.OperatorPasskeyRegistrationPending), record.CompletedAt.UTC()).First(&updated).Error; err != nil {
			return mapOperatorSetupRepositoryError(err)
		}
		if updated.SetupTokenHash == nil || record.SetupTokenMatches == nil || !record.SetupTokenMatches(*updated.SetupTokenHash) {
			return operatorsapplication.ErrOperatorForbidden
		}
		var passkeyCount int64
		if err := tx.WithContext(ctx).Model(&operatorPasskeyRecord{}).Where("operator_id = ?", record.OperatorID).Count(&passkeyCount).Error; err != nil {
			return operatorsapplication.ErrOperatorInternal
		}
		if passkeyCount != 0 {
			return operatorsapplication.ErrOperatorForbidden
		}

		updates := map[string]any{"passkey_registration_state": string(domain.OperatorPasskeyRegistrationRegistered), "setup_token_hash": nil, "setup_token_expires_at": nil, "setup_token_consumed_at": record.CompletedAt.UTC(), "updated_at": record.CompletedAt.UTC()}
		if err := tx.WithContext(ctx).Model(&operatorRecord{}).Where("id = ?", record.OperatorID).Updates(updates).Error; err != nil {
			return operatorsapplication.ErrOperatorInternal
		}
		if err := createOperatorPasskeyRecord(ctx, tx, record.OperatorID, record.Passkey, record.CompletedAt); err != nil {
			return err
		}
		return tx.WithContext(ctx).Where("id = ?", record.OperatorID).First(&updated).Error
	}, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return operatorsapplication.OperatorRecord{}, err
	}
	return adminOperatorRecordFromGORM(updated), nil
}

func (r operatorRecord) toSnapshot() adminauth.OperatorSnapshot {
	// Step 1: Admin auth use case が必要とする primitive だけを DTO として返す。
	return adminauth.OperatorSnapshot{ID: r.ID, Email: r.Email, Role: r.Role, Active: r.Active, PasskeyRegistrationState: r.PasskeyRegistrationState}
}

func createOperatorPasskeyRecord(ctx context.Context, tx *gorm.DB, operatorID string, passkey operatorsapplication.PasskeyRecord, now time.Time) error {
	// Step 1: transports は JSONB へ保存するため、検証済み string slice だけを JSON に変換する。
	transports, err := json.Marshal(passkey.Transports)
	if err != nil {
		return operatorsapplication.ErrOperatorInternal
	}
	passkeyRecord := operatorPasskeyRecord{ID: passkey.CredentialID, OperatorID: operatorID, CredentialHandle: passkey.CredentialHandle, PublicKey: passkey.PublicKey, SignCount: int64(passkey.SignCount), AAGUID: passkey.AAGUID, BackupEligible: passkey.BackupEligible, BackupState: passkey.BackupState, Transports: string(transports), CreatedAt: now.UTC(), UpdatedAt: now.UTC()}
	if err := tx.WithContext(ctx).Create(&passkeyRecord).Error; err != nil {
		return operatorsapplication.ErrOperatorInternal
	}
	return nil
}

func adminOperatorRecordFromGORM(record operatorRecord) operatorsapplication.OperatorRecord {
	// Step 1: GORM record から application DTO へ primitive 値だけを写像し、DB tag を application 層へ漏らさない。
	return operatorsapplication.OperatorRecord{OperatorID: record.ID, Email: record.Email, Role: record.Role, Active: record.Active, PasskeyRegistrationState: record.PasskeyRegistrationState, CreatedAt: record.CreatedAt}
}

func mapOperatorSetupRepositoryError(err error) error {
	// Step 1: not found は token invalid/expired/consumed と同じ non-revealing forbidden に畳む。
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return operatorsapplication.ErrOperatorForbidden
	}
	return operatorsapplication.ErrOperatorInternal
}

func mapOperatorRepositoryError(err error) error {
	// Step 1: GORM の not found は application 側で認証失敗へ畳める stable error に変換する。
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return domain.ErrSessionNotFound
	}

	// Step 2: それ以外の DB error も adapter 外へ GORM 型を公開しないよう、保存層利用不能に畳む。
	return domain.ErrAuthStoreUnavailable
}

func mapOperatorMutationError(err error) error {
	// Step 1: GORM が duplicate key を抽象 error として返す構成では、Admin API の 409 用 error に畳む。
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return operatorsapplication.ErrOperatorConflict
	}

	// Step 2: それ以外の DB error は adapter 外へ GORM 型を公開しないよう、保存層利用不能に畳む。
	return operatorsapplication.ErrOperatorInternal
}
