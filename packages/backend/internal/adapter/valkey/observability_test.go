package valkey

import (
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/redis/go-redis/v9"
)

// [VALKEY-OBS-1] raw key が namespace pattern に変換され、token/session ID が残らないことを検証する。
func TestSafeKeyPatternRedactsDynamicSegments(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		key      string
		expected string
	}{
		{name: "product key with environment prefix", key: "www-template:product:auth:recovery-token:01HXSECRET", expected: "prefix:product:auth:recovery-token:*"},
		{name: "admin operator session key", key: "admin:auth:operator-session:session-1", expected: "admin:auth:operator-session:*"},
		{name: "unknown dynamic key", key: "custom:email:alice@example.com", expected: "prefix:*"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := SafeKeyPattern(tt.key); got != tt.expected {
				t.Fatalf("SafeKeyPattern() = %q, want %q", got, tt.expected)
			}
		})
	}
}

// [VALKEY-OBS-2] EVAL target が Lua script body と ARGV を含まないことを検証する。
func TestSafeValkeyEvalTargetOmitsScriptAndArguments(t *testing.T) {
	t.Parallel()

	target := safeValkeyCommandTarget("eval", []any{
		"eval",
		"return redis.call('GET', KEYS[1])",
		1,
		"www-template:product:auth:recovery-token:token-id",
		"secret-hash-argument",
	}, "product")

	// Step 1: Lua script body、ARGV、raw token ID が target に残らないことを確認する。
	for _, leaked := range []string{"redis.call", "secret-hash-argument", "token-id"} {
		if strings.Contains(target, leaked) {
			t.Fatalf("EVAL target leaked %q: %q", leaked, target)
		}
	}
	if !strings.Contains(target, "key_count=1") || !strings.Contains(target, "prefix:product:auth:recovery-token:*") {
		t.Fatalf("EVAL target lost safe metadata: %q", target)
	}
}

// [VALKEY-OBS-3] hook が Valkey value を出さず、command metadata だけを記録することを検証する。
func TestObservationHookLogsCommandWithoutRawValue(t *testing.T) {
	previousLogger := slog.Default()
	capture := &valkeyDatastoreCaptureHandler{}
	slog.SetDefault(slog.New(capture))
	defer slog.SetDefault(previousLogger)

	hook := observationHook{surface: "product"}
	cmd := redis.NewStringCmd(context.Background(), "get", "www-template:product:auth:session:session-1")
	process := hook.ProcessHook(func(context.Context, redis.Cmder) error {
		// Step 1: Redis response value を command に設定し、hook が長さだけを記録して値自体を出さないことを確認する。
		cmd.SetVal("secret-session-json")
		return nil
	})

	if err := process(context.Background(), cmd); err != nil {
		t.Fatalf("process hook: %v", err)
	}
	if len(capture.records) != 1 {
		t.Fatalf("expected one datastore log, got %d", len(capture.records))
	}
	attrs := valkeyDatastoreAttrsToMap(capture.records[0].attrs)
	serializedAttrs := valkeyDatastoreAttrsString(capture.records[0].attrs)
	if strings.Contains(serializedAttrs, "secret-session-json") || strings.Contains(serializedAttrs, "session-1") {
		t.Fatalf("Valkey observation leaked raw value/key: %s", serializedAttrs)
	}
	if got := attrs["datastore.operation"].String(); got != "get" {
		t.Fatalf("datastore.operation = %q, want get", got)
	}
	if got := attrs["datastore.target"].String(); got != "GET prefix:product:auth:session:*" {
		t.Fatalf("datastore.target = %q, want safe key pattern", got)
	}
	if got := attrs["result_class"].String(); got != "single" {
		t.Fatalf("result_class = %q, want single", got)
	}
}

type valkeyDatastoreCapturedRecord struct {
	attrs []slog.Attr
}

type valkeyDatastoreCaptureHandler struct {
	records []valkeyDatastoreCapturedRecord
}

func (h *valkeyDatastoreCaptureHandler) Enabled(context.Context, slog.Level) bool {
	return true
}

func (h *valkeyDatastoreCaptureHandler) Handle(_ context.Context, record slog.Record) error {
	attrs := make([]slog.Attr, 0, record.NumAttrs())
	record.Attrs(func(attr slog.Attr) bool {
		attrs = append(attrs, attr)
		return true
	})
	h.records = append(h.records, valkeyDatastoreCapturedRecord{attrs: attrs})
	return nil
}

func (h *valkeyDatastoreCaptureHandler) WithAttrs([]slog.Attr) slog.Handler {
	return h
}

func (h *valkeyDatastoreCaptureHandler) WithGroup(string) slog.Handler {
	return h
}

func valkeyDatastoreAttrsToMap(attrs []slog.Attr) map[string]slog.Value {
	attrMap := make(map[string]slog.Value, len(attrs))
	for _, attr := range attrs {
		attrMap[attr.Key] = attr.Value.Resolve()
	}
	return attrMap
}

func valkeyDatastoreAttrsString(attrs []slog.Attr) string {
	parts := make([]string, 0, len(attrs))
	for _, attr := range attrs {
		parts = append(parts, attr.Key+"="+attr.Value.Resolve().String())
	}
	return strings.Join(parts, " ")
}
