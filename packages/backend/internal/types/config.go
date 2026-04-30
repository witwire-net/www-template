package types

import (
	"errors"
	"strings"
	"time"
)

const (
	defaultPort           = "8080"
	defaultAllowedOrigins = "http://localhost:5173,http://127.0.0.1:5173,http://localhost:5174,http://127.0.0.1:5174"
	defaultAppAuthValue   = "dev-app-auth"
	defaultWebAuthnRPID   = "localhost"
	defaultValkeyPrefix   = "www-template"
	defaultRecoveryBase   = "http://localhost:5174/login/recovery/consume"
	defaultSMTPPort       = 587
	defaultR2UsePathStyle = false
)

type ValkeyConfig struct {
	URL       string
	KeyPrefix string
}

type SMTPConfig struct {
	Host     string
	Port     int
	Username string
	Password string
}

type MailConfig struct {
	FromAddress string
}

type DatabaseConfig struct {
	URL string
}

type OpenSearchConfig struct {
	URL string
}

type ObjectStorageConfig struct {
	Endpoint        string
	Region          string
	Bucket          string
	AccessKeyID     string
	SecretAccessKey string
	UsePathStyle    bool
}

type InfraConfig struct {
	Database      DatabaseConfig
	Mail          MailConfig
	ObjectStorage ObjectStorageConfig
	OpenSearch    OpenSearchConfig
	SMTP          SMTPConfig
	Valkey        ValkeyConfig
}

type AuthConfig struct {
	ChallengeTTL                time.Duration
	RecoveryTokenTTL            time.Duration
	RecoverySessionTTL          time.Duration
	SessionIdleTTL              time.Duration
	SessionAbsoluteTTL          time.Duration
	PasskeyStartThrottleLimit   int
	PasskeyStartThrottleWindow  time.Duration
	RecoveryEmailThrottleLimit  int
	RecoveryEmailThrottleWindow time.Duration
	RecoveryIPThrottleLimit     int
	RecoveryIPThrottleWindow    time.Duration
	FailureLockThreshold        int
	FailureLockWindow           time.Duration
	FailureLockDuration         time.Duration
	WebAuthnRPID                string
	AccountRecoveryURLBase      string
}

type Config struct {
	AllowedOrigins []string
	AppBearerToken string
	Auth           AuthConfig
	Environment    string
	Infra          InfraConfig
	Port           string
	Observability  ObservabilityConfig
}

type ObservabilityConfig struct {
	OTELExporterOTLPEndpoint       string
	OTELExporterOTLPTracesEndpoint string
	OTELExporterOTLPLogsEndpoint   string
	OTELServiceName                string
	OTELResourceAttributes         string
}

func (c Config) AppAuthorizationValue() string {
	return "Bearer " + c.AppBearerToken
}

func (c Config) Validate() error {
	missing := make([]string, 0)
	if c.Environment != "development" && strings.TrimSpace(c.AppBearerToken) == "" {
		missing = append(missing, "app.bearer_token")
	}
	if strings.TrimSpace(c.Infra.Database.URL) == "" {
		missing = append(missing, "database.url")
	}
	if strings.TrimSpace(c.Infra.Valkey.URL) == "" {
		missing = append(missing, "valkey.url")
	}
	if strings.TrimSpace(c.Infra.SMTP.Host) == "" {
		missing = append(missing, "smtp.host")
	}
	if strings.TrimSpace(c.Infra.OpenSearch.URL) == "" {
		missing = append(missing, "opensearch.url")
	}
	if strings.TrimSpace(c.Infra.ObjectStorage.Endpoint) == "" {
		missing = append(missing, "object_storage.endpoint")
	}
	if strings.TrimSpace(c.Infra.ObjectStorage.Region) == "" {
		missing = append(missing, "object_storage.region")
	}
	if strings.TrimSpace(c.Infra.ObjectStorage.Bucket) == "" {
		missing = append(missing, "object_storage.bucket")
	}
	if strings.TrimSpace(c.Infra.ObjectStorage.AccessKeyID) == "" {
		missing = append(missing, "object_storage.access_key_id")
	}
	if strings.TrimSpace(c.Infra.ObjectStorage.SecretAccessKey) == "" {
		missing = append(missing, "object_storage.secret_access_key")
	}
	if strings.TrimSpace(c.Infra.SMTP.Host) == "" {
		missing = append(missing, "smtp.host")
	}
	if strings.TrimSpace(c.Infra.Mail.FromAddress) == "" {
		missing = append(missing, "mail.from_address")
	}
	if len(missing) > 0 {
		return errors.New(strings.Join(missing, ", ") + " is required")
	}

	return nil
}

func (c Config) AuthRuntime() AuthConfig {
	configured := c.Auth
	defaults := defaultAuthConfig()
	configured.ChallengeTTL = defaultDuration(configured.ChallengeTTL, defaults.ChallengeTTL)
	configured.RecoveryTokenTTL = defaultDuration(configured.RecoveryTokenTTL, defaults.RecoveryTokenTTL)
	configured.RecoverySessionTTL = defaultDuration(configured.RecoverySessionTTL, defaults.RecoverySessionTTL)
	configured.SessionIdleTTL = defaultDuration(configured.SessionIdleTTL, defaults.SessionIdleTTL)
	configured.SessionAbsoluteTTL = defaultDuration(configured.SessionAbsoluteTTL, defaults.SessionAbsoluteTTL)
	configured.PasskeyStartThrottleLimit = defaultInt(configured.PasskeyStartThrottleLimit, defaults.PasskeyStartThrottleLimit)
	configured.PasskeyStartThrottleWindow = defaultDuration(configured.PasskeyStartThrottleWindow, defaults.PasskeyStartThrottleWindow)
	configured.RecoveryEmailThrottleLimit = defaultInt(configured.RecoveryEmailThrottleLimit, defaults.RecoveryEmailThrottleLimit)
	configured.RecoveryEmailThrottleWindow = defaultDuration(configured.RecoveryEmailThrottleWindow, defaults.RecoveryEmailThrottleWindow)
	configured.RecoveryIPThrottleLimit = defaultInt(configured.RecoveryIPThrottleLimit, defaults.RecoveryIPThrottleLimit)
	configured.RecoveryIPThrottleWindow = defaultDuration(configured.RecoveryIPThrottleWindow, defaults.RecoveryIPThrottleWindow)
	configured.FailureLockThreshold = defaultInt(configured.FailureLockThreshold, defaults.FailureLockThreshold)
	configured.FailureLockWindow = defaultDuration(configured.FailureLockWindow, defaults.FailureLockWindow)
	configured.FailureLockDuration = defaultDuration(configured.FailureLockDuration, defaults.FailureLockDuration)
	configured.WebAuthnRPID = defaultString(configured.WebAuthnRPID, defaults.WebAuthnRPID)
	configured.AccountRecoveryURLBase = defaultString(configured.AccountRecoveryURLBase, defaults.AccountRecoveryURLBase)

	return configured
}

func defaultAuthConfig() AuthConfig {
	return AuthConfig{
		ChallengeTTL:                5 * time.Minute,
		RecoveryTokenTTL:            30 * time.Minute,
		RecoverySessionTTL:          15 * time.Minute,
		SessionIdleTTL:              12 * time.Hour,
		SessionAbsoluteTTL:          14 * 24 * time.Hour,
		PasskeyStartThrottleLimit:   5,
		PasskeyStartThrottleWindow:  5 * time.Minute,
		RecoveryEmailThrottleLimit:  3,
		RecoveryEmailThrottleWindow: time.Hour,
		RecoveryIPThrottleLimit:     10,
		RecoveryIPThrottleWindow:    time.Hour,
		FailureLockThreshold:        10,
		FailureLockWindow:           15 * time.Minute,
		FailureLockDuration:         15 * time.Minute,
		WebAuthnRPID:                defaultWebAuthnRPID,
		AccountRecoveryURLBase:      defaultRecoveryBase,
	}
}

func defaultDuration(value time.Duration, fallback time.Duration) time.Duration {
	if value == 0 {
		return fallback
	}

	return value
}

func defaultInt(value int, fallback int) int {
	if value == 0 {
		return fallback
	}

	return value
}

func defaultString(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}

	return value
}
