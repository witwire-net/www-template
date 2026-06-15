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
	"go.opentelemetry.io/otel/trace"
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

// InstallLocalTracerProviderForTesting は外部 collector へ接続しない test 用 tracer provider を global に設定する。
//
// 役割:
//   - HTTP adapter test が OTel package を直接 import せず、platform/observability 経由で trace propagation を検証できるようにする。
//   - TraceContext propagator を有効化し、otelgin が `traceparent` header を request context へ反映できる状態にする。
//   - cleanup 関数で元の provider / propagator を復元し、他テストへの global state 汚染を抑える。
//
// 戻り値:
//   - func(context.Context) error: test cleanup で呼ぶ復元関数。provider shutdown error があれば返す。
//
// 使用例:
//
//	restore := observability.InstallLocalTracerProviderForTesting()
//	t.Cleanup(func() { _ = restore(context.Background()) })
func InstallLocalTracerProviderForTesting() func(context.Context) error {
	// Step 1: 既存 global state を保存し、cleanup で元に戻せるようにする。
	previousProvider := otel.GetTracerProvider()
	previousPropagator := otel.GetTextMapPropagator()

	// Step 2: exporter なしの local provider を使い、外部 collector に依存せず server span を生成できるようにする。
	tp := sdktrace.NewTracerProvider(sdktrace.WithSampler(sdktrace.AlwaysSample()))
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	// Step 3: cleanup では provider shutdown 後に元の global provider / propagator を復元する。
	return func(ctx context.Context) error {
		err := tp.Shutdown(ctx)
		otel.SetTracerProvider(previousProvider)
		otel.SetTextMapPropagator(previousPropagator)
		return err
	}
}

// TraceIDFromContext は context に含まれる現在の trace ID を文字列として返す。
//
// 役割:
//   - adapter や application の test が OTel trace package を直接 import せず、trace propagation の有無だけを検査できるようにする。
//   - invalid span context の場合は空文字を返し、呼び出し側が fail-close な assertion を書けるようにする。
//
// 引数:
//   - ctx: request span または detached span を含む可能性がある context。
//
// 戻り値:
//   - string: valid な trace ID。trace context が無効な場合は空文字。
func TraceIDFromContext(ctx context.Context) string {
	// Step 1: context から SpanContext を取得し、valid な場合だけ trace ID を返す。
	spanContext := trace.SpanContextFromContext(ctx)
	if !spanContext.IsValid() {
		return ""
	}
	return spanContext.TraceID().String()
}
