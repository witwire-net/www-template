package domain

import (
	"time"
)

// TokenKind はリカバリートークン・セッションの用途種別を表す。
// "recovery"（アカウント復旧）または "device-link"（新規デバイス登録）を取る。
type TokenKind string

const (
	// TokenKindRecovery はアカウント復旧用のトークン・セッション種別。
	TokenKindRecovery TokenKind = "recovery"
	// TokenKindDeviceLink は新規デバイスでのパスキー追加用のトークン・セッション種別。
	TokenKindDeviceLink TokenKind = "device-link"
)

// ValidateTokenKind は TokenKind が有効な値（recovery または device-link）であることを検証する。
// 空文字列または未定義の値の場合は ErrInvalidTokenKind を返す。
func ValidateTokenKind(kind TokenKind) error {
	switch kind {
	case TokenKindRecovery, TokenKindDeviceLink:
		return nil
	default:
		return ErrInvalidTokenKind
	}
}

// RecoveryToken はアカウント復旧用のトークンを表現する。
// パスキー紛失時のアカウント復旧や新規デバイス登録に使用される。
// secret は暗号学的に安全なランダム値、kind は用途種別（recovery または device-link）。
type RecoveryToken struct {
	id         string
	accountID  AccountID
	secret     string
	kind       TokenKind
	expiresAt  time.Time
	consumedAt *time.Time
}

// RecoverySession はアカウント復旧用のセッションを表現する。
// RecoveryToken の検証後に発行され、パスキー登録等の後続操作に使用される。
type RecoverySession struct {
	id         string
	accountID  AccountID
	kind       TokenKind
	expiresAt  time.Time
	consumedAt *time.Time
}

// NewRecoveryToken は新しい RecoveryToken を生成する。
// id と accountID は有効な ULID、secret は空でない、expiresAt はゼロ値でないことを検証する。
// kind は空でないこと、かつ有効な TokenKind であることを検証する。
func NewRecoveryToken(id string, accountID AccountID, secret string, kind TokenKind, expiresAt time.Time) (RecoveryToken, error) {
	if err := ValidateAuthID(id); err != nil {
		return RecoveryToken{}, ErrInvalidAuthID
	}
	if _, err := NewAccountID(accountID.String()); err != nil {
		return RecoveryToken{}, ErrInvalidAccountID
	}
	if secret == "" {
		return RecoveryToken{}, ErrInvalidOpaqueSecret
	}
	if err := ValidateTokenKind(kind); err != nil {
		return RecoveryToken{}, err
	}
	if expiresAt.IsZero() {
		return RecoveryToken{}, ErrInvalidSessionExpiry
	}

	return RecoveryToken{id: id, accountID: accountID, secret: secret, kind: kind, expiresAt: expiresAt}, nil
}

// ReconstituteRecoveryToken は永続化層からの復元用に RecoveryToken を再構成する。
// consumedAt を含む全フィールドを設定する。
func ReconstituteRecoveryToken(id string, accountID AccountID, secret string, kind TokenKind, expiresAt time.Time, consumedAt *time.Time) (RecoveryToken, error) {
	token, err := NewRecoveryToken(id, accountID, secret, kind, expiresAt)
	if err != nil {
		return RecoveryToken{}, err
	}

	token.consumedAt = consumedAt
	return token, nil
}

// EnsureConsumable はトークンが消費可能（未消費かつ有効期限内）であることを確認する。
// 既に消費済みの場合は ErrRecoveryTokenConsumed を、有効期限切れの場合は ErrRecoveryTokenExpired を返す。
func (t RecoveryToken) EnsureConsumable(now time.Time) error {
	if t.consumedAt != nil {
		return ErrRecoveryTokenConsumed
	}
	if now.After(t.expiresAt) {
		return ErrRecoveryTokenExpired
	}

	return nil
}

// Consume はトークンを消費済みとしてマークする。消費時刻は UTC で記録される。
func (t RecoveryToken) Consume(at time.Time) RecoveryToken {
	consumedAt := at.UTC()
	t.consumedAt = &consumedAt
	return t
}

// ID はトークンの一意識別子（ULID）を返す。
func (t RecoveryToken) ID() string { return t.id }

// AccountID はトークンが紐づくアカウントの ULID を返す。
func (t RecoveryToken) AccountID() AccountID {
	return t.accountID
}

// Secret はトークンの秘密値（暗号学的に安全なランダム文字列）を返す。
func (t RecoveryToken) Secret() string { return t.secret }

// Kind はトークンの用途種別（recovery または device-link）を返す。
func (t RecoveryToken) Kind() TokenKind { return t.kind }

// ExpiresAt はトークンの有効期限を返す。
func (t RecoveryToken) ExpiresAt() time.Time { return t.expiresAt }

// ConsumedAt はトークンの消費時刻を返す。未消費の場合は nil を返す。
func (t RecoveryToken) ConsumedAt() *time.Time { return t.consumedAt }

// NewRecoverySession は新しい RecoverySession を生成する。
// id と accountID は有効な ULID、expiresAt はゼロ値でないことを検証する。
// kind は空でないこと、かつ有効な TokenKind であることを検証する。
func NewRecoverySession(id string, accountID AccountID, kind TokenKind, expiresAt time.Time) (RecoverySession, error) {
	if err := ValidateAuthID(id); err != nil {
		return RecoverySession{}, ErrInvalidAuthID
	}
	if _, err := NewAccountID(accountID.String()); err != nil {
		return RecoverySession{}, ErrInvalidAccountID
	}
	if err := ValidateTokenKind(kind); err != nil {
		return RecoverySession{}, err
	}
	if expiresAt.IsZero() {
		return RecoverySession{}, ErrInvalidSessionExpiry
	}

	return RecoverySession{id: id, accountID: accountID, kind: kind, expiresAt: expiresAt}, nil
}

// ReconstituteRecoverySession は永続化層からの復元用に RecoverySession を再構成する。
// consumedAt を含む全フィールドを設定する。
func ReconstituteRecoverySession(id string, accountID AccountID, kind TokenKind, expiresAt time.Time, consumedAt *time.Time) (RecoverySession, error) {
	session, err := NewRecoverySession(id, accountID, kind, expiresAt)
	if err != nil {
		return RecoverySession{}, err
	}

	session.consumedAt = consumedAt
	return session, nil
}

// EnsureAvailable はセッションが利用可能（未消費かつ有効期限内）であることを確認する。
// 既に消費済みの場合は ErrRecoverySessionConsumed を、有効期限切れの場合は ErrRecoverySessionExpired を返す。
func (s RecoverySession) EnsureAvailable(now time.Time) error {
	if s.consumedAt != nil {
		return ErrRecoverySessionConsumed
	}
	if now.After(s.expiresAt) {
		return ErrRecoverySessionExpired
	}

	return nil
}

// Consume はセッションを消費済みとしてマークする。消費時刻は UTC で記録される。
func (s RecoverySession) Consume(at time.Time) RecoverySession {
	consumedAt := at.UTC()
	s.consumedAt = &consumedAt
	return s
}

// ID はセッションの一意識別子（ULID）を返す。
func (s RecoverySession) ID() string { return s.id }

// AccountID はセッションが紐づくアカウントの ULID を返す。
func (s RecoverySession) AccountID() AccountID {
	return s.accountID
}

// Kind はセッションの用途種別（recovery または device-link）を返す。
func (s RecoverySession) Kind() TokenKind { return s.kind }

// ExpiresAt はセッションの有効期限を返す。
func (s RecoverySession) ExpiresAt() time.Time { return s.expiresAt }

// ConsumedAt はセッションの消費時刻を返す。未消費の場合は nil を返す。
func (s RecoverySession) ConsumedAt() *time.Time { return s.consumedAt }
