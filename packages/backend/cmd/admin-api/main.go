package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	stdhttp "net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"www-template/packages/backend/internal/app"
)

// main は Admin API 専用 GoServer binary の process entrypoint である。
// Product API の cmd/api とは別 binary として起動し、Admin runtime だけを構築する。
// 起動失敗時は構造化ログに原因を残し、process を非 0 終了して fail-close する。
func main() {
	// Step 1: stdout へ JSON 形式で出力する logger を用意し、運用環境で機械的に収集できる形にそろえる。
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	// Step 2: Admin runtime の構築と HTTP server 起動を実行し、失敗時は secret を含まない error object だけを記録する。
	if err := run(logger); err != nil {
		logger.Error("admin api runtime failed", slog.Any("error", err))
		os.Exit(1)
	}
}

// run は Admin API server の lifecycle を管理する。
// logger は起動、shutdown、close の観測用に使い、戻り値は起動または待受中に発生した致命的 error を返す。
// SIGINT / SIGTERM を受け取ると context を cancel し、HTTP server を graceful shutdown する。
func run(logger *slog.Logger) error {
	// Step 1: process signal と連動する context を作り、Admin binary の shutdown 契機を一箇所に集約する。
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Step 2: Product runtime ではなく Admin runtime を構築し、Admin binary が Product handlers を組み立てない境界を守る。
	runtime, err := app.NewAdminRuntime(ctx)
	if err != nil {
		return fmt.Errorf("build admin runtime: %w", err)
	}
	defer func() {
		// Step 3: runtime が確保した observability resource を process 終了時に解放し、close error は shutdown を妨げず観測だけ行う。
		if closeErr := runtime.Close(context.Background()); closeErr != nil {
			logger.Error("close admin runtime", slog.Any("error", closeErr))
		}
	}()

	// Step 4: Admin runtime が所有する HTTP server を取得し、Product server とは独立した graceful shutdown を設定する。
	server := runtime.Server()
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if shutdownErr := server.Shutdown(shutdownCtx); shutdownErr != nil {
			logger.Error("shutdown admin server", slog.Any("error", shutdownErr))
		}
	}()

	// Step 5: Admin API binary として待受を開始し、正常 shutdown 以外の ListenAndServe error だけを呼び出し元へ返す。
	logger.Info("www-template admin api listening", slog.String("addr", runtime.Config().Port))
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, stdhttp.ErrServerClosed) {
		return fmt.Errorf("listen and serve admin api: %w", err)
	}

	return nil
}
