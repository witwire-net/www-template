package app

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"www-template/packages/backend/internal/adapters/webauthn"
	"www-template/packages/backend/internal/auth/application"
	"www-template/packages/backend/internal/auth/domain"
	"www-template/packages/backend/internal/platform/config"
)

func TestBuildContainerUsesValkeyRepositoryWhenConfigured(t *testing.T) {
	t.Parallel()
	called := false
	fakeRepo := fakeAuthStateRepository{}
	_, err := buildContainer(
		context.Background(),
		config.Config{AppBearerToken: "dev-app-auth", Infra: config.InfraConfig{Database: config.DatabaseConfig{URL: "postgres://example"}, Valkey: config.ValkeyConfig{URL: "redis://localhost:6379/0"}}, Auth: config.AuthConfig{WebAuthnRPID: "example.com"}},
		func(context.Context, string) (application.AuthAccountRepository, func(context.Context) error, error) {
			return stubAuthAccountRepository{}, func(context.Context) error { return nil }, nil
		},
		func(context.Context, config.ValkeyConfig) (application.AuthStateRepository, func(context.Context) error, error) {
			called = true
			return fakeRepo, func(context.Context) error { return nil }, nil
		},
		fakeChallengeStoreFactory,
	)
	if err != nil {
		t.Fatalf("build container: %v", err)
	}
	if !called {
		t.Fatal("expected valkey factory to be called")
	}
}

func TestBuildContainerWiresConfiguredWebAuthnRPIDIntoAuthRuntime(t *testing.T) {
	t.Parallel()
	container, err := buildContainer(
		context.Background(),
		config.Config{AppBearerToken: "dev-app-auth", Infra: config.InfraConfig{Database: config.DatabaseConfig{URL: "postgres://example"}, Valkey: config.ValkeyConfig{URL: "redis://localhost:6379/0"}}, Auth: config.AuthConfig{WebAuthnRPID: "example.com"}},
		func(context.Context, string) (application.AuthAccountRepository, func(context.Context) error, error) {
			return stubAuthAccountRepository{}, func(context.Context) error { return nil }, nil
		},
		func(context.Context, config.ValkeyConfig) (application.AuthStateRepository, func(context.Context) error, error) {
			return fakeAuthStateRepository{}, func(context.Context) error { return nil }, nil
		},
		fakeChallengeStoreFactory,
	)
	if err != nil {
		t.Fatalf("build container: %v", err)
	}

	challenge, err := container.Auth.StartPasskeyAuthentication(context.Background(), application.StartPasskeyAuthenticationInput{Identifier: "member@example.com", ClientIP: "192.0.2.10"})
	if err != nil {
		t.Fatalf("start passkey authentication: %v", err)
	}
	if challenge.WebAuthnRPID != "example.com" {
		t.Fatalf("expected RP ID wiring to preserve config, got %q", challenge.WebAuthnRPID)
	}
}

func TestBuildContainerClosesAccountRepositoryWhenStateRepositoryFails(t *testing.T) {
	t.Parallel()

	closed := false
	_, err := buildContainer(
		context.Background(),
		config.Config{AppBearerToken: "dev-app-auth", Infra: config.InfraConfig{Database: config.DatabaseConfig{URL: "postgres://example"}, Valkey: config.ValkeyConfig{URL: "redis://localhost:6379/0"}}},
		func(context.Context, string) (application.AuthAccountRepository, func(context.Context) error, error) {
			return stubAuthAccountRepository{}, func(context.Context) error {
				closed = true
				return nil
			}, nil
		},
		func(context.Context, config.ValkeyConfig) (application.AuthStateRepository, func(context.Context) error, error) {
			return nil, nil, errors.New("valkey unavailable")
		},
		fakeChallengeStoreFactory,
	)
	if err == nil {
		t.Fatal("expected state repository error")
	}
	if !closed {
		t.Fatal("expected account repository closer to be invoked")
	}
}

type stubAuthAccountRepository struct{}

func (stubAuthAccountRepository) FindByIdentifier(context.Context, string) (domain.AuthAccount, error) {
	return emptyAuthAccountForContainerTest(), nil
}

func (stubAuthAccountRepository) FindByCredential(context.Context, string) (domain.AuthAccount, error) {
	return emptyAuthAccountForContainerTest(), nil
}

func (stubAuthAccountRepository) FindByEmail(context.Context, string) (domain.AuthAccount, error) {
	return emptyAuthAccountForContainerTest(), nil
}

func (stubAuthAccountRepository) AddPasskey(_ context.Context, _, _, _ string, _ domain.WebAuthnCredentialData) (domain.AuthAccount, error) {
	return emptyAuthAccountForContainerTest(), nil
}

func (stubAuthAccountRepository) ListPasskeys(_ context.Context, _ string) ([]domain.PasskeyCredential, error) {
	return nil, nil
}

func (stubAuthAccountRepository) DeletePasskeyByID(_ context.Context, _, _ string) error {
	return nil
}

func (stubAuthAccountRepository) FindWebAuthnCredential(_ context.Context, _ string) (domain.WebAuthnStoredCredential, error) {
	return domain.ZeroWebAuthnStoredCredential(), domain.ErrAuthAccountNotFound
}

func (stubAuthAccountRepository) UpdateWebAuthnCredentialState(_ context.Context, _ string, _ uint32, _ bool) error {
	return nil
}
func (stubAuthAccountRepository) FindByID(_ context.Context, _ string) (domain.AuthAccount, error) {
	return emptyAuthAccountForContainerTest(), nil
}

type fakeAuthStateRepository struct{}

func (fakeAuthStateRepository) SaveChallenge(context.Context, domain.AuthChallenge, time.Duration) error {
	return nil
}
func (fakeAuthStateRepository) ConsumeChallenge(context.Context, string) (domain.AuthChallenge, error) {
	return emptyChallengeForContainerTest(), nil
}
func (fakeAuthStateRepository) SaveSession(context.Context, domain.Session, time.Duration) error {
	return nil
}
func (fakeAuthStateRepository) RefreshSession(context.Context, domain.Session, time.Duration) error {
	return nil
}
func (fakeAuthStateRepository) GetSessionByToken(context.Context, string) (domain.Session, error) {
	return emptySessionForContainerTest(), nil
}
func (fakeAuthStateRepository) RevokeSession(context.Context, domain.Session, time.Duration) error {
	return nil
}
func (fakeAuthStateRepository) IssueRecoveryToken(context.Context, domain.RecoveryToken, time.Duration) error {
	return nil
}
func (fakeAuthStateRepository) SaveRecoveryDeliveryFailure(context.Context, domain.RecoveryDeliveryFailure, time.Duration) error {
	return nil
}
func (fakeAuthStateRepository) GetRecoveryTokenBySecret(context.Context, string) (domain.RecoveryToken, error) {
	return emptyRecoveryTokenForContainerTest(), nil
}
func (fakeAuthStateRepository) ConsumeRecoveryToken(context.Context, domain.RecoveryToken) error {
	return nil
}
func (fakeAuthStateRepository) ConsumeRecoveryTokenAtomic(context.Context, string, string) (domain.RecoveryToken, error) {
	return emptyRecoveryTokenForContainerTest(), domain.ErrRecoveryTokenNotFound
}
func (fakeAuthStateRepository) SaveRecoverySession(context.Context, domain.RecoverySession, time.Duration) error {
	return nil
}
func (fakeAuthStateRepository) GetRecoverySession(context.Context, string) (domain.RecoverySession, error) {
	return emptyRecoverySessionForContainerTest(), nil
}
func (fakeAuthStateRepository) ConsumeRecoverySession(context.Context, domain.RecoverySession) error {
	return nil
}
func (fakeAuthStateRepository) IncrementThrottle(context.Context, string, time.Duration) (int, error) {
	return 0, nil
}
func (fakeAuthStateRepository) SetLock(context.Context, string, time.Time, time.Duration) error {
	return nil
}
func (fakeAuthStateRepository) GetLock(context.Context, string) (domain.AuthLock, bool, error) {
	return domain.NewAuthLock(time.Time{}), false, nil
}
func (fakeAuthStateRepository) SavePasskeyOtp(context.Context, string, string, time.Duration) error {
	return nil
}
func (fakeAuthStateRepository) ConsumePasskeyOtp(context.Context, string) (string, error) {
	return "", nil
}
func (fakeAuthStateRepository) GetPasskeyOtp(context.Context, string) (string, error) {
	return "", nil
}
func (fakeAuthStateRepository) SaveReauthenticationSession(context.Context, domain.ReauthenticationSession, time.Duration) error {
	return nil
}
func (fakeAuthStateRepository) ConsumeReauthenticationSession(context.Context, string) (domain.ReauthenticationSession, error) {
	return emptyReauthenticationSessionForContainerTest(), domain.ErrReauthSessionNotFound
}
func (fakeAuthStateRepository) SaveDeviceLoginHandoff(context.Context, domain.DeviceLoginHandoff, time.Duration) error {
	return nil
}
func (fakeAuthStateRepository) FindDeviceLoginHandoffByEmailAndOtp(context.Context, string, string) (domain.DeviceLoginHandoff, error) {
	return emptyDeviceLoginHandoffForContainerTest(), domain.ErrDeviceLoginHandoffNotFound
}
func (fakeAuthStateRepository) ConsumeDeviceLoginHandoff(context.Context, string) (domain.DeviceLoginHandoff, error) {
	return emptyDeviceLoginHandoffForContainerTest(), domain.ErrDeviceLoginHandoffNotFound
}

func emptyChallengeForContainerTest() domain.AuthChallenge {
	challenge, _ := domain.NewAuthChallenge("01ARZ3NDEKTSV4RRFFQ69G5FAV", "placeholder", "placeholder", time.Unix(0, 0).UTC())
	return challenge
}

func emptySessionForContainerTest() domain.Session {
	session, _ := domain.NewSession("01ARZ3NDEKTSV4RRFFQ69G5FAV", "01ARZ3NDEKTSV4RRFFQ69G5FAW", "01ARZ3NDEKTSV4RRFFQ69G5FAX", "placeholder", time.Unix(1, 0).UTC(), time.Unix(2, 0).UTC())
	return session
}

func emptyRecoveryTokenForContainerTest() domain.RecoveryToken {
	token, _ := domain.NewRecoveryToken("01ARZ3NDEKTSV4RRFFQ69G5FAV", "01ARZ3NDEKTSV4RRFFQ69G5FAW", "placeholder", time.Unix(1, 0).UTC())
	return token
}

func emptyRecoverySessionForContainerTest() domain.RecoverySession {
	session, _ := domain.NewRecoverySession("01ARZ3NDEKTSV4RRFFQ69G5FAV", "01ARZ3NDEKTSV4RRFFQ69G5FAW", time.Unix(1, 0).UTC())
	return session
}

func emptyReauthenticationSessionForContainerTest() domain.ReauthenticationSession {
	session, _ := domain.NewReauthenticationSession(
		"01ARZ3NDEKTSV4RRFFQ69G5FAV", "01ARZ3NDEKTSV4RRFFQ69G5FAW", "01ARZ3NDEKTSV4RRFFQ69G5FAX",
		"otp-issue", "01ARZ3NDEKTSV4RRFFQ69G5FAY", time.Unix(1, 0).UTC(),
	)
	return session
}

func emptyDeviceLoginHandoffForContainerTest() domain.DeviceLoginHandoff {
	h, _ := domain.NewDeviceLoginHandoff("01ARZ3NDEKTSV4RRFFQ69G5FAV", "01ARZ3NDEKTSV4RRFFQ69G5FAW", "01ARZ3NDEKTSV4RRFFQ69G5FAX", "placeholder", "placeholder", time.Unix(1, 0).UTC())
	return h
}

func emptyAuthAccountForContainerTest() domain.AuthAccount {
	account, _ := domain.NewAuthAccount("01ARZ3NDEKTSV4RRFFQ69G5FAV", "member@example.com", "member@example.com", "01ARZ3NDEKTSV4RRFFQ69G5FB0", "existing-credential")
	return account
}

// fakeChallengeStore はテスト用のインメモリ challengeStore 実装。
type fakeChallengeStore struct {
	data map[string]string
}

func (f *fakeChallengeStore) Key(parts ...string) string {
	return strings.Join(parts, ":")
}

func (f *fakeChallengeStore) Set(_ context.Context, key string, value string, _ time.Duration) error {
	f.data[key] = value
	return nil
}

func (f *fakeChallengeStore) GetDel(_ context.Context, key string) (string, error) {
	v, ok := f.data[key]
	if !ok {
		return "", errors.New("not found")
	}
	delete(f.data, key)
	return v, nil
}

func fakeChallengeStoreFactory(context.Context, config.ValkeyConfig) (webauthn.ChallengeStore, func(context.Context) error, error) {
	return &fakeChallengeStore{data: map[string]string{}}, func(context.Context) error { return nil }, nil
}
