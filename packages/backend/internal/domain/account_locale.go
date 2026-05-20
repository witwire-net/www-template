package domain

import (
	"errors"
	"strings"
)

var (
	// ErrInvalidAccountLocale は AccountSetting.locale が対応値ではない場合に返すエラーである。
	ErrInvalidAccountLocale = errors.New("invalid account locale")
	// ErrAccountSettingNotFound は Account に属する AccountSetting が永続化層に存在しない場合に返すエラーである。
	ErrAccountSettingNotFound = errors.New("account setting not found")
)

// AccountLocale は Product Account の表示・通知に使う対応ロケール値である。
type AccountLocale string

const (
	// AccountLocaleJapanese は日本語表示・日本語通知を表す AccountSetting.locale である。
	AccountLocaleJapanese AccountLocale = "ja"
	// AccountLocaleEnglish は英語表示・英語通知を表す AccountSetting.locale である。
	AccountLocaleEnglish AccountLocale = "en"
)

// NewAccountLocale は入力文字列を正規化し、対応済み AccountLocale として返す。
func NewAccountLocale(value string) (AccountLocale, error) {
	switch AccountLocale(strings.TrimSpace(value)) {
	case AccountLocaleJapanese:
		return AccountLocaleJapanese, nil
	case AccountLocaleEnglish:
		return AccountLocaleEnglish, nil
	default:
		return "", ErrInvalidAccountLocale
	}
}

// DefaultAccountLocale は新規 Product Account に設定する既定 locale を返す。
func DefaultAccountLocale() AccountLocale {
	return AccountLocaleJapanese
}

// String は AccountLocale を API / DB に保存する文字列へ変換する。
func (l AccountLocale) String() string {
	return string(l)
}

func validateAccountID(id string) error {
	_, err := NewAccountID(id)
	return err
}
