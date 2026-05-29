package admin

import (
	"context"
	"database/sql"
	"time"

	"gorm.io/gorm"

	adminauth "www-template/packages/backend/internal/application/admin/auth"
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
