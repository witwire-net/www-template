package postgres

import (
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// [POSTGRES-LOGGER-1] buildGORMLoggerConfig がセキュリティ要件を満たす設定を返すことを検証する。
//
// 検証内容:
//   - IgnoreRecordNotFoundError: true → passkey 0 件のアカウントでの not found が noisy ログにならない。
//   - ParameterizedQueries: true → SQL ログにメールアドレス等の PII が含まれない。
//   - LogLevel: Warn → 実際の DB エラーだけがログに残る。
//   - SlowThreshold: 200ms → スロークエリを検出する。
func TestBuildGORMLoggerConfigMeetsSecurityRequirements(t *testing.T) {
	t.Parallel()

	cfg := buildGORMLoggerConfig()

	if !cfg.IgnoreRecordNotFoundError {
		t.Error("expected IgnoreRecordNotFoundError=true to suppress noisy record-not-found logs")
	}
	if !cfg.ParameterizedQueries {
		t.Error("expected ParameterizedQueries=true to prevent PII in SQL logs")
	}
	if cfg.LogLevel != logger.Warn {
		t.Errorf("expected LogLevel=Warn for production safety, got %v", cfg.LogLevel)
	}
	if cfg.SlowThreshold != 200*1e6 { // 200ms in nanoseconds
		t.Errorf("expected SlowThreshold=200ms, got %v", cfg.SlowThreshold)
	}
}

// [POSTGRES-LOGGER-2] slogWriter が logger.Writer イーフェース（Printf メソット）を実装することを検証する。
func TestSlogWriterImplementsLoggerWriter(t *testing.T) {
	t.Parallel()

	// slogWriter が logger.Writer イーフェースを実装していることを確認する（コンパイル時検証）。
	var _ logger.Writer = &slogWriter{logger: slog.Default()}

	// Printf が panic せずに実行できることを確認する。
	writer := &slogWriter{logger: slog.Default()}
	writer.Printf("test message: %s", "value")
}

// [POSTGRES-LOGGER-3] observedGORMLogger が SQL literal を SigNoz 属性へ出さず、安全な template だけを記録することを検証する。
func TestObservedGORMLoggerSanitizesStatementMetadata(t *testing.T) {
	previousLogger := slog.Default()
	capture := &postgresDatastoreCaptureHandler{}
	slog.SetDefault(slog.New(capture))
	defer slog.SetDefault(previousLogger)

	observedLogger := newObservedGORMLogger(logger.Discard)
	observedLogger.Trace(context.Background(), time.Now().Add(-25*time.Millisecond), func() (string, int64) {
		// Step 1: email と secret 風 token を含む SQL を渡し、観測属性では placeholder 化されることを確認する。
		return "SELECT * FROM accounts WHERE email = 'alice@example.com' AND recovery_token = 'secret-token-value' AND retry_count = 12 LIMIT 1", 1
	}, nil)

	if len(capture.records) != 1 {
		t.Fatalf("expected one datastore log, got %d", len(capture.records))
	}
	attrs := postgresDatastoreAttrsToMap(capture.records[0].attrs)
	target := attrs["datastore.target"].String()
	if strings.Contains(target, "alice@example.com") || strings.Contains(target, "secret-token-value") || strings.Contains(target, "12") {
		t.Fatalf("datastore.target leaked raw SQL value: %q", target)
	}
	if got := attrs["datastore.operation"].String(); got != "select" {
		t.Fatalf("datastore.operation = %q, want select", got)
	}
	if got := attrs["error_class"].String(); got != "none" {
		t.Fatalf("error_class = %q, want none", got)
	}
}

// [POSTGRES-LOGGER-4] GORM record not found が raw error ではなく not_found class として記録されることを検証する。
func TestObservedGORMLoggerClassifiesRecordNotFound(t *testing.T) {
	previousLogger := slog.Default()
	capture := &postgresDatastoreCaptureHandler{}
	slog.SetDefault(slog.New(capture))
	defer slog.SetDefault(previousLogger)

	observedLogger := newObservedGORMLogger(logger.Discard)
	observedLogger.Trace(context.Background(), time.Now(), func() (string, int64) {
		// Step 1: record not found の代表 query を渡し、存在確認情報を raw error ではなく分類だけで残す。
		return "SELECT * FROM passkey_credentials WHERE account_id = ? ORDER BY id ASC LIMIT 1", -1
	}, gorm.ErrRecordNotFound)

	if len(capture.records) != 1 {
		t.Fatalf("expected one datastore log, got %d", len(capture.records))
	}
	attrs := postgresDatastoreAttrsToMap(capture.records[0].attrs)
	if got := attrs["datastore.status"].String(); got != "not_found" {
		t.Fatalf("datastore.status = %q, want not_found", got)
	}
	if got := attrs["error_class"].String(); got != "not_found" {
		t.Fatalf("error_class = %q, want not_found", got)
	}
	if _, exists := attrs["error"]; exists {
		t.Fatal("raw error attribute must not be present")
	}
}

type postgresDatastoreCapturedRecord struct {
	attrs []slog.Attr
}

type postgresDatastoreCaptureHandler struct {
	records []postgresDatastoreCapturedRecord
}

func (h *postgresDatastoreCaptureHandler) Enabled(context.Context, slog.Level) bool {
	return true
}

func (h *postgresDatastoreCaptureHandler) Handle(_ context.Context, record slog.Record) error {
	attrs := make([]slog.Attr, 0, record.NumAttrs())
	record.Attrs(func(attr slog.Attr) bool {
		attrs = append(attrs, attr)
		return true
	})
	h.records = append(h.records, postgresDatastoreCapturedRecord{attrs: attrs})
	return nil
}

func (h *postgresDatastoreCaptureHandler) WithAttrs([]slog.Attr) slog.Handler {
	return h
}

func (h *postgresDatastoreCaptureHandler) WithGroup(string) slog.Handler {
	return h
}

func postgresDatastoreAttrsToMap(attrs []slog.Attr) map[string]slog.Value {
	attrMap := make(map[string]slog.Value, len(attrs))
	for _, attr := range attrs {
		attrMap[attr.Key] = attr.Value.Resolve()
	}
	return attrMap
}
