package config

import (
	"errors"
	"net"
	"net/netip"
	"net/url"
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
	Host            string
	Port            int
	Username        string
	Password        string
	SecureTransport bool
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
	ChallengeTTL                    time.Duration
	RecoveryTokenTTL                time.Duration
	RecoverySessionTTL              time.Duration
	ReauthSessionTTL                time.Duration
	SessionIdleTTL                  time.Duration
	SessionAbsoluteTTL              time.Duration
	RefreshTokenTTL                 time.Duration
	PasskeyStartThrottleLimit       int
	PasskeyStartGlobalThrottleLimit int
	PasskeyStartThrottleWindow      time.Duration
	SecretHashKey                   string
	RecoveryEmailThrottleLimit      int
	RecoveryEmailThrottleWindow     time.Duration
	RecoveryIPThrottleLimit         int
	RecoveryIPThrottleWindow        time.Duration
	FailureLockThreshold            int
	FailureLockWindow               time.Duration
	FailureLockDuration             time.Duration
	WebAuthnRPID                    string
	AccountRecoveryURLBase          string
	AuthBodyLimitBytes              int
	JWTSecret                       string
}

type Config struct {
	AllowedOrigins     []string
	AppBearerToken     string
	Auth               AuthConfig
	Environment        string
	Infra              InfraConfig
	Port               string
	Observability      ObservabilityConfig
	TrustedProxyCIDRs  []string
	ServerReadTimeout  time.Duration
	ServerWriteTimeout time.Duration
	ServerIdleTimeout  time.Duration
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
	if err := c.validateRequiredFields(); err != nil {
		return err
	}

	// production 環境では追加のセキュリティ検証を fail-close で実施する
	if c.Environment != "development" {
		if err := c.validateProductionSecurity(); err != nil {
			return err
		}
	}

	return nil
}

// validateRequiredFields はインフラ必須フィールドの欠如を検証する。
func (c Config) validateRequiredFields() error {
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

// validateProductionSecurity は production 環境でのみ呼び出され、
// 認証基盤に関する危険な設定値を検出すると即座にエラーを返して startup を阻止する。
// これにより、誤設定による fail-open を防ぐ。
func (c Config) validateProductionSecurity() error {
	var errs []string

	errs = c.validateProductionAllowedOrigins(errs)
	errs = c.validateProductionWebAuthnRPID(errs)
	errs = c.validateProductionRecoveryURL(errs)
	errs = c.validateProductionTrustedProxy(errs)
	errs = c.validateProductionAuthBodyLimit(errs)
	errs = c.validateProductionMailTransport(errs)
	errs = c.validateProductionSecretHashKey(errs)

	if len(errs) > 0 {
		return errors.New("production security validation failed: " + strings.Join(errs, "; "))
	}
	return nil
}

// validateProductionSecretHashKey は production で SecretHashKey が設定されており、
// 既知の default や短すぎる値でないことを検証する。
func (c Config) validateProductionSecretHashKey(errs []string) []string {
	secret := strings.TrimSpace(c.Auth.SecretHashKey)
	if secret == "" {
		return append(errs, "secret_hash_key is required in production")
	}
	if secret == "dev-pepper-change-in-production" {
		return append(errs, "secret_hash_key must not use the default dev value in production")
	}
	if len(secret) < 32 {
		return append(errs, "secret_hash_key must be at least 32 characters in production")
	}

	jwtSecret := strings.TrimSpace(c.Auth.JWTSecret)
	if jwtSecret == "" {
		return append(errs, "jwt_secret is required in production")
	}
	if jwtSecret == "change-this-to-a-long-random-jwt-secret-in-production" {
		return append(errs, "jwt_secret must not use the default dev value in production")
	}
	if len(jwtSecret) < 32 {
		return append(errs, "jwt_secret must be at least 32 characters in production")
	}
	if jwtSecret == secret {
		return append(errs, "jwt_secret must not be the same as secret_hash_key in production")
	}
	return errs
}

// validateProductionAllowedOrigins は allowed_origins が HTTPS のみであり、
// localhost/loopback/plain HTTP/wildcard を含まないことを検証する。
func (c Config) validateProductionAllowedOrigins(errs []string) []string {
	if len(c.AllowedOrigins) == 0 {
		return append(errs, "allowed_origins is required in production")
	}
	for _, origin := range c.AllowedOrigins {
		if err := validateProductionOrigin(origin); err != nil {
			errs = append(errs, "allowed_origins: "+err.Error())
		}
	}
	return errs
}

// validateProductionWebAuthnRPID は webauthn_rp_id が allowed origin host のいずれかと一致することを検証する。
func (c Config) validateProductionWebAuthnRPID(errs []string) []string {
	if c.Auth.WebAuthnRPID == "" {
		return append(errs, "webauthn_rp_id is required in production")
	}
	matched := false
	for _, origin := range c.AllowedOrigins {
		u, err := url.Parse(origin)
		if err != nil {
			continue
		}
		if u.Hostname() == c.Auth.WebAuthnRPID {
			matched = true
			break
		}
	}
	if !matched {
		return append(errs, "webauthn_rp_id must match allowed origin host in production")
	}
	return errs
}

// validateProductionRecoveryURL は account_recovery_url_base が HTTPS かつ
// localhost/loopback/wildcard/private IP でないことを検証する。
func (c Config) validateProductionRecoveryURL(errs []string) []string {
	if c.Auth.AccountRecoveryURLBase == "" {
		return append(errs, "account_recovery_url_base is required in production")
	}
	if err := validateProductionRecoveryURLValue(c.Auth.AccountRecoveryURLBase); err != nil {
		return append(errs, err.Error())
	}
	return errs
}

// validateProductionTrustedProxy は trusted_proxy_cidrs が設定されており、
// すべて有効な CIDR 形式であることを検証する。
func (c Config) validateProductionTrustedProxy(errs []string) []string {
	if len(c.TrustedProxyCIDRs) == 0 {
		return append(errs, "trusted_proxy_cidrs is required in production")
	}
	for _, cidr := range c.TrustedProxyCIDRs {
		if _, err := netip.ParsePrefix(cidr); err != nil {
			errs = append(errs, "trusted_proxy_cidrs contains invalid CIDR: "+cidr)
		}
	}
	return errs
}

// validateProductionAuthBodyLimit は auth_body_limit_bytes が正の値に設定されていることを検証する。
func (c Config) validateProductionAuthBodyLimit(errs []string) []string {
	if c.Auth.AuthBodyLimitBytes <= 0 {
		return append(errs, "auth_body_limit_bytes must be set in production")
	}
	return errs
}

// validateProductionMailTransport は production で SMTP が secure transport を使用することを検証する。
func (c Config) validateProductionMailTransport(errs []string) []string {
	if !c.Infra.SMTP.SecureTransport {
		return append(errs, "mail_secure_transport is required in production")
	}
	return errs
}

// validateProductionHost は production で許可される host が
// localhost/loopback/private IP/wildcard でないことを検証する共通ヘルパー。
func validateProductionHost(host string) error {
	if host == "" {
		return errors.New("host is empty")
	}
	if host == "localhost" || host == "127.0.0.1" || host == "::1" {
		return errors.New("localhost/loopback host is not allowed in production: " + host)
	}
	if strings.Contains(host, "*") {
		return errors.New("wildcard host is not allowed in production: " + host)
	}
	if ip := net.ParseIP(host); ip != nil {
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
			return errors.New("private/loopback IP host is not allowed in production: " + host)
		}
	}
	return nil
}

// validateProductionOrigin は production で許可される origin が HTTPS かつ
// localhost/loopback/private IP/wildcard でないことを検証する。
func validateProductionOrigin(origin string) error {
	u, err := url.Parse(origin)
	if err != nil {
		return errors.New("invalid origin URL: " + origin)
	}
	if u.Scheme != "https" {
		return errors.New("origin must use HTTPS: " + origin)
	}
	if err := validateProductionHost(u.Hostname()); err != nil {
		return errors.New("origin " + err.Error() + ": " + origin)
	}
	return nil
}

// validateProductionRecoveryURLValue は production で許可される recovery URL が
// HTTPS スキームかつ localhost/loopback/wildcard/private IP でないことを検証する。
func validateProductionRecoveryURLValue(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return errors.New("account_recovery_url_base is invalid URL: " + rawURL)
	}
	if u.Scheme != "https" {
		return errors.New("account_recovery_url_base must use HTTPS: " + rawURL)
	}
	if err := validateProductionHost(u.Hostname()); err != nil {
		return errors.New("account_recovery_url_base " + err.Error() + ": " + rawURL)
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
	configured.PasskeyStartGlobalThrottleLimit = defaultInt(configured.PasskeyStartGlobalThrottleLimit, defaults.PasskeyStartGlobalThrottleLimit)
	configured.PasskeyStartThrottleWindow = defaultDuration(configured.PasskeyStartThrottleWindow, defaults.PasskeyStartThrottleWindow)
	configured.SecretHashKey = defaultString(configured.SecretHashKey, defaults.SecretHashKey)
	configured.RecoveryEmailThrottleLimit = defaultInt(configured.RecoveryEmailThrottleLimit, defaults.RecoveryEmailThrottleLimit)
	configured.RecoveryEmailThrottleWindow = defaultDuration(configured.RecoveryEmailThrottleWindow, defaults.RecoveryEmailThrottleWindow)
	configured.RecoveryIPThrottleLimit = defaultInt(configured.RecoveryIPThrottleLimit, defaults.RecoveryIPThrottleLimit)
	configured.RecoveryIPThrottleWindow = defaultDuration(configured.RecoveryIPThrottleWindow, defaults.RecoveryIPThrottleWindow)
	configured.FailureLockThreshold = defaultInt(configured.FailureLockThreshold, defaults.FailureLockThreshold)
	configured.FailureLockWindow = defaultDuration(configured.FailureLockWindow, defaults.FailureLockWindow)
	configured.FailureLockDuration = defaultDuration(configured.FailureLockDuration, defaults.FailureLockDuration)
	configured.WebAuthnRPID = defaultString(configured.WebAuthnRPID, defaults.WebAuthnRPID)
	configured.AccountRecoveryURLBase = defaultString(configured.AccountRecoveryURLBase, defaults.AccountRecoveryURLBase)
	configured.AuthBodyLimitBytes = defaultInt(configured.AuthBodyLimitBytes, defaults.AuthBodyLimitBytes)
	configured.JWTSecret = defaultString(configured.JWTSecret, defaults.JWTSecret)

	return configured
}

func defaultAuthConfig() AuthConfig {
	return AuthConfig{
		ChallengeTTL:                    5 * time.Minute,
		RecoveryTokenTTL:                30 * time.Minute,
		RecoverySessionTTL:              15 * time.Minute,
		SessionIdleTTL:                  12 * time.Hour,
		SessionAbsoluteTTL:              14 * 24 * time.Hour,
		PasskeyStartThrottleLimit:       5,
		PasskeyStartGlobalThrottleLimit: 1000,
		PasskeyStartThrottleWindow:      5 * time.Minute,
		SecretHashKey:                   "dev-pepper-change-in-production",
		JWTSecret:                       "change-this-to-a-long-random-jwt-secret-in-production",
		RecoveryEmailThrottleLimit:      3,
		RecoveryEmailThrottleWindow:     time.Hour,
		RecoveryIPThrottleLimit:         10,
		RecoveryIPThrottleWindow:        time.Hour,
		FailureLockThreshold:            10,
		FailureLockWindow:               15 * time.Minute,
		FailureLockDuration:             15 * time.Minute,
		WebAuthnRPID:                    defaultWebAuthnRPID,
		AccountRecoveryURLBase:          defaultRecoveryBase,
		AuthBodyLimitBytes:              1 << 20, // 1 MiB の development safe default
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
