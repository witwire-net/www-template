package postgres

import (
	"context"
	"errors"

	"gorm.io/gorm"

	domain "www-template/packages/backend/internal/domain"
)

type gormAccountSettingRecord struct {
	AccountID string `gorm:"column:account_id;primaryKey"`
	Locale    string `gorm:"column:locale"`
}

func (gormAccountSettingRecord) TableName() string {
	return "account_settings"
}

// GormAccountSettingRepository は Product AccountSetting を PostgreSQL に保存する repository adapter である。
type GormAccountSettingRepository struct {
	db *gorm.DB
}

// NewGormAccountSettingRepository は GormAccountSettingRepository を構築する。
func NewGormAccountSettingRepository(db *gorm.DB) *GormAccountSettingRepository {
	return &GormAccountSettingRepository{db: db}
}

// Get は account_settings から AccountID に紐づく AccountSetting を読み込む。
func (r *GormAccountSettingRepository) Get(ctx context.Context, accountID domain.AccountID) (domain.AccountSetting, error) {
	var record gormAccountSettingRecord
	if err := r.db.WithContext(ctx).Where("account_id = ?", accountID.String()).First(&record).Error; err != nil {
		return emptyAccountSetting(), mapAccountSettingError(err)
	}
	return record.toDomain()
}

// CreateDefault は Account 作成時に同じ Account の必須 child として既定 AccountSetting を作成する。
func (r *GormAccountSettingRepository) CreateDefault(ctx context.Context, accountID domain.AccountID) (domain.AccountSetting, error) {
	setting, err := domain.NewDefaultAccountSetting(accountID)
	if err != nil {
		return emptyAccountSetting(), err
	}
	record := gormAccountSettingRecord{AccountID: setting.AccountID().String(), Locale: setting.Locale().String()}
	if err := r.db.WithContext(ctx).Create(&record).Error; err != nil {
		return emptyAccountSetting(), mapAccountSettingError(err)
	}
	return setting, nil
}

// UpdateLocale は AccountSetting.locale を対応済み locale に更新し、更新後の値を返す。
func (r *GormAccountSettingRepository) UpdateLocale(ctx context.Context, accountID domain.AccountID, locale domain.AccountLocale) (domain.AccountSetting, error) {
	result := r.db.WithContext(ctx).Model(&gormAccountSettingRecord{}).
		Where("account_id = ?", accountID.String()).
		Update("locale", locale.String())
	if result.Error != nil {
		return emptyAccountSetting(), mapAccountSettingError(result.Error)
	}
	if result.RowsAffected == 0 {
		return emptyAccountSetting(), domain.ErrAccountSettingNotFound
	}
	return r.Get(ctx, accountID)
}

func (r gormAccountSettingRecord) toDomain() (domain.AccountSetting, error) {
	locale, err := domain.NewAccountLocale(r.Locale)
	if err != nil {
		return emptyAccountSetting(), err
	}
	accountID, err := domain.NewAccountID(r.AccountID)
	if err != nil {
		return emptyAccountSetting(), err
	}
	return domain.NewAccountSetting(accountID, locale)
}

func emptyAccountSetting() domain.AccountSetting {
	accountID, _ := domain.NewAccountID("01ARZ3NDEKTSV4RRFFQ69G5FAV")
	setting, _ := domain.NewDefaultAccountSetting(accountID)
	return setting
}

func mapAccountSettingError(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return domain.ErrAccountSettingNotFound
	}
	if err != nil {
		return err
	}
	return err
}
