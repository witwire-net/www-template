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
	accountID           string
	identifier          string
	email               string
	status              string
	sessionRevokedAfter *time.Time
	credentials         []PasskeyCredential
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
		status:      "active",
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
		status:      "active",
		credentials: credentials,
	}, nil
}

func (a AuthAccount) AccountID() string  { return a.accountID }
func (a AuthAccount) Identifier() string { return a.identifier }
func (a AuthAccount) Email() string      { return a.email }
func (a AuthAccount) Status() string     { return a.status }

// IsSuspended はアカウントが管理者操作で停止中かどうかを返す。
// status は Product DB の accounts.status から復元された値であり、
// suspended の場合は新規 token pair 発行、refresh rotation、bearer 認可を拒否する。
// 未設定または active の場合は false を返す。
func (a AuthAccount) IsSuspended() bool { return a.status == "suspended" }

// SessionRevokedAfter はアカウント停止時に設定されたセッション失効時刻を返す。
// nil の場合は失効時刻が設定されていないことを示す。
func (a AuthAccount) SessionRevokedAfter() *time.Time { return a.sessionRevokedAfter }

// WithStatus は status と sessionRevokedAfter を設定した新しい AuthAccount を返す。
// リポジトリ層で DB から読み出した値を設定する際に使用する。
func (a AuthAccount) WithStatus(status string, sessionRevokedAfter *time.Time) AuthAccount {
	a.status = status
	a.sessionRevokedAfter = sessionRevokedAfter
	return a
}

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
