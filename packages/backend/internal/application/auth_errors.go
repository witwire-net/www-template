package application

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	domain "www-template/packages/backend/internal/domain"
)

var (
	ErrUnauthenticated            = errors.New("unauthenticated")
	ErrSessionExpired             = errors.New("session-expired")
	ErrInternalError              = errors.New("internal-error")
	ErrBadRequest                 = errors.New("bad auth request")
	ErrLastPasskeyCannotBeDeleted = errors.New("last passkey cannot be deleted")
	// ErrAccountSuspended はアカウントが管理者により停止されている場合に返すエラー。
	// 停止中アカウントに対しては新規トークン発行、セッション認可、refresh rotation を拒否する。
	// HTTP レスポンスでは 403 Forbidden + error="account-suspended" で返す。
	ErrAccountSuspended = errors.New("account-suspended")
)

// generateURLToken は tokenID に基づいて独立した暗号論的乱数 secret を生成し、
// URL token（tokenID.hexSecret 形式）と平文 secret を返す。
// secret は crypto/rand による 32 バイトの乱数を hex エンコードしたもの（64 文字）。
// URL token はメールに埋め込まれ、平文 secret は Valkey に HMAC ハッシュとして保存される。
// 生成に失敗した場合は error を返す。
func generateURLToken(tokenID string) (string, string, error) {
	secretBytes := make([]byte, 32)
	if _, err := rand.Read(secretBytes); err != nil {
		return "", "", fmt.Errorf("generateURLToken: crypto/rand.Read: %w", err)
	}
	plainSecret := hex.EncodeToString(secretBytes)
	urlToken := tokenID + "." + plainSecret
	return urlToken, plainSecret, nil
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

// parseURLToken は URL token（"tokenID.secret" 形式）から tokenID と平文 secret を抽出する。
// 想定外の形式の場合はエラーを返す。
func parseURLToken(token string) (string, string, error) {
	dotIdx := strings.Index(token, ".")
	if dotIdx < 1 {
		return "", "", errors.New("invalid token format")
	}
	tokenID := token[:dotIdx]
	secret := token[dotIdx+1:]
	if tokenID == "" || secret == "" {
		return "", "", errors.New("invalid token format")
	}
	return tokenID, secret, nil
}

func (s *AuthService) mapAuthStoreError(err error) error {
	if errors.Is(err, domain.ErrAuthStoreUnavailable) {
		return ErrInternalError
	}
	if errors.Is(err, domain.ErrAccountAuthNotFound) {
		return ErrBadRequest
	}

	return ErrBadRequest
}
