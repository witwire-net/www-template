package app

import (
	"context"
	stdhttp "net/http"
	"net/http/httptest"
	"testing"
	"time"

	adminhttp "www-template/packages/backend/internal/adapter/http/admin"
	"www-template/packages/backend/internal/platform/config"
)

func TestNewProductRuntimeWithConfigFailsClosedWithoutTokenOutsideDevelopment(t *testing.T) {
	t.Parallel()

	_, err := NewProductRuntimeWithConfig(context.Background(), config.Config{
		AllowedOrigins: []string{"http://www.localhost:5173", "http://localhost:5174"},
		Environment:    "production",
		Port:           "8080",
	})
	if err == nil {
		t.Fatal("expected error for missing production bearer token")
	}
}

func TestNewProductRuntimeWithConfigFailsClosedWhenRequiredInfrastructureIsMissing(t *testing.T) {
	t.Parallel()

	_, err := NewProductRuntimeWithConfig(context.Background(), config.Config{
		AllowedOrigins: []string{"http://www.localhost:5173", "http://localhost:5174"},
		AppBearerToken: "dev-app-auth",
		Environment:    "development",
		Port:           "8080",
	})
	if err == nil {
		t.Fatal("expected error for missing infrastructure settings")
	}
}

func TestNewAdminRuntimeWithConfigFailsClosedWithoutAdminSurfaceConfig(t *testing.T) {
	t.Parallel()

	// Step 1: Product runtime 用の完全な development 設定を流用し、Admin surface 専用 field が空の状態を作る。
	cfg := fullyConfiguredDevelopmentConfig()

	// Step 2: Admin runtime 構築が infrastructure 接続へ進む前に、Admin 固有設定不足で fail-close することを確認する。
	_, err := NewAdminRuntimeWithConfig(context.Background(), cfg)
	if err == nil {
		t.Fatal("expected error for missing Admin surface runtime config")
	}
}

func TestAdminOperatorAuthRuntimeConfigUsesOperatorSemantics(t *testing.T) {
	t.Parallel()

	// Step 1: Admin operator auth に渡す runtime TTL を作り、Product account config 名ではなく operator session config DTO へ変換する。
	authRuntime := config.AuthConfig{
		SessionIdleTTL:     15 * time.Minute,
		RefreshTokenTTL:    30 * 24 * time.Hour,
		SessionAbsoluteTTL: 60 * 24 * time.Hour,
		WebAuthnRPID:       "admin.example.com",
	}

	// Step 2: 変換結果の field 名と値が operator access/refresh session semantics を表すことを [ADMIN-AUTH-BE-S087] として固定する。
	operatorConfig := operatorAuthConfigFromRuntime(authRuntime)
	if operatorConfig.OperatorAccessTokenTTL != 15*time.Minute || operatorConfig.OperatorRefreshSessionTTL != 30*24*time.Hour || operatorConfig.OperatorRefreshCookieLifetime != 30*24*time.Hour {
		t.Fatalf("[ADMIN-AUTH-BE-S087] unexpected operator auth config: %+v", operatorConfig)
	}
	if operatorConfig.WebAuthnRPID != "admin.example.com" {
		t.Fatalf("[ADMIN-AUTH-BE-S087] expected Admin operator RP ID, got %q", operatorConfig.WebAuthnRPID)
	}
}

// fullyConfiguredDevelopmentConfig は開発環境で起動可能な最小限のインフラ設定を持つ Config を返す。
// これにより、TTL 検証の前に infrastructure missing で落ちることを防ぐ。
func fullyConfiguredDevelopmentConfig() config.Config {
	return config.Config{
		AllowedOrigins: []string{"http://www.localhost:5173", "http://localhost:5174"},
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
func TestNewProductRuntimeWithConfigRejectsShortRefreshTokenTTL(t *testing.T) {
	t.Parallel()

	cfg := fullyConfiguredDevelopmentConfig()
	cfg.Auth.RefreshTokenTTL = 23 * time.Hour
	_, err := NewProductRuntimeWithConfig(context.Background(), cfg)
	if err == nil {
		t.Fatal("expected error for short refresh token TTL")
	}
}

// [UT-AUTH-BE-S040] Negative TTL blocks startup
func TestNewProductRuntimeWithConfigRejectsNegativeRefreshTokenTTL(t *testing.T) {
	t.Parallel()

	cfg := fullyConfiguredDevelopmentConfig()
	cfg.Auth.RefreshTokenTTL = -1 * time.Hour
	_, err := NewProductRuntimeWithConfig(context.Background(), cfg)
	if err == nil {
		t.Fatal("expected error for negative refresh token TTL")
	}
}

// TestAdminHTTPAdapterDoesNotRegisterProductOperations は Admin HTTP adapter が Product operation を公開しないことを検証する。
// Admin adapter 接続後の 4.7 境界では、Admin binary に接続される adapter が Product router を流用しないことを `/api/v1/*` の Product path への 404 で固定する。
func TestAdminHTTPAdapterDoesNotRegisterProductOperations(t *testing.T) {
	t.Parallel()

	// Step 1: Admin runtime が使う Admin HTTP adapter を直接作成し、Product runtime の Gin router や Product generated handlers を経由しないことを確認対象にする。
	mux := adminhttp.NewRouter(fullyConfiguredDevelopmentConfig(), adminhttp.Dependencies{})
	productRoutes := []struct {
		method string
		path   string
	}{
		{method: stdhttp.MethodGet, path: "/api/v1/status"},
		{method: stdhttp.MethodPost, path: "/api/v1/auth/refresh"},
		{method: stdhttp.MethodPost, path: "/api/v1/auth/logout"},
		{method: stdhttp.MethodPost, path: "/api/v1/auth/passkey/register/start"},
		{method: stdhttp.MethodPost, path: "/api/v1/auth/passkey/register"},
		{method: stdhttp.MethodGet, path: "/api/v1/passkeys"},
		{method: stdhttp.MethodDelete, path: "/api/v1/passkeys/01ARZ3NDEKTSV4RRFFQ69G5FAV"},
		{method: stdhttp.MethodGet, path: "/api/v1/sessions"},
		{method: stdhttp.MethodDelete, path: "/api/v1/sessions/others"},
		{method: stdhttp.MethodDelete, path: "/api/v1/sessions/01ARZ3NDEKTSV4RRFFQ69G5FAV"},
	}

	// Step 2: Product 専用 path が Admin mux で 404 になることを確認し、Admin binary への Product operation 混入を検出できるようにする。
	for _, route := range productRoutes {
		request := httptest.NewRequest(route.method, route.path, nil)
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, request)
		if response.Code != stdhttp.StatusNotFound {
			t.Fatalf("expected Admin mux to reject Product route %s %s with 404, got %d body=%s", route.method, route.path, response.Code, response.Body.String())
		}
	}
}
