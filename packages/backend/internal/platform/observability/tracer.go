package observability

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

func InitTracer(ctx context.Context, endpoint, serviceName string) (func(context.Context) error, error) {
	// Step 1: 空 endpoint は OTLP 無効化として扱わず、共通の gRPC 既定 endpoint へ正規化する。
	endpoint = resolveOTLPEndpoint(endpoint)

	if serviceName == "" {
		// Step 2: serviceName 未設定時も traces が無名 service にならないよう Product API 名へ正規化する。
		serviceName = "www-template-api"
	}

	// Step 3: OTLP trace exporter を gRPC で構築し、collector へ trace batch を送信できるようにする。
	exporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(endpoint),
		otlptracegrpc.WithInsecure(),
	)
	if err != nil {
		return nil, fmt.Errorf("create otlp trace exporter: %w", err)
	}

	// Step 4: trace resource に service 情報と process 情報を付与し、SigNoz 上で backend process を識別できるようにする。
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

	// Step 5: batch exporter を使う TracerProvider を作り、過剰な同期送信で request path を遅くしない。
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter, sdktrace.WithBatchTimeout(1*time.Second)),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)

	// Step 6: global tracer provider と propagation を設定し、HTTP middleware と application code が同じ trace context を使うようにする。
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	// Step 7: runtime shutdown 時に未送信 trace を flush できる closer を返す。
	return func(ctx context.Context) error {
		return tp.Shutdown(ctx)
	}, nil
}
