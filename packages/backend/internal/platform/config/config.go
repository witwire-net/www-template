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
		Origin            string   `toml:"origin"`
		ProductOrigin     string   `toml:"product_origin"`
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
		RefreshTokenTTL        string `toml:"refresh_token_ttl"`
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
		JWTSecret              string `toml:"jwt_secret"`
		RPID                   string `toml:"rp_id"`
		RPName                 string `toml:"rp_name"`
	} `toml:"auth"`
	Database struct {
		URL       string `toml:"url"`
		AdminRole string `toml:"admin_role"`
	} `toml:"database"`
	Valkey struct {
		URL        string `toml:"url"`
		AdminURL   string `toml:"admin_url"`
		ProductURL string `toml:"product_url"`
		KeyPrefix  string `toml:"key_prefix"`
	} `toml:"valkey"`
	Cookie struct {
		Name     string `toml:"name"`
		Domain   string `toml:"domain"`
		Path     string `toml:"path"`
		Secure   bool   `toml:"secure"`
		SameSite string `toml:"same_site"`
	} `toml:"cookie"`
	OpenSearch struct {
		URL                   string `toml:"url"`
		AdminAuditIndexPrefix string `toml:"admin_audit_index_prefix"`
		ProductIndexPrefix    string `toml:"product_index_prefix"`
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
		ProductName string `toml:"product_name"`
	} `toml:"mail"`
	Bootstrap struct {
		Enabled    bool   `toml:"enabled"`
		SecretHash string `toml:"secret_hash"`
		ExpiresAt  string `toml:"expires_at"`
	} `toml:"bootstrap"`
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

func resolveAdminConfigPath() string {
	// Step 1: Admin binary 専用の環境変数を最優先し、Product CONFIG_PATH と明確に分離して運用できるようにする。
	if envPath := os.Getenv("ADMIN_CONFIG_PATH"); envPath != "" {
		return envPath
	}

	// Step 2: 既存の process manager が CONFIG_PATH だけを渡す場合にも Admin binary を明示 config で起動できるようにする。
	if envPath := os.Getenv("CONFIG_PATH"); envPath != "" {
		return envPath
	}

	// Step 3: repository root からの開発起動では Admin 専用 local config を探索する。
	if _, err := os.Stat(".config/local.admin.toml"); err == nil {
		abs, _ := filepath.Abs(".config/local.admin.toml")
		return abs
	}

	// Step 4: backend package 配下からの開発起動でも同じ Admin config を見つけられるようにする。
	if _, err := os.Stat("../../.config/local.admin.toml"); err == nil {
		abs, _ := filepath.Abs("../../.config/local.admin.toml")
		return abs
	}

	return ""
}

func LoadConfig() Config {
	configPath := resolveConfigPath()
	raw := loadRawConfigFromPath(configPath, "config file not found. Set CONFIG_PATH or place .config/local.toml at the project root")
	return buildConfig(raw)
}

// LoadAdminConfig は Admin API binary 専用の TOML 設定を読み込む。
//
// 読み込み順:
//   - ADMIN_CONFIG_PATH
//   - CONFIG_PATH
//   - .config/local.admin.toml
//
// 戻り値:
//   - Config: 共通 runtime 設定に AdminRuntimeConfig を含めた値。
//
// エラーケース:
//   - 設定ファイルが見つからない、読み込めない、TOML として parse できない場合は既存 LoadConfig と同じく panic する。
//
// 利用例:
//
//	cfg := config.LoadAdminConfig()
func LoadAdminConfig() Config {
	configPath := resolveAdminConfigPath()
	raw := loadRawConfigFromPath(configPath, "admin config file not found. Set ADMIN_CONFIG_PATH or place .config/local.admin.toml at the project root")
	cfg := buildConfig(raw)
	applyAdminInfraAliases(&cfg, raw)
	return cfg
}

func loadRawConfigFromPath(configPath string, missingMessage string) tomlConfig {
	// Step 1: caller が解決した path が空なら、surface ごとの案内文で fail-close する。
	if configPath == "" {
		panic(missingMessage)
	}

	// Step 2: path traversal の意図せぬ解釈を避けるため filepath.Clean 後の path から TOML を読み込む。
	data, err := os.ReadFile(filepath.Clean(configPath))
	if err != nil {
		panic(fmt.Sprintf("read config file %s: %v", configPath, err))
	}

	// Step 3: TOML を構造体へ decode し、未知 field は Go 側で無視して後方の section 追加に耐える。
	var raw tomlConfig
	if err := toml.Unmarshal(data, &raw); err != nil {
		panic(fmt.Sprintf("parse config file %s: %v", configPath, err))
	}

	// Step 4: decode 済みの raw config を返し、Product / Admin それぞれの loader が surface ごとの alias 適用を決める。
	return raw
}

func buildConfig(raw tomlConfig) Config {
	// Step 1: app environment と allowed origins に Product runtime 互換の default を適用する。
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

	authCfg, err := buildAuthConfig(raw.Auth)
	if err != nil {
		panic(fmt.Sprintf("invalid auth config: %v", err))
	}

	// Step 3: 共通 Config と AdminRuntimeConfig を同じ値として返し、Product runtime は Admin field を無視できる状態に保つ。
	return Config{
		AllowedOrigins:     allowedOrigins,
		Admin:              buildAdminRuntimeConfig(raw),
		AppBearerToken:     appBearerToken,
		Auth:               authCfg,
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
				ProductName: defaultString(raw.Mail.ProductName, "www-template"),
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
				URL:                   strings.TrimSpace(raw.OpenSearch.URL),
				AdminAuditIndexPrefix: strings.TrimSpace(raw.OpenSearch.AdminAuditIndexPrefix),
				ProductIndexPrefix:    strings.TrimSpace(raw.OpenSearch.ProductIndexPrefix),
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

func applyAdminInfraAliases(cfg *Config, raw tomlConfig) {
	// Step 1: Admin TOML の database.url を Admin backend の最小権限 DB 接続先として使う。
	cfg.Infra.Database.URL = strings.TrimSpace(raw.Database.URL)

	// Step 2: Admin runtime の共通 Valkey slot には Admin URL を写し、Product runtime の valkey.url default には影響させない。
	valkeyURL := defaultString(raw.Valkey.AdminURL, raw.Valkey.URL)
	cfg.Infra.Valkey.URL = strings.TrimSpace(valkeyURL)
}

func buildAdminRuntimeConfig(raw tomlConfig) AdminRuntimeConfig {
	// Step 1: Admin Valkey URL は Admin 専用 field を優先し、未設定の場合だけ共通 valkey.url を許可する。
	adminValkeyURL := defaultString(raw.Valkey.AdminURL, raw.Valkey.URL)

	// Step 2: Product Valkey URL は Admin 起動時の logical DB 衝突検証だけに使い、Admin runtime の接続先へは渡さない。
	productValkeyURL := strings.TrimSpace(raw.Valkey.ProductURL)

	// Step 3: bootstrap expires_at は RFC3339 のみ受け付け、parse 不能値は zero time として validation で fail-close させる。
	var bootstrapExpiresAt time.Time
	if trimmed := strings.TrimSpace(raw.Bootstrap.ExpiresAt); trimmed != "" {
		parsed, err := time.Parse(time.RFC3339, trimmed)
		if err == nil {
			bootstrapExpiresAt = parsed.UTC()
		}
	}

	// Step 4: Admin Cookie は [cookie] section からそのまま読み、必須性や production secure policy は ValidateAdminRuntime に集約する。
	return AdminRuntimeConfig{
		Domain:        strings.TrimSpace(raw.Server.Origin),
		ProductDomain: strings.TrimSpace(raw.Server.ProductOrigin),
		Cookie: AdminCookieConfig{
			Name:     strings.TrimSpace(raw.Cookie.Name),
			Domain:   strings.TrimSpace(raw.Cookie.Domain),
			Path:     strings.TrimSpace(raw.Cookie.Path),
			Secure:   raw.Cookie.Secure,
			SameSite: strings.TrimSpace(raw.Cookie.SameSite),
		},
		Database: AdminDatabaseConfig{
			Role: strings.TrimSpace(raw.Database.AdminRole),
		},
		Bootstrap: AdminBootstrapConfig{
			Enabled:    raw.Bootstrap.Enabled,
			SecretHash: strings.TrimSpace(raw.Bootstrap.SecretHash),
			ExpiresAt:  bootstrapExpiresAt,
		},
		Valkey: ValkeyConfig{
			URL:       strings.TrimSpace(adminValkeyURL),
			KeyPrefix: defaultString(raw.Valkey.KeyPrefix, defaultValkeyPrefix),
		},
		ProductValkey: ValkeyConfig{
			URL:       productValkeyURL,
			KeyPrefix: defaultString(raw.Valkey.KeyPrefix, defaultValkeyPrefix),
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
	RefreshTokenTTL        string `toml:"refresh_token_ttl"`
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
	JWTSecret              string `toml:"jwt_secret"`
	RPID                   string `toml:"rp_id"`
	RPName                 string `toml:"rp_name"`
}) (AuthConfig, error) {
	defaults := defaultAuthConfig()

	refreshTTL, err := parseRefreshTokenTTL(raw.RefreshTokenTTL)
	if err != nil {
		return AuthConfig{}, err
	}

	// Step 2: Product TOML の webauthn_rp_id と Admin TOML の rp_id alias を同じ AuthConfig へ正規化する。
	webAuthnRPID := defaultString(raw.WebAuthnRPID, raw.RPID)

	return AuthConfig{
		ChallengeTTL:                parseDuration(raw.ChallengeTTL, defaults.ChallengeTTL),
		RecoveryTokenTTL:            parseDuration(raw.RecoveryTokenTTL, defaults.RecoveryTokenTTL),
		RecoverySessionTTL:          parseDuration(raw.RecoverySessionTTL, defaults.RecoverySessionTTL),
		ReauthSessionTTL:            parseDuration(raw.ReauthSessionTTL, defaults.ReauthSessionTTL),
		SessionIdleTTL:              parseDuration(raw.SessionIdleTTL, defaults.SessionIdleTTL),
		SessionAbsoluteTTL:          parseDuration(raw.SessionAbsoluteTTL, defaults.SessionAbsoluteTTL),
		RefreshTokenTTL:             refreshTTL,
		PasskeyStartThrottleLimit:   defaultInt(raw.PasskeyStartLimit, defaults.PasskeyStartThrottleLimit),
		PasskeyStartThrottleWindow:  parseDuration(raw.PasskeyStartWindow, defaults.PasskeyStartThrottleWindow),
		RecoveryEmailThrottleLimit:  defaultInt(raw.RecoveryEmailLimit, defaults.RecoveryEmailThrottleLimit),
		RecoveryEmailThrottleWindow: parseDuration(raw.RecoveryEmailWindow, defaults.RecoveryEmailThrottleWindow),
		RecoveryIPThrottleLimit:     defaultInt(raw.RecoveryIPLimit, defaults.RecoveryIPThrottleLimit),
		RecoveryIPThrottleWindow:    parseDuration(raw.RecoveryIPWindow, defaults.RecoveryIPThrottleWindow),
		FailureLockThreshold:        defaultInt(raw.FailureThreshold, defaults.FailureLockThreshold),
		FailureLockWindow:           parseDuration(raw.FailureWindow, defaults.FailureLockWindow),
		FailureLockDuration:         parseDuration(raw.FailureDuration, defaults.FailureLockDuration),
		WebAuthnRPID:                defaultString(webAuthnRPID, defaults.WebAuthnRPID),
		AccountRecoveryURLBase:      defaultString(raw.AccountRecoveryURLBase, defaults.AccountRecoveryURLBase),
		AuthBodyLimitBytes:          defaultInt(raw.AuthBodyLimitBytes, defaults.AuthBodyLimitBytes),
		SecretHashKey:               defaultString(raw.SecretHashKey, defaults.SecretHashKey),
		JWTSecret:                   defaultString(raw.JWTSecret, defaults.JWTSecret),
	}, nil
}

// parseRefreshTokenTTL は refresh_token_ttl の strict パースを行う。
// 空文字列の場合は 0（無期限）を返す。
// 有効な duration 文字列の場合はその値を返す。
// 無効な duration 文字列の場合はエラーを返し、fallback しない。
func parseRefreshTokenTTL(value string) (time.Duration, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, nil
	}
	d, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("invalid refresh_token_ttl %q: %w", value, err)
	}
	return d, nil
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
