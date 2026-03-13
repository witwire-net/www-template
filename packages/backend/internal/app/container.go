package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	"witwire.net/www-template/packages/backend/internal/domain"
	"witwire.net/www-template/packages/backend/internal/persistence"
	"witwire.net/www-template/packages/backend/internal/types"
	"witwire.net/www-template/packages/backend/internal/usecases"
)

const defaultReadHeaderTimeout = 5 * time.Second

type Container struct {
	Profiles *usecases.ProfilesService
	close    func(context.Context) error
}

func BuildContainer(ctx context.Context, cfg types.Config) (*Container, error) {
	_ = ctx

	var (
		repo    domain.ProfileRepository
		closeFn = func(context.Context) error { return nil }
	)

	if strings.EqualFold(cfg.ProfileStore, "gorm") {
		db, err := persistence.OpenGormDatabase(cfg.DatabaseURL)
		if err != nil {
			return nil, err
		}

		sqlDB, err := db.DB()
		if err != nil {
			return nil, fmt.Errorf("open sql db: %w", err)
		}

		repo = persistence.NewGormProfileRepository(db)
		closeFn = func(context.Context) error {
			return sqlDB.Close()
		}
	} else {
		repo = persistence.NewMemoryProfileRepository()
	}

	return &Container{
		Profiles: usecases.NewProfilesService(repo, func() time.Time {
			return time.Now().UTC()
		}),
		close: closeFn,
	}, nil
}

func (c *Container) Close(ctx context.Context) error {
	return c.close(ctx)
}
