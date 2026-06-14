package app

import (
	"context"
	"errors"
	stdhttp "net/http"
	"time"

	adminhttp "www-template/packages/backend/internal/adapter/http/admin"
	"www-template/packages/backend/internal/adapter/postgres"
	"www-template/packages/backend/internal/platform/config"
	"www-template/packages/backend/internal/platform/health"
	"www-template/packages/backend/internal/platform/observability"
)

const defaultAdminOperatorSetupTokenTTL = 24 * time.Hour

// AdminRuntime は Admin API 専用 binary の runtime 構成を保持する。
// Product Runtime と別型にすることで、Product application container や Product HTTP router を誤って共有しない。
// config は起動時に検証済みの設定値、server は Admin binary が所有する HTTP server、closeObs は tracer / meter の解放関数である。
type AdminRuntime struct {
	config    config.Config
	container *AdminContainer
	server    *stdhttp.Server
	closeObs  func(context.Context) error
}

// NewAdminRuntime は repository 設定ファイルから Admin API runtime を構築する。
// ctx は startup 時の infrastructure 検証と observability 初期化に使う。
// 戻り値は Admin API 専用 runtime であり、Product runtime の container / router は含まない。
// 設定ファイルが見つからない場合、または必須 infrastructure 検証に失敗した場合は error を返す前に panic する可能性がある LoadAdminConfig の挙動を引き継ぐ。
func NewAdminRuntime(ctx context.Context) (*AdminRuntime, error) {
	// Step 1: Product binary 用 loader ではなく Admin 専用 loader を使い、Admin domain / cookie / Valkey URL を必須入力として読み込む。
	return NewAdminRuntimeWithConfig(ctx, config.LoadAdminConfig())
}

// NewAdminRuntimeWithConfig は検証済み設定から Admin API runtime を構築する。
// ctx は database / OpenSearch の到達性確認と observability 初期化に使う。
// cfg は Admin binary 用に渡された設定であり、Product runtime へは渡さない。
// 戻り値は Admin 専用 HTTP server を持つ runtime で、Product handlers を登録しない。
// 設定検証、認証 TTL 検証、infrastructure 検証、observability 初期化のいずれかに失敗した場合は error を返す。
func NewAdminRuntimeWithConfig(ctx context.Context, cfg config.Config) (*AdminRuntime, error) {
	// Step 1: Admin surface 固有の domain / cookie / DB role / Valkey URL を検証し、Product 設定だけで Admin binary が起動することを防ぐ。
	if err := cfg.ValidateAdminRuntime(); err != nil {
		return nil, err
	}

	// Step 2: Product と同じ認証 TTL policy を Admin binary でも起動時に守り、短すぎる refresh token 設定を拒否する。
	if err := validateAuthConfig(cfg.Auth); err != nil {
		return nil, err
	}

	// Step 3: Admin binary が依存する backend infrastructure の到達性を起動前に検証し、Product 専用 object storage 等へ依存しない。
	if err := verifyAdminInfrastructure(ctx, cfg); err != nil {
		return nil, err
	}

	// Step 4: Admin API process 用の tracer / meter / logger を初期化し、startup 以後の観測情報を収集できる状態にする。
	obs := cfg.Observability
	// Step 4-1: traces/metrics/logs の endpoint を起動前に解決し、設定の fallback と exporter の実利用先を一致させる。
	otlpEndpoints := observability.ResolveOTLPEndpoints(obs.OTELExporterOTLPEndpoint, obs.OTELExporterOTLPTracesEndpoint, obs.OTELExporterOTLPLogsEndpoint)
	// Step 4-2: SigNoz collector が OTLP gRPC を受けられない状態なら fail-fast し、実行中の exporter retry ログを発生させない。
	if err := observability.VerifyOTLPEndpoints(ctx, otlpEndpoints); err != nil {
		return nil, err
	}

	// Step 4-3: 検証済み traces endpoint で tracer を初期化し、Admin API の trace を SigNoz に送信する。
	closeTracer, err := observability.InitTracer(ctx, otlpEndpoints.Traces, obs.OTELServiceName)
	if err != nil {
		return nil, err
	}

	// Step 4-4: 検証済み metrics endpoint で meter を初期化し、Admin runtime metrics を SigNoz に送信する。
	closeMeter, err := observability.InitMeter(ctx, otlpEndpoints.Metrics, obs.OTELServiceName)
	if err != nil {
		_ = closeTracer(ctx)
		return nil, err
	}

	// Step 4-5: 検証済み logs endpoint で logger を初期化し、stdout と SigNoz の両方へ Admin backend logs を送信する。
	closeLogger, err := observability.InitLogger(ctx, otlpEndpoints.Logs, obs.OTELServiceName)
	if err != nil {
		_ = closeMeter(ctx)
		_ = closeTracer(ctx)
		return nil, err
	}

	// Step 5: tracer / meter / logger の close を一つの関数へまとめ、cmd/admin-api から安全に解放できるようにする。
	closeObs := func(ctx context.Context) error {
		_ = closeLogger(ctx)
		_ = closeTracer(ctx)
		_ = closeMeter(ctx)
		return nil
	}

	// Step 6: Admin account 管理 use case と audit projection を runtime で構成し、HTTP adapter へ具象 repository を直接持ち込まない。
	container, err := BuildAdminContainer(ctx, cfg)
	if err != nil {
		_ = closeObs(ctx)
		return nil, err
	}

	// Step 7: Admin 専用 HTTP adapter を設定し、Product router / Product generated bindings を Admin binary に持ち込まない。
	server := &stdhttp.Server{
		Addr: ":" + cfg.Port,
		Handler: adminhttp.NewRouter(cfg, adminhttp.Dependencies{
			OperatorSessions:          adminhttp.NewOperatorSessionValidator(container.OperatorAuth),
			OperatorAuth:              container.OperatorAuth,
			OperatorPasskeyAuth:       container.OperatorPasskeyLogin,
			OperatorPasskeyVerifier:   container.OperatorPasskeyVerifier,
			OperatorSetup:             container.OperatorSetup,
			OperatorPasskeyManagement: container.OperatorPasskeys,
			AccountCreation:           container.AccountCreation,
			AccountSearch:             container.AccountSearch,
		}),
		ReadHeaderTimeout: defaultReadHeaderTimeout,
		ReadTimeout:       cfg.ServerReadTimeout,
		WriteTimeout:      cfg.ServerWriteTimeout,
		IdleTimeout:       cfg.ServerIdleTimeout,
	}

	return &AdminRuntime{
		config:    cfg,
		container: container,
		server:    server,
		closeObs:  closeObs,
	}, nil
}

func verifyAdminInfrastructure(ctx context.Context, cfg config.Config) error {
	// Step 1: Admin account management と operator persistence が使う DB URL が空なら、DB 接続前に fail-close する。
	if cfg.Infra.Database.URL == "" {
		return errors.New("database url is required")
	}

	// Step 2: Admin runtime が使用する DB へ接続し、起動前に到達性を確認する。
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

	// Step 3: Admin audit projection の接続先が未設定または到達不能な場合、観測不能な Admin mutation を避けるため起動を止める。
	return health.CheckOpenSearch(ctx, cfg.Infra.OpenSearch)
}

// Close は Admin runtime が確保した observability resource と application container を解放する。
//
// 役割:
//   - OTel tracer / meter を解放し、観測データの送信を停止する。
//   - application container が管理する DB 接続と Admin Valkey 接続を閉じる。
//
// 引数:
//   - ctx: shutdown の deadline / cancel を伝えるためのコンテキスト。
//
// 戻り値:
//   - container close が返した最初のエラー。OTel close のエラーは process 終了を妨げないよう吸収する。
//
// エラーケース:
//   - container close が失敗した場合はそのエラーを返す。
//   - closeObs のエラーは常に吸収し、戻り値に含めない。
//
// 使用例:
//
//	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
//	defer cancel()
//	if err := runtime.Close(ctx); err != nil {
//		slog.Error("admin runtime close failed", "error", err)
//	}
func (r *AdminRuntime) Close(ctx context.Context) error {
	// Step 1: observability 初期化前に呼ばれても安全なよう nil を確認し、二次障害を避ける。
	if r.closeObs != nil {
		_ = r.closeObs(ctx)
	}
	if r.container != nil && r.container.close != nil {
		return r.container.close(ctx)
	}
	return nil
}

// Config は Admin runtime の起動に使われた設定値を返す。
// 戻り値は値コピーであり、呼び出し側の変更が runtime 内部状態へ副作用を与えない。
func (r *AdminRuntime) Config() config.Config {
	return r.config
}

// Server は Admin API 専用 HTTP server を返す。
// 戻り値の server は cmd/admin-api が ListenAndServe と Shutdown を管理するために使う。
func (r *AdminRuntime) Server() *stdhttp.Server {
	return r.server
}
