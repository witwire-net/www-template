package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

const defaultInfrastructureTimeout = 3 * time.Second

func OpenDatabase(databaseURL string) (*gorm.DB, error) {
	if strings.TrimSpace(databaseURL) == "" {
		return nil, errors.New("DATABASE_URL is required")
	}

	db, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("open gorm database: %w", err)
	}

	return db, nil
}

func PingDatabase(ctx context.Context, db *gorm.DB) error {
	if db == nil {
		return errors.New("gorm database is required")
	}

	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("open sql db: %w", err)
	}

	pingContext, cancel := context.WithTimeout(ctx, defaultInfrastructureTimeout)
	defer cancel()
	if err := sqlDB.PingContext(pingContext); err != nil {
		return fmt.Errorf("ping postgres: %w", err)
	}

	return nil
}
