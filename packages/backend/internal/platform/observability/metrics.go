package observability

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// InitMeter initializes the OTel MeterProvider and starts Go runtime metrics collection.
func InitMeter(ctx context.Context, endpoint, serviceName string) (func(context.Context) error, error) {
	// Step 1: 空 endpoint は OTLP 無効化として扱わず、共通の gRPC 既定 endpoint へ正規化する。
	endpoint = resolveOTLPEndpoint(endpoint)

	if serviceName == "" {
		// Step 2: serviceName 未設定時も metrics が無名 service にならないよう Product API 名へ正規化する。
		serviceName = "www-template-api"
	}

	// Step 3: OTLP metric exporter を gRPC で構築し、runtime/application metrics を collector へ送信できるようにする。
	exporter, err := otlpmetricgrpc.New(ctx,
		otlpmetricgrpc.WithEndpoint(endpoint),
		otlpmetricgrpc.WithInsecure(),
	)
	if err != nil {
		return nil, fmt.Errorf("create otlp metric exporter: %w", err)
	}

	// Step 4: metric resource に service 情報と process 情報を付与し、SigNoz 上で backend process を識別できるようにする。
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(serviceName),
			semconv.ServiceVersionKey.String("0.1.0"),
			semconv.DeploymentEnvironmentKey.String("development"),
		),
		resource.WithFromEnv(),
		resource.WithProcess(),
	)
	if err != nil {
		return nil, fmt.Errorf("create otel resource: %w", err)
	}

	// Step 5: periodic reader を使い、15 秒ごとに metrics を collector へ送信する。
	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(exporter, sdkmetric.WithInterval(15*time.Second))),
		sdkmetric.WithResource(res),
	)

	// Step 6: Go runtime metrics を同じ MeterProvider に接続し、GC/メモリ等の基礎情報を収集する。
	if err := runtime.Start(
		runtime.WithMeterProvider(mp),
		runtime.WithMinimumReadMemStatsInterval(15*time.Second),
	); err != nil {
		return nil, fmt.Errorf("start runtime metrics: %w", err)
	}

	// Step 7: runtime shutdown 時に metrics reader/exporter を停止できる closer を返す。
	return func(ctx context.Context) error {
		return mp.Shutdown(ctx)
	}, nil
}
