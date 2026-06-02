package app

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"www-template/packages/backend/internal/adapter/mailer"
	"www-template/packages/backend/internal/adapter/postgres"
	productpostgres "www-template/packages/backend/internal/adapter/postgres/product"
	"www-template/packages/backend/internal/adapter/valkey"
	productvalkey "www-template/packages/backend/internal/adapter/valkey/product"
	"www-template/packages/backend/internal/adapter/webauthn"
	productaccounts "www-template/packages/backend/internal/application/accounts"
	productauth "www-template/packages/backend/internal/application/auth"
	domain "www-template/packages/backend/internal/domain"
	"www-template/packages/backend/internal/platform/config"
	"www-template/packages/backend/internal/platform/observability"
)

const defaultReadHeaderTimeout = 5 * time.Second

type Container struct {
	Auth            productauth.ProductAuthService
	AccountSetting  *productaccounts.AccountSettingService
	AccountSnapshot *productaccounts.AccountSettingSnapshotService
	TokenService    productauth.ProductContextRefreshService
	SessionService  productauth.ProductSessionService
	close           func(context.Context) error
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

func BuildContainer(ctx context.Context, cfg config.Config) (*Container, error) {
	return buildContainer(ctx, cfg, newGormAccountAuthRepository, newValkeyAuthStateRepository, newValkeyChallengeStore)
}

func buildContainer(ctx context.Context, cfg config.Config, newAccountAuthRepository accountAuthRepositoryFactory, newAuthStateRepository authStateRepositoryFactory, newChallengeStore challengeStoreFactory) (*Container, error) {
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
	tokenService := productauth.NewProductContextRefreshService(productLifecycle)
	sessionService := productauth.NewProductSessionService(productLifecycle)
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

	return &Container{
		Auth:            authSvc,
		AccountSetting:  accountSettingService,
		AccountSnapshot: accountSnapshotService,
		TokenService:    tokenService,
		SessionService:  sessionService,
		close:           composeClosers(closeChallengeStore, closeStateRepo, closeAccountRepo, func(context.Context) error { return productAuthStore.Close() }),
	}, nil
}

// newValkeyChallengeStore は production 用の Valkey challengeStore を構築する。
func newValkeyChallengeStore(_ context.Context, config config.ValkeyConfig) (webauthn.ChallengeStore, func(context.Context) error, error) {
	store, err := valkey.NewStore(config)
	if err != nil {
		return nil, nil, err
	}
	return store, func(context.Context) error { return store.Close() }, nil
}

func newGormAccountAuthRepository(ctx context.Context, databaseURL string) (productauth.PasskeyAccountRepository, productaccounts.AccountSettingRepository, func(context.Context) error, error) {
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

	return productpostgres.NewGormAccountAuthRepository(db), productpostgres.NewGormAccountSettingRepository(db), func(context.Context) error {
		return sqlDB.Close()
	}, nil
}

func newValkeyAuthStateRepository(ctx context.Context, cfg config.ValkeyConfig, authConfig config.AuthConfig) (productauth.AuthStateRepository, func(context.Context) error, error) {
	store, err := valkey.NewStore(cfg)
	if err != nil {
		return nil, nil, err
	}
	if err := store.Ping(ctx); err != nil {
		_ = store.Close()
		return nil, nil, err
	}

	repo, err := valkey.NewAuthStateRepository(store, authConfig.SecretHashKey)
	if err != nil {
		_ = store.Close()
		return nil, nil, err
	}

	return repo, func(context.Context) error {
		return repo.Close()
	}, nil
}

func composeClosers(closers ...func(context.Context) error) func(context.Context) error {
	return func(ctx context.Context) error {
		for _, closeFn := range closers {
			if closeFn == nil {
				continue
			}
			if err := closeFn(ctx); err != nil {
				return err
			}
		}
		return nil
	}
}

func (c *Container) Close(ctx context.Context) error {
	return c.close(ctx)
}
