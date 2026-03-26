package app

import (
	"context"
	"time"

	"www-template/packages/backend/internal/persistence"
	"www-template/packages/backend/internal/types"
	"www-template/packages/backend/internal/usecases"
)

const defaultReadHeaderTimeout = 5 * time.Second

type Container struct {
	Auth  *usecases.AuthService
	close func(context.Context) error
}

type authAccountRepositoryFactory func(context.Context, string) (usecases.AuthAccountRepository, func(context.Context) error, error)
type authStateRepositoryFactory func(context.Context, types.ValkeyConfig) (usecases.AuthStateRepository, func(context.Context) error, error)

type rejectingInvitationPasskeyRegistrar struct{}

func (rejectingInvitationPasskeyRegistrar) RegisterInvitationPasskey(context.Context, usecases.InvitationPasskeyRegistrationInput) (usecases.AuthSession, error) {
	return usecases.AuthSession{}, usecases.ErrBadRequest
}

func BuildContainer(ctx context.Context, cfg types.Config) (*Container, error) {
	return buildContainer(ctx, cfg, newGormAuthAccountRepository, newValkeyAuthStateRepository)
}

func buildContainer(ctx context.Context, cfg types.Config, newAuthAccountRepository authAccountRepositoryFactory, newAuthStateRepository authStateRepositoryFactory) (*Container, error) {
	authConfig := cfg.AuthRuntime()
	idPolicy := newAuthIDPolicy()

	accountRepo, closeAccountRepo, err := newAuthAccountRepository(ctx, cfg.Infra.Database.URL)
	if err != nil {
		return nil, err
	}

	stateRepo, closeStateRepo, err := newAuthStateRepository(ctx, cfg.Infra.Valkey)
	if err != nil {
		_ = closeAccountRepo(ctx)
		return nil, err
	}

	smtpSender := NewSMTPSender(cfg.Infra)
	recoverySender := NewAccountRecoverySender(smtpSender, cfg.Infra)

	return &Container{
		Auth: usecases.NewAuthService(stateRepo, accountRepo, recoverySender, rejectingInvitationPasskeyRegistrar{}, func() time.Time {
			return time.Now().UTC()
		}, idPolicy, authConfig),
		close: composeClosers(closeStateRepo, closeAccountRepo),
	}, nil
}

func newGormAuthAccountRepository(ctx context.Context, databaseURL string) (usecases.AuthAccountRepository, func(context.Context) error, error) {
	db, err := persistence.OpenGormDatabase(databaseURL)
	if err != nil {
		return nil, nil, err
	}
	if err := persistence.PingGormDatabase(ctx, db); err != nil {
		sqlDB, dbErr := db.DB()
		if dbErr == nil {
			_ = sqlDB.Close()
		}
		return nil, nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, nil, err
	}

	return persistence.NewGormAuthAccountRepository(db), func(context.Context) error {
		return sqlDB.Close()
	}, nil
}

func newValkeyAuthStateRepository(ctx context.Context, config types.ValkeyConfig) (usecases.AuthStateRepository, func(context.Context) error, error) {
	store, err := persistence.NewValkeyStore(config)
	if err != nil {
		return nil, nil, err
	}
	if err := store.Ping(ctx); err != nil {
		_ = store.Close()
		return nil, nil, err
	}

	repo, err := persistence.NewAuthStateRepository(store)
	if err != nil {
		_ = store.Close()
		return nil, nil, err
	}

	return repo, func(context.Context) error {
		return repo.Close()
	}, nil
}

func composeClosers(closers ...func(context.Context) error) func(context.Context) error {
	return func(ctx context.Context) error {
		for _, closeFn := range closers {
			if closeFn == nil {
				continue
			}
			if err := closeFn(ctx); err != nil {
				return err
			}
		}
		return nil
	}
}

func (c *Container) Close(ctx context.Context) error {
	return c.close(ctx)
}
