package app

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"www-template/packages/backend/internal/observability"
	"www-template/packages/backend/internal/persistence"
	"www-template/packages/backend/internal/types"
	"www-template/packages/backend/internal/usecases"
)

const defaultReadHeaderTimeout = 5 * time.Second

type Container struct {
	Auth  *usecases.AuthService
	close func(context.Context) error
}

type authAccountRepositoryFactory func(context.Context, string) (usecases.AuthAccountRepository, func(context.Context) error, error)
type authStateRepositoryFactory func(context.Context, types.ValkeyConfig) (usecases.AuthStateRepository, func(context.Context) error, error)
type challengeStoreFactory func(context.Context, types.ValkeyConfig) (challengeStore, func(context.Context) error, error)

type rejectingInvitationPasskeyRegistrar struct{}

func (rejectingInvitationPasskeyRegistrar) RegisterInvitationPasskey(context.Context, usecases.InvitationPasskeyRegistrationInput) (usecases.AuthSession, error) {
	return usecases.AuthSession{}, usecases.ErrBadRequest
}

// slogAuditNotifier は slog を使って認証 audit event を標準出力に出力する実装。
// secret（OTP、credential raw data）は含めず、accountID・passkeyID・requestID のみを記録する。
type slogAuditNotifier struct {
	logger *slog.Logger
}

func newSlogAuditNotifier(logger *slog.Logger) *slogAuditNotifier {
	return &slogAuditNotifier{logger: logger}
}

func (n *slogAuditNotifier) EmitPasskeyAddedByOTP(ctx context.Context, accountID string, passkeyID string, requestID string) {
	n.logger.InfoContext(ctx, "audit: passkey added by OTP",
		slog.String("event_type", "passkey.added_by_otp"),
		slog.String("account_id", accountID),
		slog.String("passkey_id", passkeyID),
		slog.String("request_id", requestID),
	)
}

func (n *slogAuditNotifier) EmitCredentialStateUpdateFailure(ctx context.Context, credentialHandle string, err error) {
	n.logger.ErrorContext(ctx, "audit: credential state update failed",
		slog.String("event_type", "credential.state_update_failed"),
		slog.String("credential_handle", credentialHandle),
		slog.String("error", err.Error()),
	)
}

func BuildContainer(ctx context.Context, cfg types.Config) (*Container, error) {
	return buildContainer(ctx, cfg, newGormAuthAccountRepository, newValkeyAuthStateRepository, newValkeyChallengeStore)
}

func buildContainer(ctx context.Context, cfg types.Config, newAuthAccountRepository authAccountRepositoryFactory, newAuthStateRepository authStateRepositoryFactory, newChallengeStore challengeStoreFactory) (*Container, error) {
	authConfig := cfg.AuthRuntime()
	idPolicy := newAuthIDPolicy()

	accountRepo, closeAccountRepo, err := newAuthAccountRepository(ctx, cfg.Infra.Database.URL)
	if err != nil {
		return nil, err
	}

	stateRepo, closeStateRepo, err := newAuthStateRepository(ctx, cfg.Infra.Valkey)
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

	smtpSender := NewSMTPSender(cfg.Infra)
	recoverySender := NewAccountRecoverySender(smtpSender, cfg.Infra)

	authSvc := usecases.NewAuthService(stateRepo, accountRepo, recoverySender, rejectingInvitationPasskeyRegistrar{}, func() time.Time {
		return time.Now().UTC()
	}, idPolicy, authConfig)
	authSvc.UsePasskeyOtpSender(recoverySender)
	authSvc.UseAuditNotifier(newSlogAuditNotifier(observability.Logger()))

	// RPID が未設定の場合は起動を拒否する（fail-closed）。
	if authConfig.WebAuthnRPID == "" {
		_ = composeClosers(closeChallengeStore, closeStateRepo, closeAccountRepo)(ctx)
		return nil, fmt.Errorf("webauthn RPID is required: set AUTH_WEBAUTHN_RPID")
	}

	webAuthnProv, webAuthnErr := newWebAuthnProvider(authConfig.WebAuthnRPID, cfg.AllowedOrigins, authConfig.ChallengeTTL, challengeStore)
	if webAuthnErr != nil {
		_ = composeClosers(closeChallengeStore, closeStateRepo, closeAccountRepo)(ctx)
		return nil, fmt.Errorf("webauthn provider init: %w", webAuthnErr)
	}
	authSvc.UseWebAuthnProvider(webAuthnProv)

	return &Container{
		Auth:  authSvc,
		close: composeClosers(closeChallengeStore, closeStateRepo, closeAccountRepo),
	}, nil
}

// newValkeyChallengeStore は production 用の Valkey challengeStore を構築する。
func newValkeyChallengeStore(_ context.Context, config types.ValkeyConfig) (challengeStore, func(context.Context) error, error) {
	store, err := persistence.NewValkeyStore(config)
	if err != nil {
		return nil, nil, err
	}
	return store, func(context.Context) error { return store.Close() }, nil
}

func newGormAuthAccountRepository(ctx context.Context, databaseURL string) (usecases.AuthAccountRepository, func(context.Context) error, error) {
	db, err := persistence.OpenGormDatabase(databaseURL)
	if err != nil {
		return nil, nil, err
	}
	if err := persistence.PingGormDatabase(ctx, db); err != nil {
		sqlDB, dbErr := db.DB()
		if dbErr == nil {
			_ = sqlDB.Close()
		}
		return nil, nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, nil, err
	}

	return persistence.NewGormAuthAccountRepository(db), func(context.Context) error {
		return sqlDB.Close()
	}, nil
}

func newValkeyAuthStateRepository(ctx context.Context, config types.ValkeyConfig) (usecases.AuthStateRepository, func(context.Context) error, error) {
	store, err := persistence.NewValkeyStore(config)
	if err != nil {
		return nil, nil, err
	}
	if err := store.Ping(ctx); err != nil {
		_ = store.Close()
		return nil, nil, err
	}

	repo, err := persistence.NewAuthStateRepository(store)
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
