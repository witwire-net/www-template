package postgres

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const defaultInfrastructureTimeout = 3 * time.Second

// slogWriter は slog.Logger を GORM の logger.Writer イーフェースに適合させるアダプターである。
//
// 役割:
//   - GORM の logger.New は logger.Writer イーフェース（Printf メソッド）を受け取る。
//   - slog.Logger は Printf メソッドを持たないため、このアダプターで橋渡しする。
type slogWriter struct {
	logger *slog.Logger
}

// Printf は GORM ロガーからの出力を slog 経由で出力する。
func (w *slogWriter) Printf(format string, args ...any) {
	w.logger.Info(fmt.Sprintf(format, args...))
}

// buildGORMLoggerConfig は GORM ロガーの設定を構築する。
//
// セキュリティ要件:
//   - IgnoreRecordNotFoundError: true → passkey 0 件のアカウントでの not found が noisy ログにならない。
//   - ParameterizedQueries: true → SQL ログにメールアドレス等の PII が含まれない。
//   - LogLevel: Warn → 実際の DB エラーだけがログに残る。
//   - SlowThreshold: 200ms → スロークエリを検出する。
func buildGORMLoggerConfig() logger.Config {
	return logger.Config{
		IgnoreRecordNotFoundError: true,
		ParameterizedQueries:      true,
		LogLevel:                  logger.Warn,
		SlowThreshold:             200 * time.Millisecond,
	}
}

// OpenDatabase は GORM を使って PostgreSQL への接続を開く。
//
// GORM ロガー設定:
//   - IgnoreRecordNotFoundError: true にすることで、通常の not found クエリ（passkey 0 件検索など）が
//     noisy なログとして出力されないようにする。これはセキュリティ上重要であり、
//     account existence をログから漏らさないための措置である。
//   - ParameterizedQueries: true にすることで、SQL ログにバインドパラメータ（メールアドレス、トークンなど）が
//     含まれないようにする。これにより PII やセキュリティ情報のログ漏洩を防ぐ。
//   - LogLevel: Warn に設定し、実際の DB エラーはログに残しつつ、通常の not found は抑制する。
func OpenDatabase(databaseURL string) (*gorm.DB, error) {
	if strings.TrimSpace(databaseURL) == "" {
		return nil, errors.New("DATABASE_URL is required")
	}

	// GORM ロガーを設定し、record not found のノイズと SQL パラメータの露出を抑制する。
	gormLogger := logger.New(
		&slogWriter{logger: slog.Default()},
		buildGORMLoggerConfig(),
	)

	db, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{
		Logger: newObservedGORMLogger(gormLogger),
	})
	if err != nil {
		return nil, fmt.Errorf("open gorm database: %w", err)
	}

	return db, nil
}

func PingDatabase(ctx context.Context, db *gorm.DB) error {
	if db == nil {
		return errors.New("gorm database is required")
	}

	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("open sql db: %w", err)
	}

	pingContext, cancel := context.WithTimeout(ctx, defaultInfrastructureTimeout)
	defer cancel()
	pingStartedAt := time.Now()
	pingErr := sqlDB.PingContext(pingContext)
	observePostgresPing(ctx, pingStartedAt, pingErr)
	if pingErr != nil {
		return fmt.Errorf("ping postgres: %w", pingErr)
	}

	return nil
}
