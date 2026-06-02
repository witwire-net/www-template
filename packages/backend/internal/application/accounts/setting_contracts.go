package accounts

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
//
// 役割:
//   - 認証済み Product Account に属する保存済み表示・通知設定を application 境界で表す。
//   - AccountID は bearer session から確定した所有者、Locale は domain.AccountLocale を文字列化した保存値である。
//   - Auth 情報や Admin operator 設定は含めず、AccountSetting 専用 use case の出力に限定する。
type AccountSetting struct {
	AccountID domain.AccountID
	Locale    string
}

// AccountSettingSnapshot は refresh response に合成する Product AccountSetting snapshot DTO である。
//
// 役割:
//   - refresh token rotation 成功後、Product UI が現在 locale を同期できるよう最小値だけを返す。
//   - AccountID は caller context で既に確定しているため含めず、transport 合成用の Locale だけを保持する。
type AccountSettingSnapshot struct {
	Locale string
}

// AccountSettingRepository は AccountSetting の永続化を抽象化する port である。
//
// 役割:
//   - Product AccountSetting の作成・取得・locale 更新を application 層から GORM/SQL 実装へ直接依存させない。
//   - accountID は domain.AccountID、locale は domain.AccountLocale として受け取り、repository 側で raw 入力検証を再実装しない。
//   - not found / invalid / store unavailable は domain error または application error として返し、service が stable error へ写像する。
type AccountSettingRepository interface {
	// CreateDefault は Account 作成時に既定 locale の AccountSetting を保存する。
	// ctx は保存単位のキャンセル情報、accountID は対象 Product Account ID である。
	// 保存層障害や不正 ID の場合は error を返す。
	CreateDefault(ctx context.Context, accountID domain.AccountID) (domain.AccountSetting, error)
	// Get は AccountID に紐づく AccountSetting を取得する。
	// 不在時は domain.ErrAccountSettingNotFound、保存層障害時は実装固有または domain の error を返す。
	Get(ctx context.Context, accountID domain.AccountID) (domain.AccountSetting, error)
	// UpdateLocale は AccountID に紐づく AccountSetting.locale を更新して保存後の domain object を返す。
	// locale は検証済み domain.AccountLocale を受け取り、不在・保存層障害・不正 ID の場合は error を返す。
	UpdateLocale(ctx context.Context, accountID domain.AccountID, locale domain.AccountLocale) (domain.AccountSetting, error)
}

func mapDomainSetting(setting domain.AccountSetting) AccountSetting {
	return AccountSetting{AccountID: setting.AccountID(), Locale: setting.Locale().String()}
}

func mapDomainSnapshot(snapshot domain.AccountSettingSnapshot) AccountSettingSnapshot {
	return AccountSettingSnapshot{Locale: snapshot.Locale().String()}
}
