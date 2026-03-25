package persistence

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"www-template/packages/backend/internal/domain"
)

type gormProfileRecord struct {
	ID        int64     `gorm:"column:id;primaryKey;autoIncrement"`
	Name      string    `gorm:"column:name"`
	Email     string    `gorm:"column:email"`
	CreatedAt time.Time `gorm:"column:created_at"`
}

func (gormProfileRecord) TableName() string {
	return "profiles"
}

type GormProfileRepository struct {
	db *gorm.DB
}

func OpenGormDatabase(databaseURL string) (*gorm.DB, error) {
	if strings.TrimSpace(databaseURL) == "" {
		return nil, errors.New("DATABASE_URL is required when APP_PROFILE_STORE=gorm")
	}

	db, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("open gorm database: %w", err)
	}

	return db, nil
}

func NewGormProfileRepository(db *gorm.DB) *GormProfileRepository {
	return &GormProfileRepository{db: db}
}

func (r *GormProfileRepository) List(ctx context.Context) ([]domain.Profile, error) {
	records := make([]gormProfileRecord, 0)
	if err := r.db.WithContext(ctx).Order("id ASC").Find(&records).Error; err != nil {
		return nil, fmt.Errorf("list profiles: %w", err)
	}

	profiles := make([]domain.Profile, 0, len(records))
	for _, record := range records {
		profile, err := toDomainProfile(record)
		if err != nil {
			return nil, fmt.Errorf("reconstitute profile %d: %w", record.ID, err)
		}

		profiles = append(profiles, profile)
	}

	return profiles, nil
}

func (r *GormProfileRepository) GetByID(ctx context.Context, id int64) (domain.Profile, error) {
	var record gormProfileRecord
	if err := r.db.WithContext(ctx).First(&record, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			var empty domain.Profile
			return empty, domain.ErrProfileNotFound
		}

		var empty domain.Profile
		return empty, fmt.Errorf("get profile: %w", err)
	}

	profile, err := toDomainProfile(record)
	if err != nil {
		var empty domain.Profile
		return empty, fmt.Errorf("reconstitute profile %d: %w", record.ID, err)
	}

	return profile, nil
}

func (r *GormProfileRepository) Create(ctx context.Context, input domain.CreateProfileInput) (domain.Profile, error) {
	record := gormProfileRecord{
		Email: input.Email(),
		Name:  input.Name(),
	}
	if err := r.db.WithContext(ctx).Create(&record).Error; err != nil {
		var empty domain.Profile
		return empty, fmt.Errorf("create profile: %w", err)
	}

	profile, err := toDomainProfile(record)
	if err != nil {
		var empty domain.Profile
		return empty, fmt.Errorf("reconstitute profile %d: %w", record.ID, err)
	}

	return profile, nil
}

func toDomainProfile(record gormProfileRecord) (domain.Profile, error) {
	return domain.ReconstituteProfile(record.ID, record.Email, record.Name, record.CreatedAt)
}
