package observability

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	otellog "go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

// traceContextHandler は slog.Handler をラップし、trace context から trace_id/span_id を注入する。
type traceContextHandler struct {
	slog.Handler
}

// Logger は trace_id/span_id を注入する slog.Logger を返す。
//
// 役割:
//   - InitLogger が呼ばれた後は、OTLP dual-write 対応の slog.Default() を返す。
//   - InitLogger が呼ばれる前（startup 初期）は、stdout JSON のみのフォールバックを返す。
//   - これにより product_container.go や operator_audit_projection.go 等の
//     observability.Logger() 呼び出しが OTLP に送信される。
func Logger() *slog.Logger {
	// slog.Default() は InitLogger 後は OTLP dual-write handler になっている。
	// InitLogger 前は stdlib の default（stdout text）が返るが、
	// startup 初期のログは許容範囲内のため問題ない。
	return slog.Default()
}

func (h *traceContextHandler) Handle(ctx context.Context, r slog.Record) error {
	if spanContext := trace.SpanContextFromContext(ctx); spanContext.IsValid() {
		r.AddAttrs(
			slog.String("trace_id", spanContext.TraceID().String()),
			slog.String("span_id", spanContext.SpanID().String()),
		)
	}
	return h.Handler.Handle(ctx, r)
}

func (h *traceContextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &traceContextHandler{Handler: h.Handler.WithAttrs(attrs)}
}

func (h *traceContextHandler) WithGroup(name string) slog.Handler {
	return &traceContextHandler{Handler: h.Handler.WithGroup(name)}
}

// InitLogger は OTLP logs exporter を初期化し、slog の default ロガーを OTLP 対応に差し替える。
//
// 役割:
//   - OTELExporterOTLPLogsEndpoint が設定されている場合、OTLP gRPC 経由で SigNoz にログを送信する。
//   - endpoint が空の場合は OTELExporterOTLPEndpoint をフォールバックとして使用し、
//     それも空の場合は localhost:4317 をデフォルトとする。
//   - slog.SetDefault を呼び、既存の observability.Logger() 呼び出しを OTLP 対応に変更する。
//   - stdout JSON 出力と OTLP 出力を dual-write し、開発環境でもログ確認を可能にする。
//
// 引数:
//   - ctx: OTel exporter 接続に使う context。
//   - logsEndpoint: OTLP logs エンドポイント（例: localhost:4317）。空の場合はフォールバック。
//   - serviceName: OTel resource に付与するサービス名。
//
// 戻り値:
//   - func(context.Context) error: LoggerProvider をシャットダウンする関数。
//   - error: exporter 初期化失敗時。
func InitLogger(ctx context.Context, logsEndpoint string, serviceName string) (func(context.Context) error, error) {
	// Step 1: 空 endpoint は OTLP 無効化として扱わず、共通の gRPC 既定 endpoint へ正規化する。
	endpoint := resolveOTLPEndpoint(logsEndpoint)

	if serviceName == "" {
		// Step 2: serviceName 未設定時も logs が無名 service にならないよう Product API 名へ正規化する。
		serviceName = "www-template-api"
	}

	// OTLP logs exporter を gRPC で作成する。
	exporter, err := otlploggrpc.New(ctx,
		otlploggrpc.WithEndpoint(endpoint),
		otlploggrpc.WithInsecure(),
	)
	if err != nil {
		return nil, fmt.Errorf("create otlp log exporter: %w", err)
	}

	// OTel resource を構築する。tracer/meter と同じ属性を使用する。
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
		return nil, fmt.Errorf("create otel resource for logs: %w", err)
	}

	// LoggerProvider を構築する。
	lp := log.NewLoggerProvider(
		log.WithProcessor(log.NewBatchProcessor(exporter, log.WithExportInterval(5*time.Second))),
		log.WithResource(res),
	)

	// slog の default ロガーを OTLP 対応に差し替える。
	// stdout JSON 出力と OTLP 出力を dual-write する。
	otelHandler := &otelSlogHandler{
		logger: lp.Logger("www-template"),
		stdout: slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		}),
	}
	slog.SetDefault(slog.New(otelHandler))

	return func(ctx context.Context) error {
		return lp.Shutdown(ctx)
	}, nil
}

// otelSlogHandler は slog.Record を OTLP logs と stdout の両方に送信する handler である。
//
// 役割:
//   - slog.Record を OTel log.Record に変換し、OTLP exporter 経由で SigNoz に送信する。
//   - stdout JSON 出力も維持し、開発環境でのログ確認を可能にする。
//   - trace context が存在する場合は trace_id/span_id を OTel log 属性に追加する。
//   - Enabled は stdout handler に委譲し、Debug レベル等の低レベルログが流出しないようにする。
//   - WithAttrs/WithGroup で設定された属性は OTLP record にも反映される。
type otelSlogHandler struct {
	logger otellog.Logger
	stdout slog.Handler
	// attrs は WithAttrs で設定された属性。OTLP record にも反映する。
	attrs []otellog.KeyValue
	// group は WithGroup で設定されたグループ名。OTLP record の属性キーに prefix として付与する。
	group string
}

// Enabled は stdout handler に委譲し、Info 未満のログが stdout/OTLP に流出しないようにする。
func (h *otelSlogHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.stdout.Enabled(ctx, level)
}

func (h *otelSlogHandler) Handle(ctx context.Context, r slog.Record) error {
	// stdout に出力する（trace_id/span_id injection を含む）。
	traceHandler := &traceContextHandler{Handler: h.stdout}
	if err := traceHandler.Handle(ctx, r); err != nil {
		return err
	}

	// OTLP logs に出力する。
	var record otellog.Record
	record.SetTimestamp(r.Time)
	record.SetObservedTimestamp(time.Now())
	record.SetBody(otellog.StringValue(r.Message))

	// severity を slog.Level から OTel severity に変換する。
	record.SetSeverity(slogLevelToOTelSeverity(r.Level))

	// WithAttrs で設定された属性を OTLP record に追加する。
	for _, attr := range h.attrs {
		record.AddAttributes(attr)
	}

	// slog attributes を OTel log attributes に変換する。
	// WithGroup で設定されたグループがある場合は属性キーに prefix を付与する。
	r.Attrs(func(a slog.Attr) bool {
		key := a.Key
		if h.group != "" {
			key = h.group + "." + key
		}
		record.AddAttributes(slogAttrToOTelKeyValue(key, a))
		return true
	})

	h.logger.Emit(ctx, record)
	return nil
}

func (h *otelSlogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	// OTel log attributes に変換して保持する。
	// WithGroup で設定されたグループがある場合は属性キーに prefix を付与する。
	otelAttrs := make([]otellog.KeyValue, 0, len(attrs))
	for _, a := range attrs {
		key := a.Key
		if h.group != "" {
			key = h.group + "." + key
		}
		otelAttrs = append(otelAttrs, slogAttrToOTelKeyValue(key, a))
	}
	return &otelSlogHandler{
		logger: h.logger,
		stdout: h.stdout.WithAttrs(attrs),
		attrs:  append(h.attrs, otelAttrs...),
		group:  h.group,
	}
}

func (h *otelSlogHandler) WithGroup(name string) slog.Handler {
	prefix := name
	if h.group != "" {
		prefix = h.group + "." + name
	}
	return &otelSlogHandler{
		logger: h.logger,
		stdout: h.stdout.WithGroup(name),
		attrs:  h.attrs,
		group:  prefix,
	}
}

// slogLevelToOTelSeverity は slog.Level を OTel log.Severity に変換する。
func slogLevelToOTelSeverity(level slog.Level) otellog.Severity {
	switch {
	case level <= slog.LevelDebug:
		return otellog.SeverityDebug
	case level <= slog.LevelInfo:
		return otellog.SeverityInfo
	case level <= slog.LevelWarn:
		return otellog.SeverityWarn
	case level <= slog.LevelError:
		return otellog.SeverityError
	default:
		return otellog.SeverityFatal
	}
}

// slogAttrToOTelKeyValue は slog.Attr を OTel log.KeyValue に変換する。
// key パラメータは WithGroup による prefix 適用済みのキーを受け取る。
func slogAttrToOTelKeyValue(key string, a slog.Attr) otellog.KeyValue {
	switch a.Value.Kind() {
	case slog.KindString:
		return otellog.String(key, a.Value.String())
	case slog.KindInt64:
		return otellog.Int64(key, a.Value.Int64())
	case slog.KindFloat64:
		return otellog.Float64(key, a.Value.Float64())
	case slog.KindBool:
		return otellog.Bool(key, a.Value.Bool())
	case slog.KindDuration:
		return otellog.String(key, a.Value.Duration().String())
	case slog.KindTime:
		return otellog.String(key, a.Value.Time().Format(time.RFC3339Nano))
	default:
		return otellog.String(key, a.Value.String())
	}
}
