package application

import (
	"context"
	"errors"

	domain "www-template/packages/backend/internal/domain"
)

// AccountSettingService は認証済み Product Account の AccountSetting を取得・更新する use case である。
type AccountSettingService struct {
	repository AccountSettingRepository
}

// NewAccountSettingService は AccountSettingService を構築する。
func NewAccountSettingService(repository AccountSettingRepository) *AccountSettingService {
	return &AccountSettingService{repository: repository}
}

// Get は認証済み AccountID に紐づく AccountSetting を返す。
func (s *AccountSettingService) Get(ctx context.Context, accountID domain.AccountID) (AccountSetting, error) {
	if s == nil || s.repository == nil {
		return AccountSetting{}, ErrAccountSettingUnavailable
	}
	setting, err := s.repository.Get(ctx, accountID)
	if err != nil {
		return AccountSetting{}, mapRepositoryError(err)
	}
	return mapDomainSetting(setting), nil
}

// Update は認証済み AccountID に紐づく AccountSetting.locale を更新して返す。
func (s *AccountSettingService) Update(ctx context.Context, accountID domain.AccountID, localeValue string) (AccountSetting, error) {
	if s == nil || s.repository == nil {
		return AccountSetting{}, ErrAccountSettingUnavailable
	}
	locale, err := domain.NewAccountLocale(localeValue)
	if err != nil {
		return AccountSetting{}, ErrInvalidAccountSetting
	}
	setting, err := s.repository.UpdateLocale(ctx, accountID, locale)
	if err != nil {
		return AccountSetting{}, mapRepositoryError(err)
	}
	return mapDomainSetting(setting), nil
}

func mapRepositoryError(err error) error {
	if errors.Is(err, domain.ErrInvalidAccountLocale) || errors.Is(err, domain.ErrInvalidAccountID) {
		return ErrInvalidAccountSetting
	}
	if errors.Is(err, domain.ErrAccountSettingNotFound) || errors.Is(err, ErrAccountSettingNotFound) {
		return ErrAccountSettingNotFound
	}
	return ErrAccountSettingUnavailable
}
