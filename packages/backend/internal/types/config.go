package types

import (
	"errors"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultPort           = "8080"
	defaultAllowedOrigins = "http://localhost:5173,http://127.0.0.1:5173,http://localhost:5174,http://127.0.0.1:5174"
	defaultAppAuthValue   = "dev-app-auth"
	defaultProfileStore   = "memory"
	defaultWebAuthnRPID   = "localhost"
	defaultValkeyPrefix   = "www-template"
	defaultRecoveryBase   = "http://localhost:5173/app/login/recovery/consume"
	defaultSMTPPort       = 587
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

type InfraConfig struct {
	Valkey ValkeyConfig
	SMTP   SMTPConfig
	Mail   MailConfig
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
	DatabaseURL    string
	Environment    string
	Infra          InfraConfig
	Port           string
	ProfileStore   string
}

func LoadConfig() Config {
	environment := getEnv("APP_ENV", "development")
	allowedOriginsValue := getEnv("ALLOWED_ORIGINS", defaultAllowedOrigins)
	allowedOrigins := make([]string, 0)
	for _, rawOrigin := range strings.Split(allowedOriginsValue, ",") {
		origin := strings.TrimSpace(rawOrigin)
		if origin != "" {
			allowedOrigins = append(allowedOrigins, origin)
		}
	}

	appBearerToken := strings.TrimSpace(os.Getenv("APP_BEARER_TOKEN"))
	if environment == "development" && appBearerToken == "" {
		appBearerToken = defaultAppAuthValue
	}

	return Config{
		AllowedOrigins: allowedOrigins,
		AppBearerToken: appBearerToken,
		Auth:           loadAuthConfig(),
		DatabaseURL:    strings.TrimSpace(os.Getenv("DATABASE_URL")),
		Environment:    environment,
		Infra:          loadInfraConfig(),
		Port:           getEnv("PORT", defaultPort),
		ProfileStore:   getEnv("APP_PROFILE_STORE", defaultProfileStore),
	}
}

func (c Config) AppAuthorizationValue() string {
	return "Bearer " + c.AppBearerToken
}

func (c Config) Validate() error {
	if c.Environment != "development" && strings.TrimSpace(c.AppBearerToken) == "" {
		return errors.New("APP_BEARER_TOKEN is required when APP_ENV is not development")
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

func loadInfraConfig() InfraConfig {
	return InfraConfig{
		Valkey: ValkeyConfig{
			URL:       strings.TrimSpace(os.Getenv("VALKEY_URL")),
			KeyPrefix: getEnv("VALKEY_KEY_PREFIX", defaultValkeyPrefix),
		},
		SMTP: SMTPConfig{
			Host:     strings.TrimSpace(os.Getenv("SMTP_HOST")),
			Port:     getEnvInt("SMTP_PORT", defaultSMTPPort),
			Username: strings.TrimSpace(os.Getenv("SMTP_USERNAME")),
			Password: strings.TrimSpace(os.Getenv("SMTP_PASSWORD")),
		},
		Mail: MailConfig{
			FromAddress: strings.TrimSpace(os.Getenv("MAIL_FROM_ADDRESS")),
		},
	}
}

func getEnv(key string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	return value
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

func loadAuthConfig() AuthConfig {
	defaults := defaultAuthConfig()

	return AuthConfig{
		ChallengeTTL:                defaults.ChallengeTTL,
		RecoveryTokenTTL:            defaults.RecoveryTokenTTL,
		RecoverySessionTTL:          defaults.RecoverySessionTTL,
		SessionIdleTTL:              defaults.SessionIdleTTL,
		SessionAbsoluteTTL:          defaults.SessionAbsoluteTTL,
		PasskeyStartThrottleLimit:   defaults.PasskeyStartThrottleLimit,
		PasskeyStartThrottleWindow:  defaults.PasskeyStartThrottleWindow,
		RecoveryEmailThrottleLimit:  defaults.RecoveryEmailThrottleLimit,
		RecoveryEmailThrottleWindow: defaults.RecoveryEmailThrottleWindow,
		RecoveryIPThrottleLimit:     defaults.RecoveryIPThrottleLimit,
		RecoveryIPThrottleWindow:    defaults.RecoveryIPThrottleWindow,
		FailureLockThreshold:        defaults.FailureLockThreshold,
		FailureLockWindow:           defaults.FailureLockWindow,
		FailureLockDuration:         defaults.FailureLockDuration,
		WebAuthnRPID:                getEnv("WEBAUTHN_RP_ID", defaultWebAuthnRPID),
		AccountRecoveryURLBase:      getEnv("ACCOUNT_RECOVERY_URL_BASE", defaultRecoveryBase),
	}
}

func getEnvInt(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}

	return parsed
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
