package persistence

import (
	"context"
	"errors"
	"strings"

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
	ID               string `gorm:"column:id;primaryKey"`
	AccountID        string `gorm:"column:account_id"`
	Identifier       string `gorm:"column:identifier"`
	CredentialHandle string `gorm:"column:credential_handle"`
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

func (r *GormAuthAccountRepository) ReplacePasskey(ctx context.Context, accountID string, passkeyCredentialID string, credential string) (domain.AuthAccount, error) {
	validatedAccount, err := domain.NewAuthAccount(accountID, "placeholder@example.com", "placeholder@example.com", passkeyCredentialID, credential)
	if err != nil {
		return emptyAuthAccount(), err
	}
	var (
		account         gormAccountRecord
		passkey         gormPasskeyCredentialRecord
		replacedAccount = validatedAccount
	)
	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("id = ?", accountID).First(&account).Error; err != nil {
			return mapAuthAccountError(err)
		}
		replacedAccount, err = domain.NewAuthAccount(account.ID, account.Email, account.Email, passkeyCredentialID, credential)
		if err != nil {
			return err
		}
		if err := tx.Where("account_id = ?", accountID).Delete(&gormPasskeyCredentialRecord{}).Error; err != nil {
			return domain.ErrAuthStoreUnavailable
		}

		passkey = gormPasskeyCredentialRecord{ID: replacedAccount.PasskeyCredentialID(), AccountID: replacedAccount.AccountID(), Identifier: replacedAccount.Identifier(), CredentialHandle: replacedAccount.CredentialHandle()}
		if err := tx.Save(&passkey).Error; err != nil {
			return domain.ErrAuthStoreUnavailable
		}

		return nil
	})
	if err != nil {
		if errors.Is(err, domain.ErrAuthAccountNotFound) || errors.Is(err, domain.ErrAuthStoreUnavailable) {
			return emptyAuthAccount(), err
		}
		return emptyAuthAccount(), mapAuthAccountError(err)
	}

	return replacedAccount, nil
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
