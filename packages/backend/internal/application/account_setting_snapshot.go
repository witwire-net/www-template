package application

import (
	"context"

	domain "www-template/packages/backend/internal/domain"
)

// AccountSettingSnapshotService は refresh response に合成する AccountSetting snapshot を読み込む use case である。
type AccountSettingSnapshotService struct {
	repository AccountSettingRepository
}

// NewAccountSettingSnapshotService は AccountSettingSnapshotService を構築する。
func NewAccountSettingSnapshotService(repository AccountSettingRepository) *AccountSettingSnapshotService {
	return &AccountSettingSnapshotService{repository: repository}
}

// Load は AccountID から AccountSetting を読み取り、transport 合成用 snapshot DTO として返す。
func (s *AccountSettingSnapshotService) Load(ctx context.Context, accountID domain.AccountID) (AccountSettingSnapshot, error) {
	if s == nil || s.repository == nil {
		return AccountSettingSnapshot{}, ErrAccountSettingUnavailable
	}
	setting, err := s.repository.Get(ctx, accountID)
	if err != nil {
		return AccountSettingSnapshot{}, mapRepositoryError(err)
	}
	return mapDomainSnapshot(setting.Snapshot()), nil
}
