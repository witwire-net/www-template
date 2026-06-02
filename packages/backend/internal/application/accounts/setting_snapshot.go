package accounts

import (
	"context"

	domain "www-template/packages/backend/internal/domain"
)

// AccountSettingSnapshotService は refresh response に合成する AccountSetting snapshot を読み込む use case である。
//
// 役割:
//   - Auth lifecycle が確定した AccountID から Product AccountSetting.locale を読み取り、refresh response 用の最小 snapshot に変換する。
//   - 認証や token rotation は扱わず、AccountSetting の read model composition だけを担当する。
//   - repository 欠落や保存層障害は ErrAccountSettingUnavailable へ fail-closed に写像する。
type AccountSettingSnapshotService struct {
	repository AccountSettingRepository
}

// NewAccountSettingSnapshotService は AccountSettingSnapshotService を構築する。
//
// 引数:
//   - repository: AccountSetting の永続化 port。nil の場合、Load は ErrAccountSettingUnavailable を返す。
//
// 戻り値:
//   - *AccountSettingSnapshotService: refresh response composition 用の use case instance。
func NewAccountSettingSnapshotService(repository AccountSettingRepository) *AccountSettingSnapshotService {
	return &AccountSettingSnapshotService{repository: repository}
}

// Load は AccountID から AccountSetting を読み取り、transport 合成用 snapshot DTO として返す。
//
// 引数:
//   - ctx: 呼び出し単位のキャンセル・期限情報。
//   - accountID: refresh rotation などで確定済みの Product Account ID。
//
// 戻り値:
//   - AccountSettingSnapshot: response に埋め込む locale snapshot。
//   - error: repository 欠落/障害は ErrAccountSettingUnavailable、不在は ErrAccountSettingNotFound、不正 ID は ErrInvalidAccountSetting。
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
