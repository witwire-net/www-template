package domain

import (
	"errors"
	"strings"
	"time"
)

var (
	ErrInvalidAccountID         = errors.New("account id is required")
	ErrInvalidPasskeyCredential = errors.New("passkey credential id is required")
	ErrInvalidSessionID         = errors.New("session id is required")
	ErrInvalidSessionToken      = errors.New("session token is required")
	ErrInvalidSessionExpiry     = errors.New("session expiry is required")
)

type Session struct {
	id                  string
	accountID           string
	passkeyCredentialID string
	token               string
	idleExpiresAt       time.Time
	absoluteExpiresAt   time.Time
	revokedAt           *time.Time
}

func NewSession(id string, accountID string, passkeyCredentialID string, token string, idleExpiresAt time.Time, absoluteExpiresAt time.Time) (Session, error) {
	if err := ValidateAuthID(id); err != nil {
		return Session{}, ErrInvalidSessionID
	}
	if err := ValidateAuthID(accountID); err != nil {
		return Session{}, ErrInvalidAccountID
	}
	if err := ValidateAuthID(passkeyCredentialID); err != nil {
		return Session{}, ErrInvalidPasskeyCredential
	}
	if strings.TrimSpace(token) == "" {
		return Session{}, ErrInvalidSessionToken
	}
	if idleExpiresAt.IsZero() || absoluteExpiresAt.IsZero() || !idleExpiresAt.Before(absoluteExpiresAt) {
		return Session{}, ErrInvalidSessionExpiry
	}

	return Session{
		id:                  id,
		accountID:           accountID,
		passkeyCredentialID: passkeyCredentialID,
		token:               token,
		idleExpiresAt:       idleExpiresAt,
		absoluteExpiresAt:   absoluteExpiresAt,
	}, nil
}

func ReconstituteSession(id string, accountID string, passkeyCredentialID string, token string, idleExpiresAt time.Time, absoluteExpiresAt time.Time, revokedAt *time.Time) (Session, error) {
	session, err := NewSession(id, accountID, passkeyCredentialID, token, idleExpiresAt, absoluteExpiresAt)
	if err != nil {
		return Session{}, err
	}

	session.revokedAt = revokedAt
	return session, nil
}

func (s Session) EnsureActive(now time.Time) error {
	if s.revokedAt != nil {
		return ErrSessionRevoked
	}
	if now.After(s.idleExpiresAt) || now.After(s.absoluteExpiresAt) {
		return ErrSessionExpired
	}

	return nil
}

func (s Session) Revoke(at time.Time) Session {
	revokedAt := at.UTC()
	s.revokedAt = &revokedAt
	return s
}

func (s Session) RefreshIdle(now time.Time, ttl time.Duration) Session {
	s.idleExpiresAt = now.UTC().Add(ttl)
	if s.idleExpiresAt.After(s.absoluteExpiresAt) {
		s.idleExpiresAt = s.absoluteExpiresAt
	}

	return s
}

func (s Session) RevocationTTL(now time.Time) time.Duration {
	if now.After(s.absoluteExpiresAt) {
		return 0
	}

	return s.absoluteExpiresAt.Sub(now)
}

func (s Session) ID() string                   { return s.id }
func (s Session) AccountID() string            { return s.accountID }
func (s Session) PasskeyCredentialID() string  { return s.passkeyCredentialID }
func (s Session) Token() string                { return s.token }
func (s Session) IdleExpiresAt() time.Time     { return s.idleExpiresAt }
func (s Session) AbsoluteExpiresAt() time.Time { return s.absoluteExpiresAt }
func (s Session) RevokedAt() *time.Time        { return s.revokedAt }
