package domain

import (
	"strings"
	"time"
)

// PasskeyCredential は 1 件のパスキー credential を表す値オブジェクト。
type PasskeyCredential struct {
	id               string
	accountID        string
	identifier       string
	credentialHandle string
	createdAt        time.Time
}

// NewPasskeyCredential は PasskeyCredential を生成する。
// id または accountID が ULID でない場合は ErrInvalidAuthID を返す。
// credentialHandle が空の場合は ErrInvalidPasskeyCredential を返す。
func NewPasskeyCredential(id string, accountID string, identifier string, credentialHandle string, createdAt time.Time) (PasskeyCredential, error) {
	if err := ValidateAuthID(id); err != nil {
		return PasskeyCredential{}, ErrInvalidAuthID
	}
	if err := ValidateAuthID(accountID); err != nil {
		return PasskeyCredential{}, ErrInvalidAccountID
	}
	if strings.TrimSpace(credentialHandle) == "" {
		return PasskeyCredential{}, ErrInvalidPasskeyCredential
	}
	return PasskeyCredential{
		id:               id,
		accountID:        accountID,
		identifier:       strings.TrimSpace(identifier),
		credentialHandle: strings.TrimSpace(credentialHandle),
		createdAt:        createdAt,
	}, nil
}

func (c PasskeyCredential) ID() string               { return c.id }
func (c PasskeyCredential) AccountID() string        { return c.accountID }
func (c PasskeyCredential) Identifier() string       { return c.identifier }
func (c PasskeyCredential) CredentialHandle() string { return c.credentialHandle }
func (c PasskeyCredential) CreatedAt() time.Time     { return c.createdAt }

// AuthAccount はひとつの認証済みアカウントを表すドメインモデル。
// credentials に 1 件以上の PasskeyCredential を保持する。
type AuthAccount struct {
	accountID   string
	identifier  string
	email       string
	credentials []PasskeyCredential
}

// NewAuthAccount は後方互換のための単一パスキー版コンストラクタ。
func NewAuthAccount(accountID string, identifier string, email string, passkeyCredentialID string, credentialHandle string) (AuthAccount, error) {
	if err := ValidateAuthID(accountID); err != nil {
		return AuthAccount{}, ErrInvalidAccountID
	}
	if err := ValidateAuthID(passkeyCredentialID); err != nil {
		return AuthAccount{}, ErrInvalidPasskeyCredential
	}
	if strings.TrimSpace(identifier) == "" || strings.TrimSpace(email) == "" || strings.TrimSpace(credentialHandle) == "" {
		return AuthAccount{}, ErrInvalidChallenge
	}

	cred := PasskeyCredential{
		id:               passkeyCredentialID,
		accountID:        accountID,
		identifier:       strings.TrimSpace(identifier),
		credentialHandle: strings.TrimSpace(credentialHandle),
		createdAt:        time.Time{},
	}

	return AuthAccount{
		accountID:   accountID,
		identifier:  strings.TrimSpace(identifier),
		email:       strings.TrimSpace(email),
		credentials: []PasskeyCredential{cred},
	}, nil
}

// NewAuthAccountWithCredentials は複数 credential を受け取るコンストラクタ。
// credentials が空の場合は ErrInvalidPasskeyCredential を返す。
func NewAuthAccountWithCredentials(accountID string, identifier string, email string, credentials []PasskeyCredential) (AuthAccount, error) {
	if err := ValidateAuthID(accountID); err != nil {
		return AuthAccount{}, ErrInvalidAccountID
	}
	if strings.TrimSpace(identifier) == "" || strings.TrimSpace(email) == "" {
		return AuthAccount{}, ErrInvalidChallenge
	}
	if len(credentials) == 0 {
		return AuthAccount{}, ErrInvalidPasskeyCredential
	}

	return AuthAccount{
		accountID:   accountID,
		identifier:  strings.TrimSpace(identifier),
		email:       strings.TrimSpace(email),
		credentials: credentials,
	}, nil
}

func (a AuthAccount) AccountID() string  { return a.accountID }
func (a AuthAccount) Identifier() string { return a.identifier }
func (a AuthAccount) Email() string      { return a.email }

// Credentials は保持する全 PasskeyCredential のスライスを返す。
func (a AuthAccount) Credentials() []PasskeyCredential { return a.credentials }

// PasskeyCredentialID は後方互換アクセサ。先頭 credential の ID を返す。
func (a AuthAccount) PasskeyCredentialID() string {
	if len(a.credentials) == 0 {
		return ""
	}
	return a.credentials[0].ID()
}

// CredentialHandle は後方互換アクセサ。先頭 credential の CredentialHandle を返す。
func (a AuthAccount) CredentialHandle() string {
	if len(a.credentials) == 0 {
		return ""
	}
	return a.credentials[0].CredentialHandle()
}
