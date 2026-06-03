package app

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"time"

	"www-template/packages/backend/internal/adapter/mailer"
	"www-template/packages/backend/internal/adapter/postgres"
	adminvalkey "www-template/packages/backend/internal/adapter/valkey/admin"
	webauthnadapter "www-template/packages/backend/internal/adapter/webauthn"
	accountsapplication "www-template/packages/backend/internal/application/accounts"
	auditapplication "www-template/packages/backend/internal/application/audit"
	adminauth "www-template/packages/backend/internal/application/auth"
	operatorsapplication "www-template/packages/backend/internal/application/operators"
	"www-template/packages/backend/internal/platform/config"
	"www-template/packages/backend/internal/platform/id"
	"www-template/packages/backend/internal/platform/secret"
)

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

// adminAccountIDGenerator は Admin account creation use case で使う AccountID 発行器である。
//
// 役割:
//   - Admin schema の account 作成時に ULID を発行し、domain.NewAccountID で形式検証する。
//   - Product runtime の IDGenerator と共有せず、Admin account namespace を分離する。
//
// 使用例:
//
//	id, err := adminAccountIDGenerator{}.Next()
type adminAccountIDGenerator struct{}

// Next は Admin account 用の ULID を発行する。
//
// 戻り値:
//   - string: 発行された ULID 文字列。
//   - error: 乱数生成に失敗した場合。
func (adminAccountIDGenerator) Next() (string, error) {
	// Step 1: AccountID は Admin account creation use case の外側で ULID として発行し、domain.NewAccountID で形式検証する。
	return id.NewULID(time.Now().UTC(), rand.Reader)
}

// adminOpaqueTokenGenerator は Admin operator session の refresh token 用乱数トークン生成器である。
//
// 役割:
//   - Admin operator の refresh token として 64 byte の暗号学的乱数を Base64URL エンコードする。
//   - Product runtime の TokenGenerator と共有せず、Admin session namespace を分離する。
//
// 使用例:
//
//	token, err := adminOpaqueTokenGenerator{}.NewToken()
type adminOpaqueTokenGenerator struct{}

// NewToken は Admin operator refresh token 用の暗号学的乱数トークンを生成する。
//
// 戻り値:
//   - string: 64 byte の暗号学的乱数を Base64URL エンコードしたトークン。
//   - error: 乱数生成に失敗した場合。
func (adminOpaqueTokenGenerator) NewToken() (string, error) {
	// Step 1: refreshToken 用に 64 byte の暗号学的乱数を生成し、弱い fallback は行わない。
	raw := make([]byte, 64)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}

	// Step 2: Cookie で扱いやすい padding なし Base64URL 文字列へ変換する。
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

// gormDatabaseHandle は GORM の DB handle が提供する sql.DB 取得インターフェースである。
//
// 役割:
//   - closeGormDatabase 関数が GORM の DB() メソッドを呼び出すためのインターフェース。
//   - 具象型への依存を避け、テスト時のスタブ注入を可能にする。
type gormDatabaseHandle interface {
	DB() (*sql.DB, error)
}

// closeGormDatabase は GORM の DB handle から database/sql handle を取り出し、connection pool を閉じる。
//
// 役割:
//   - Admin runtime 専用 DB connection pool を閉じ、Product runtime container の pool へ副作用を与えない。
//   - BuildAdminContainer の error path と AdminContainer.close から使われる。
//
// 引数:
//   - db: GORM の DB handle。DB() メソッドで sql.DB を返すインターフェース。
//
// 戻り値:
//   - error: sql.DB の取得または close に失敗した場合。
func closeGormDatabase(db gormDatabaseHandle) error {
	// Step 1: GORM handle から database/sql handle を取り出し、既に取得不能な場合だけ close 対象なしとして扱う。
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}

	// Step 2: Admin runtime 専用 DB connection pool を閉じ、Product runtime container の pool へ副作用を与えない。
	return sqlDB.Close()
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
	accountRepo := postgres.NewAccountManagementRepository(db)
	operatorRepo := postgres.NewOperatorRepository(db)
	operatorPasskeyRepo := postgres.NewOperatorPasskeyRepository(db)
	auditRepo := postgres.NewOperatorAuditRepository(db)
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
	projector, err := NewOperatorAuditOpenSearchProjector(cfg.Infra.OpenSearch)
	if err != nil {
		return nil, err
	}
	accountCreation, err := accountsapplication.NewAccountCreationService(accountRepo, auditService, adminAccountIDGenerator{}, projector, NewOperatorAuditProjectionWarningObserver())
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
			Operators:       operators,
			RefreshSessions: sessions,
			Signer:          signer,
			TokenGenerator:  adminOpaqueTokenGenerator{},
			IDGenerator:     newAuthIDPolicy(),
			Clock:           func() time.Time { return time.Now().UTC() },
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
