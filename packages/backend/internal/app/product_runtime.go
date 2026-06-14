package app

import (
	"context"
	stdhttp "net/http"

	producthttp "www-template/packages/backend/internal/adapter/http/product"
	"www-template/packages/backend/internal/adapter/postgres"
	productvalkey "www-template/packages/backend/internal/adapter/valkey/product"
	"www-template/packages/backend/internal/platform/config"
	"www-template/packages/backend/internal/platform/health"
	"www-template/packages/backend/internal/platform/observability"
)

// ProductRuntime は Product Account 向け HTTP API サーバーの起動・停止・設定アクセスを束ねる runtime である。
//
// 役割:
//   - config、DI container、HTTP server、observability closer を保持し、bootstrap から graceful shutdown までの lifecycle を管理する。
//   - Admin runtime とは別型にし、Product account auth と Admin operator auth の credential や session store を混在させない。
//   - Command 層（cmd/api）から NewProductRuntime 経由で生成し、Config()/Server()/Close() だけを公開する。
//
// フィールド（非公開）:
//   - config: Product 向け runtime 設定（port、auth TTL、DB/Valkey/OpenSearch/ObjectStorage URL）。
//   - container: Account auth、setting、token refresh、session lifecycle を束ねる DI container。
//   - server: Gin engine を Handler に持つ *net/http.Server。
//   - closeObs: OTel tracer と meter の終了処理をまとめた関数。
//
// 使用例:
//
//	runtime, err := NewProductRuntime(ctx)
//	if err != nil {
//		return err
//	}
//	defer runtime.Close(ctx)
//	server := runtime.Server()
//	server.ListenAndServe()
type ProductRuntime struct {
	config    config.Config
	container *ProductContainer
	server    *stdhttp.Server
	closeObs  func(context.Context) error
}

// NewProductRuntime は環境変数から設定を読み込み、Product runtime を生成する。
//
// 内部で config.LoadConfig を呼び、NewProductRuntimeWithConfig へ委譲する。
// 設定ファイルの読み取りや環境変数の解決に失敗した場合は error を返す。
//
// 引数:
//   - ctx: OTel exporter 接続や infrastructure ping に使う context。
//
// 戻り値:
//   - *ProductRuntime: 起動準備済みの Product runtime。
//   - error: 設定読み取り、検証、infrastructure 接続、observability 初期化のいずれかが失敗した場合。
//
// 使用例:
//
//	runtime, err := NewProductRuntime(ctx)
//	if err != nil {
//		log.Fatalf("failed to create runtime: %v", err)
//	}
func NewProductRuntime(ctx context.Context) (*ProductRuntime, error) {
	return NewProductRuntimeWithConfig(ctx, config.LoadConfig())
}

// NewProductRuntimeWithConfig は明示的な設定値から Product runtime を生成する。
//
// テストや fail-close 検証のために設定を外部注入する際に使用する。
// cfg の検証、認証設定の検証、infrastructure の接続確認、OTel の初期化を順に行い、
// いずれかが失敗した場合は即座に error を返す（fail-fast）。
//
// 引数:
//   - ctx: infrastructure ping および OTel exporter 接続に使う context。
//   - cfg: Validate() 済みである必要がある Product 向け runtime 設定。
//
// 戻り値:
//   - *ProductRuntime: 全依存が解決済みで ListenAndServe 可能な runtime。
//   - error: 設定検証、インフラ接続、OTel 初期化、container 構築のいずれかが失敗した場合。
//
// 使用例:
//
//	runtime, err := NewProductRuntimeWithConfig(ctx, testConfig())
//	if err != nil {
//		t.Fatalf("unexpected error: %v", err)
//	}
func NewProductRuntimeWithConfig(ctx context.Context, cfg config.Config) (*ProductRuntime, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	if err := validateAuthConfig(cfg.Auth); err != nil {
		return nil, err
	}
	if err := verifyInfrastructure(ctx, cfg); err != nil {
		return nil, err
	}

	obs := cfg.Observability
	// Step 1: traces/metrics/logs の endpoint を起動前に解決し、設定の fallback と exporter の実利用先を一致させる。
	otlpEndpoints := observability.ResolveOTLPEndpoints(obs.OTELExporterOTLPEndpoint, obs.OTELExporterOTLPTracesEndpoint, obs.OTELExporterOTLPLogsEndpoint)
	// Step 2: SigNoz collector が OTLP gRPC を受けられない状態なら fail-fast し、実行中の exporter retry ログを発生させない。
	if err := observability.VerifyOTLPEndpoints(ctx, otlpEndpoints); err != nil {
		return nil, err
	}

	// Step 3: 検証済み traces endpoint で tracer を初期化し、Product API の trace を SigNoz に送信する。
	closeTracer, err := observability.InitTracer(ctx, otlpEndpoints.Traces, obs.OTELServiceName)
	if err != nil {
		return nil, err
	}

	// Step 4: 検証済み metrics endpoint で meter を初期化し、runtime metrics を SigNoz に送信する。
	closeMeter, err := observability.InitMeter(ctx, otlpEndpoints.Metrics, obs.OTELServiceName)
	if err != nil {
		_ = closeTracer(ctx)
		return nil, err
	}

	// Step 5: 検証済み logs endpoint で logger を初期化し、stdout と SigNoz の両方へ backend logs を送信する。
	closeLogger, err := observability.InitLogger(ctx, otlpEndpoints.Logs, obs.OTELServiceName)
	if err != nil {
		_ = closeMeter(ctx)
		_ = closeTracer(ctx)
		return nil, err
	}

	closeObs := func(ctx context.Context) error {
		_ = closeLogger(ctx)
		_ = closeTracer(ctx)
		_ = closeMeter(ctx)
		return nil
	}

	container, err := BuildProductContainer(ctx, cfg)
	if err != nil {
		_ = closeObs(ctx)
		return nil, err
	}

	handler := producthttp.NewRouter(cfg, producthttp.Dependencies{
		Auth:            container.AccountAuth,
		AccountSetting:  container.AccountSetting,
		AccountSnapshot: container.AccountSnapshot,
		TokenService:    container.AccountContextRefresh,
		SessionService:  container.AccountSessions,
	})
	server := &stdhttp.Server{
		Addr:              ":" + cfg.Port,
		Handler:           handler,
		ReadHeaderTimeout: defaultReadHeaderTimeout,
		ReadTimeout:       cfg.ServerReadTimeout,
		WriteTimeout:      cfg.ServerWriteTimeout,
		IdleTimeout:       cfg.ServerIdleTimeout,
	}

	return &ProductRuntime{
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

	store, err := productvalkey.NewStore(cfg.Infra.Valkey)
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

// Close は Product runtime の全リソースを解放する。
//
// OTel tracer/meter の shutdown を実行し、次に ProductContainer.Close を呼んで
// Valkey/DB 接続を閉じる。いずれかの close でエラーが発生しても、後続の close は継続する。
// nil guard は r.closeObs に対して行い、r.container が nil の場合は container.Close が nil pointer dereference を起こし得る。
//
// 引数:
//   - ctx: shutdown 処理の deadline を制御する context。
//
// 戻り値:
//   - error: ProductContainer.Close が返したエラー。OTel close のエラーは破棄する。
//
// 使用例:
//
//	defer runtime.Close(ctx)
func (r *ProductRuntime) Close(ctx context.Context) error {
	if r.closeObs != nil {
		_ = r.closeObs(ctx)
	}
	return r.container.Close(ctx)
}

// Config は Product runtime の設定を返す。
//
// 戻り値は起動時に検証済みの設定であり、読み取り専用として利用する。
// 設定値の変更は runtime の再起動なしには反映されない。
//
// 戻り値:
//   - config.Config: port、auth TTL、DB/Valkey 接続情報などを含む検証済み設定。
func (r *ProductRuntime) Config() config.Config {
	return r.config
}

// Server は Product HTTP server の *net/http.Server を返す。
//
// cmd/api が ListenAndServe または Shutdown を呼ぶために使用する。
// server は NewProductRuntimeWithConfig の内部で Gin engine を Handler に設定済みである。
//
// 戻り値:
//   - *net/http.Server: Addr、Handler、timeout 設定済みの HTTP server。
func (r *ProductRuntime) Server() *stdhttp.Server {
	return r.server
}
