package usecases

import (
	"errors"
	"strings"

	"www-template/packages/backend/internal/domain"
)

var (
	ErrUnauthenticated            = errors.New("unauthenticated")
	ErrSessionExpired             = errors.New("session-expired")
	ErrInternalError              = errors.New("internal-error")
	ErrBadRequest                 = errors.New("bad auth request")
	ErrLastPasskeyCannotBeDeleted = errors.New("last passkey cannot be deleted")
	ErrInvalidOtp                 = errors.New("invalid otp")
	ErrOtpExpiredOrConsumed       = errors.New("otp expired or consumed")
)

func toAuthSession(requestID string, session domain.Session) AuthSession {
	return AuthSession{
		RequestID:           requestID,
		AccountID:           session.AccountID(),
		PasskeyCredentialID: session.PasskeyCredentialID(),
		SessionID:           session.ID(),
		SessionToken:        session.Token(),
		ExpiresAt:           session.AbsoluteExpiresAt(),
	}
}

func opaqueValue(id string) string {
	return "opaque-" + id
}

func selectorCount(recoverySession string, invitationSession string) int {
	count := 0
	if strings.TrimSpace(recoverySession) != "" {
		count++
	}
	if strings.TrimSpace(invitationSession) != "" {
		count++
	}
	return count
}

func passkeyStartKey(identifier string, clientIP string) string {
	return "start:" + strings.TrimSpace(identifier) + ":" + strings.TrimSpace(clientIP)
}

func recoveryEmailKey(email string) string {
	return "recovery:email:" + strings.TrimSpace(email)
}

func recoveryIPKey(clientIP string) string {
	return "recovery:ip:" + strings.TrimSpace(clientIP)
}

func failureLockKey(subject string, clientIP string) string {
	return "lock:" + strings.TrimSpace(subject) + ":" + strings.TrimSpace(clientIP)
}

func failureWindowKey(key string) string {
	return "failures:" + key
}

func (s *AuthService) mapSessionError(err error) error {
	switch {
	case errors.Is(err, domain.ErrSessionNotFound):
		return ErrUnauthenticated
	case errors.Is(err, domain.ErrSessionExpired), errors.Is(err, domain.ErrSessionRevoked):
		return ErrSessionExpired
	case errors.Is(err, domain.ErrAuthStoreUnavailable):
		return ErrInternalError
	default:
		return ErrInternalError
	}
}

func (s *AuthService) mapRecoveryConsumeError(err error) error {
	switch {
	case errors.Is(err, domain.ErrAuthStoreUnavailable):
		return ErrInternalError
	case errors.Is(err, domain.ErrRecoveryTokenNotFound),
		errors.Is(err, domain.ErrRecoveryTokenExpired),
		errors.Is(err, domain.ErrRecoveryTokenConsumed),
		errors.Is(err, domain.ErrRecoverySessionNotFound),
		errors.Is(err, domain.ErrRecoverySessionExpired),
		errors.Is(err, domain.ErrRecoverySessionConsumed):
		return ErrBadRequest
	default:
		return ErrBadRequest
	}
}

func otpKey(otp string) string {
	return "passkey-otp:" + strings.TrimSpace(otp)
}

func otpChallengeKey(otp string) string {
	return "passkey-otp-challenge:" + strings.TrimSpace(otp)
}

func (s *AuthService) mapAuthStoreError(err error) error {
	if errors.Is(err, domain.ErrAuthStoreUnavailable) {
		return ErrInternalError
	}
	if errors.Is(err, domain.ErrAuthAccountNotFound) {
		return ErrBadRequest
	}

	return ErrBadRequest
}
