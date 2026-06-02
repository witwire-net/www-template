package accounts

import (
	"context"
	"errors"

	domain "www-template/packages/backend/internal/domain"
)

// AccountSettingService は認証済み Product Account の AccountSetting を取得・更新する use case である。
//
// 役割:
//   - HTTP adapter から受け取った認証済み AccountID を使い、Product AccountSetting だけを読み書きする。
//   - locale の raw 文字列検証は domain.NewAccountLocale に委譲し、application 内の inline business validation を避ける。
//   - repository 欠落や保存層障害は ErrAccountSettingUnavailable へ fail-closed に写像する。
type AccountSettingService struct {
	repository AccountSettingRepository
}

// NewAccountSettingService は AccountSettingService を構築する。
//
// 引数:
//   - repository: AccountSetting の永続化 port。nil の場合、各 method は ErrAccountSettingUnavailable を返す。
//
// 戻り値:
//   - *AccountSettingService: repository を保持する use case instance。
func NewAccountSettingService(repository AccountSettingRepository) *AccountSettingService {
	return &AccountSettingService{repository: repository}
}

// Get は認証済み AccountID に紐づく AccountSetting を返す。
//
// 引数:
//   - ctx: 呼び出し単位のキャンセル・期限情報。
//   - accountID: bearer session から確定した Product Account の ID。
//
// 戻り値:
//   - AccountSetting: transport へ返す locale 設定 DTO。
//   - error: repository 欠落/障害は ErrAccountSettingUnavailable、不在は ErrAccountSettingNotFound、不正 ID は ErrInvalidAccountSetting。
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
//
// 引数:
//   - ctx: 呼び出し単位のキャンセル・期限情報。
//   - accountID: bearer session から確定した Product Account の ID。
//   - localeValue: client が保存したい locale 文字列。domain.NewAccountLocale で検証する。
//
// 戻り値:
//   - AccountSetting: 更新後の保存済み locale 設定 DTO。
//   - error: locale 不正は ErrInvalidAccountSetting、不在は ErrAccountSettingNotFound、保存層障害は ErrAccountSettingUnavailable。
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
