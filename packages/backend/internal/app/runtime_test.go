package app

import (
	"context"
	"testing"

	"witwire.net/www-template/packages/backend/internal/types"
)

func TestNewRuntimeWithConfigFailsClosedWithoutTokenOutsideDevelopment(t *testing.T) {
	t.Parallel()

	_, err := NewRuntimeWithConfig(context.Background(), types.Config{
		AllowedOrigins: []string{"http://localhost:5173"},
		Environment:    "production",
		Port:           "8080",
		ProfileStore:   "memory",
	})
	if err == nil {
		t.Fatal("expected error for missing production bearer token")
	}
}
