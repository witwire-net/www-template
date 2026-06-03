package app

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"www-template/packages/backend/internal/adapter/mailer"
	"www-template/packages/backend/internal/adapter/postgres"
	productvalkey "www-template/packages/backend/internal/adapter/valkey/product"
	"www-template/packages/backend/internal/adapter/webauthn"
	productaccounts "www-template/packages/backend/internal/application/accounts"
	productauth "www-template/packages/backend/internal/application/auth"
	domain "www-template/packages/backend/internal/domain"
	"www-template/packages/backend/internal/platform/config"
	"www-template/packages/backend/internal/platform/observability"
)

// ProductContainer は Product Account 向けの application service と adapter 接続を束ねる DI container である。
//
// 役割:
//   - Account の passkey 認証、アカウント設定、token コンテキスト refresh、session lifecycle の各 service を保持する。
//   - Admin Container とは別型にし、Product Account credentials と Admin Operator credentials の混在を防ぐ。
//   - Valkey/DB 接続の close 関数を内部に保持し、ProductRuntime.Close から委譲されてリソース解放を行う。
//
// フィールド（公開）:
//   - AccountAuth: passkey 登録・認証・recovery・device-link を扱う認証 facade。
//   - AccountSetting: account locale 等の設定取得。
//   - AccountSnapshot: account setting の snapshot 取得。
//   - AccountContextRefresh: refresh token rotation と account context 提供。
//   - AccountSessions: session 一覧・失効。
//
// フィールド（非公開）:
//   - close: Valkey store、DB 接続などをまとめて閉じる関数。
//
// 使用例:
//
//	container, err := BuildProductContainer(ctx, cfg)
//	if err != nil {
//		return err
//	}
//	defer container.Close(ctx)
type ProductContainer struct {
	AccountAuth           productauth.ProductAuthService
	AccountSetting        *productaccounts.AccountSettingService
	AccountSnapshot       *productaccounts.AccountSettingSnapshotService
	AccountContextRefresh productauth.ProductContextRefreshService
	AccountSessions       productauth.ProductSessionService
	close                 func(context.Context) error
}

type accountAuthRepositoryFactory func(context.Context, string) (productauth.PasskeyAccountRepository, productaccounts.AccountSettingRepository, func(context.Context) error, error)
type authStateRepositoryFactory func(context.Context, config.ValkeyConfig, config.AuthConfig) (productauth.AuthStateRepository, func(context.Context) error, error)
type challengeStoreFactory func(context.Context, config.ValkeyConfig) (webauthn.ChallengeStore, func(context.Context) error, error)

type rejectingInvitationPasskeyRegistrar struct{}

func (rejectingInvitationPasskeyRegistrar) RegisterInvitationPasskey(context.Context, productauth.InvitationPasskeyRegistrationInput) (productauth.AuthSession, error) {
	return productauth.AuthSession{}, productauth.ErrBadRequest
}

//	slogAuditNotifier は slog を使って認証 audit event を標準出力に出力する実装。
//
// secret（credential raw data）は含めず、安全な識別子のみを記録する。
type slogAuditNotifier struct {
	logger *slog.Logger
}

func newSlogAuditNotifier(logger *slog.Logger) *slogAuditNotifier {
	return &slogAuditNotifier{logger: logger}
}

func (n *slogAuditNotifier) EmitCredentialStateUpdateFailure(ctx context.Context, credentialHandle string, err error) {
	n.logger.ErrorContext(ctx, "audit: credential state update failed",
		slog.String("event_type", "credential.state_update_failed"),
		slog.String("credential_handle", credentialHandle),
		slog.String("error", err.Error()),
	)
}

func (n *slogAuditNotifier) EmitDeviceLinkDeliveryFailure(ctx context.Context, requestID string, accountID domain.AccountID, err error) {
	n.logger.ErrorContext(ctx, "audit: device-link delivery failed",
		slog.String("event_type", "device_link.delivery_failed"),
		slog.String("request_id", requestID),
		slog.String("account_id", accountID.String()),
		slog.String("error", err.Error()),
	)
}

func (n *slogAuditNotifier) EmitRecoverySessionRevokeFailure(ctx context.Context, accountID domain.AccountID, err error) {
	n.logger.ErrorContext(ctx, "audit: recovery session revoke failed",
		slog.String("event_type", "recovery.session_revoke_failed"),
		slog.String("account_id", accountID.String()),
		slog.String("error", err.Error()),
	)
}

func (n *slogAuditNotifier) EmitRecoveryCompleteDeliveryFailure(ctx context.Context, accountID domain.AccountID, err error) {
	n.logger.ErrorContext(ctx, "audit: recovery complete delivery failed",
		slog.String("event_type", "recovery.complete_delivery_failed"),
		slog.String("account_id", accountID.String()),
		slog.String("error", err.Error()),
	)
}

func (n *slogAuditNotifier) EmitDeviceLinkCompleteDeliveryFailure(ctx context.Context, accountID domain.AccountID, err error) {
	n.logger.ErrorContext(ctx, "audit: device-link complete delivery failed",
		slog.String("event_type", "device_link.complete_delivery_failed"),
		slog.String("account_id", accountID.String()),
		slog.String("error", err.Error()),
	)
}

// BuildProductContainer は Product Account 向けの全依存を解決し、ProductContainer を生成する。
//
// 内部で production factory を使って DB repository、Valkey store、WebAuthn provider を構築する。
// テストでは buildProductContainer に test factory を注入して検証する。
//
// 引数:
//   - ctx: DB 接続や Valkey ping に使う context。
//   - cfg: 検証済みの Product runtime 設定。
//
// 戻り値:
//   - *ProductContainer: Account auth、setting、token refresh、session lifecycle の全 service を保持する container。
//   - error: DB/Valkey 接続失敗、WebAuthn RPID 未設定、service 構築失敗のいずれか。
//     エラー時は内部で構築済みの resource を close してから返す（fail-closed）。
//
// 使用例:
//
//	container, err := BuildProductContainer(ctx, cfg)
//	if err != nil {
//		return fmt.Errorf("build product container: %w", err)
//	}
func BuildProductContainer(ctx context.Context, cfg config.Config) (*ProductContainer, error) {
	return buildProductContainer(ctx, cfg, newAccountAuthRepository, newValkeyAuthStateRepository, newValkeyChallengeStore)
}

func buildProductContainer(ctx context.Context, cfg config.Config, newAccountAuthRepository accountAuthRepositoryFactory, newAuthStateRepository authStateRepositoryFactory, newChallengeStore challengeStoreFactory) (*ProductContainer, error) {
	authConfig := cfg.AuthRuntime()
	idPolicy := newAuthIDPolicy()

	accountRepo, accountSettingRepo, closeAccountRepo, err := newAccountAuthRepository(ctx, cfg.Infra.Database.URL)
	if err != nil {
		return nil, err
	}

	stateRepo, closeStateRepo, err := newAuthStateRepository(ctx, cfg.Infra.Valkey, authConfig)
	if err != nil {
		_ = closeAccountRepo(ctx)
		return nil, err
	}

	// WebAuthn challenge を Valkey-backed で保存するため、provider 専用の store を構築する。
	challengeStore, closeChallengeStore, challengeStoreErr := newChallengeStore(ctx, cfg.Infra.Valkey)
	if challengeStoreErr != nil {
		_ = composeClosers(closeStateRepo, closeAccountRepo)(ctx)
		return nil, fmt.Errorf("challenge store init: %w", challengeStoreErr)
	}

	accountSettingService := productaccounts.NewAccountSettingService(accountSettingRepo)
	accountSnapshotService := productaccounts.NewAccountSettingSnapshotService(accountSettingRepo)
	smtpSender := mailer.NewSMTPSender(cfg.Infra)
	recoverySender := mailer.NewAccountRecoverySender(smtpSender, cfg.Infra, accountSnapshotService)

	// Step 1: Product account auth の canonical lifecycle owner を production container で構築し、session issuance / refresh / bearer validation を root legacy TokenService から切り離す。
	productAuthStore, err := productvalkey.NewStore(cfg.Infra.Valkey)
	if err != nil {
		_ = composeClosers(closeChallengeStore, closeStateRepo, closeAccountRepo)(ctx)
		return nil, fmt.Errorf("product auth store init: %w", err)
	}
	productSigner, err := productauth.NewTokenJSONSignVerifier([]byte(authConfig.JWTSecret))
	if err != nil {
		_ = composeClosers(func(context.Context) error { return productAuthStore.Close() }, closeChallengeStore, closeStateRepo, closeAccountRepo)(ctx)
		return nil, fmt.Errorf("product auth signer init: %w", err)
	}
	refreshTokenTTL := authConfig.RefreshTokenTTL
	if refreshTokenTTL == 0 {
		// Step 2: 既存 config 互換で refresh TTL 未設定時は絶対 session TTL を使い、canonical lifecycle の TTL validation を無期限 token で失敗させない。
		refreshTokenTTL = authConfig.SessionAbsoluteTTL
	}
	productLifecycle, err := productauth.NewAccountSessionService(productauth.AccountSessionDependencies{
		Accounts:        accountRepo,
		RefreshSessions: productvalkey.NewAccountRefreshSessionStore(productAuthStore),
		Sessions:        productvalkey.NewAccountSessionMetadataStore(productAuthStore),
		Signer:          productSigner,
		IDGenerator:     idPolicy,
		TokenGenerator:  productauth.NewCryptoOpaqueTokenGenerator(),
		Clock:           func() time.Time { return time.Now().UTC() },
	}, productauth.AccountSessionConfig{AccessTokenTTL: domain.AccessTokenTTL, RefreshTokenTTL: refreshTokenTTL, RefreshCookieLifetime: refreshTokenTTL})
	if err != nil {
		_ = composeClosers(func(context.Context) error { return productAuthStore.Close() }, closeChallengeStore, closeStateRepo, closeAccountRepo)(ctx)
		return nil, fmt.Errorf("product auth lifecycle init: %w", err)
	}
	accountContextRefresh := productauth.NewProductContextRefreshService(productLifecycle)
	accountSessions := productauth.NewProductSessionService(productLifecycle)
	// RPID が未設定の場合は起動を拒否する（fail-closed）。
	if authConfig.WebAuthnRPID == "" {
		_ = composeClosers(func(context.Context) error { return productAuthStore.Close() }, closeChallengeStore, closeStateRepo, closeAccountRepo)(ctx)
		return nil, fmt.Errorf("webauthn RPID is required: set AUTH_WEBAUTHN_RPID")
	}

	webAuthnProv, webAuthnErr := webauthn.NewWebAuthnProvider(authConfig.WebAuthnRPID, cfg.AllowedOrigins, authConfig.ChallengeTTL, challengeStore)
	if webAuthnErr != nil {
		_ = composeClosers(func(context.Context) error { return productAuthStore.Close() }, closeChallengeStore, closeStateRepo, closeAccountRepo)(ctx)
		return nil, fmt.Errorf("webauthn provider init: %w", webAuthnErr)
	}
	authSvc, err := productauth.NewProductAuthService(stateRepo, accountRepo, recoverySender, rejectingInvitationPasskeyRegistrar{}, productLifecycle, productauth.AuthServiceOptionalPorts{WebAuthn: webAuthnProv, DeviceLinkSender: recoverySender, RecoveryCompleteSender: recoverySender, DeviceLinkCompleteSender: recoverySender, AuditNotifier: newSlogAuditNotifier(observability.Logger())}, func() time.Time {
		return time.Now().UTC()
	}, idPolicy, authConfig)
	if err != nil {
		_ = composeClosers(func(context.Context) error { return productAuthStore.Close() }, closeChallengeStore, closeStateRepo, closeAccountRepo)(ctx)
		return nil, fmt.Errorf("product auth facade init: %w", err)
	}

	return &ProductContainer{
		AccountAuth:           authSvc,
		AccountSetting:        accountSettingService,
		AccountSnapshot:       accountSnapshotService,
		AccountContextRefresh: accountContextRefresh,
		AccountSessions:       accountSessions,
		close:                 composeClosers(closeChallengeStore, closeStateRepo, closeAccountRepo, func(context.Context) error { return productAuthStore.Close() }),
	}, nil
}

// newValkeyChallengeStore は production 用の Valkey challengeStore を構築する。
func newValkeyChallengeStore(_ context.Context, config config.ValkeyConfig) (webauthn.ChallengeStore, func(context.Context) error, error) {
	store, err := productvalkey.NewStore(config)
	if err != nil {
		return nil, nil, err
	}
	return store, func(context.Context) error { return store.Close() }, nil
}

func newAccountAuthRepository(ctx context.Context, databaseURL string) (productauth.PasskeyAccountRepository, productaccounts.AccountSettingRepository, func(context.Context) error, error) {
	db, err := postgres.OpenDatabase(databaseURL)
	if err != nil {
		return nil, nil, nil, err
	}
	if err := postgres.PingDatabase(ctx, db); err != nil {
		sqlDB, dbErr := db.DB()
		if dbErr == nil {
			_ = sqlDB.Close()
		}
		return nil, nil, nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, nil, nil, err
	}

	return postgres.NewAccountAuthRepository(db), postgres.NewAccountSettingRepository(db), func(context.Context) error {
		return sqlDB.Close()
	}, nil
}

func newValkeyAuthStateRepository(ctx context.Context, cfg config.ValkeyConfig, authConfig config.AuthConfig) (productauth.AuthStateRepository, func(context.Context) error, error) {
	store, err := productvalkey.NewStore(cfg)
	if err != nil {
		return nil, nil, err
	}
	if err := store.Ping(ctx); err != nil {
		_ = store.Close()
		return nil, nil, err
	}

	repo, err := productvalkey.NewAuthStateRepository(store, authConfig.SecretHashKey)
	if err != nil {
		_ = store.Close()
		return nil, nil, err
	}

	return repo, func(context.Context) error {
		return repo.Close()
	}, nil
}

// Close は Product container が保持する Valkey store と DB 接続を解放する。
//
// BuildProductContainer 内で composeClosers により構築された close 関数を呼び出す。
// c が nil の場合は nil pointer dereference を起こし得るため、呼び出し側が nil guard すること。
//
// 引数:
//   - ctx: close 処理の deadline を制御する context。
//
// 戻り値:
//   - error: close 処理中に発生した最初のエラー。composeClosers は fail-fast であるため、
//     エラー発生以降の closer は実行されない。エラーがなければ nil を返す。
//
// 使用例:
//
//	defer container.Close(ctx)
func (c *ProductContainer) Close(ctx context.Context) error {
	return c.close(ctx)
}
