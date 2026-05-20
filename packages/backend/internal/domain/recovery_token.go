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

type RecoveryToken struct {
	id         string
	accountID  AccountID
	secret     string
	kind       TokenKind
	expiresAt  time.Time
	consumedAt *time.Time
}

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

func (t RecoveryToken) EnsureConsumable(now time.Time) error {
	if t.consumedAt != nil {
		return ErrRecoveryTokenConsumed
	}
	if now.After(t.expiresAt) {
		return ErrRecoveryTokenExpired
	}

	return nil
}

func (t RecoveryToken) Consume(at time.Time) RecoveryToken {
	consumedAt := at.UTC()
	t.consumedAt = &consumedAt
	return t
}

func (t RecoveryToken) ID() string { return t.id }
func (t RecoveryToken) AccountID() AccountID {
	return t.accountID
}
func (t RecoveryToken) Secret() string         { return t.secret }
func (t RecoveryToken) Kind() TokenKind        { return t.kind }
func (t RecoveryToken) ExpiresAt() time.Time   { return t.expiresAt }
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

func (s RecoverySession) EnsureAvailable(now time.Time) error {
	if s.consumedAt != nil {
		return ErrRecoverySessionConsumed
	}
	if now.After(s.expiresAt) {
		return ErrRecoverySessionExpired
	}

	return nil
}

func (s RecoverySession) Consume(at time.Time) RecoverySession {
	consumedAt := at.UTC()
	s.consumedAt = &consumedAt
	return s
}

func (s RecoverySession) ID() string { return s.id }
func (s RecoverySession) AccountID() AccountID {
	return s.accountID
}
func (s RecoverySession) Kind() TokenKind        { return s.kind }
func (s RecoverySession) ExpiresAt() time.Time   { return s.expiresAt }
func (s RecoverySession) ConsumedAt() *time.Time { return s.consumedAt }
