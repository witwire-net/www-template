package postgres

import (
	"log/slog"
	"testing"

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
