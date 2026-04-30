package app

import (
	"context"
	stdhttp "net/http"

	backendhttp "www-template/packages/backend/internal/http"
	"www-template/packages/backend/internal/observability"
	"www-template/packages/backend/internal/persistence"
	"www-template/packages/backend/internal/types"
)

type Runtime struct {
	config    types.Config
	container *Container
	server    *stdhttp.Server
	closeObs  func(context.Context) error
}

func NewRuntime(ctx context.Context) (*Runtime, error) {
	return NewRuntimeWithConfig(ctx, LoadConfig())
}

func NewRuntimeWithConfig(ctx context.Context, cfg types.Config) (*Runtime, error) {
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
	}

	return &Runtime{
		config:    cfg,
		container: container,
		server:    server,
		closeObs:  closeObs,
	}, nil
}

func verifyInfrastructure(ctx context.Context, cfg types.Config) error {
	db, err := persistence.OpenGormDatabase(cfg.Infra.Database.URL)
	if err != nil {
		return err
	}
	if err := persistence.PingGormDatabase(ctx, db); err != nil {
		return err
	}
	sqlDB, err := db.DB()
	if err == nil {
		defer func() {
			_ = sqlDB.Close()
		}()
	}

	store, err := persistence.NewValkeyStore(cfg.Infra.Valkey)
	if err != nil {
		return err
	}
	defer func() {
		_ = store.Close()
	}()
	if err := store.Ping(ctx); err != nil {
		return err
	}

	if err := persistence.CheckOpenSearch(ctx, cfg.Infra.OpenSearch); err != nil {
		return err
	}
	return persistence.CheckObjectStorage(ctx, cfg.Infra.ObjectStorage)
}

func (r *Runtime) Close(ctx context.Context) error {
	if r.closeObs != nil {
		_ = r.closeObs(ctx)
	}
	return r.container.Close(ctx)
}

func (r *Runtime) Config() types.Config {
	return r.config
}

func (r *Runtime) Server() *stdhttp.Server {
	return r.server
}
