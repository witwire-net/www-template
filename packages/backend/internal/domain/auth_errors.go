package domain

import "errors"

var (
	ErrAuthStoreUnavailable    = errors.New("auth store unavailable")
	ErrAuthAccountNotFound     = errors.New("auth account not found")
	ErrChallengeNotFound       = errors.New("challenge not found")
	ErrChallengeExpired        = errors.New("challenge expired")
	ErrInvalidChallenge        = errors.New("challenge is invalid")
	ErrSessionNotFound         = errors.New("session not found")
	ErrSessionExpired          = errors.New("session expired")
	ErrSessionRevoked          = errors.New("session revoked")
	ErrRecoveryTokenNotFound   = errors.New("recovery token not found")
	ErrRecoveryTokenExpired    = errors.New("recovery token expired")
	ErrRecoveryTokenConsumed   = errors.New("recovery token consumed")
	ErrRecoverySessionNotFound = errors.New("recovery session not found")
	ErrRecoverySessionExpired  = errors.New("recovery session expired")
	ErrRecoverySessionConsumed = errors.New("recovery session consumed")
	ErrAuthTemporarilyLocked   = errors.New("auth flow is temporarily locked")
	ErrAuthBranchAmbiguous     = errors.New("exactly one auth branch selector is required")
	ErrRecoveryStateRequired   = errors.New("recovery session is required")
	ErrInvalidOpaqueSecret     = errors.New("opaque secret is required")
	// ErrOtpNotFound は OTP が存在しない・期限切れ・消費済みの場合に返す。
	ErrOtpNotFound = errors.New("otp not found or expired")
)
