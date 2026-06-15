package app

import (
	"context"

	"www-template/packages/backend/internal/platform/config"
	"www-template/packages/backend/internal/platform/observability"
)

// initRuntimeObservability は Product/Admin runtime 共通の OTel tracer/meter/logger を初期化する。
// infrastructure health check より前に呼ぶことで、startup の DB/Valkey/OpenSearch I/O も SigNoz へ記録できる。
//
// 引数:
//   - ctx: OTLP endpoint 検証と exporter 初期化の cancellation を伝える context。
//   - obs: 設定ファイルから読み込んだ observability 設定。
//
// 戻り値:
//   - func(context.Context) error: tracer/meter/logger をまとめて shutdown する closer。
//   - error: OTLP endpoint 到達性または exporter 初期化に失敗した場合。
func initRuntimeObservability(ctx context.Context, obs config.ObservabilityConfig) (func(context.Context) error, error) {
	// Step 1: traces/metrics/logs の endpoint を起動前に解決し、設定 fallback と exporter の実利用先を一致させる。
	otlpEndpoints := observability.ResolveOTLPEndpoints(obs.OTELExporterOTLPEndpoint, obs.OTELExporterOTLPTracesEndpoint, obs.OTELExporterOTLPLogsEndpoint)

	// Step 2: SigNoz collector が OTLP gRPC を受けられない状態なら fail-fast し、実行中の exporter retry ログを発生させない。
	if err := observability.VerifyOTLPEndpoints(ctx, otlpEndpoints); err != nil {
		return nil, err
	}

	// Step 3: 検証済み traces endpoint で tracer を初期化し、HTTP と datastore span を同じ trace provider へ送る。
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

	// Step 6: runtime shutdown と startup failure path の両方から同じ closer を呼べるよう、順序付きにまとめる。
	return func(ctx context.Context) error {
		_ = closeLogger(ctx)
		_ = closeTracer(ctx)
		_ = closeMeter(ctx)
		return nil
	}, nil
}
