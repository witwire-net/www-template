package admin

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"gorm.io/gorm"

	adminauth "www-template/packages/backend/internal/application/auth"
	domain "www-template/packages/backend/internal/domain"
)

// OperatorPasskeyRepository は Admin operator passkey credential を PostgreSQL から読み書きする adapter である。
//
// 役割:
//   - admin.operator_passkeys だけを扱い、Product account_passkey_credentials へ依存しない。
//   - application には非秘匿 DTO だけを返し、credential_handle、public_key、sign_count は一覧 response に漏らさない。
//   - 削除時は operator_id と credential id の両方で絞り、最後の 1 件削除を repository 側でも防ぐ。
type OperatorPasskeyRepository struct {
	db *gorm.DB
}

// FindWebAuthnCredential は credentialHandle から Admin operator WebAuthn stored credential を復元する。
//
// credentialHandle は WebAuthn provider が raw ID から導出した base64url handle である。
// 戻り値は署名検証に必要な public key / sign count / backup state だけを含み、HTTP response へは返さない。
func (r *OperatorPasskeyRepository) FindWebAuthnCredential(ctx context.Context, handle string) (domain.WebAuthnStoredCredential, error) {
	// Step 1: Admin operator_passkeys だけを credential_handle で検索し、Product credential table を参照しない。
	var record operatorPasskeyRecord
	if err := r.db.WithContext(ctx).Where("credential_handle = ?", handle).First(&record).Error; err != nil {
		return domain.ZeroWebAuthnStoredCredential(), domain.ErrAccountAuthNotFound
	}

	// Step 2: JSONB transports を provider 用 primitive slice へ戻し、壊れた保存値は store unavailable として fail-close にする。
	transports, err := operatorPasskeyTransports(record.Transports)
	if err != nil {
		return domain.ZeroWebAuthnStoredCredential(), domain.ErrAuthStoreUnavailable
	}
	signCount, err := operatorPasskeySignCount(record.SignCount)
	if err != nil {
		return domain.ZeroWebAuthnStoredCredential(), domain.ErrAuthStoreUnavailable
	}

	// Step 3: WebAuthn provider が署名検証に使う domain DTO へ写像する。
	return domain.ReconstitueWebAuthnStoredCredential(record.CredentialHandle, record.PublicKey, signCount, record.AAGUID, record.BackupEligible, record.BackupState, transports), nil
}

// UpdateWebAuthnCredentialState は passkey login 成功後の sign count と backup state を保存する。
//
// handle は検証済み credential handle、newSignCount / newBackupState は WebAuthn provider が assertion から返した最新状態である。
// 保存失敗または対象不在の場合は認証 store 障害として扱い、古い replay 検出状態を残した成功応答を防ぐ。
func (r *OperatorPasskeyRepository) UpdateWebAuthnCredentialState(ctx context.Context, handle string, newSignCount uint32, newBackupState bool) error {
	// Step 1: credential_handle で対象 Admin passkey だけを更新し、operator owner は credential handle の一意性に委ねる。
	result := r.db.WithContext(ctx).Model(&operatorPasskeyRecord{}).
		Where("credential_handle = ?", handle).
		Updates(map[string]any{"sign_count": newSignCount, "backup_state": newBackupState})
	if result.Error != nil {
		return domain.ErrAuthStoreUnavailable
	}
	if result.RowsAffected == 0 {
		return domain.ErrAccountAuthNotFound
	}

	// Step 2: WebAuthn credential state の更新が完了したため成功とする。
	return nil
}

type operatorPasskeyListRecord struct {
	ID         string     `gorm:"column:id;primaryKey"`
	OperatorID string     `gorm:"column:operator_id"`
	CreatedAt  time.Time  `gorm:"column:created_at"`
	LastUsedAt *time.Time `gorm:"column:last_used_at"`
}

func (operatorPasskeyListRecord) TableName() string {
	return "admin.operator_passkeys"
}

// NewOperatorPasskeyRepository は Admin operator passkey repository を構築する。
//
// db は Admin schema へ接続できる GORM handle であり、nil 検証や接続検証は runtime composition が行う。
func NewOperatorPasskeyRepository(db *gorm.DB) *OperatorPasskeyRepository {
	// Step 1: DB handle を保持し、repository method ごとに context 付き query として使う。
	return &OperatorPasskeyRepository{db: db}
}

// ListOperatorPasskeys は operatorID に紐づく Admin operator passkey credential の非秘匿一覧を返す。
func (r *OperatorPasskeyRepository) ListOperatorPasskeys(ctx context.Context, operatorID string) ([]adminauth.OperatorPasskeyCredential, error) {
	// Step 1: operator_id で所有者を限定し、Product credential table や他 Operator credential を読まない。
	var records []operatorPasskeyListRecord
	if err := r.db.WithContext(ctx).Where("operator_id = ?", operatorID).Order("created_at ASC, id ASC").Find(&records).Error; err != nil {
		return nil, domain.ErrAuthStoreUnavailable
	}

	// Step 2: GORM record から application DTO へ写像し、credential handle や public key を落として返す。
	passkeys := make([]adminauth.OperatorPasskeyCredential, 0, len(records))
	for _, record := range records {
		passkeys = append(passkeys, adminauth.OperatorPasskeyCredential{ID: record.ID, CreatedAt: record.CreatedAt, LastUsedAt: record.LastUsedAt})
	}
	return passkeys, nil
}

// DeleteOperatorPasskey は operatorID と passkeyID が一致する Admin operator passkey credential を削除する。
func (r *OperatorPasskeyRepository) DeleteOperatorPasskey(ctx context.Context, operatorID string, passkeyID string) error {
	// Step 1: SERIALIZABLE transaction 内で件数判定と削除を実行し、並行削除による write skew で最後の 1 件が消えないようにする。
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		passkeys, err := r.listOperatorPasskeysForDeletion(ctx, tx, operatorID)
		if err != nil {
			return err
		}
		if err := domain.EnsureOperatorPasskeyDeletionAllowed(len(passkeys)); err != nil {
			return err
		}

		// Step 2: credential id と operator_id の両方で削除し、path ID だけによる越権削除を防ぐ。
		result := tx.WithContext(ctx).Where("id = ? AND operator_id = ?", passkeyID, operatorID).Delete(&operatorPasskeyListRecord{})
		if result.Error != nil {
			return domain.ErrAuthStoreUnavailable
		}
		if result.RowsAffected == 0 {
			return domain.ErrSessionNotFound
		}
		return nil
	}, &sql.TxOptions{Isolation: sql.LevelSerializable})
}

func (r *OperatorPasskeyRepository) listOperatorPasskeysForDeletion(ctx context.Context, tx *gorm.DB, operatorID string) ([]operatorPasskeyListRecord, error) {
	// Step 1: operator_id に限定した credential 行を SERIALIZABLE transaction 内で読み、同一 snapshot で削除可否を判定する。
	var records []operatorPasskeyListRecord
	if err := tx.WithContext(ctx).
		Where("operator_id = ?", operatorID).
		Order("created_at ASC, id ASC").
		Find(&records).Error; err != nil {
		return nil, domain.ErrAuthStoreUnavailable
	}

	// Step 2: transaction 内の credential 一覧を返し、呼び出し側が同じ isolation level で件数判定と削除を続けられるようにする。
	return records, nil
}

func operatorPasskeyTransports(raw string) ([]string, error) {
	// Step 1: 空 transports は保存時に未指定だった credential として扱い、nil slice を返す。
	if raw == "" {
		return nil, nil
	}

	// Step 2: JSONB 文字列を provider DTO 用の string slice に戻す。
	var transports []string
	if err := json.Unmarshal([]byte(raw), &transports); err != nil {
		return nil, err
	}
	return transports, nil
}

func operatorPasskeySignCount(raw int64) (uint32, error) {
	// Step 1: DB の int64 値を WebAuthn authenticator の uint32 sign count 範囲へ安全に収める。
	if raw < 0 || raw > int64(^uint32(0)) {
		return 0, domain.ErrAuthStoreUnavailable
	}

	// Step 2: 範囲検証済みの値だけを uint32 へ変換する。
	return uint32(raw), nil
}
