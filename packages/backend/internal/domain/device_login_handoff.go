package domain

import (
	"errors"
	"time"
)

var (
	// ErrDeviceLoginHandoffNotFound は指定された handoff record が存在しない場合のエラー。
	ErrDeviceLoginHandoffNotFound = errors.New("device login handoff not found")
	// ErrDeviceLoginHandoffExpired は handoff record の有効期限が切れている場合のエラー。
	ErrDeviceLoginHandoffExpired = errors.New("device login handoff expired")
	// ErrDeviceLoginHandoffConsumed は handoff record が既に消費済みである場合のエラー。
	ErrDeviceLoginHandoffConsumed = errors.New("device login handoff consumed")
	// ErrDeviceLoginHandoffLocked は handoff 対象が一時ロック中である場合のエラー。
	ErrDeviceLoginHandoffLocked = errors.New("device login handoff locked")
)

// DeviceLoginHandoff は OTP ベースの新端末ログイン有効化のための一時状態を表す。
// account、issuing session、normalized email、handoff ID、expiration、attempt counters、challenge binding を含む。
type DeviceLoginHandoff struct {
	handoffID        string
	accountID        string
	issuingSessionID string
	emailHash        string
	otpHash          string
	challengeID      string
	expiresAt        time.Time
	attemptCount     int
	consumedAt       *time.Time
}

// NewDeviceLoginHandoff は新しい DeviceLoginHandoff を生成する。
// handoffID、accountID、issuingSessionID は有効な ULID でなければならない。
// emailHash と otpHash は空文字列を許可しない。
// expiresAt はゼロ値を許可しない。
func NewDeviceLoginHandoff(handoffID string, accountID string, issuingSessionID string, emailHash string, otpHash string, expiresAt time.Time) (DeviceLoginHandoff, error) {
	if err := ValidateAuthID(handoffID); err != nil {
		return DeviceLoginHandoff{}, ErrInvalidAuthID
	}
	if err := ValidateAuthID(accountID); err != nil {
		return DeviceLoginHandoff{}, ErrInvalidAccountID
	}
	if err := ValidateAuthID(issuingSessionID); err != nil {
		return DeviceLoginHandoff{}, ErrInvalidSessionID
	}
	if emailHash == "" || otpHash == "" {
		return DeviceLoginHandoff{}, errors.New("email hash and otp hash are required")
	}
	if expiresAt.IsZero() {
		return DeviceLoginHandoff{}, errors.New("expiresAt is required")
	}
	return DeviceLoginHandoff{
		handoffID:        handoffID,
		accountID:        accountID,
		issuingSessionID: issuingSessionID,
		emailHash:        emailHash,
		otpHash:          otpHash,
		expiresAt:        expiresAt,
	}, nil
}

// BindChallenge は WebAuthn challenge ID を handoff に紐付ける。
// start 時に呼び出され、finish 時の challenge 検証に使用される。
func (h DeviceLoginHandoff) BindChallenge(challengeID string) DeviceLoginHandoff {
	h.challengeID = challengeID
	return h
}

// IncrementAttempt は検証試行回数を 1 増やす。
// OTP 検証失敗時に呼び出され、失敗カウントの加算に使用される。
func (h DeviceLoginHandoff) IncrementAttempt() DeviceLoginHandoff {
	h.attemptCount++
	return h
}

// Consume は handoff を消費済みとしてマークする。
// atomic consume 時に呼び出され、二度目の使用を防ぐ。
func (h DeviceLoginHandoff) Consume(now time.Time) DeviceLoginHandoff {
	at := now.UTC()
	h.consumedAt = &at
	return h
}

// EnsureAvailable は handoff が有効（未消費かつ期限内）であることを確認する。
// 消費済みまたは期限切れの場合はエラーを返す。
func (h DeviceLoginHandoff) EnsureAvailable(now time.Time) error {
	if h.consumedAt != nil {
		return ErrDeviceLoginHandoffConsumed
	}
	if now.After(h.expiresAt) {
		return ErrDeviceLoginHandoffExpired
	}
	return nil
}

// EmptyDeviceLoginHandoff はゼロ値の DeviceLoginHandoff を返す。
// エラー発生時など、有効な handoff が存在しないことを表す用途でのみ使用する。
// この値は EnsureAvailable を呼び出すと常にエラーとなり、安全に「無効」状態を表現できる。
func EmptyDeviceLoginHandoff() DeviceLoginHandoff {
	return DeviceLoginHandoff{}
}

// ID は handoff ID を返す。
func (h DeviceLoginHandoff) ID() string { return h.handoffID }

// AccountID は紐づくアカウント ID を返す。
func (h DeviceLoginHandoff) AccountID() string { return h.accountID }

// IssuingSessionID は発行元セッション ID を返す。
func (h DeviceLoginHandoff) IssuingSessionID() string { return h.issuingSessionID }

// EmailHash は正規化・ハッシュ済みのメールアドレスを返す。
func (h DeviceLoginHandoff) EmailHash() string { return h.emailHash }

// OtpHash はハッシュ済みの OTP を返す。
func (h DeviceLoginHandoff) OtpHash() string { return h.otpHash }

// ChallengeID は紐づく WebAuthn challenge ID を返す。
func (h DeviceLoginHandoff) ChallengeID() string { return h.challengeID }

// ExpiresAt は handoff の有効期限を返す。
func (h DeviceLoginHandoff) ExpiresAt() time.Time { return h.expiresAt }

// AttemptCount は検証試行回数を返す。
func (h DeviceLoginHandoff) AttemptCount() int { return h.attemptCount }

// ConsumedAt は消費日時を返す。未消費の場合は nil。
func (h DeviceLoginHandoff) ConsumedAt() *time.Time { return h.consumedAt }
