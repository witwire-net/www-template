package types

import "testing"

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
