package app

import (
	"context"
	stdhttp "net/http"

	backendhttp "www-template/packages/backend/internal/adapters/http"
	"www-template/packages/backend/internal/adapters/persistence/postgres"
	"www-template/packages/backend/internal/adapters/persistence/valkey"
	"www-template/packages/backend/internal/platform/config"
	"www-template/packages/backend/internal/platform/health"
	"www-template/packages/backend/internal/platform/observability"
)

type Runtime struct {
	config    config.Config
	container *Container
	server    *stdhttp.Server
	closeObs  func(context.Context) error
}

func NewRuntime(ctx context.Context) (*Runtime, error) {
	return NewRuntimeWithConfig(ctx, config.LoadConfig())
}

func NewRuntimeWithConfig(ctx context.Context, cfg config.Config) (*Runtime, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	if err := verifyInfrastructure(ctx, cfg); err != nil {
		return nil, err
	}

	obs := cfg.Observability
	closeTracer, err := observability.InitTracer(ctx, obs.OTELExporterOTLPEndpoint, obs.OTELServiceName)
	if err != nil {
		return nil, err
	}

	closeMeter, err := observability.InitMeter(ctx, obs.OTELExporterOTLPEndpoint, obs.OTELServiceName)
	if err != nil {
		_ = closeTracer(ctx)
		return nil, err
	}

	closeObs := func(ctx context.Context) error {
		_ = closeTracer(ctx)
		_ = closeMeter(ctx)
		return nil
	}

	container, err := BuildContainer(ctx, cfg)
	if err != nil {
		_ = closeObs(ctx)
		return nil, err
	}

	handler := backendhttp.NewRouter(cfg, backendhttp.Dependencies{Auth: container.Auth})
	server := &stdhttp.Server{
		Addr:              ":" + cfg.Port,
		Handler:           handler,
		ReadHeaderTimeout: defaultReadHeaderTimeout,
		ReadTimeout:       cfg.ServerReadTimeout,
		WriteTimeout:      cfg.ServerWriteTimeout,
		IdleTimeout:       cfg.ServerIdleTimeout,
	}

	return &Runtime{
		config:    cfg,
		container: container,
		server:    server,
		closeObs:  closeObs,
	}, nil
}

func verifyInfrastructure(ctx context.Context, cfg config.Config) error {
	db, err := postgres.OpenDatabase(cfg.Infra.Database.URL)
	if err != nil {
		return err
	}
	if err := postgres.PingDatabase(ctx, db); err != nil {
		return err
	}
	sqlDB, err := db.DB()
	if err == nil {
		defer func() {
			_ = sqlDB.Close()
		}()
	}

	store, err := valkey.NewStore(cfg.Infra.Valkey)
	if err != nil {
		return err
	}
	defer func() {
		_ = store.Close()
	}()
	if err := store.Ping(ctx); err != nil {
		return err
	}

	if err := health.CheckOpenSearch(ctx, cfg.Infra.OpenSearch); err != nil {
		return err
	}
	return health.CheckObjectStorage(ctx, cfg.Infra.ObjectStorage)
}

func (r *Runtime) Close(ctx context.Context) error {
	if r.closeObs != nil {
		_ = r.closeObs(ctx)
	}
	return r.container.Close(ctx)
}

func (r *Runtime) Config() config.Config {
	return r.config
}

func (r *Runtime) Server() *stdhttp.Server {
	return r.server
}
