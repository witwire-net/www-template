package domain

import "errors"

var (
	ErrAuthStoreUnavailable      = errors.New("auth store unavailable")
	ErrAccountAuthNotFound       = errors.New("account auth not found")
	ErrChallengeNotFound         = errors.New("challenge not found")
	ErrChallengeExpired          = errors.New("challenge expired")
	ErrInvalidChallenge          = errors.New("challenge is invalid")
	ErrSessionNotFound           = errors.New("session not found")
	ErrSessionExpired            = errors.New("session expired")
	ErrSessionRevoked            = errors.New("session revoked")
	ErrRecoveryTokenNotFound     = errors.New("recovery token not found")
	ErrRecoveryTokenExpired      = errors.New("recovery token expired")
	ErrRecoveryTokenConsumed     = errors.New("recovery token consumed")
	ErrRecoverySessionNotFound   = errors.New("recovery session not found")
	ErrRecoverySessionExpired    = errors.New("recovery session expired")
	ErrRecoverySessionConsumed   = errors.New("recovery session consumed")
	ErrReauthSessionNotFound     = errors.New("reauthentication session not found")
	ErrReauthSessionExpired      = errors.New("reauthentication session expired")
	ErrReauthSessionConsumed     = errors.New("reauthentication session consumed")
	ErrReauthSessionKindMismatch = errors.New("reauthentication session kind mismatch")
	ErrAuthTemporarilyLocked     = errors.New("auth flow is temporarily locked")
	ErrAuthBranchAmbiguous       = errors.New("exactly one auth branch selector is required")
	ErrRecoveryStateRequired     = errors.New("recovery session is required")
	ErrInvalidOpaqueSecret       = errors.New("opaque secret is required")
	ErrInvalidPasskeyCredential  = errors.New("passkey credential id is required")
	ErrInvalidSessionID          = errors.New("session id is required")
	ErrInvalidToken              = errors.New("token is required")
	ErrInvalidSessionExpiry      = errors.New("session expiry is required")
	// ErrInvalidTokenKind は recovery token/session の kind が空または無効な場合に返す。
	ErrInvalidTokenKind = errors.New("token kind is required")
)
