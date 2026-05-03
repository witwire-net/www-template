package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/pelletier/go-toml/v2"
)

const (
	defaultReadTimeout  = 30 * time.Second
	defaultWriteTimeout = 30 * time.Second
	defaultIdleTimeout  = 120 * time.Second
)

// tomlConfig mirrors the structure of .config/*.toml files.
type tomlConfig struct {
	App struct {
		Environment string `toml:"environment"`
		BearerToken string `toml:"bearer_token"`
	} `toml:"app"`
	Server struct {
		Port              int      `toml:"port"`
		AllowedOrigins    []string `toml:"allowed_origins"`
		TrustedProxyCIDRs []string `toml:"trusted_proxy_cidrs"`
		ReadTimeout       string   `toml:"read_timeout"`
		WriteTimeout      string   `toml:"write_timeout"`
		IdleTimeout       string   `toml:"idle_timeout"`
	} `toml:"server"`
	Auth struct {
		WebAuthnRPID           string `toml:"webauthn_rp_id"`
		AccountRecoveryURLBase string `toml:"account_recovery_url_base"`
		ChallengeTTL           string `toml:"challenge_ttl"`
		RecoveryTokenTTL       string `toml:"recovery_token_ttl"`
		RecoverySessionTTL     string `toml:"recovery_session_ttl"`
		ReauthSessionTTL       string `toml:"reauth_session_ttl"`
		SessionIdleTTL         string `toml:"session_idle_ttl"`
		SessionAbsoluteTTL     string `toml:"session_absolute_ttl"`
		PasskeyStartLimit      int    `toml:"passkey_start_limit"`
		PasskeyStartWindow     string `toml:"passkey_start_window"`
		RecoveryEmailLimit     int    `toml:"recovery_email_limit"`
		RecoveryEmailWindow    string `toml:"recovery_email_window"`
		RecoveryIPLimit        int    `toml:"recovery_ip_limit"`
		RecoveryIPWindow       string `toml:"recovery_ip_window"`
		FailureThreshold       int    `toml:"failure_threshold"`
		FailureWindow          string `toml:"failure_window"`
		FailureDuration        string `toml:"failure_duration"`
		AuthBodyLimitBytes     int    `toml:"auth_body_limit_bytes"`
		SecretHashKey          string `toml:"secret_hash_key"`
	} `toml:"auth"`
	Database struct {
		URL string `toml:"url"`
	} `toml:"database"`
	Valkey struct {
		URL       string `toml:"url"`
		KeyPrefix string `toml:"key_prefix"`
	} `toml:"valkey"`
	OpenSearch struct {
		URL string `toml:"url"`
	} `toml:"opensearch"`
	ObjectStorage struct {
		Endpoint        string `toml:"endpoint"`
		Region          string `toml:"region"`
		Bucket          string `toml:"bucket"`
		AccessKeyID     string `toml:"access_key_id"`
		SecretAccessKey string `toml:"secret_access_key"`
		UsePathStyle    bool   `toml:"use_path_style"`
	} `toml:"object_storage"`
	SMTP struct {
		Host            string `toml:"host"`
		Port            int    `toml:"port"`
		Username        string `toml:"username"`
		Password        string `toml:"password"`
		SecureTransport bool   `toml:"secure_transport"`
	} `toml:"smtp"`
	Mail struct {
		FromAddress string `toml:"from_address"`
	} `toml:"mail"`
	Observability struct {
		OTELExporterOTLPEndpoint       string `toml:"otel_exporter_otlp_endpoint"`
		OTELExporterOTLPTracesEndpoint string `toml:"otel_exporter_otlp_traces_endpoint"`
		OTELExporterOTLPLogsEndpoint   string `toml:"otel_exporter_otlp_logs_endpoint"`
		OTELServiceName                string `toml:"otel_service_name"`
		OTELResourceAttributes         string `toml:"otel_resource_attributes"`
	} `toml:"observability"`
}

func resolveConfigPath() string {
	if envPath := os.Getenv("CONFIG_PATH"); envPath != "" {
		return envPath
	}

	// Try relative to working directory (project root).
	if _, err := os.Stat(".config/local.toml"); err == nil {
		abs, _ := filepath.Abs(".config/local.toml")
		return abs
	}

	// Try relative to this package (packages/backend/internal/app).
	if _, err := os.Stat("../../.config/local.toml"); err == nil {
		abs, _ := filepath.Abs("../../.config/local.toml")
		return abs
	}

	return ""
}

func LoadConfig() Config {
	configPath := resolveConfigPath()
	if configPath == "" {
		panic("config file not found. Set CONFIG_PATH or place .config/local.toml at the project root")
	}

	data, err := os.ReadFile(filepath.Clean(configPath))
	if err != nil {
		panic(fmt.Sprintf("read config file %s: %v", configPath, err))
	}

	var raw tomlConfig
	if err := toml.Unmarshal(data, &raw); err != nil {
		panic(fmt.Sprintf("parse config file %s: %v", configPath, err))
	}

	environment := defaultString(raw.App.Environment, "development")
	allowedOrigins := raw.Server.AllowedOrigins
	if len(allowedOrigins) == 0 {
		allowedOrigins = strings.Split(defaultAllowedOrigins, ",")
		for i := range allowedOrigins {
			allowedOrigins[i] = strings.TrimSpace(allowedOrigins[i])
		}
	}

	appBearerToken := strings.TrimSpace(raw.App.BearerToken)
	if environment == "development" && appBearerToken == "" {
		appBearerToken = defaultAppAuthValue
	}

	return Config{
		AllowedOrigins:     allowedOrigins,
		AppBearerToken:     appBearerToken,
		Auth:               buildAuthConfig(raw.Auth),
		Environment:        environment,
		TrustedProxyCIDRs:  raw.Server.TrustedProxyCIDRs,
		ServerReadTimeout:  parseDuration(raw.Server.ReadTimeout, defaultReadTimeout),
		ServerWriteTimeout: parseDuration(raw.Server.WriteTimeout, defaultWriteTimeout),
		ServerIdleTimeout:  parseDuration(raw.Server.IdleTimeout, defaultIdleTimeout),
		Infra: InfraConfig{
			Database: DatabaseConfig{
				URL: strings.TrimSpace(raw.Database.URL),
			},
			Mail: MailConfig{
				FromAddress: strings.TrimSpace(raw.Mail.FromAddress),
			},
			ObjectStorage: ObjectStorageConfig{
				Endpoint:        strings.TrimSpace(raw.ObjectStorage.Endpoint),
				Region:          strings.TrimSpace(raw.ObjectStorage.Region),
				Bucket:          strings.TrimSpace(raw.ObjectStorage.Bucket),
				AccessKeyID:     strings.TrimSpace(raw.ObjectStorage.AccessKeyID),
				SecretAccessKey: strings.TrimSpace(raw.ObjectStorage.SecretAccessKey),
				UsePathStyle:    raw.ObjectStorage.UsePathStyle,
			},
			OpenSearch: OpenSearchConfig{
				URL: strings.TrimSpace(raw.OpenSearch.URL),
			},
			Valkey: ValkeyConfig{
				URL:       strings.TrimSpace(raw.Valkey.URL),
				KeyPrefix: defaultString(raw.Valkey.KeyPrefix, defaultValkeyPrefix),
			},
			SMTP: SMTPConfig{
				Host:            strings.TrimSpace(raw.SMTP.Host),
				Port:            defaultInt(raw.SMTP.Port, defaultSMTPPort),
				Username:        strings.TrimSpace(raw.SMTP.Username),
				Password:        strings.TrimSpace(raw.SMTP.Password),
				SecureTransport: raw.SMTP.SecureTransport,
			},
		},
		Port: defaultString(strconv.Itoa(raw.Server.Port), defaultPort),
		Observability: ObservabilityConfig{
			OTELExporterOTLPEndpoint:       strings.TrimSpace(raw.Observability.OTELExporterOTLPEndpoint),
			OTELExporterOTLPTracesEndpoint: strings.TrimSpace(raw.Observability.OTELExporterOTLPTracesEndpoint),
			OTELExporterOTLPLogsEndpoint:   strings.TrimSpace(raw.Observability.OTELExporterOTLPLogsEndpoint),
			OTELServiceName:                strings.TrimSpace(raw.Observability.OTELServiceName),
			OTELResourceAttributes:         strings.TrimSpace(raw.Observability.OTELResourceAttributes),
		},
	}
}

func buildAuthConfig(raw struct {
	WebAuthnRPID           string `toml:"webauthn_rp_id"`
	AccountRecoveryURLBase string `toml:"account_recovery_url_base"`
	ChallengeTTL           string `toml:"challenge_ttl"`
	RecoveryTokenTTL       string `toml:"recovery_token_ttl"`
	RecoverySessionTTL     string `toml:"recovery_session_ttl"`
	ReauthSessionTTL       string `toml:"reauth_session_ttl"`
	SessionIdleTTL         string `toml:"session_idle_ttl"`
	SessionAbsoluteTTL     string `toml:"session_absolute_ttl"`
	PasskeyStartLimit      int    `toml:"passkey_start_limit"`
	PasskeyStartWindow     string `toml:"passkey_start_window"`
	RecoveryEmailLimit     int    `toml:"recovery_email_limit"`
	RecoveryEmailWindow    string `toml:"recovery_email_window"`
	RecoveryIPLimit        int    `toml:"recovery_ip_limit"`
	RecoveryIPWindow       string `toml:"recovery_ip_window"`
	FailureThreshold       int    `toml:"failure_threshold"`
	FailureWindow          string `toml:"failure_window"`
	FailureDuration        string `toml:"failure_duration"`
	AuthBodyLimitBytes     int    `toml:"auth_body_limit_bytes"`
	SecretHashKey          string `toml:"secret_hash_key"`
}) AuthConfig {
	defaults := defaultAuthConfig()

	return AuthConfig{
		ChallengeTTL:                parseDuration(raw.ChallengeTTL, defaults.ChallengeTTL),
		RecoveryTokenTTL:            parseDuration(raw.RecoveryTokenTTL, defaults.RecoveryTokenTTL),
		RecoverySessionTTL:          parseDuration(raw.RecoverySessionTTL, defaults.RecoverySessionTTL),
		ReauthSessionTTL:            parseDuration(raw.ReauthSessionTTL, defaults.ReauthSessionTTL),
		SessionIdleTTL:              parseDuration(raw.SessionIdleTTL, defaults.SessionIdleTTL),
		SessionAbsoluteTTL:          parseDuration(raw.SessionAbsoluteTTL, defaults.SessionAbsoluteTTL),
		PasskeyStartThrottleLimit:   defaultInt(raw.PasskeyStartLimit, defaults.PasskeyStartThrottleLimit),
		PasskeyStartThrottleWindow:  parseDuration(raw.PasskeyStartWindow, defaults.PasskeyStartThrottleWindow),
		RecoveryEmailThrottleLimit:  defaultInt(raw.RecoveryEmailLimit, defaults.RecoveryEmailThrottleLimit),
		RecoveryEmailThrottleWindow: parseDuration(raw.RecoveryEmailWindow, defaults.RecoveryEmailThrottleWindow),
		RecoveryIPThrottleLimit:     defaultInt(raw.RecoveryIPLimit, defaults.RecoveryIPThrottleLimit),
		RecoveryIPThrottleWindow:    parseDuration(raw.RecoveryIPWindow, defaults.RecoveryIPThrottleWindow),
		FailureLockThreshold:        defaultInt(raw.FailureThreshold, defaults.FailureLockThreshold),
		FailureLockWindow:           parseDuration(raw.FailureWindow, defaults.FailureLockWindow),
		FailureLockDuration:         parseDuration(raw.FailureDuration, defaults.FailureLockDuration),
		WebAuthnRPID:                defaultString(raw.WebAuthnRPID, defaults.WebAuthnRPID),
		AccountRecoveryURLBase:      defaultString(raw.AccountRecoveryURLBase, defaults.AccountRecoveryURLBase),
		AuthBodyLimitBytes:          defaultInt(raw.AuthBodyLimitBytes, defaults.AuthBodyLimitBytes),
		SecretHashKey:               defaultString(raw.SecretHashKey, defaults.SecretHashKey),
	}
}

func parseDuration(value string, fallback time.Duration) time.Duration {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	d, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return d
}
