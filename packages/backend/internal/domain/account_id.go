package domain

import (
	"errors"
	"strings"
)

var (
	// ErrInvalidAccountID は Product Account の canonical ULID が不正な場合に返すエラーである。
	ErrInvalidAccountID = errors.New("invalid account id")
)

// AccountID は Product Account を表す canonical ULID 値オブジェクトである。
//
// raw 文字列を直接レイヤー間で受け渡すと、Auth 側の識別子や session ID と混同しやすい。
// そのため flat な domain package が AccountID の正を提供し、Auth domain を含む他レイヤーは
// この型を import して Account への従属関係を明示する。
type AccountID string

// NewAccountID は raw 文字列を検証し、canonical ULID の AccountID として返す。
//
// raw は前後空白を除去した後、26 文字の Crockford Base32 ULID だけを受け付ける。
// 不正な値の場合は ErrInvalidAccountID を返し、呼び出し側が fail-closed に扱えるようにする。
func NewAccountID(raw string) (AccountID, error) {
	trimmed := strings.TrimSpace(raw)
	if len(trimmed) != 26 {
		return "", ErrInvalidAccountID
	}
	for _, r := range trimmed {
		if !strings.ContainsRune("0123456789ABCDEFGHJKMNPQRSTVWXYZ", r) {
			return "", ErrInvalidAccountID
		}
	}
	return AccountID(trimmed), nil
}

// String は AccountID を API、DB、JWT claim へ渡すための canonical 文字列へ変換する。
func (id AccountID) String() string {
	return string(id)
}
