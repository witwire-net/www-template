package persistence

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"

	"www-template/packages/backend/internal/domain"
)

type gormAccountRecord struct {
	ID    string `gorm:"column:id;primaryKey"`
	Email string `gorm:"column:email"`
}

func (gormAccountRecord) TableName() string {
	return "accounts"
}

type gormPasskeyCredentialRecord struct {
	ID               string    `gorm:"column:id;primaryKey"`
	AccountID        string    `gorm:"column:account_id"`
	Identifier       string    `gorm:"column:identifier"`
	CredentialHandle string    `gorm:"column:credential_handle"`
	CreatedAt        time.Time `gorm:"column:created_at;autoCreateTime"`
	// WebAuthn Credential Record fields (migration 000003).
	// Nullable to maintain compatibility with rows created before migration 000003.
	PublicKey      []byte   `gorm:"column:public_key"`
	SignCount      uint32   `gorm:"column:sign_count;default:0"`
	AAGUID         []byte   `gorm:"column:aaguid"`
	BackupEligible bool     `gorm:"column:backup_eligible;default:false"`
	BackupState    bool     `gorm:"column:backup_state;default:false"`
	Transports     []string `gorm:"column:transports;serializer:json"`
}

func (gormPasskeyCredentialRecord) TableName() string {
	return "passkey_credentials"
}

type GormAuthAccountRepository struct {
	db *gorm.DB
}

func NewGormAuthAccountRepository(db *gorm.DB) *GormAuthAccountRepository {
	return &GormAuthAccountRepository{db: db}
}

func (r *GormAuthAccountRepository) FindByIdentifier(ctx context.Context, identifier string) (domain.AuthAccount, error) {
	return r.findByPasskey(ctx, "identifier = ?", strings.TrimSpace(identifier))
}

func (r *GormAuthAccountRepository) FindByCredential(ctx context.Context, credential string) (domain.AuthAccount, error) {
	return r.findByPasskey(ctx, "credential_handle = ?", strings.TrimSpace(credential))
}

// FindByEmail は email でアカウントを検索し、最古の passkey credential（id ASC 先頭）を
// 含む AuthAccount を返す。複数パスキーがある場合も先頭 1 件を返す挙動を維持する。
func (r *GormAuthAccountRepository) FindByEmail(ctx context.Context, email string) (domain.AuthAccount, error) {
	var account gormAccountRecord
	if err := r.db.WithContext(ctx).Where("email = ?", strings.TrimSpace(email)).First(&account).Error; err != nil {
		return emptyAuthAccount(), mapAuthAccountError(err)
	}

	var passkey gormPasskeyCredentialRecord
	if err := r.db.WithContext(ctx).Where("account_id = ?", account.ID).Order("id ASC").First(&passkey).Error; err != nil {
		return emptyAuthAccount(), mapAuthAccountError(err)
	}

	return normalizeDomainAccount(account, passkey)
}

// ListPasskeys は accountID に紐づく全 passkey credential を返す。
// account が存在しない場合は domain.ErrAuthAccountNotFound を返す。
func (r *GormAuthAccountRepository) ListPasskeys(ctx context.Context, accountID string) ([]domain.PasskeyCredential, error) {
	// account の存在確認
	var account gormAccountRecord
	if err := r.db.WithContext(ctx).Where("id = ?", accountID).First(&account).Error; err != nil {
		return nil, mapAuthAccountError(err)
	}

	var records []gormPasskeyCredentialRecord
	if err := r.db.WithContext(ctx).Where("account_id = ?", accountID).Order("id ASC").Find(&records).Error; err != nil {
		return nil, domain.ErrAuthStoreUnavailable
	}
	credentials := make([]domain.PasskeyCredential, 0, len(records))
	for _, rec := range records {
		cred, err := domain.NewPasskeyCredential(rec.ID, rec.AccountID, rec.Identifier, rec.CredentialHandle, rec.CreatedAt)
		if err != nil {
			return nil, domain.ErrAuthStoreUnavailable
		}
		credentials = append(credentials, cred)
	}
	return credentials, nil
}

// AddPasskey は既存パスキーを削除せず 1 件追加する。
// credData に WebAuthn credential record のデータを渡す（provider なしの場合は zero value で可）。
// 返却する AuthAccount は追加前から存在する先頭 credential（id ASC）をベースに構築する。
// これは既存認証フロー（FindByCredential, FindByEmail）との一貫性を保つ意図仕様であり、
// session の passkeyCredentialId は「先頭 credential」を指す。
// 新規追加 credential を passkeyCredentialId として返したい場合は呼び出し側で ListPasskeys を呼ぶこと。
func (r *GormAuthAccountRepository) AddPasskey(ctx context.Context, accountID string, credentialID string, handle string, credData domain.WebAuthnCredentialData) (domain.AuthAccount, error) {
	var account gormAccountRecord
	if err := r.db.WithContext(ctx).Where("id = ?", accountID).First(&account).Error; err != nil {
		return emptyAuthAccount(), mapAuthAccountError(err)
	}

	// identifier は FindByEmail に準じ既存 passkey の identifier を流用
	var firstPasskey gormPasskeyCredentialRecord
	if err := r.db.WithContext(ctx).Where("account_id = ?", accountID).Order("id ASC").First(&firstPasskey).Error; err != nil {
		return emptyAuthAccount(), mapAuthAccountError(err)
	}

	newRecord := gormPasskeyCredentialRecord{
		ID:               credentialID,
		AccountID:        accountID,
		Identifier:       firstPasskey.Identifier,
		CredentialHandle: strings.TrimSpace(handle),
		CreatedAt:        time.Now().UTC(),
		PublicKey:        credData.PublicKey,
		SignCount:        credData.SignCount,
		AAGUID:           credData.AAGUID,
		BackupEligible:   credData.BackupEligible,
		BackupState:      credData.BackupState,
		Transports:       credData.Transports,
	}
	if err := r.db.WithContext(ctx).Create(&newRecord).Error; err != nil {
		return emptyAuthAccount(), domain.ErrAuthStoreUnavailable
	}

	return normalizeDomainAccount(account, firstPasskey)
}

// DeletePasskeyByID は account_id と id の両方で絞り込んで削除し、他アカウントの誤削除を防ぐ。
func (r *GormAuthAccountRepository) DeletePasskeyByID(ctx context.Context, accountID string, credentialID string) error {
	result := r.db.WithContext(ctx).
		Where("id = ? AND account_id = ?", credentialID, accountID).
		Delete(&gormPasskeyCredentialRecord{})
	if result.Error != nil {
		return domain.ErrAuthStoreUnavailable
	}
	if result.RowsAffected == 0 {
		return domain.ErrAuthAccountNotFound
	}
	return nil
}

// FindWebAuthnCredential は credentialHandle（base64url rawID）から WebAuthn stored credential を返す。
// FinishLogin 時の署名検証に必要な public key 等を提供する。
func (r *GormAuthAccountRepository) FindWebAuthnCredential(ctx context.Context, handle string) (domain.WebAuthnStoredCredential, error) {
	var rec gormPasskeyCredentialRecord
	if err := r.db.WithContext(ctx).Where("credential_handle = ?", strings.TrimSpace(handle)).First(&rec).Error; err != nil {
		return domain.ZeroWebAuthnStoredCredential(), mapAuthAccountError(err)
	}
	return domain.ReconstitueWebAuthnStoredCredential(
		rec.CredentialHandle,
		rec.PublicKey,
		rec.SignCount,
		rec.AAGUID,
		rec.BackupEligible,
		rec.BackupState,
		rec.Transports,
	), nil
}

// UpdateWebAuthnCredentialState は FinishLogin 成功後に credential の SignCount と BackupState を更新する。
// SignCount はリプレイ攻撃検出に使用するため、login 成功のたびに最新値へ更新する必要がある。
func (r *GormAuthAccountRepository) UpdateWebAuthnCredentialState(ctx context.Context, handle string, newSignCount uint32, newBackupState bool) error {
	result := r.db.WithContext(ctx).Model(&gormPasskeyCredentialRecord{}).
		Where("credential_handle = ?", strings.TrimSpace(handle)).
		Updates(map[string]any{
			"sign_count":   newSignCount,
			"backup_state": newBackupState,
		})
	if result.Error != nil {
		return domain.ErrAuthStoreUnavailable
	}
	return nil
}

func (r *GormAuthAccountRepository) findByPasskey(ctx context.Context, query string, value string) (domain.AuthAccount, error) {
	var passkey gormPasskeyCredentialRecord
	if err := r.db.WithContext(ctx).Where(query, value).First(&passkey).Error; err != nil {
		return emptyAuthAccount(), mapAuthAccountError(err)
	}

	var account gormAccountRecord
	if err := r.db.WithContext(ctx).Where("id = ?", passkey.AccountID).First(&account).Error; err != nil {
		return emptyAuthAccount(), mapAuthAccountError(err)
	}

	return normalizeDomainAccount(account, passkey)
}

func toDomainAuthAccount(account gormAccountRecord, passkey gormPasskeyCredentialRecord) (domain.AuthAccount, error) {
	return domain.NewAuthAccount(account.ID, passkey.Identifier, account.Email, passkey.ID, passkey.CredentialHandle)
}

func normalizeDomainAccount(account gormAccountRecord, passkey gormPasskeyCredentialRecord) (domain.AuthAccount, error) {
	authAccount, err := toDomainAuthAccount(account, passkey)
	if err != nil {
		return emptyAuthAccount(), domain.ErrAuthStoreUnavailable
	}
	return authAccount, nil
}

func mapAuthAccountError(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return domain.ErrAuthAccountNotFound
	}

	if err != nil {
		return domain.ErrAuthStoreUnavailable
	}

	return err
}

func emptyAuthAccount() domain.AuthAccount {
	account, _ := domain.NewAuthAccount(
		"01ARZ3NDEKTSV4RRFFQ69G5FAV",
		"placeholder@example.com",
		"placeholder@example.com",
		"01ARZ3NDEKTSV4RRFFQ69G5FAW",
		"placeholder-credential",
	)
	return account
}
