package app

import (
	"context"
	stdhttp "net/http"

	backendhttp "www-template/packages/backend/internal/http"
	"www-template/packages/backend/internal/types"
)

type Runtime struct {
	config    types.Config
	container *Container
	server    *stdhttp.Server
}

func NewRuntime(ctx context.Context) (*Runtime, error) {
	return NewRuntimeWithConfig(ctx, types.LoadConfig())
}

func NewRuntimeWithConfig(ctx context.Context, cfg types.Config) (*Runtime, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	container, err := BuildContainer(ctx, cfg)
	if err != nil {
		return nil, err
	}

	handler := backendhttp.NewRouter(cfg, backendhttp.Dependencies{Auth: container.Auth, Profiles: container.Profiles})
	server := &stdhttp.Server{
		Addr:              ":" + cfg.Port,
		Handler:           handler,
		ReadHeaderTimeout: defaultReadHeaderTimeout,
	}

	return &Runtime{
		config:    cfg,
		container: container,
		server:    server,
	}, nil
}

func (r *Runtime) Close(ctx context.Context) error {
	return r.container.Close(ctx)
}

func (r *Runtime) Config() types.Config {
	return r.config
}

func (r *Runtime) Server() *stdhttp.Server {
	return r.server
}
