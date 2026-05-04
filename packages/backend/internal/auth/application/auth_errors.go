package application

import (
	"errors"
	"strings"

	"www-template/packages/backend/internal/auth/domain"
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

// parseOpaqueTokenID は opaque token（"opaque-<tokenID>"）から tokenID を抽出する。
// 想定外の形式の場合はエラーを返す。
func parseOpaqueTokenID(token string) (string, error) {
	const prefix = "opaque-"
	if !strings.HasPrefix(token, prefix) {
		return "", errors.New("invalid token format")
	}
	tokenID := strings.TrimPrefix(token, prefix)
	if tokenID == "" {
		return "", errors.New("empty token id")
	}
	return tokenID, nil
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
