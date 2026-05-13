package config

import (
	"testing"
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
