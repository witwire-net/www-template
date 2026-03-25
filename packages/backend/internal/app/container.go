package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	"www-template/packages/backend/internal/domain"
	"www-template/packages/backend/internal/persistence"
	"www-template/packages/backend/internal/types"
	"www-template/packages/backend/internal/usecases"
)

const defaultReadHeaderTimeout = 5 * time.Second

type Container struct {
	Auth     *usecases.AuthService
	Profiles *usecases.ProfilesService
	close    func(context.Context) error
}

type authStateRepositoryFactory func(types.ValkeyConfig) (usecases.AuthStateRepository, error)
type valkeyCloser interface{ Close() error }

type rejectingInvitationPasskeyRegistrar struct{}

func (rejectingInvitationPasskeyRegistrar) RegisterInvitationPasskey(context.Context, usecases.InvitationPasskeyRegistrationInput) (usecases.AuthSession, error) {
	return usecases.AuthSession{}, usecases.ErrBadRequest
}

func BuildContainer(ctx context.Context, cfg types.Config) (*Container, error) {
	return buildContainer(ctx, cfg, func(valkeyCfg types.ValkeyConfig) (usecases.AuthStateRepository, error) {
		store, err := persistence.NewValkeyStore(valkeyCfg)
		if err != nil {
			return nil, err
		}
		return persistence.NewAuthStateRepository(store)
	})
}

func buildContainer(ctx context.Context, cfg types.Config, newAuthStateRepository authStateRepositoryFactory) (*Container, error) {
	_ = ctx
	authConfig := cfg.AuthRuntime()
	idPolicy := newAuthIDPolicy()

	var (
		profileRepo domain.ProfileRepository
		stateRepo   usecases.AuthStateRepository
		accountRepo usecases.AuthAccountRepository
		closeFn     = func(context.Context) error { return nil }
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

		profileRepo = persistence.NewGormProfileRepository(db)
		accountRepo = persistence.NewGormAuthAccountRepository(db)
		closeFn = func(context.Context) error {
			return sqlDB.Close()
		}
	} else {
		profileRepo = persistence.NewInMemoryProfileRepository()
		accountRepo = persistence.NewInMemoryAuthAccountRepository()
	}

	if strings.TrimSpace(cfg.Infra.Valkey.URL) != "" {
		valkeyRepo, err := newAuthStateRepository(cfg.Infra.Valkey)
		if err != nil {
			return nil, err
		}
		stateRepo = valkeyRepo
		if closer, ok := valkeyRepo.(valkeyCloser); ok {
			previousClose := closeFn
			closeFn = func(ctx context.Context) error {
				if err := closer.Close(); err != nil {
					return err
				}
				return previousClose(ctx)
			}
		}
	} else {
		stateRepo = persistence.NewInMemoryStateRepository(func() time.Time {
			return time.Now().UTC()
		})
	}
	smtpSender := NewSMTPSender(cfg.Infra)
	recoverySender := NewAccountRecoverySender(smtpSender, cfg.Infra)

	return &Container{
		Auth: usecases.NewAuthService(stateRepo, accountRepo, recoverySender, rejectingInvitationPasskeyRegistrar{}, func() time.Time {
			return time.Now().UTC()
		}, idPolicy, authConfig),
		Profiles: usecases.NewProfilesService(profileRepo, func() time.Time {
			return time.Now().UTC()
		}),
		close: closeFn,
	}, nil
}

func (c *Container) Close(ctx context.Context) error {
	return c.close(ctx)
}
