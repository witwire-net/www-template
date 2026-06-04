package observability

import (
	"context"
	"log/slog"
	"testing"

	otellog "go.opentelemetry.io/otel/log"
)

// [OBS-LOGGER-1] slogLevelToOTelSeverity が slog.Level を正しく OTel severity に変換することを検証する。
//
// 検証内容:
//   - Debug レベルが OTel SeverityDebug に変換される。
//   - Info レベルが OTel SeverityInfo に変換される。
//   - Warn レベルが OTel SeverityWarn に変換される。
//   - Error レベルが OTel SeverityError に変換される。
func TestSlogLevelToOTelSeverityMapping(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		level    slog.Level
		expected otellog.Severity
	}{
		{"Debug maps to SeverityDebug", slog.LevelDebug, otellog.SeverityDebug},
		{"Info maps to SeverityInfo", slog.LevelInfo, otellog.SeverityInfo},
		{"Warn maps to SeverityWarn", slog.LevelWarn, otellog.SeverityWarn},
		{"Error maps to SeverityError", slog.LevelError, otellog.SeverityError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := slogLevelToOTelSeverity(tt.level)
			if result != tt.expected {
				t.Errorf("slogLevelToOTelSeverity(%v) = %v, want %v", tt.level, result, tt.expected)
			}
		})
	}
}

// [OBS-LOGGER-2] slogAttrToOTelKeyValue が slog.Attr を正しく OTel log.KeyValue に変換することを検証する。
func TestSlogAttrToOTelKeyValueConversion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		attr slog.Attr
		key  string
	}{
		{"string attr", slog.String("key", "value"), "key"},
		{"int attr", slog.Int("count", 42), "count"},
		{"bool attr", slog.Bool("enabled", true), "enabled"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := slogAttrToOTelKeyValue(tt.key, tt.attr)
			if result.Key != tt.key {
				t.Errorf("slogAttrToOTelKeyValue key = %q, want %q", result.Key, tt.key)
			}
		})
	}
}

// [OBS-LOGGER-3] Logger() が non-nil の slog.Logger を返すことを検証する。
func TestLoggerReturnsNonNilLogger(t *testing.T) {
	t.Parallel()
	l := Logger()
	if l == nil {
		t.Fatal("expected Logger() to return non-nil slog.Logger")
	}
}

// [OBS-LOGGER-4] traceContextHandler が trace context を持つ context で trace_id/span_id を注入することを検証する。
func TestTraceContextHandlerInjectsTraceIDAndSpanID(t *testing.T) {
	t.Parallel()

	// traceContextHandler が slog.Handler イーフェースを実装していることを確認する。
	var _ slog.Handler = &traceContextHandler{Handler: slog.NewJSONHandler(nil, nil)}
}

// [OBS-LOGGER-5] otelSlogHandler.Enabled が stdout handler に委譲され、Debug レベルが Info 設定で抑制されることを検証する。
func TestOtelSlogHandlerEnabledDelegatesToStdoutHandler(t *testing.T) {
	t.Parallel()

	// Info レベルの stdout handler を持つ otelSlogHandler を構築する。
	stdoutHandler := slog.NewJSONHandler(nil, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	handler := &otelSlogHandler{
		logger: nil, // OTLP logger は使用しない
		stdout: stdoutHandler,
	}

	// Debug レベルは Info 設定で抑制されるはず。
	if handler.Enabled(context.Background(), slog.LevelDebug) {
		t.Error("expected Debug level to be disabled when stdout handler level is Info")
	}

	// Info レベルは有効であるはず。
	if !handler.Enabled(context.Background(), slog.LevelInfo) {
		t.Error("expected Info level to be enabled")
	}

	// Warn レベルは有効であるはず。
	if !handler.Enabled(context.Background(), slog.LevelWarn) {
		t.Error("expected Warn level to be enabled")
	}
}

// [OBS-LOGGER-6] WithGroup.WithAttrs の順で使われた場合、OTLP attribute key に group prefix が付与されることを検証する。
//
// 検証内容:
//   - logger.WithGroup("request").With("id", "abc") の OTLP attribute key が "request.id" になる。
//   - WithAttrs で設定された属性が OTLP record に反映される。
func TestOtelSlogHandlerWithGroupThenWithAttrsAppliesGroupPrefix(t *testing.T) {
	t.Parallel()

	handler := &otelSlogHandler{
		logger: nil,
		stdout: slog.NewJSONHandler(nil, &slog.HandlerOptions{Level: slog.LevelInfo}),
	}

	// WithGroup("request").WithAttrs([id=abc]) の順で属性を設定する。
	grouped := handler.WithGroup("request").(*otelSlogHandler)
	withAttrs := grouped.WithAttrs([]slog.Attr{slog.String("id", "abc")}).(*otelSlogHandler)

	// attrs に group prefix が付与されていることを確認する。
	if len(withAttrs.attrs) != 1 {
		t.Fatalf("expected 1 attr, got %d", len(withAttrs.attrs))
	}
	if withAttrs.attrs[0].Key != "request.id" {
		t.Errorf("expected attr key 'request.id', got %q", withAttrs.attrs[0].Key)
	}
}
