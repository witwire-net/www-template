package domain

import "time"

// AuthChallenge は認証チャレンジ（パスキー署名要求やリカバリートークン検証等）を表現する。
// identifier はチャレンジの対象（メールアドレス等）、challenge は Base64URL エンコードされたランダム値。
type AuthChallenge struct {
	id         string
	identifier string
	challenge  string
	expiresAt  time.Time
}

// AuthLock は認証フローの一時的なロック状態を表現する。
// レート制限やスロットリングの結果として発生し、lockedUntil までは認証試行を拒否する。
type AuthLock struct {
	lockedUntil time.Time
}

// NewAuthChallenge は新しい AuthChallenge を生成する。
// id は有効な ULID であること、identifier と challenge は空でないこと、expiresAt はゼロ値でないことを検証する。
func NewAuthChallenge(id string, identifier string, challenge string, expiresAt time.Time) (AuthChallenge, error) {
	if err := ValidateAuthID(id); err != nil {
		return AuthChallenge{}, ErrInvalidAuthID
	}
	if identifier == "" || challenge == "" || expiresAt.IsZero() {
		return AuthChallenge{}, ErrInvalidChallenge
	}

	return AuthChallenge{id: id, identifier: identifier, challenge: challenge, expiresAt: expiresAt}, nil
}

// EnsureAvailable はチャレンジが有効期限内であることを確認する。
// 有効期限を過ぎている場合は ErrChallengeExpired を返す。
func (c AuthChallenge) EnsureAvailable(now time.Time) error {
	if now.After(c.expiresAt) {
		return ErrChallengeExpired
	}

	return nil
}

// ID はチャレンジの一意識別子（ULID）を返す。
func (c AuthChallenge) ID() string { return c.id }

// Identifier はチャレンジの対象（メールアドレス等）を返す。
func (c AuthChallenge) Identifier() string { return c.identifier }

// Challenge は Base64URL エンコードされたランダムチャレンジ値を返す。
func (c AuthChallenge) Challenge() string { return c.challenge }

// ExpiresAt はチャレンジの有効期限を返す。
func (c AuthChallenge) ExpiresAt() time.Time { return c.expiresAt }

// NewAuthLock は指定された時刻まで認証をロックする AuthLock を生成する。
func NewAuthLock(until time.Time) AuthLock {
	return AuthLock{lockedUntil: until}
}

// Active は指定された時刻においてロックが有効かどうかを判定する。
// now が lockedUntil より前の場合は true を返す。
func (l AuthLock) Active(now time.Time) bool {
	return now.Before(l.lockedUntil)
}

// LockedUntil はロック解除時刻を返す。
func (l AuthLock) LockedUntil() time.Time { return l.lockedUntil }
