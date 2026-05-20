package application

import (
	"context"
	"errors"

	domain "www-template/packages/backend/internal/domain"
)

var (
	// ErrAccountSettingNotFound は AccountSetting が存在しない場合に返す application error である。
	ErrAccountSettingNotFound = errors.New("account setting not found")
	// ErrInvalidAccountSetting は AccountSetting 更新入力が不正な場合に返す application error である。
	ErrInvalidAccountSetting = errors.New("invalid account setting")
	// ErrAccountSettingUnavailable は永続化層などが利用できない場合に返す application error である。
	ErrAccountSettingUnavailable = errors.New("account setting unavailable")
)

// AccountSetting は transport へ返す Product AccountSetting DTO である。
type AccountSetting struct {
	AccountID domain.AccountID
	Locale    string
}

// AccountSettingSnapshot は refresh response に合成する Product AccountSetting snapshot DTO である。
type AccountSettingSnapshot struct {
	Locale string
}

// AccountSettingRepository は AccountSetting の永続化を抽象化する port である。
type AccountSettingRepository interface {
	CreateDefault(ctx context.Context, accountID domain.AccountID) (domain.AccountSetting, error)
	Get(ctx context.Context, accountID domain.AccountID) (domain.AccountSetting, error)
	UpdateLocale(ctx context.Context, accountID domain.AccountID, locale domain.AccountLocale) (domain.AccountSetting, error)
}

func mapDomainSetting(setting domain.AccountSetting) AccountSetting {
	return AccountSetting{AccountID: setting.AccountID(), Locale: setting.Locale().String()}
}

func mapDomainSnapshot(snapshot domain.AccountSettingSnapshot) AccountSettingSnapshot {
	return AccountSettingSnapshot{Locale: snapshot.Locale().String()}
}
