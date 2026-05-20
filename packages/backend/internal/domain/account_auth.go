package domain

import (
	"strings"
	"time"
)

// PasskeyCredential は Account.Auth に属する 1 件のパスキー credential を表す値オブジェクトである。
//
// id と accountID は canonical ULID でなければならない。
// credentialHandle は WebAuthn の credential raw ID を安全に参照するための永続値であり、空文字を許可しない。
// この値オブジェクトは認証用 projection の一部だけを表し、AccountSetting や locale を保持しない。
func NewPasskeyCredential(id string, accountID AccountID, identifier string, credentialHandle string, createdAt time.Time) (PasskeyCredential, error) {
	if err := ValidateAuthID(id); err != nil {
		return PasskeyCredential{}, ErrInvalidAuthID
	}
	if _, err := NewAccountID(accountID.String()); err != nil {
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

// PasskeyCredential は Account.Auth projection が認証に使う credential の最小情報である。
type PasskeyCredential struct {
	id               string
	accountID        AccountID
	identifier       string
	credentialHandle string
	createdAt        time.Time
}

// ID は credential 自体の canonical ULID を返す。
func (c PasskeyCredential) ID() string { return c.id }

// AccountID は credential を所有する Product Account の canonical ULID を返す。
func (c PasskeyCredential) AccountID() AccountID { return c.accountID }

// Identifier は認証開始時に使用する identifier を返す。
func (c PasskeyCredential) Identifier() string { return c.identifier }

// CredentialHandle は WebAuthn credential lookup に使う handle を返す。
func (c PasskeyCredential) CredentialHandle() string { return c.credentialHandle }

// CreatedAt は credential が作成された時刻を返す。
func (c PasskeyCredential) CreatedAt() time.Time { return c.createdAt }

// NewAccountAuth は単一 credential から Account.Auth projection を生成する。
//
// accountID と passkeyCredentialID は canonical ULID でなければならない。
// identifier、email、credentialHandle は空白を除去した上で空文字を拒否する。
// 戻り値は認証に必要な AccountID、identifier、email、status、credential だけを持ち、AccountSetting を持たない。
func NewAccountAuth(accountID AccountID, identifier string, email string, passkeyCredentialID string, credentialHandle string) (AccountAuth, error) {
	if _, err := NewAccountID(accountID.String()); err != nil {
		return AccountAuth{}, ErrInvalidAccountID
	}
	if err := ValidateAuthID(passkeyCredentialID); err != nil {
		return AccountAuth{}, ErrInvalidPasskeyCredential
	}
	if strings.TrimSpace(identifier) == "" || strings.TrimSpace(email) == "" || strings.TrimSpace(credentialHandle) == "" {
		return AccountAuth{}, ErrInvalidChallenge
	}

	cred := PasskeyCredential{
		id:               passkeyCredentialID,
		accountID:        accountID,
		identifier:       strings.TrimSpace(identifier),
		credentialHandle: strings.TrimSpace(credentialHandle),
		createdAt:        time.Time{},
	}

	return AccountAuth{
		accountID:   accountID,
		identifier:  strings.TrimSpace(identifier),
		email:       strings.TrimSpace(email),
		status:      "active",
		credentials: []PasskeyCredential{cred},
	}, nil
}

// NewAccountAuthWithCredentials は複数 credential から Account.Auth projection を生成する。
//
// credentials が空の場合は Account.Auth として認証不能な projection になるため ErrInvalidPasskeyCredential を返す。
// この constructor も AccountSetting / locale を受け取らず、Auth が設定境界を所有しないことを保証する。
func NewAccountAuthWithCredentials(accountID AccountID, identifier string, email string, credentials []PasskeyCredential) (AccountAuth, error) {
	if _, err := NewAccountID(accountID.String()); err != nil {
		return AccountAuth{}, ErrInvalidAccountID
	}
	if strings.TrimSpace(identifier) == "" || strings.TrimSpace(email) == "" {
		return AccountAuth{}, ErrInvalidChallenge
	}
	if len(credentials) == 0 {
		return AccountAuth{}, ErrInvalidPasskeyCredential
	}

	return AccountAuth{
		accountID:   accountID,
		identifier:  strings.TrimSpace(identifier),
		email:       strings.TrimSpace(email),
		status:      "active",
		credentials: credentials,
	}, nil
}

// AccountAuth は Account にぶら下がる認証用 projection である。
//
// AccountID、identifier、email、status、sessionRevokedAfter、passkey credentials だけを保持する。
// 表示・通知設定である AccountSetting や locale は Product Account 側の責務であり、この型には存在しない。
type AccountAuth struct {
	accountID           AccountID
	identifier          string
	email               string
	status              string
	sessionRevokedAfter *time.Time
	credentials         []PasskeyCredential
}

// AccountID は Product Account の canonical ULID を返す。
func (a AccountAuth) AccountID() AccountID { return a.accountID }

// Identifier は認証開始で使用する identifier を返す。
func (a AccountAuth) Identifier() string { return a.identifier }

// Email は Account.Auth projection に必要な通知先メールアドレスを返す。
func (a AccountAuth) Email() string { return a.email }

// Status は Product Account の認証可否判断に使う status を返す。
func (a AccountAuth) Status() string { return a.status }

// IsSuspended は Product Account が停止中で認証を拒否すべきかを返す。
func (a AccountAuth) IsSuspended() bool { return a.status == "suspended" }

// SessionRevokedAfter は停止や管理操作により既存 session を拒否すべき境界時刻を返す。
func (a AccountAuth) SessionRevokedAfter() *time.Time { return a.sessionRevokedAfter }

// WithStatus は DB から復元した status と sessionRevokedAfter を反映した新しい AccountAuth を返す。
func (a AccountAuth) WithStatus(status string, sessionRevokedAfter *time.Time) AccountAuth {
	a.status = status
	a.sessionRevokedAfter = sessionRevokedAfter
	return a
}

// Credentials は Account.Auth に属する全 passkey credentials を返す。
func (a AccountAuth) Credentials() []PasskeyCredential { return a.credentials }
