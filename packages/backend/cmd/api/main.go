package main

import (
	"context"
	"errors"
	"fmt"
	stdhttp "net/http"
	"os/signal"
	"syscall"
	"time"

	"log"

	"www-template/packages/backend/internal/app"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("api runtime failed: %v", err)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)

	runtime, err := app.NewRuntime(ctx)
	if err != nil {
		stop()
		return fmt.Errorf("build runtime: %w", err)
	}
	defer stop()
	defer func() {
		if closeErr := runtime.Close(context.Background()); closeErr != nil {
			log.Printf("close runtime: %v", closeErr)
		}
	}()

	server := runtime.Server()

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if shutdownErr := server.Shutdown(shutdownCtx); shutdownErr != nil {
			log.Printf("shutdown server: %v", shutdownErr)
		}
	}()

	log.Printf("www-template api listening on %s", runtime.Config().Port)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, stdhttp.ErrServerClosed) {
		return fmt.Errorf("listen and serve: %w", err)
	}

	return nil
}
