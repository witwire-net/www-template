package domain

import (
	"errors"
	"regexp"
	"strings"
)

var (
	// ErrInvalidAuthID は認証フローで使用される ID（challenge ID, session ID, token ID 等）が
	// 有効な ULID 形式でない場合に返すエラー。
	ErrInvalidAuthID = errors.New("auth id must be a valid ULID")
	ulidPattern      = regexp.MustCompile(`^[0-9A-HJKMNP-TV-Z]{26}$`)
)

// ValidateAuthID は認証フローで使用される ID が有効な ULID 形式であることを検証する。
// 前後の空白は除去してから判定する。有効な場合は nil を返し、無効な場合は ErrInvalidAuthID を返す。
func ValidateAuthID(value string) error {
	if !ulidPattern.MatchString(strings.TrimSpace(value)) {
		return ErrInvalidAuthID
	}

	return nil
}
