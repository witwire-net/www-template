package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestConfigValidateRequiresInfrastructureSettings(t *testing.T) {
	t.Parallel()

	err := Config{Environment: "development"}.Validate()
	if err == nil {
		t.Fatal("expected error for missing infrastructure settings")
	}
}

func TestConfigValidateAcceptsFullyConfiguredDevelopmentRuntime(t *testing.T) {
	t.Parallel()

	err := Config{
		AppBearerToken: "dev-app-auth",
		Environment:    "development",
		Infra: InfraConfig{
			Database: DatabaseConfig{URL: "postgres://template:template@postgres:5432/template?sslmode=disable"},
			Mail:     MailConfig{FromAddress: "noreply@example.com"},
			SMTP:     SMTPConfig{Host: "mailpit", Port: 1025},
			ObjectStorage: ObjectStorageConfig{
				Endpoint:        "http://minio:9000",
				Region:          "us-east-1",
				Bucket:          "template",
				AccessKeyID:     "minioadmin",
				SecretAccessKey: "minioadmin",
				UsePathStyle:    true,
			},
			OpenSearch: OpenSearchConfig{URL: "http://opensearch:9200"},
			Valkey:     ValkeyConfig{URL: "redis://valkey:6379/0"},
		},
	}.Validate()
	if err != nil {
		t.Fatalf("expected valid config, got %v", err)
	}
}

func TestConfigValidateAdminRuntimeRequiresSurfaceSettings(t *testing.T) {
	t.Parallel()

	// Step 1: Product runtime としては有効な設定でも、Admin 専用 surface 設定がなければ Admin startup が失敗することを固定する。
	cfg := fullyConfiguredAdminValidationBase()
	cfg.Admin = AdminRuntimeConfig{}
	err := cfg.ValidateAdminRuntime()
	if err == nil {
		t.Fatal("expected error for missing Admin runtime settings")
	}

	// Step 2: 主要な必須 field が error に列挙され、運用者が欠落箇所を特定できることを確認する。
	message := fmt.Sprint(err)
	for _, want := range []string{"admin.domain is required", "admin.product_domain is required", "admin.cookie.name is required", "admin.database.role is required", "admin.valkey.url is required", "admin.product_valkey.url is required"} {
		if !strings.Contains(message, want) {
			t.Fatalf("expected validation error to contain %q, got %q", want, message)
		}
	}
}

func TestConfigValidateAdminRuntimeAcceptsDevelopmentSurfaceSettings(t *testing.T) {
	t.Parallel()

	// Step 1: development では localhost の http origin と insecure local cookie を許可し、local 開発を Product production policy で壊さない。
	err := fullyConfiguredAdminValidationBase().ValidateAdminRuntime()
	if err != nil {
		t.Fatalf("expected valid Admin runtime config, got %v", err)
	}
}

func TestConfigValidateAdminRuntimeRejectsNonBcryptBootstrapHash(t *testing.T) {
	t.Parallel()

	// Step 1: bootstrap gate が有効な状態で高速 digest 形式の hash を設定し、起動時に fail-close されることを固定する。
	cfg := fullyConfiguredAdminValidationBase()
	cfg.Admin.Bootstrap = AdminBootstrapConfig{
		Enabled:    true,
		SecretHash: "cGFIBQC2yFy4n7fRpQS_RGruxzrq5UwXJpkxlyLj1QQ",
		ExpiresAt:  time.Now().UTC().Add(time.Hour),
	}
	err := cfg.ValidateAdminRuntime()
	if err == nil {
		t.Fatal("expected error for non-bcrypt bootstrap hash")
	}
	if !strings.Contains(fmt.Sprint(err), "admin.bootstrap.secret_hash must be a bcrypt hash") {
		t.Fatalf("expected bcrypt bootstrap hash error, got %v", err)
	}
}

func TestConfigValidateAdminRuntimeRejectsDomainCollision(t *testing.T) {
	t.Parallel()

	// Step 1: Admin と Product の hostname が同じで port だけ違う設定を作り、domain 分離の失敗を validation で検出する。
	cfg := fullyConfiguredAdminValidationBase()
	cfg.Admin.ProductDomain = "http://admin.localhost:5174"
	err := cfg.ValidateAdminRuntime()
	if err == nil {
		t.Fatal("expected error for Admin/Product domain collision")
	}
	if !strings.Contains(fmt.Sprint(err), "admin.domain must differ from admin.product_domain") {
		t.Fatalf("expected domain collision error, got %v", err)
	}
}

func TestConfigValidateAdminRuntimeRejectsMismatchedCookieDomain(t *testing.T) {
	t.Parallel()

	// Step 1: Cookie domain が Admin origin と違う場合、refreshToken が別 surface に送られる危険があるため拒否されることを確認する。
	cfg := fullyConfiguredAdminValidationBase()
	cfg.Admin.Cookie.Domain = "app.localhost"
	err := cfg.ValidateAdminRuntime()
	if err == nil {
		t.Fatal("expected error for mismatched Admin cookie domain")
	}
	if !strings.Contains(fmt.Sprint(err), "admin.cookie.domain must match admin.domain host") {
		t.Fatalf("expected cookie domain error, got %v", err)
	}
}

func TestConfigValidateAdminRuntimeRejectsInsecureProductionCookie(t *testing.T) {
	t.Parallel()

	// Step 1: [OpenSpec Task 4.48] insecure production cookie rejection の追跡点として、production 設定では Secure=false の Admin refreshToken Cookie を許さないことを固定する。
	cfg := fullyConfiguredAdminValidationBase()
	cfg.Environment = "production"
	cfg.Admin.Domain = "https://admin.example.com"
	cfg.Admin.ProductDomain = "https://app.example.com"
	cfg.Admin.Cookie.Domain = "admin.example.com"
	cfg.Admin.Cookie.Secure = false
	err := cfg.ValidateAdminRuntime()
	if err == nil {
		t.Fatal("expected error for insecure production Admin cookie")
	}
	if !strings.Contains(fmt.Sprint(err), "admin.cookie.secure is required outside development") {
		t.Fatalf("expected secure cookie error, got %v", err)
	}
}

func TestConfigValidateAdminRuntimeRejectsValkeyLogicalDBCollision(t *testing.T) {
	t.Parallel()

	// Step 1: Admin と Product が同じ Valkey endpoint / logical DB を指す誤設定を作り、operator auth state と Product auth state の混在を再現する。
	cfg := fullyConfiguredAdminValidationBase()
	cfg.Admin.ProductValkey.URL = "redis://valkey:6379/1"
	err := cfg.ValidateAdminRuntime()
	if err == nil {
		t.Fatal("expected error for Admin/Product Valkey logical DB collision")
	}

	// Step 2: startup validation が logical DB 衝突専用の error を返し、他の Admin surface 設定エラーに紛れないことを固定する。
	if !strings.Contains(fmt.Sprint(err), "admin.valkey.url must not share logical DB with admin.product_valkey.url") {
		t.Fatalf("expected Valkey logical DB collision error, got %v", err)
	}
}

func TestConfigValidateAdminRuntimeRejectsOpenSearchNamespaceCollision(t *testing.T) {
	t.Parallel()

	// Step 1: [ADMIN-CONSOLE-BE-S086] Admin audit prefix と Product prefix が同一の場合、同じ index 群へ書き込む危険があるため startup validation で拒否する。
	cfg := fullyConfiguredAdminValidationBase()
	cfg.Infra.OpenSearch.ProductIndexPrefix = "admin-audit"
	err := cfg.ValidateAdminRuntime()
	if err == nil {
		t.Fatal("expected error for OpenSearch namespace collision")
	}
	if !strings.Contains(fmt.Sprint(err), "opensearch.admin_audit_index_prefix must not collide with opensearch.product_index_prefix") {
		t.Fatalf("expected OpenSearch namespace collision error, got %v", err)
	}
}

func TestConfigValidateAdminRuntimeRejectsOpenSearchNamespaceContainment(t *testing.T) {
	t.Parallel()

	// Step 1: [ADMIN-CONSOLE-BE-S086] prefix の包含関係も wildcard pattern が交差するため、同一 prefix と同じ fail-close 対象にする。
	cfg := fullyConfiguredAdminValidationBase()
	cfg.Infra.OpenSearch.ProductIndexPrefix = "audit"
	err := cfg.ValidateAdminRuntime()
	if err == nil {
		t.Fatal("expected error for OpenSearch namespace containment")
	}
	if !strings.Contains(fmt.Sprint(err), "opensearch.admin_audit_index_prefix must not collide with opensearch.product_index_prefix") {
		t.Fatalf("expected OpenSearch namespace containment error, got %v", err)
	}
}

func TestLoadAdminConfigMapsAdminTOMLSurfaceFields(t *testing.T) {
	// Step 1: Admin 専用 TOML を一時ファイルへ作成し、Product loader ではなく Admin loader の alias だけを検証する。
	configPath := filepath.Join(t.TempDir(), "admin.toml")
	data := []byte(`
[app]
environment = "development"

[server]
origin = "http://admin.localhost:5176"
product_origin = "http://app.localhost:5174"

[cookie]
name = "www_template_admin_refresh"
domain = "admin.localhost"
path = "/"
secure = false
same_site = "Lax"

[auth]
rp_id = "admin.localhost"
jwt_secret = "dev-admin-jwt-secret-change-in-production"

[database]
admin_role = "admin_console_write"
url = "postgres://admin_console:admin_console@postgres:5432/www-template?sslmode=disable"

[valkey]
admin_url = "redis://valkey:6379/1"
product_url = "redis://valkey:6379/0"

[opensearch]
url = "http://opensearch:9200"
admin_audit_index_prefix = "admin-audit"
product_index_prefix = "product-domain"

[observability]
otel_exporter_otlp_endpoint = "otel-collector:4317"
otel_service_name = "www-template-admin-api"
`)
	if err := os.WriteFile(configPath, data, 0o600); err != nil {
		t.Fatalf("write temp admin config: %v", err)
	}
	t.Setenv("ADMIN_CONFIG_PATH", configPath)
	t.Setenv("CONFIG_PATH", "")

	// Step 2: Admin loader が TOML の Admin surface field と Admin-only infra alias を Config へ写すことを確認する。
	cfg := LoadAdminConfig()
	if cfg.Admin.Domain != "http://admin.localhost:5176" {
		t.Fatalf("expected Admin domain mapping, got %q", cfg.Admin.Domain)
	}
	if cfg.Admin.ProductDomain != "http://app.localhost:5174" {
		t.Fatalf("expected Product domain mapping, got %q", cfg.Admin.ProductDomain)
	}
	if cfg.Admin.Cookie.Name != "www_template_admin_refresh" || cfg.Admin.Cookie.Domain != "admin.localhost" {
		t.Fatalf("expected Admin cookie mapping, got %#v", cfg.Admin.Cookie)
	}
	if cfg.Admin.Database.Role != "admin_console_write" {
		t.Fatalf("expected Admin database role mapping, got %q", cfg.Admin.Database.Role)
	}
	if cfg.Admin.Valkey.URL != "redis://valkey:6379/1" || cfg.Infra.Valkey.URL != "redis://valkey:6379/1" {
		t.Fatalf("expected Admin Valkey URL mapping, got admin=%q infra=%q", cfg.Admin.Valkey.URL, cfg.Infra.Valkey.URL)
	}
	if cfg.Admin.ProductValkey.URL != "redis://valkey:6379/0" {
		t.Fatalf("expected Product Valkey URL mapping for Admin validation, got %q", cfg.Admin.ProductValkey.URL)
	}
	if cfg.Infra.Database.URL == "" || !strings.Contains(cfg.Infra.Database.URL, "admin_console") {
		t.Fatalf("expected Admin database URL mapping, got %q", cfg.Infra.Database.URL)
	}
	assertAdminConfigSurfaceInfrastructure(t, cfg)
}

func assertAdminConfigSurfaceInfrastructure(t *testing.T, cfg Config) {
	t.Helper()

	if cfg.Infra.OpenSearch.AdminAuditIndexPrefix != "admin-audit" || cfg.Infra.OpenSearch.ProductIndexPrefix != "product-domain" {
		t.Fatalf("expected OpenSearch namespace mapping, got %#v", cfg.Infra.OpenSearch)
	}
	if cfg.Observability.OTELExporterOTLPEndpoint != "otel-collector:4317" || cfg.Observability.OTELServiceName != "www-template-admin-api" {
		t.Fatalf("expected Admin observability mapping, got %#v", cfg.Observability)
	}
}

func fullyConfiguredAdminValidationBase() Config {
	return Config{
		Environment: "development",
		Admin: AdminRuntimeConfig{
			Domain:        "http://admin.localhost:5176",
			ProductDomain: "http://app.localhost:5174",
			Cookie: AdminCookieConfig{
				Name:     "www_template_admin_refresh",
				Domain:   "admin.localhost",
				Path:     "/",
				SameSite: "Lax",
			},
			Database:      AdminDatabaseConfig{Role: "admin_console_write"},
			Valkey:        ValkeyConfig{URL: "redis://valkey:6379/1"},
			ProductValkey: ValkeyConfig{URL: "redis://valkey:6379/0"},
		},
		Infra: InfraConfig{OpenSearch: OpenSearchConfig{URL: "http://opensearch:9200", AdminAuditIndexPrefix: "admin-audit", ProductIndexPrefix: "product-domain"}},
	}
}

var unsafeProductionConfigCases = []struct {
	name   string
	modify func(*Config)
}{
	{
		name: "[AUTH-BE-S025] localhost origin",
		modify: func(c *Config) {
			c.AllowedOrigins = []string{"https://example.com", "http://localhost:5173"}
		},
	},
	{
		name: "[AUTH-BE-S025] plain HTTP origin",
		modify: func(c *Config) {
			c.AllowedOrigins = []string{"http://example.com"}
		},
	},
	{
		name: "[AUTH-BE-S025] wildcard origin",
		modify: func(c *Config) {
			c.AllowedOrigins = []string{"https://*.example.com"}
		},
	},
	{
		name: "[AUTH-BE-S025] mismatched RP ID",
		modify: func(c *Config) {
			c.Auth.WebAuthnRPID = "evil.com"
		},
	},
	{
		name: "[AUTH-BE-S025] plain HTTP recovery URL",
		modify: func(c *Config) {
			c.Auth.AccountRecoveryURLBase = "http://example.com/recover"
		},
	},
	{
		name: "[AUTH-BE-S025] missing trusted proxy",
		modify: func(c *Config) {
			c.TrustedProxyCIDRs = nil
		},
	},
	{
		name: "[AUTH-BE-S025] invalid trusted proxy CIDR",
		modify: func(c *Config) {
			c.TrustedProxyCIDRs = []string{"not-a-cidr"}
		},
	},
	{
		name: "[AUTH-BE-S025] missing auth body limit",
		modify: func(c *Config) {
			c.Auth.AuthBodyLimitBytes = 0
		},
	},
	{
		name: "[AUTH-BE-S025] insecure mail transport",
		modify: func(c *Config) {
			c.Infra.SMTP.SecureTransport = false
		},
	},
	{
		name: "[AUTH-BE-S025] missing secret hash key",
		modify: func(c *Config) {
			c.Auth.SecretHashKey = ""
		},
	},
	{
		name: "[AUTH-BE-S025] default secret hash key",
		modify: func(c *Config) {
			c.Auth.SecretHashKey = "dev-pepper-change-in-production"
		},
	},
	{
		name: "[AUTH-BE-S025] short secret hash key",
		modify: func(c *Config) {
			c.Auth.SecretHashKey = "short"
		},
	},
	{
		name: "[AUTH-BE-S025] space-padded default secret hash key",
		modify: func(c *Config) {
			c.Auth.SecretHashKey = "  dev-pepper-change-in-production  "
		},
	},
	{
		name: "[AUTH-BE-S025] missing jwt secret",
		modify: func(c *Config) {
			c.Auth.JWTSecret = ""
		},
	},
	{
		name: "[AUTH-BE-S025] default jwt secret",
		modify: func(c *Config) {
			c.Auth.JWTSecret = "change-this-to-a-long-random-jwt-secret-in-production"
		},
	},
	{
		name: "[AUTH-BE-S025] short jwt secret",
		modify: func(c *Config) {
			c.Auth.JWTSecret = "short"
		},
	},
	{
		name: "[AUTH-BE-S025] jwt secret same as secret hash key",
		modify: func(c *Config) {
			c.Auth.JWTSecret = c.Auth.SecretHashKey
		},
	},
}

func TestConfigValidateRejectsUnsafeProductionConfig(t *testing.T) {
	t.Parallel()

	for _, tc := range unsafeProductionConfigCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			runProductionValidationCase(t, tc.modify)
		})
	}
}

// runProductionValidationCase は安全な production 構成のベースラインに対して modify を適用し、
// Validate がエラーを返すことを検証するヘルパー。
func runProductionValidationCase(t *testing.T, modify func(*Config)) {
	t.Helper()

	cfg := &Config{
		AppBearerToken: "prod-app-auth",
		Environment:    "production",
		AllowedOrigins: []string{"https://example.com"},
		Auth: AuthConfig{
			WebAuthnRPID:           "example.com",
			AccountRecoveryURLBase: "https://example.com/recover",
			AuthBodyLimitBytes:     1 << 20,
			SecretHashKey:          "a-very-long-production-pepper-key-that-is-safe-to-use-12345",
			JWTSecret:              "a-very-long-production-jwt-secret-that-is-safe-to-use-12345",
		},
		TrustedProxyCIDRs: []string{"10.0.0.0/8", "172.16.0.0/12"},
		Infra: InfraConfig{
			Database: DatabaseConfig{URL: "postgres://template:template@postgres:5432/template?sslmode=disable"},
			Mail:     MailConfig{FromAddress: "noreply@example.com"},
			SMTP:     SMTPConfig{Host: "mailpit", Port: 465, SecureTransport: true},
			ObjectStorage: ObjectStorageConfig{
				Endpoint:        "http://minio:9000",
				Region:          "us-east-1",
				Bucket:          "template",
				AccessKeyID:     "minioadmin",
				SecretAccessKey: "minioadmin",
				UsePathStyle:    true,
			},
			OpenSearch: OpenSearchConfig{URL: "http://opensearch:9200"},
			Valkey:     ValkeyConfig{URL: "redis://valkey:6379/0"},
		},
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected valid production config, got %v", err)
	}
}
