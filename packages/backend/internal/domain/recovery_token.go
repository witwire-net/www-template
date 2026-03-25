package domain

import (
	"time"
)

type RecoveryToken struct {
	id         string
	accountID  string
	secret     string
	expiresAt  time.Time
	consumedAt *time.Time
}

type RecoverySession struct {
	id         string
	accountID  string
	expiresAt  time.Time
	consumedAt *time.Time
}

func NewRecoveryToken(id string, accountID string, secret string, expiresAt time.Time) (RecoveryToken, error) {
	if err := ValidateAuthID(id); err != nil {
		return RecoveryToken{}, ErrInvalidAuthID
	}
	if err := ValidateAuthID(accountID); err != nil {
		return RecoveryToken{}, ErrInvalidAccountID
	}
	if secret == "" {
		return RecoveryToken{}, ErrInvalidOpaqueSecret
	}
	if expiresAt.IsZero() {
		return RecoveryToken{}, ErrInvalidSessionExpiry
	}

	return RecoveryToken{id: id, accountID: accountID, secret: secret, expiresAt: expiresAt}, nil
}

func ReconstituteRecoveryToken(id string, accountID string, secret string, expiresAt time.Time, consumedAt *time.Time) (RecoveryToken, error) {
	token, err := NewRecoveryToken(id, accountID, secret, expiresAt)
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

func (t RecoveryToken) ID() string             { return t.id }
func (t RecoveryToken) AccountID() string      { return t.accountID }
func (t RecoveryToken) Secret() string         { return t.secret }
func (t RecoveryToken) ExpiresAt() time.Time   { return t.expiresAt }
func (t RecoveryToken) ConsumedAt() *time.Time { return t.consumedAt }

func NewRecoverySession(id string, accountID string, expiresAt time.Time) (RecoverySession, error) {
	if err := ValidateAuthID(id); err != nil {
		return RecoverySession{}, ErrInvalidAuthID
	}
	if err := ValidateAuthID(accountID); err != nil {
		return RecoverySession{}, ErrInvalidAccountID
	}
	if expiresAt.IsZero() {
		return RecoverySession{}, ErrInvalidSessionExpiry
	}

	return RecoverySession{id: id, accountID: accountID, expiresAt: expiresAt}, nil
}

func ReconstituteRecoverySession(id string, accountID string, expiresAt time.Time, consumedAt *time.Time) (RecoverySession, error) {
	session, err := NewRecoverySession(id, accountID, expiresAt)
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

func (s RecoverySession) ID() string             { return s.id }
func (s RecoverySession) AccountID() string      { return s.accountID }
func (s RecoverySession) ExpiresAt() time.Time   { return s.expiresAt }
func (s RecoverySession) ConsumedAt() *time.Time { return s.consumedAt }
