package app

import (
	"context"
	"testing"
	"time"

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

// fullyConfiguredDevelopmentConfig は開発環境で起動可能な最小限のインフラ設定を持つ Config を返す。
// これにより、TTL 検証の前に infrastructure missing で落ちることを防ぐ。
func fullyConfiguredDevelopmentConfig() config.Config {
	return config.Config{
		AllowedOrigins: []string{"http://localhost:5173", "http://localhost:5174"},
		AppBearerToken: "dev-app-auth",
		Environment:    "development",
		Port:           "8080",
		Infra: config.InfraConfig{
			Database: config.DatabaseConfig{URL: "postgres://template:template@postgres:5432/template?sslmode=disable"},
			Mail:     config.MailConfig{FromAddress: "noreply@example.com"},
			SMTP:     config.SMTPConfig{Host: "mailpit", Port: 1025},
			ObjectStorage: config.ObjectStorageConfig{
				Endpoint:        "http://minio:9000",
				Region:          "us-east-1",
				Bucket:          "template",
				AccessKeyID:     "minioadmin",
				SecretAccessKey: "minioadmin",
				UsePathStyle:    true,
			},
			OpenSearch: config.OpenSearchConfig{URL: "http://opensearch:9200"},
			Valkey:     config.ValkeyConfig{URL: "redis://valkey:6379/0"},
		},
	}
}

// [UT-AUTH-BE-S040] Short TTL blocks startup
func TestNewRuntimeWithConfigRejectsShortRefreshTokenTTL(t *testing.T) {
	t.Parallel()

	cfg := fullyConfiguredDevelopmentConfig()
	cfg.Auth.RefreshTokenTTL = 23 * time.Hour
	_, err := NewRuntimeWithConfig(context.Background(), cfg)
	if err == nil {
		t.Fatal("expected error for short refresh token TTL")
	}
}

// [UT-AUTH-BE-S040] Negative TTL blocks startup
func TestNewRuntimeWithConfigRejectsNegativeRefreshTokenTTL(t *testing.T) {
	t.Parallel()

	cfg := fullyConfiguredDevelopmentConfig()
	cfg.Auth.RefreshTokenTTL = -1 * time.Hour
	_, err := NewRuntimeWithConfig(context.Background(), cfg)
	if err == nil {
		t.Fatal("expected error for negative refresh token TTL")
	}
}
