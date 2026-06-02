package app

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	stdhttp "net/http"
	"time"

	adminhttp "www-template/packages/backend/internal/adapter/http/admin"
	"www-template/packages/backend/internal/adapter/mailer"
	"www-template/packages/backend/internal/adapter/postgres"
	accountspostgres "www-template/packages/backend/internal/adapter/postgres/accounts"
	adminpostgres "www-template/packages/backend/internal/adapter/postgres/admin"
	auditpostgres "www-template/packages/backend/internal/adapter/postgres/audit"
	adminvalkey "www-template/packages/backend/internal/adapter/valkey/admin"
	webauthnadapter "www-template/packages/backend/internal/adapter/webauthn"
	accountsapplication "www-template/packages/backend/internal/application/accounts"
	auditapplication "www-template/packages/backend/internal/application/audit"
	adminauth "www-template/packages/backend/internal/application/auth"
	operatorsapplication "www-template/packages/backend/internal/application/operators"
	"www-template/packages/backend/internal/platform/config"
	"www-template/packages/backend/internal/platform/health"
	"www-template/packages/backend/internal/platform/id"
	"www-template/packages/backend/internal/platform/observability"
	"www-template/packages/backend/internal/platform/secret"
)

const defaultAdminOperatorSetupTokenTTL = 24 * time.Hour

// AdminRuntime は Admin API 専用 binary の runtime 構成を保持する。
// Product Runtime と別型にすることで、Product application container や Product HTTP router を誤って共有しない。
// config は起動時に検証済みの設定値、server は Admin binary が所有する HTTP server、closeObs は tracer / meter の解放関数である。
type AdminRuntime struct {
	config    config.Config
	container *AdminContainer
	server    *stdhttp.Server
	closeObs  func(context.Context) error
}

type bcryptSecretHashVerifier struct{}

func (bcryptSecretHashVerifier) HashSecret(secretValue string) (string, error) {
	// Step 1: Admin setup/bootstrap secret の保存形式は platform/secret の bcrypt helper に委譲する。
	return secret.HashBcryptSecret(secretValue)
}

func (bcryptSecretHashVerifier) MatchesSecret(hash string, secretValue string) bool {
	// Step 1: Admin setup/bootstrap secret の照合は platform/secret の bcrypt helper に委譲し、高速 digest fallback を作らない。
	return secret.MatchesBcryptSecret(hash, secretValue)
}

// AdminContainer は Admin API binary 専用の application service と close 関数を保持する。
//
// 役割:
//   - Product container と共有せず、Admin schema / Admin audit projection を使う service だけを runtime へ渡す。
//   - OperatorAuth は middleware 用 session validator と refresh/current/logout の source である。
//   - OperatorPasskeyLogin は WebAuthn login outer flow を担当し、session lifecycle と challenge 発行の責務を分離する。
//   - OperatorPasskeyVerifier は Admin WebAuthn assertion を検証し、raw credential handle を session 発行へ直通させない。
//   - AccountCreation は Admin audit OpenSearch projection を含む mutation use case である。
//   - AccountSearch は Product Account read model を Admin 権限で読む read use case である。
type AdminContainer struct {
	OperatorAuth            *adminauth.OperatorSessionService
	OperatorPasskeyLogin    *adminauth.OperatorPasskeyLoginService
	OperatorPasskeyVerifier adminauth.OperatorPasskeyVerifier
	OperatorPasskeys        *adminauth.OperatorCredentialService
	OperatorSetup           *operatorsapplication.OperatorService
	AccountCreation         *accountsapplication.AccountCreationService
	AccountSearch           *accountsapplication.AccountSearchService
	close                   func(context.Context) error
}

type adminAccountIDGenerator struct{}

type adminOpaqueTokenGenerator struct{}

type gormDatabaseHandle interface {
	DB() (*sql.DB, error)
}

// NewAdminRuntime は repository 設定ファイルから Admin API runtime を構築する。
// ctx は startup 時の infrastructure 検証と observability 初期化に使う。
// 戻り値は Admin API 専用 runtime であり、Product runtime の container / router は含まない。
// 設定ファイルが見つからない場合、または必須 infrastructure 検証に失敗した場合は error を返す前に panic する可能性がある LoadAdminConfig の挙動を引き継ぐ。
func NewAdminRuntime(ctx context.Context) (*AdminRuntime, error) {
	// Step 1: Product binary 用 loader ではなく Admin 専用 loader を使い、Admin domain / cookie / Valkey URL を必須入力として読み込む。
	return NewAdminRuntimeWithConfig(ctx, config.LoadAdminConfig())
}

// NewAdminRuntimeWithConfig は検証済み設定から Admin API runtime を構築する。
// ctx は database / OpenSearch の到達性確認と observability 初期化に使う。
// cfg は Admin binary 用に渡された設定であり、Product runtime へは渡さない。
// 戻り値は Admin 専用 HTTP server を持つ runtime で、Product handlers を登録しない。
// 設定検証、認証 TTL 検証、infrastructure 検証、observability 初期化のいずれかに失敗した場合は error を返す。
func NewAdminRuntimeWithConfig(ctx context.Context, cfg config.Config) (*AdminRuntime, error) {
	// Step 1: Admin surface 固有の domain / cookie / DB role / Valkey URL を検証し、Product 設定だけで Admin binary が起動することを防ぐ。
	if err := cfg.ValidateAdminRuntime(); err != nil {
		return nil, err
	}

	// Step 2: Product と同じ認証 TTL policy を Admin binary でも起動時に守り、短すぎる refresh token 設定を拒否する。
	if err := validateAuthConfig(cfg.Auth); err != nil {
		return nil, err
	}

	// Step 3: Admin binary が依存する backend infrastructure の到達性を起動前に検証し、Product 専用 object storage 等へ依存しない。
	if err := verifyAdminInfrastructure(ctx, cfg); err != nil {
		return nil, err
	}

	// Step 4: Admin API process 用の tracer / meter を初期化し、startup 以後の観測情報を収集できる状態にする。
	obs := cfg.Observability
	closeTracer, err := observability.InitTracer(ctx, obs.OTELExporterOTLPEndpoint, obs.OTELServiceName)
	if err != nil {
		return nil, err
	}

	closeMeter, err := observability.InitMeter(ctx, obs.OTELExporterOTLPEndpoint, obs.OTELServiceName)
	if err != nil {
		_ = closeTracer(ctx)
		return nil, err
	}

	// Step 5: tracer と meter の close を一つの関数へまとめ、cmd/admin-api から安全に解放できるようにする。
	closeObs := func(ctx context.Context) error {
		_ = closeTracer(ctx)
		_ = closeMeter(ctx)
		return nil
	}

	// Step 6: Admin account 管理 use case と audit projection を runtime で構成し、HTTP adapter へ具象 repository を直接持ち込まない。
	container, err := BuildAdminContainer(ctx, cfg)
	if err != nil {
		_ = closeObs(ctx)
		return nil, err
	}

	// Step 7: Admin 専用 HTTP adapter を設定し、Product router / Product generated bindings を Admin binary に持ち込まない。
	server := &stdhttp.Server{
		Addr: ":" + cfg.Port,
		Handler: adminhttp.NewRouterWithDependencies(cfg, adminhttp.Dependencies{
			OperatorSessions:          adminhttp.NewOperatorSessionValidator(container.OperatorAuth),
			OperatorAuth:              container.OperatorAuth,
			OperatorPasskeyAuth:       container.OperatorPasskeyLogin,
			OperatorPasskeyVerifier:   container.OperatorPasskeyVerifier,
			OperatorSetup:             container.OperatorSetup,
			OperatorPasskeyManagement: container.OperatorPasskeys,
			AccountCreation:           container.AccountCreation,
			AccountSearch:             container.AccountSearch,
		}),
		ReadHeaderTimeout: defaultReadHeaderTimeout,
		ReadTimeout:       cfg.ServerReadTimeout,
		WriteTimeout:      cfg.ServerWriteTimeout,
		IdleTimeout:       cfg.ServerIdleTimeout,
	}

	return &AdminRuntime{
		config:    cfg,
		container: container,
		server:    server,
		closeObs:  closeObs,
	}, nil
}

// BuildAdminContainer は Admin API runtime 用の account 管理 service 群を構築する。
//
// ctx は DB ping と将来の store 初期化に使う cancellation context である。
// cfg は Admin runtime validation 済みの設定であり、Product runtime container へは渡さない。
// 戻り値は Admin-only service container で、OpenSearch projection failure observer を含む。
func BuildAdminContainer(ctx context.Context, cfg config.Config) (*AdminContainer, error) {
	// Step 1: Admin runtime 用 DB handle を開き、Product runtime repository と lifecycle を共有しない。
	db, err := postgres.OpenDatabase(cfg.Infra.Database.URL)
	if err != nil {
		return nil, err
	}
	if err := postgres.PingDatabase(ctx, db); err != nil {
		_ = closeGormDatabase(db)
		return nil, err
	}
	var valkeyStore *adminvalkey.Store
	cleanupOnError := true
	defer func() {
		// Step 2: container 構築中にどこかで失敗した場合だけ、確保済み DB/Valkey resource をまとめて解放する。
		if cleanupOnError {
			closeAdminContainerResources(db, valkeyStore)
		}
	}()

	// Step 3: Admin schema repository と account repository を同じ DB handle で構成し、account 作成 transaction と audit intent/outcome を同じ database 境界へ置く。
	accountRepo := accountspostgres.NewAccountRepository(db)
	operatorRepo := adminpostgres.NewOperatorRepository(db)
	operatorPasskeyRepo := adminpostgres.NewOperatorPasskeyRepository(db)
	auditRepo := auditpostgres.NewRepository(db)
	auditService, err := auditapplication.NewAuditService(auditRepo, func() time.Time { return time.Now().UTC() })
	if err != nil {
		return nil, err
	}

	// Step 4: Admin operator session state 用 Valkey store を構成し、Product Valkey key namespace と package 境界を共有しない。
	valkeyStore, err = openAdminRuntimeValkeyStore(ctx, cfg)
	if err != nil {
		return nil, err
	}

	// Step 5: Admin WebAuthn provider を Admin logical DB の challenge store と組み合わせ、login/setup registration ceremony を同じ provider で検証する。
	webAuthnProvider, registrationProvider, challengeProvider, err := newAdminWebAuthnProviders(cfg, valkeyStore)
	if err != nil {
		return nil, err
	}

	// Step 6: Admin operator session lifecycle service は current/refresh/logout/session 発行を提供し、Product auth service を共有しない。
	authService, err := newAdminOperatorAuthService(cfg, operatorRepo, adminvalkey.NewOperatorRefreshSessionStore(valkeyStore))
	if err != nil {
		return nil, err
	}
	operatorPasskeyLogin, err := newAdminOperatorPasskeyLoginService(cfg, operatorRepo, challengeProvider, authService)
	if err != nil {
		return nil, err
	}
	operatorPasskeyVerifier, err := adminauth.NewOperatorPasskeyVerifier(webAuthnProvider, operatorPasskeyRepo)
	if err != nil {
		return nil, err
	}
	operatorCredentials, err := adminauth.NewOperatorCredentialService(operatorPasskeyRepo)
	if err != nil {
		return nil, err
	}

	// Step 7: Admin operator setup / creation use case を構成し、setup token 平文は mailer delivery port だけへ渡す。
	secretHashVerifier := bcryptSecretHashVerifier{}
	operatorSetup, err := operatorsapplication.NewOperatorService(operatorRepo, auditService, adminAccountIDGenerator{}, adminOpaqueTokenGenerator{}, registrationProvider, authService, mailer.NewSetupTokenDeliveryPort(mailer.NewSMTPSender(cfg.Infra), cfg), secretHashVerifier, secretHashVerifier, func() time.Time { return time.Now().UTC() }, operatorsapplication.BootstrapConfig{Enabled: cfg.Admin.Bootstrap.Enabled, SecretHash: cfg.Admin.Bootstrap.SecretHash, ExpiresAt: cfg.Admin.Bootstrap.ExpiresAt}, defaultAdminOperatorSetupTokenTTL)
	if err != nil {
		return nil, err
	}

	// Step 8: Admin audit projection は Go backend 側 projector と warning observer を注入し、packages/admin の OpenSearch client を不要にする。
	projector, err := NewAdminAuditOpenSearchProjector(cfg.Infra.OpenSearch)
	if err != nil {
		return nil, err
	}
	accountCreation, err := accountsapplication.NewAccountCreationService(accountRepo, auditService, adminAccountIDGenerator{}, projector, NewAdminAuditProjectionWarningObserver())
	if err != nil {
		return nil, err
	}
	accountSearch, err := accountsapplication.NewAccountSearchService(accountRepo)
	if err != nil {
		return nil, err
	}

	// Step 9: 構成済み use case と DB/Valkey close をまとめて返し、AdminRuntime.Close が一括解放できるようにする。
	cleanupOnError = false
	return &AdminContainer{
		OperatorAuth:            authService,
		OperatorPasskeyLogin:    operatorPasskeyLogin,
		OperatorPasskeyVerifier: operatorPasskeyVerifier,
		OperatorPasskeys:        operatorCredentials,
		OperatorSetup:           operatorSetup,
		AccountCreation:         accountCreation,
		AccountSearch:           accountSearch,
		close: composeClosers(
			func(context.Context) error { return valkeyStore.Close() },
			func(context.Context) error { return closeGormDatabase(db) },
		),
	}, nil
}

func openAdminRuntimeValkeyStore(ctx context.Context, cfg config.Config) (*adminvalkey.Store, error) {
	// Step 1: Admin logical DB / namespace 用 Valkey store を開き、Product session store と key 空間を分離する。
	store, err := adminvalkey.NewStore(cfg.Infra.Valkey)
	if err != nil {
		return nil, err
	}

	// Step 2: 起動時 ping に失敗した store は session 永続化に使えないため、返却前に閉じて fail-close する。
	if err := store.Ping(ctx); err != nil {
		_ = store.Close()
		return nil, err
	}
	return store, nil
}

func newAdminWebAuthnProviders(cfg config.Config, store *adminvalkey.Store) (adminauth.WebAuthnProvider, adminauth.OperatorPasskeyRegistrationProvider, adminauth.OperatorPasskeyChallengeProvider, error) {
	// Step 1: Admin origin/domain 専用の WebAuthn provider を生成し、Product RP origin を Admin setup/login ceremony に混在させない。
	webAuthnProvider, err := webauthnadapter.NewWebAuthnProvider(cfg.AuthRuntime().WebAuthnRPID, []string{cfg.Admin.Domain}, cfg.AuthRuntime().ChallengeTTL, store)
	if err != nil {
		return nil, nil, nil, err
	}

	// Step 2: setup registration に必要な provider interface を満たさない場合は、credential 登録を開始せず起動を止める。
	registrationProvider, ok := webAuthnProvider.(adminauth.OperatorPasskeyRegistrationProvider)
	if !ok {
		return nil, nil, nil, errors.New("admin webauthn provider does not implement operator registration")
	}

	// Step 3: login challenge に必要な provider interface を満たさない場合は、認証開始 route を fail-close にするため起動を止める。
	challengeProvider, ok := webAuthnProvider.(adminauth.OperatorPasskeyChallengeProvider)
	if !ok {
		return nil, nil, nil, errors.New("admin webauthn provider does not implement operator login challenge")
	}
	return webAuthnProvider, registrationProvider, challengeProvider, nil
}

func closeAdminContainerResources(db gormDatabaseHandle, store *adminvalkey.Store) {
	// Step 1: Valkey store が確保済みなら先に閉じ、session/challenge store の connection を残さない。
	if store != nil {
		_ = store.Close()
	}

	// Step 2: DB handle が確保済みなら connection pool を閉じ、構築失敗時に backend process へ pool を残さない。
	if db != nil {
		_ = closeGormDatabase(db)
	}
}

func newAdminOperatorAuthService(cfg config.Config, operators adminauth.OperatorRepository, sessions adminauth.OperatorRefreshSessionStore) (*adminauth.OperatorSessionService, error) {
	// Step 1: runtime default を反映した auth config を使い、Admin session TTL と署名 secret の未設定を development default で補完する。
	authRuntime := cfg.AuthRuntime()
	signer, err := adminauth.NewTokenJSONSignVerifier([]byte(authRuntime.JWTSecret))
	if err != nil {
		return nil, err
	}

	// Step 2: Admin session lifecycle service には challenge provider を渡さず、WebAuthn outer flow と session 発行を分離する。
	return adminauth.NewOperatorSessionService(
		adminauth.OperatorSessionDependencies{
			Operators: operators,
			Sessions:  sessions,
			Signer:    signer,
			Secrets:   adminOpaqueTokenGenerator{},
			IDs:       newAuthIDPolicy(),
			Clock:     func() time.Time { return time.Now().UTC() },
		},
		operatorAuthConfigFromRuntime(authRuntime),
	)
}

func newAdminOperatorPasskeyLoginService(cfg config.Config, operators adminauth.OperatorRepository, challenges adminauth.OperatorPasskeyChallengeProvider, sessions adminauth.OperatorSessionIssuer) (*adminauth.OperatorPasskeyLoginService, error) {
	// Step 1: session lifecycle と同じ auth runtime config を使い、Admin RP ID と TTL policy を login facade に共有する。
	authRuntime := cfg.AuthRuntime()

	// Step 2: WebAuthn challenge provider と session issuer を passkey login service に渡し、OperatorSessionService から outer flow 依存を排除する。
	return adminauth.NewOperatorPasskeyLoginService(
		adminauth.OperatorPasskeyLoginDependencies{Operators: operators, Challenges: challenges, Sessions: sessions},
		operatorAuthConfigFromRuntime(authRuntime),
	)
}

func operatorAuthConfigFromRuntime(authRuntime config.AuthConfig) adminauth.OperatorSessionConfig {
	// Step 1: Admin operator refresh session は共通 auth runtime の refresh TTL を operator session TTL として解釈し、未設定時は絶対 session TTL に丸めて無期限 operator session を避ける。
	refreshSessionTTL := authRuntime.RefreshTokenTTL
	if refreshSessionTTL == 0 {
		refreshSessionTTL = authRuntime.SessionAbsoluteTTL
	}

	// Step 2: Admin operator refresh Cookie lifetime を server-side operator session TTL と同じ長さにし、server 期限を超えて refresh Cookie が残らないようにする。
	return adminauth.OperatorSessionConfig{
		OperatorAccessTokenTTL:        authRuntime.SessionIdleTTL,
		OperatorRefreshSessionTTL:     refreshSessionTTL,
		OperatorRefreshCookieLifetime: refreshSessionTTL,
		WebAuthnRPID:                  authRuntime.WebAuthnRPID,
	}
}

func verifyAdminInfrastructure(ctx context.Context, cfg config.Config) error {
	// Step 1: Admin account management と operator persistence が使う DB URL が空なら、DB 接続前に fail-close する。
	if cfg.Infra.Database.URL == "" {
		return errors.New("database url is required")
	}

	// Step 2: Admin runtime が使用する DB へ接続し、起動前に到達性を確認する。
	db, err := postgres.OpenDatabase(cfg.Infra.Database.URL)
	if err != nil {
		return err
	}
	if err := postgres.PingDatabase(ctx, db); err != nil {
		return err
	}
	sqlDB, err := db.DB()
	if err == nil {
		defer func() {
			_ = sqlDB.Close()
		}()
	}

	// Step 3: Admin audit projection の接続先が未設定または到達不能な場合、観測不能な Admin mutation を避けるため起動を止める。
	return health.CheckOpenSearch(ctx, cfg.Infra.OpenSearch)
}

// Close は Admin runtime が確保した observability resource を解放する。
// ctx は shutdown の deadline / cancel を伝えるために使う。
// 戻り値は現在 nil 固定であり、各 close 関数の error は process 終了を妨げないよう吸収する。
func (r *AdminRuntime) Close(ctx context.Context) error {
	// Step 1: observability 初期化前に呼ばれても安全なよう nil を確認し、二次障害を避ける。
	if r.closeObs != nil {
		_ = r.closeObs(ctx)
	}
	if r.container != nil && r.container.close != nil {
		return r.container.close(ctx)
	}
	return nil
}

func (adminAccountIDGenerator) Next() (string, error) {
	// Step 1: AccountID は Admin account creation use case の外側で ULID として発行し、domain.NewAccountID で形式検証する。
	return id.NewULID(time.Now().UTC(), rand.Reader)
}

func (adminOpaqueTokenGenerator) NewToken() (string, error) {
	// Step 1: refreshToken 用に 64 byte の暗号学的乱数を生成し、弱い fallback は行わない。
	raw := make([]byte, 64)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}

	// Step 2: Cookie で扱いやすい padding なし Base64URL 文字列へ変換する。
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func closeGormDatabase(db gormDatabaseHandle) error {
	// Step 1: GORM handle から database/sql handle を取り出し、既に取得不能な場合だけ close 対象なしとして扱う。
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}

	// Step 2: Admin runtime 専用 DB connection pool を閉じ、Product runtime container の pool へ副作用を与えない。
	return sqlDB.Close()
}

// Config は Admin runtime の起動に使われた設定値を返す。
// 戻り値は値コピーであり、呼び出し側の変更が runtime 内部状態へ副作用を与えない。
func (r *AdminRuntime) Config() config.Config {
	return r.config
}

// Server は Admin API 専用 HTTP server を返す。
// 戻り値の server は cmd/admin-api が ListenAndServe と Shutdown を管理するために使う。
func (r *AdminRuntime) Server() *stdhttp.Server {
	return r.server
}
