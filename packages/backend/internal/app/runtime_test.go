package app

import (
	"context"
	"testing"

	"www-template/packages/backend/internal/platform/config"
)

func TestNewRuntimeWithConfigFailsClosedWithoutTokenOutsideDevelopment(t *testing.T) {
	t.Parallel()

	_, err := NewRuntimeWithConfig(context.Background(), config.Config{
		AllowedOrigins: []string{"http://localhost:5173", "http://localhost:5174"},
		Environment:    "production",
		Port:           "8080",
	})
	if err == nil {
		t.Fatal("expected error for missing production bearer token")
	}
}

func TestNewRuntimeWithConfigFailsClosedWhenRequiredInfrastructureIsMissing(t *testing.T) {
	t.Parallel()

	_, err := NewRuntimeWithConfig(context.Background(), config.Config{
		AllowedOrigins: []string{"http://localhost:5173", "http://localhost:5174"},
		AppBearerToken: "dev-app-auth",
		Environment:    "development",
		Port:           "8080",
	})
	if err == nil {
		t.Fatal("expected error for missing infrastructure settings")
	}
}
