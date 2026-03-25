package domain

import "time"

type AuthChallenge struct {
	id         string
	identifier string
	challenge  string
	expiresAt  time.Time
}

type AuthLock struct {
	lockedUntil time.Time
}

func NewAuthChallenge(id string, identifier string, challenge string, expiresAt time.Time) (AuthChallenge, error) {
	if err := ValidateAuthID(id); err != nil {
		return AuthChallenge{}, ErrInvalidAuthID
	}
	if identifier == "" || challenge == "" || expiresAt.IsZero() {
		return AuthChallenge{}, ErrInvalidChallenge
	}

	return AuthChallenge{id: id, identifier: identifier, challenge: challenge, expiresAt: expiresAt}, nil
}

func (c AuthChallenge) EnsureAvailable(now time.Time) error {
	if now.After(c.expiresAt) {
		return ErrChallengeExpired
	}

	return nil
}

func (c AuthChallenge) ID() string           { return c.id }
func (c AuthChallenge) Identifier() string   { return c.identifier }
func (c AuthChallenge) Challenge() string    { return c.challenge }
func (c AuthChallenge) ExpiresAt() time.Time { return c.expiresAt }

func NewAuthLock(until time.Time) AuthLock {
	return AuthLock{lockedUntil: until}
}

func (l AuthLock) Active(now time.Time) bool {
	return now.Before(l.lockedUntil)
}

func (l AuthLock) LockedUntil() time.Time { return l.lockedUntil }
