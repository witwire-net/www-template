package application_test

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"testing"
	"time"

	"www-template/packages/backend/internal/auth/application"
	"www-template/packages/backend/internal/auth/domain"
	"www-template/packages/backend/internal/platform/config"
	"www-template/packages/backend/internal/platform/id"
)

// ─── stubs ───────────────────────────────────────────────────────────────────

type stubStateRepo struct {
	challenges       map[string]domain.AuthChallenge
	sessions         map[string]domain.Session
	recoveryTokens   map[string]domain.RecoveryToken
	recoverySessions map[string]domain.RecoverySession
	counters         map[string]int
	locks            map[string]time.Time
	otpStore         map[string]string
	handoffs         map[string]domain.DeviceLoginHandoff
	clock            func() time.Time
}

// stubAuditNotifier は AuditNotifier のテスト用スタブ。
// EmitPasskeyAddedByOTP の呼び出し回数と引数を記録する。
type stubAuditNotifier struct {
	emitCount     int
	lastAccountID string
	lastPasskeyID string
	lastRequestID string
}

func (n *stubAuditNotifier) EmitPasskeyAddedByOTP(_ context.Context, accountID string, passkeyID string, requestID string) {
	n.emitCount++
	n.lastAccountID = accountID
	n.lastPasskeyID = passkeyID
	n.lastRequestID = requestID
}

func (n *stubAuditNotifier) EmitCredentialStateUpdateFailure(_ context.Context, _ string, _ error) {
}

func newStubStateRepo(clock func() time.Time) *stubStateRepo {
	return &stubStateRepo{
		challenges:       map[string]domain.AuthChallenge{},
		sessions:         map[string]domain.Session{},
		recoveryTokens:   map[string]domain.RecoveryToken{},
		recoverySessions: map[string]domain.RecoverySession{},
		counters:         map[string]int{},
		locks:            map[string]time.Time{},
		otpStore:         map[string]string{},
		handoffs:         map[string]domain.DeviceLoginHandoff{},
		clock:            clock,
	}
}

func (r *stubStateRepo) SaveChallenge(_ context.Context, c domain.AuthChallenge, _ time.Duration) error {
	r.challenges[c.Challenge()] = c
	return nil
}
func (r *stubStateRepo) ConsumeChallenge(_ context.Context, secret string) (domain.AuthChallenge, error) {
	c, ok := r.challenges[secret]
	if !ok {
		return emptyChallenge(), domain.ErrChallengeNotFound
	}
	delete(r.challenges, secret)
	return c, nil
}
func (r *stubStateRepo) SaveSession(_ context.Context, s domain.Session, _ time.Duration) error {
	r.sessions[s.Token()] = s
	return nil
}
func (r *stubStateRepo) RefreshSession(_ context.Context, s domain.Session, _ time.Duration) error {
	r.sessions[s.Token()] = s
	return nil
}
func (r *stubStateRepo) GetSessionByToken(_ context.Context, token string) (domain.Session, error) {
	s, ok := r.sessions[token]
	if !ok {
		return emptySession(), domain.ErrSessionNotFound
	}
	return s, nil
}
func (r *stubStateRepo) RevokeSession(_ context.Context, s domain.Session, _ time.Duration) error {
	r.sessions[s.Token()] = s
	return nil
}
func (r *stubStateRepo) IssueRecoveryToken(_ context.Context, t domain.RecoveryToken, _ time.Duration) error {
	r.recoveryTokens[t.Secret()] = t
	return nil
}
func (r *stubStateRepo) SaveRecoveryDeliveryFailure(_ context.Context, _ domain.RecoveryDeliveryFailure, _ time.Duration) error {
	return nil
}
func (r *stubStateRepo) GetRecoveryTokenBySecret(_ context.Context, secret string) (domain.RecoveryToken, error) {
	t, ok := r.recoveryTokens[secret]
	if !ok {
		return emptyRecoveryToken(), domain.ErrRecoveryTokenNotFound
	}
	return t, nil
}
func (r *stubStateRepo) ConsumeRecoveryToken(_ context.Context, t domain.RecoveryToken) error {
	r.recoveryTokens[t.Secret()] = t
	return nil
}

func (r *stubStateRepo) ConsumeRecoveryTokenAtomic(_ context.Context, tokenID string, secret string) (domain.RecoveryToken, error) {
	t, ok := r.recoveryTokens[secret]
	if !ok {
		return emptyRecoveryToken(), domain.ErrRecoveryTokenNotFound
	}
	delete(r.recoveryTokens, secret)
	return t, nil
}
func (r *stubStateRepo) SaveRecoverySession(_ context.Context, s domain.RecoverySession, _ time.Duration) error {
	r.recoverySessions[s.ID()] = s
	return nil
}
func (r *stubStateRepo) GetRecoverySession(_ context.Context, id string) (domain.RecoverySession, error) {
	s, ok := r.recoverySessions[id]
	if !ok {
		return emptyRecoverySession(), domain.ErrRecoverySessionNotFound
	}
	return s, nil
}
func (r *stubStateRepo) ConsumeRecoverySession(_ context.Context, s domain.RecoverySession) error {
	r.recoverySessions[s.ID()] = s
	return nil
}
func (r *stubStateRepo) IncrementThrottle(_ context.Context, key string, _ time.Duration) (int, error) {
	r.counters[key]++
	return r.counters[key], nil
}
func (r *stubStateRepo) SetLock(_ context.Context, key string, until time.Time, _ time.Duration) error {
	r.locks[key] = until
	return nil
}
func (r *stubStateRepo) GetLock(_ context.Context, key string) (domain.AuthLock, bool, error) {
	until, ok := r.locks[key]
	if !ok {
		return domain.NewAuthLock(time.Time{}), false, nil
	}
	return domain.NewAuthLock(until), true, nil
}
func (r *stubStateRepo) SavePasskeyOtp(_ context.Context, otpKey string, value string, _ time.Duration) error {
	r.otpStore[otpKey] = value
	return nil
}
func (r *stubStateRepo) ConsumePasskeyOtp(_ context.Context, otpKey string) (string, error) {
	v, ok := r.otpStore[otpKey]
	if !ok {
		return "", domain.ErrOtpNotFound
	}
	delete(r.otpStore, otpKey)
	return v, nil
}
func (r *stubStateRepo) GetPasskeyOtp(_ context.Context, otpKey string) (string, error) {
	v, ok := r.otpStore[otpKey]
	if !ok {
		return "", domain.ErrOtpNotFound
	}
	return v, nil
}

func (r *stubStateRepo) SaveReauthenticationSession(_ context.Context, _ domain.ReauthenticationSession, _ time.Duration) error {
	return nil
}

func (r *stubStateRepo) ConsumeReauthenticationSession(_ context.Context, _ string) (domain.ReauthenticationSession, error) {
	session, _ := domain.NewReauthenticationSession(
		"01ARZ3NDEKTSV4RRFFQ69G5FAV", "01ARZ3NDEKTSV4RRFFQ69G5FAW", "01ARZ3NDEKTSV4RRFFQ69G5FAX",
		"otp-issue", "01ARZ3NDEKTSV4RRFFQ69G5FAY", time.Unix(1, 0).UTC(),
	)
	return session, domain.ErrReauthSessionNotFound
}

func (r *stubStateRepo) SaveDeviceLoginHandoff(_ context.Context, handoff domain.DeviceLoginHandoff, _ time.Duration) error {
	r.handoffs[handoff.ID()] = handoff
	return nil
}

func (r *stubStateRepo) FindDeviceLoginHandoffByEmailAndOtp(_ context.Context, emailHash string, otpHash string) (domain.DeviceLoginHandoff, error) {
	for _, h := range r.handoffs {
		if h.EmailHash() == emailHash && h.OtpHash() == otpHash {
			return h, nil
		}
	}
	return emptyDeviceLoginHandoff(), domain.ErrDeviceLoginHandoffNotFound
}

func (r *stubStateRepo) ConsumeDeviceLoginHandoff(_ context.Context, handoffID string) (domain.DeviceLoginHandoff, error) {
	h, ok := r.handoffs[handoffID]
	if !ok {
		return emptyDeviceLoginHandoff(), domain.ErrDeviceLoginHandoffNotFound
	}
	delete(r.handoffs, handoffID)
	return h, nil
}

// stubAccountRepo は AuthAccountRepository の in-memory スタブ。
type stubAccountRepo struct {
	accounts map[string]domain.AuthAccount // keyed by accountID
}

func newStubAccountRepoWithAccount(account domain.AuthAccount) *stubAccountRepo {
	return &stubAccountRepo{accounts: map[string]domain.AuthAccount{account.AccountID(): account}}
}

func (r *stubAccountRepo) FindByID(_ context.Context, accountID string) (domain.AuthAccount, error) {
	a, ok := r.accounts[accountID]
	if !ok {
		return emptyAuthAccount(), domain.ErrAuthAccountNotFound
	}
	return a, nil
}
func (r *stubAccountRepo) FindByIdentifier(_ context.Context, identifier string) (domain.AuthAccount, error) {
	for _, a := range r.accounts {
		if a.Identifier() == identifier {
			return a, nil
		}
	}
	return emptyAuthAccount(), domain.ErrAuthAccountNotFound
}
func (r *stubAccountRepo) FindByCredential(_ context.Context, handle string) (domain.AuthAccount, error) {
	for _, a := range r.accounts {
		for _, c := range a.Credentials() {
			if c.CredentialHandle() == handle {
				return a, nil
			}
		}
	}
	return emptyAuthAccount(), domain.ErrAuthAccountNotFound
}
func (r *stubAccountRepo) FindByEmail(_ context.Context, email string) (domain.AuthAccount, error) {
	for _, a := range r.accounts {
		if a.Email() == email {
			return a, nil
		}
	}
	return emptyAuthAccount(), domain.ErrAuthAccountNotFound
}
func (r *stubAccountRepo) AddPasskey(_ context.Context, accountID string, credentialID string, handle string, _ domain.WebAuthnCredentialData) (domain.AuthAccount, error) {
	a, ok := r.accounts[accountID]
	if !ok {
		return emptyAuthAccount(), domain.ErrAuthAccountNotFound
	}
	newCred, err := domain.NewPasskeyCredential(credentialID, accountID, a.Identifier(), handle, time.Time{})
	if err != nil {
		return emptyAuthAccount(), err
	}
	updated, err := domain.NewAuthAccountWithCredentials(a.AccountID(), a.Identifier(), a.Email(), append(a.Credentials(), newCred))
	if err != nil {
		return emptyAuthAccount(), err
	}
	r.accounts[accountID] = updated
	return updated, nil
}
func (r *stubAccountRepo) ListPasskeys(_ context.Context, accountID string) ([]domain.PasskeyCredential, error) {
	a, ok := r.accounts[accountID]
	if !ok {
		return nil, domain.ErrAuthAccountNotFound
	}
	return a.Credentials(), nil
}
func (r *stubAccountRepo) DeletePasskeyByID(_ context.Context, accountID string, credentialID string) error {
	a, ok := r.accounts[accountID]
	if !ok {
		return domain.ErrAuthAccountNotFound
	}
	creds := a.Credentials()
	for _, c := range creds {
		if c.ID() == credentialID {
			return nil
		}
	}
	return domain.ErrAuthAccountNotFound
}

func (r *stubAccountRepo) FindWebAuthnCredential(_ context.Context, handle string) (domain.WebAuthnStoredCredential, error) {
	for _, a := range r.accounts {
		for _, c := range a.Credentials() {
			if c.CredentialHandle() == handle {
				return domain.ReconstitueWebAuthnStoredCredential(handle, nil, 0, nil, false, false, nil), nil
			}
		}
	}
	return domain.ZeroWebAuthnStoredCredential(), domain.ErrAuthAccountNotFound
}

func (r *stubAccountRepo) UpdateWebAuthnCredentialState(_ context.Context, _ string, _ uint32, _ bool) error {
	return nil
}

// trackingAccountRepo は UpdateWebAuthnCredentialState の呼び出しを記録する stubAccountRepo の拡張。
type trackingAccountRepo struct {
	stubAccountRepo
	updateStateCallCount int
	updateStateErr       error
}

func newTrackingAccountRepo(account domain.AuthAccount) *trackingAccountRepo {
	return &trackingAccountRepo{
		stubAccountRepo: stubAccountRepo{accounts: map[string]domain.AuthAccount{account.AccountID(): account}},
	}
}

func (r *trackingAccountRepo) UpdateWebAuthnCredentialState(_ context.Context, _ string, _ uint32, _ bool) error {
	r.updateStateCallCount++
	return r.updateStateErr
}

// ─── helpers ──────────────────────────────────────────────────────────────────

func emptyChallenge() domain.AuthChallenge {
	c, _ := domain.NewAuthChallenge("01ARZ3NDEKTSV4RRFFQ69G5FAV", "placeholder", "placeholder", time.Unix(0, 0).UTC())
	return c
}
func emptySession() domain.Session {
	s, _ := domain.NewSession("01ARZ3NDEKTSV4RRFFQ69G5FAV", "01ARZ3NDEKTSV4RRFFQ69G5FAW", "01ARZ3NDEKTSV4RRFFQ69G5FAX", "placeholder", time.Unix(1, 0).UTC(), time.Unix(2, 0).UTC())
	return s
}
func emptyRecoveryToken() domain.RecoveryToken {
	t, _ := domain.NewRecoveryToken("01ARZ3NDEKTSV4RRFFQ69G5FAV", "01ARZ3NDEKTSV4RRFFQ69G5FAW", "placeholder", time.Unix(1, 0).UTC())
	return t
}
func emptyDeviceLoginHandoff() domain.DeviceLoginHandoff {
	h, _ := domain.NewDeviceLoginHandoff("01ARZ3NDEKTSV4RRFFQ69G5FAV", "01ARZ3NDEKTSV4RRFFQ69G5FAW", "01ARZ3NDEKTSV4RRFFQ69G5FAX", "placeholder", "placeholder", time.Unix(1, 0).UTC())
	return h
}
func emptyRecoverySession() domain.RecoverySession {
	s, _ := domain.NewRecoverySession("01ARZ3NDEKTSV4RRFFQ69G5FAV", "01ARZ3NDEKTSV4RRFFQ69G5FAW", time.Unix(1, 0).UTC())
	return s
}
func emptyAuthAccount() domain.AuthAccount {
	a, _ := domain.NewAuthAccount("01ARZ3NDEKTSV4RRFFQ69G5FAV", "placeholder@example.com", "placeholder@example.com", "01ARZ3NDEKTSV4RRFFQ69G5FAW", "placeholder")
	return a
}

var ids = []string{
	"01ARZ3NDEKTSV4RRFFQ69G5FAV", "01ARZ3NDEKTSV4RRFFQ69G5FAW", "01ARZ3NDEKTSV4RRFFQ69G5FAX",
	"01ARZ3NDEKTSV4RRFFQ69G5FAY", "01ARZ3NDEKTSV4RRFFQ69G5FAZ", "01ARZ3NDEKTSV4RRFFQ69G5FB1",
	"01ARZ3NDEKTSV4RRFFQ69G5FB2", "01ARZ3NDEKTSV4RRFFQ69G5FB3", "01ARZ3NDEKTSV4RRFFQ69G5FB4",
	"01ARZ3NDEKTSV4RRFFQ69G5FB5", "01ARZ3NDEKTSV4RRFFQ69G5FB6", "01ARZ3NDEKTSV4RRFFQ69G5FB7",
	"01ARZ3NDEKTSV4RRFFQ69G5FB8", "01ARZ3NDEKTSV4RRFFQ69G5FB9", "01ARZ3NDEKTSV4RRFFQ69G5FBA",
	"01ARZ3NDEKTSV4RRFFQ69G5FBB", "01ARZ3NDEKTSV4RRFFQ69G5FBC", "01ARZ3NDEKTSV4RRFFQ69G5FBD",
	"01ARZ3NDEKTSV4RRFFQ69G5FBE", "01ARZ3NDEKTSV4RRFFQ69G5FBF",
}

func newSeqPolicy() id.AuthIDPolicy {
	next := 0
	return id.AuthIDPolicy{
		New:      func() string { v := ids[next]; next++; return v },
		Validate: domain.ValidateAuthID,
	}
}

func fixedClock() func() time.Time {
	t := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	return func() time.Time { return t }
}

func accountWithTwoPasskeys(t *testing.T) (domain.AuthAccount, *stubAccountRepo) {
	t.Helper()
	cred1, err := domain.NewPasskeyCredential(
		"01ARZ3NDEKTSV4RRFFQ69G5FAV",
		"01ARZ3NDEKTSV4RRFFQ69G5FAW",
		"user@example.com",
		"handle-one",
		time.Time{},
	)
	if err != nil {
		t.Fatalf("NewPasskeyCredential cred1: %v", err)
	}
	cred2, err := domain.NewPasskeyCredential(
		"01ARZ3NDEKTSV4RRFFQ69G5FAX",
		"01ARZ3NDEKTSV4RRFFQ69G5FAW",
		"user@example.com",
		"handle-two",
		time.Time{},
	)
	if err != nil {
		t.Fatalf("NewPasskeyCredential cred2: %v", err)
	}
	account, err := domain.NewAuthAccountWithCredentials(
		"01ARZ3NDEKTSV4RRFFQ69G5FAW",
		"user@example.com",
		"user@example.com",
		[]domain.PasskeyCredential{cred1, cred2},
	)
	if err != nil {
		t.Fatalf("NewAuthAccountWithCredentials: %v", err)
	}
	return account, newStubAccountRepoWithAccount(account)
}

func accountWithOnePasskey(t *testing.T) (domain.AuthAccount, *stubAccountRepo) {
	t.Helper()
	account, err := domain.NewAuthAccount(
		"01ARZ3NDEKTSV4RRFFQ69G5FAW",
		"user@example.com",
		"user@example.com",
		"01ARZ3NDEKTSV4RRFFQ69G5FAV",
		"handle-one",
	)
	if err != nil {
		t.Fatalf("NewAuthAccount: %v", err)
	}
	return account, newStubAccountRepoWithAccount(account)
}

func newTestAuthService(stateRepo application.AuthStateRepository, accountRepo application.AuthAccountRepository) *application.AuthService {
	return application.NewAuthService(stateRepo, accountRepo, nil, nil, fixedClock(), newSeqPolicy(), config.AuthConfig{
		ChallengeTTL:                    5 * time.Minute,
		SessionIdleTTL:                  30 * time.Minute,
		SessionAbsoluteTTL:              24 * time.Hour,
		RecoveryTokenTTL:                30 * time.Minute,
		RecoverySessionTTL:              10 * time.Minute,
		RecoveryEmailThrottleLimit:      3,
		RecoveryIPThrottleLimit:         5,
		RecoveryEmailThrottleWindow:     time.Hour,
		RecoveryIPThrottleWindow:        time.Hour,
		PasskeyStartThrottleLimit:       5,
		PasskeyStartGlobalThrottleLimit: 1000,
		HandoffGlobalThrottleLimit:      1000,
		SecretHashKey:                   "test-pepper",
		PasskeyStartThrottleWindow:      time.Minute,
		FailureLockThreshold:            10,
		FailureLockDuration:             15 * time.Minute,
		FailureLockWindow:               time.Minute,
		WebAuthnRPID:                    "example.com",
		AccountRecoveryURLBase:          "https://example.com/recover",
	})
}

// ─── MockWebAuthnProvider ─────────────────────────────────────────────────────

// mockWebAuthnProvider は WebAuthnProvider interface のテスト用スタブ。
// テストごとに BeginLogin/FinishLogin/BeginRegistration/FinishRegistration の
// 戻り値・エラーをカスタマイズできる。
type mockWebAuthnProvider struct {
	beginLoginFn         func(ctx context.Context, identifier string) (string, []byte, error)
	finishLoginFn        func(ctx context.Context, challengeKey string, credential application.WebAuthnAssertionCredentialDTO, lookupCredential func(ctx context.Context, handle string) (domain.WebAuthnStoredCredential, error)) (string, uint32, bool, bool, error)
	beginRegistrationFn  func(ctx context.Context, accountID string) (string, []byte, error)
	finishRegistrationFn func(ctx context.Context, challengeKey string, accountID string, credential application.WebAuthnAttestationCredentialDTO) (string, domain.WebAuthnCredentialData, error)
}

func (m *mockWebAuthnProvider) BeginLogin(ctx context.Context, identifier string) (string, []byte, error) {
	if m.beginLoginFn != nil {
		return m.beginLoginFn(ctx, identifier)
	}
	return "challenge-key-login", []byte(`{"challenge":"challenge-key-login"}`), nil
}

func (m *mockWebAuthnProvider) FinishLogin(ctx context.Context, challengeKey string, credential application.WebAuthnAssertionCredentialDTO, lookupCredential func(ctx context.Context, handle string) (domain.WebAuthnStoredCredential, error)) (string, uint32, bool, bool, error) {
	if m.finishLoginFn != nil {
		return m.finishLoginFn(ctx, challengeKey, credential, lookupCredential)
	}
	return "handle-one", 1, false, true, nil
}

func (m *mockWebAuthnProvider) BeginRegistration(ctx context.Context, accountID string) (string, []byte, error) {
	if m.beginRegistrationFn != nil {
		return m.beginRegistrationFn(ctx, accountID)
	}
	return "challenge-key-reg", []byte(`{"challenge":"challenge-key-reg"}`), nil
}

func (m *mockWebAuthnProvider) FinishRegistration(ctx context.Context, challengeKey string, accountID string, credential application.WebAuthnAttestationCredentialDTO) (string, domain.WebAuthnCredentialData, error) {
	if m.finishRegistrationFn != nil {
		return m.finishRegistrationFn(ctx, challengeKey, accountID, credential)
	}
	return credential.ID, domain.ZeroWebAuthnCredentialData(), nil
}

// newTestAuthServiceWithProvider は WebAuthn provider を注入した AuthService を返す。
func newTestAuthServiceWithProvider(stateRepo application.AuthStateRepository, accountRepo application.AuthAccountRepository, provider application.WebAuthnProvider) *application.AuthService {
	svc := newTestAuthService(stateRepo, accountRepo)
	svc.UseWebAuthnProvider(provider)
	return svc
}

// ─── Provider-on tests ────────────────────────────────────────────────────────

// [AUTH-BE-WA-1] StartPasskeyAuthentication with provider: returns WebAuthnOptions from provider.
func TestStartPasskeyAuthenticationWithProvider(t *testing.T) {
	t.Parallel()
	_, accountRepo := accountWithOnePasskey(t)
	stateRepo := newStubStateRepo(fixedClock())
	provider := &mockWebAuthnProvider{
		beginLoginFn: func(_ context.Context, _ string) (string, []byte, error) {
			return "ck-login-001", []byte(`{"publicKey":{"challenge":"ck-login-001"}}`), nil
		},
	}
	svc := newTestAuthServiceWithProvider(stateRepo, accountRepo, provider)

	ch, err := svc.StartPasskeyAuthentication(context.Background(), application.StartPasskeyAuthenticationInput{
		Identifier: "user@example.com",
		ClientIP:   "127.0.0.1",
	})
	if err != nil {
		t.Fatalf("StartPasskeyAuthentication: %v", err)
	}
	if ch.Challenge != "ck-login-001" {
		t.Errorf("expected Challenge=ck-login-001, got %q", ch.Challenge)
	}
	if ch.ChallengeID != "ck-login-001" {
		t.Errorf("expected ChallengeID=ck-login-001, got %q", ch.ChallengeID)
	}
	if len(ch.WebAuthnOptions) == 0 {
		t.Error("expected non-empty WebAuthnOptions")
	}
}

// [AUTH-BE-WA-2] FinishPasskeyAuthentication with provider: calls FinishLogin and issues session.
func TestFinishPasskeyAuthenticationWithProvider(t *testing.T) {
	t.Parallel()
	account, accountRepo := accountWithOnePasskey(t)
	stateRepo := newStubStateRepo(fixedClock())

	// provider は "handle-one" を返す（account に handle-one が登録済み）
	provider := &mockWebAuthnProvider{
		finishLoginFn: func(_ context.Context, _ string, _ application.WebAuthnAssertionCredentialDTO, lookupFn func(context.Context, string) (domain.WebAuthnStoredCredential, error)) (string, uint32, bool, bool, error) {
			// lookupCredential が正しく渡されていることを確認
			if lookupFn == nil {
				t.Error("lookupCredential callback must not be nil")
			}
			return "handle-one", 1, false, true, nil
		},
	}
	svc := newTestAuthServiceWithProvider(stateRepo, accountRepo, provider)

	result, err := svc.FinishPasskeyAuthentication(context.Background(), application.FinishPasskeyAuthenticationInput{
		Credential: application.WebAuthnAssertionCredentialDTO{
			ID:    "handle-one",
			RawID: "handle-one",
			Type:  "public-key",
			Response: application.WebAuthnAssertionResponseDTO{
				ClientDataJSON:    "clientdata",
				AuthenticatorData: "authdata",
				Signature:         "sig",
			},
		},
		ClientIP: "127.0.0.1",
	})
	if err != nil {
		t.Fatalf("FinishPasskeyAuthentication: %v", err)
	}
	if result.AccountID != account.AccountID() {
		t.Errorf("expected AccountID=%q, got %q", account.AccountID(), result.AccountID)
	}
	if result.SessionToken == "" {
		t.Error("expected non-empty SessionToken")
	}
}

// [AUTH-BE-WA-3] FinishPasskeyAuthentication with provider: FinishLogin error → ErrBadRequest.
func TestFinishPasskeyAuthenticationWithProviderError(t *testing.T) {
	t.Parallel()
	_, accountRepo := accountWithOnePasskey(t)
	stateRepo := newStubStateRepo(fixedClock())

	provider := &mockWebAuthnProvider{
		finishLoginFn: func(_ context.Context, _ string, _ application.WebAuthnAssertionCredentialDTO, _ func(context.Context, string) (domain.WebAuthnStoredCredential, error)) (string, uint32, bool, bool, error) {
			return "", 0, false, false, context.DeadlineExceeded // simulate provider error
		},
	}
	svc := newTestAuthServiceWithProvider(stateRepo, accountRepo, provider)

	_, err := svc.FinishPasskeyAuthentication(context.Background(), application.FinishPasskeyAuthenticationInput{
		Credential: application.WebAuthnAssertionCredentialDTO{
			ID:       "bad-handle",
			Response: application.WebAuthnAssertionResponseDTO{ClientDataJSON: "x"},
		},
		ClientIP: "127.0.0.1",
	})
	if err != application.ErrBadRequest {
		t.Fatalf("expected ErrBadRequest, got %v", err)
	}
}

// [AUTH-BE-WA-4] StartPasskeyRegistration with provider: returns WebAuthnOptions.
func TestStartPasskeyRegistrationWithProvider(t *testing.T) {
	t.Parallel()
	stateRepo := newStubStateRepo(fixedClock())
	_, accountRepo := accountWithOnePasskey(t)

	// recovery session をセットアップ
	accountID := "01ARZ3NDEKTSV4RRFFQ69G5FAW"
	recoverySession, err := domain.NewRecoverySession("01ARZ3NDEKTSV4RRFFQ69G5FAV", accountID, time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("NewRecoverySession: %v", err)
	}
	stateRepo.recoverySessions[recoverySession.ID()] = recoverySession

	provider := &mockWebAuthnProvider{
		beginRegistrationFn: func(_ context.Context, _ string) (string, []byte, error) {
			return "ck-reg-001", []byte(`{"publicKey":{"challenge":"ck-reg-001"}}`), nil
		},
	}
	svc := newTestAuthServiceWithProvider(stateRepo, accountRepo, provider)

	ch, err := svc.StartPasskeyRegistration(context.Background(), application.StartPasskeyRegistrationInput{
		RecoverySession: recoverySession.ID(),
		ClientIP:        "127.0.0.1",
	})
	if err != nil {
		t.Fatalf("StartPasskeyRegistration: %v", err)
	}
	if ch.Challenge != "ck-reg-001" {
		t.Errorf("expected Challenge=ck-reg-001, got %q", ch.Challenge)
	}
	if len(ch.WebAuthnOptions) == 0 {
		t.Error("expected non-empty WebAuthnOptions")
	}
}

// [AUTH-BE-WA-5] RegisterPasskey with provider: FinishRegistration resolves handle and stores credential.
func TestRegisterPasskeyWithProvider(t *testing.T) {
	t.Parallel()
	stateRepo := newStubStateRepo(fixedClock())
	_, accountRepo := accountWithOnePasskey(t)

	accountID := "01ARZ3NDEKTSV4RRFFQ69G5FAW"
	recoverySession, err := domain.NewRecoverySession("01ARZ3NDEKTSV4RRFFQ69G5FAV", accountID, time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("NewRecoverySession: %v", err)
	}
	stateRepo.recoverySessions[recoverySession.ID()] = recoverySession

	provider := &mockWebAuthnProvider{
		finishRegistrationFn: func(_ context.Context, _ string, _ string, cred application.WebAuthnAttestationCredentialDTO) (string, domain.WebAuthnCredentialData, error) {
			return "new-handle-webauthn", domain.NewWebAuthnCredentialData([]byte("pubkey"), 0, make([]byte, 16), false, false, nil), nil
		},
	}
	svc := newTestAuthServiceWithProvider(stateRepo, accountRepo, provider)

	result, err := svc.RegisterPasskey(context.Background(), application.RegisterPasskeyInput{
		RecoverySession: recoverySession.ID(),
		Credential: application.WebAuthnAttestationCredentialDTO{
			ID:    "new-handle-webauthn",
			RawID: "new-handle-webauthn",
			Type:  "public-key",
			Response: application.WebAuthnAttestationResponseDTO{
				ClientDataJSON:    "clientdata",
				AttestationObject: "attestation",
			},
		},
		ClientIP: "127.0.0.1",
	})
	if err != nil {
		t.Fatalf("RegisterPasskey: %v", err)
	}
	if result.AccountID != accountID {
		t.Errorf("expected AccountID=%q, got %q", accountID, result.AccountID)
	}
}

// [AUTH-BE-WA-6] StartAddPasskey with provider: returns WebAuthnOptions.
func TestStartAddPasskeyWithProvider(t *testing.T) {
	t.Parallel()
	stateRepo := newStubStateRepo(fixedClock())
	_, accountRepo := accountWithOnePasskey(t)

	provider := &mockWebAuthnProvider{
		beginRegistrationFn: func(_ context.Context, _ string) (string, []byte, error) {
			return "ck-add-001", []byte(`{"publicKey":{"challenge":"ck-add-001"}}`), nil
		},
	}
	svc := newTestAuthServiceWithProvider(stateRepo, accountRepo, provider)

	ch, err := svc.StartAddPasskey(context.Background(), "01ARZ3NDEKTSV4RRFFQ69G5FAW")
	if err != nil {
		t.Fatalf("StartAddPasskey: %v", err)
	}
	if ch.Challenge != "ck-add-001" {
		t.Errorf("expected Challenge=ck-add-001, got %q", ch.Challenge)
	}
}

// [AUTH-BE-WA-7] FinishAddPasskey with provider: adds new passkey without challenge lookup.
func TestFinishAddPasskeyWithProvider(t *testing.T) {
	t.Parallel()
	stateRepo := newStubStateRepo(fixedClock())
	_, accountRepo := accountWithOnePasskey(t)

	provider := &mockWebAuthnProvider{
		finishRegistrationFn: func(_ context.Context, _ string, _ string, _ application.WebAuthnAttestationCredentialDTO) (string, domain.WebAuthnCredentialData, error) {
			return "handle-added", domain.ZeroWebAuthnCredentialData(), nil
		},
	}
	svc := newTestAuthServiceWithProvider(stateRepo, accountRepo, provider)

	creds, err := svc.FinishAddPasskey(context.Background(), "01ARZ3NDEKTSV4RRFFQ69G5FAW", application.WebAuthnAttestationCredentialDTO{
		ID:    "handle-added",
		RawID: "handle-added",
		Type:  "public-key",
		Response: application.WebAuthnAttestationResponseDTO{
			ClientDataJSON:    "clientdata",
			AttestationObject: "attestation",
		},
	})
	if err != nil {
		t.Fatalf("FinishAddPasskey: %v", err)
	}
	// 元の 1 件 + 新規 1 件 = 2 件
	if len(creds) != 2 {
		t.Errorf("expected 2 credentials, got %d", len(creds))
	}
}

// ─── Task 4.7: DeletePasskey – last passkey cannot be deleted ─────────────────

// [AUTH-BE-4.7] DeletePasskey が最終 1 件のとき ErrLastPasskeyCannotBeDeleted を返す
func TestDeletePasskeyRejectsLastCredential(t *testing.T) {
	t.Parallel()
	_, accountRepo := accountWithOnePasskey(t)
	svc := newTestAuthService(newStubStateRepo(fixedClock()), accountRepo)

	accountID := "01ARZ3NDEKTSV4RRFFQ69G5FAW"
	passkeyID := "01ARZ3NDEKTSV4RRFFQ69G5FAV" // #nosec G101

	err := svc.DeletePasskey(context.Background(), accountID, passkeyID)
	if err == nil {
		t.Fatal("expected ErrLastPasskeyCannotBeDeleted, got nil")
	}
	if err != application.ErrLastPasskeyCannotBeDeleted {
		t.Fatalf("expected ErrLastPasskeyCannotBeDeleted, got %v", err)
	}
}

// ─── Task 4.8: DeletePasskey – other account's credential is rejected ─────────

// [AUTH-BE-4.8] DeletePasskey で account A の accountID を使って account B の credentialID を
// 指定した場合、適切なエラー（domain.ErrAuthAccountNotFound）を返す（cross-account deletion rejection）。
func TestDeletePasskeyRejectsCrossAccountCredential(t *testing.T) {
	t.Parallel()

	// account A: 2 件のパスキーを持つ
	accountA, _ := accountWithTwoPasskeys(t)

	// account B: 別のアカウント（別の credentialID を持つ）
	accountBCred, err := domain.NewPasskeyCredential(
		"01ARZ3NDEKTSV4RRFFQ69G5FB1", // account B の credentialID
		"01ARZ3NDEKTSV4RRFFQ69G5FB2", // account B の accountID
		"other@example.com",
		"handle-b",
		time.Time{},
	)
	if err != nil {
		t.Fatalf("NewPasskeyCredential accountB: %v", err)
	}
	accountB, err := domain.NewAuthAccountWithCredentials(
		"01ARZ3NDEKTSV4RRFFQ69G5FB2",
		"other@example.com",
		"other@example.com",
		[]domain.PasskeyCredential{accountBCred},
	)
	if err != nil {
		t.Fatalf("NewAuthAccountWithCredentials accountB: %v", err)
	}

	// 両アカウントを含む repo
	accountRepo := &stubAccountRepo{accounts: map[string]domain.AuthAccount{
		accountA.AccountID(): accountA,
		accountB.AccountID(): accountB,
	}}
	svc := newTestAuthService(newStubStateRepo(fixedClock()), accountRepo)

	// account A の accountID で account B の credentialID を削除しようとする
	err = svc.DeletePasskey(context.Background(), accountA.AccountID(), accountBCred.ID())
	if err == nil {
		t.Fatal("expected error for cross-account credential deletion, got nil")
	}
	// account A には account B の credentialID が存在しないので ErrAuthAccountNotFound を返す
	if err != domain.ErrAuthAccountNotFound {
		t.Fatalf("expected domain.ErrAuthAccountNotFound, got %v", err)
	}
}

// ─── Task 4.12: StartAddPasskeyByOtp – expired/missing OTP ───────────────────

// [AUTH-BE-4.12] StartAddPasskeyByOtp で存在しない OTP を指定すると ErrOtpExpiredOrConsumed を返す
func TestStartAddPasskeyByOtpRejectsMissingOtp(t *testing.T) {
	t.Parallel()
	stateRepo := newStubStateRepo(fixedClock())
	svc := newTestAuthService(stateRepo, newStubAccountRepoWithAccount(emptyAuthAccount()))

	_, err := svc.StartAddPasskeyByOtp(context.Background(), "test@example.com", "999999", "127.0.0.1")
	if err == nil {
		t.Fatal("expected ErrOtpExpiredOrConsumed, got nil")
	}
	if err != application.ErrOtpExpiredOrConsumed {
		t.Fatalf("expected ErrOtpExpiredOrConsumed, got %v", err)
	}
}

// ─── Task 4.13: FinishAddPasskeyByOtp – consumed OTP ─────────────────────────

// [AUTH-BE-4.13] FinishAddPasskeyByOtp で消費済み OTP を指定すると ErrOtpExpiredOrConsumed を返す。
// 1 回目の FinishAddPasskeyByOtp を成功させて OTP が消費されることを確認し、
// 2 回目の呼び出しで ErrOtpExpiredOrConsumed が返ることを確認する。
func TestFinishAddPasskeyByOtpRejectsConsumedOtp(t *testing.T) {
	t.Parallel()
	stateRepo := newStubStateRepo(fixedClock())
	_, accountRepo := accountWithOnePasskey(t)
	svc := newTestAuthServiceWithProvider(stateRepo, accountRepo, &mockWebAuthnProvider{})

	accountID := "01ARZ3NDEKTSV4RRFFQ69G5FAW"

	// Step 1: OTP を発行する
	otp, err := svc.IssuePasskeyOtp(context.Background(), accountID, "01ARZ3NDEKTSV4RRFFQ69G5FB1")
	if err != nil {
		t.Fatalf("IssuePasskeyOtp: %v", err)
	}

	// Step 2: StartAddPasskeyByOtp で OTP を使ってチャレンジを取得する（OTP は消費されない = GetPasskeyOtp）
	passkeyChallenge, err := svc.StartAddPasskeyByOtp(context.Background(), "user@example.com", otp, "127.0.0.1")
	if err != nil {
		t.Fatalf("StartAddPasskeyByOtp: %v", err)
	}

	// Step 3: 1 回目の FinishAddPasskeyByOtp を成功させる
	// legacy path: ID に handle、Response.ClientDataJSON に challenge を格納
	credential := application.WebAuthnAttestationCredentialDTO{
		ID:    "handle-new",
		RawID: "handle-new",
		Type:  "public-key",
		Response: application.WebAuthnAttestationResponseDTO{
			ClientDataJSON: passkeyChallenge.Challenge,
		},
	}
	if err := svc.FinishAddPasskeyByOtp(context.Background(), "user@example.com", otp, credential, "127.0.0.1"); err != nil {
		t.Fatalf("FinishAddPasskeyByOtp (1st call) should succeed, got: %v", err)
	}

	// Step 4: 2 回目の FinishAddPasskeyByOtp: OTP は既に 1 回目で消費されている
	if err := svc.FinishAddPasskeyByOtp(context.Background(), "user@example.com", otp, credential, "127.0.0.1"); err == nil {
		t.Fatal("expected ErrOtpExpiredOrConsumed on 2nd call, got nil")
	} else if err != application.ErrOtpExpiredOrConsumed {
		t.Fatalf("expected ErrOtpExpiredOrConsumed, got %v", err)
	}
}

// ─── Regression: challenge mismatch protects account B's session ─────────────

// [AUTH-BE-REG-1] FinishAddPasskeyByOtp に別 OTP セッションの challengeValue を渡した場合、
// ErrBadRequest を返す（差し替え攻撃への回帰テスト）。
// 仕様: otpA の OTP/challenge は消費されるが、otpB 側のセッションは一切影響を受けない。
func TestFinishAddPasskeyByOtpRejectsChallengeMismatch(t *testing.T) {
	t.Parallel()
	stateRepo := newStubStateRepo(fixedClock())

	// アカウント A と B を個別に作成して repo に入れる
	accountA, err := domain.NewAuthAccount("01ARZ3NDEKTSV4RRFFQ69G5FAW", "user@example.com", "user@example.com", "01ARZ3NDEKTSV4RRFFQ69G5FAV", "handle-one")
	if err != nil {
		t.Fatalf("NewAuthAccount A: %v", err)
	}
	accountB, err := domain.NewAuthAccount("01ARZ3NDEKTSV4RRFFQ69G5FAV", "other@example.com", "other@example.com", "01ARZ3NDEKTSV4RRFFQ69G5FB0", "handle-b")
	if err != nil {
		t.Fatalf("NewAuthAccount B: %v", err)
	}
	accountRepo := &stubAccountRepo{accounts: map[string]domain.AuthAccount{
		accountA.AccountID(): accountA,
		accountB.AccountID(): accountB,
	}}

	// アカウント A と B でそれぞれ異なる challengeKey を返すように beginRegistrationFn をカスタマイズする。
	// これにより challengeA.Challenge != challengeB.Challenge となり、
	// provider の FinishRegistration で challenge mismatch を検出できる。
	callCount := 0
	provider := &mockWebAuthnProvider{
		beginRegistrationFn: func(_ context.Context, _ string) (string, []byte, error) {
			callCount++
			key := fmt.Sprintf("challenge-key-reg-%d", callCount)
			return key, []byte(`{"challenge":"` + key + `"}`), nil
		},
		finishRegistrationFn: func(_ context.Context, challengeKey string, _ string, credential application.WebAuthnAttestationCredentialDTO) (string, domain.WebAuthnCredentialData, error) {
			// storedChallengeKey と credential の ClientDataJSON が一致しない場合は mismatch エラー
			if challengeKey != "" && credential.Response.ClientDataJSON != challengeKey {
				return "", domain.ZeroWebAuthnCredentialData(), fmt.Errorf("challenge mismatch: stored=%q, got=%q", challengeKey, credential.Response.ClientDataJSON)
			}
			return credential.ID, domain.ZeroWebAuthnCredentialData(), nil
		},
	}
	svc := newTestAuthServiceWithProvider(stateRepo, accountRepo, provider)

	accountID := "01ARZ3NDEKTSV4RRFFQ69G5FAW"

	// Step 1: account A の OTP を発行し、チャレンジ C1 を取得する
	otpA, err := svc.IssuePasskeyOtp(context.Background(), accountID, "01ARZ3NDEKTSV4RRFFQ69G5FB1")
	if err != nil {
		t.Fatalf("IssuePasskeyOtp A: %v", err)
	}
	challengeA, err := svc.StartAddPasskeyByOtp(context.Background(), "user@example.com", otpA, "127.0.0.1")
	if err != nil {
		t.Fatalf("StartAddPasskeyByOtp A: %v", err)
	}
	_ = challengeA // challengeA は使用しない（チャレンジが生成されたことの確認のみ）

	// Step 2: 別のアカウントで OTP を発行し、チャレンジ C2 を取得する
	otherAccountID := "01ARZ3NDEKTSV4RRFFQ69G5FAV"
	otpB, err := svc.IssuePasskeyOtp(context.Background(), otherAccountID, "01ARZ3NDEKTSV4RRFFQ69G5FB2")
	if err != nil {
		t.Fatalf("IssuePasskeyOtp B: %v", err)
	}
	challengeB, err := svc.StartAddPasskeyByOtp(context.Background(), "other@example.com", otpB, "127.0.0.1")
	if err != nil {
		t.Fatalf("StartAddPasskeyByOtp B: %v", err)
	}

	// mismatch 前に handoff の数を記録する（otpA と otpB の 2 件が存在する）
	handoffCountBefore := len(stateRepo.handoffs)
	if handoffCountBefore != 2 {
		t.Fatalf("expected 2 handoff entries before mismatch, got %d", handoffCountBefore)
	}

	// Step 3: account A の OTP を使い、C2（別セッションの challenge）を credential に埋め込んで送る
	// → provider の FinishRegistration で storedChallengeKey(C1) != credential.ClientDataJSON(C2) なので ErrBadRequest
	// → otpA の handoff は消費されるが、otpB の handoff は残る
	mismatchCredential := application.WebAuthnAttestationCredentialDTO{
		ID:    "handle-new",
		RawID: "handle-new",
		Type:  "public-key",
		Response: application.WebAuthnAttestationResponseDTO{
			ClientDataJSON: challengeB.Challenge, // mismatch: A の OTP に B の challenge を渡す
		},
	}
	if err := svc.FinishAddPasskeyByOtp(context.Background(), "user@example.com", otpA, mismatchCredential, "127.0.0.1"); err == nil {
		t.Fatal("expected ErrBadRequest for challenge mismatch, got nil")
	} else if err != application.ErrBadRequest {
		t.Fatalf("expected ErrBadRequest, got %v", err)
	}

	// Step 4: otpA 側の handoff は消費されており、otpB 側は一切影響を受けていない（差し替え攻撃の影響が波及しないことの確認）
	handoffCountAfter := len(stateRepo.handoffs)
	if handoffCountAfter != 1 {
		t.Fatalf("expected 1 handoff remaining after A's mismatch attempt (B only), got %d", handoffCountAfter)
	}
}

// ─── WebAuthn failure lock / error classification tests ──────────────────────

// [AUTH-BE-WA-8] FinishPasskeyAuthentication with provider: FinishLogin error increments failure lock.
func TestFinishPasskeyAuthenticationWithProviderErrorIncrementsFailureLock(t *testing.T) {
	t.Parallel()
	_, accountRepo := accountWithOnePasskey(t)
	stateRepo := newStubStateRepo(fixedClock())

	provider := &mockWebAuthnProvider{
		finishLoginFn: func(_ context.Context, _ string, _ application.WebAuthnAssertionCredentialDTO, _ func(context.Context, string) (domain.WebAuthnStoredCredential, error)) (string, uint32, bool, bool, error) {
			return "", 0, false, false, fmt.Errorf("invalid signature")
		},
	}
	svc := newTestAuthServiceWithProvider(stateRepo, accountRepo, provider)

	input := application.FinishPasskeyAuthenticationInput{
		Credential: application.WebAuthnAssertionCredentialDTO{
			ID:       "bad-handle",
			Response: application.WebAuthnAssertionResponseDTO{ClientDataJSON: "x"},
		},
		ClientIP: "10.0.0.1",
	}
	_, err := svc.FinishPasskeyAuthentication(context.Background(), input)
	if err != application.ErrBadRequest {
		t.Fatalf("expected ErrBadRequest, got %v", err)
	}

	// failure window カウンタが増加していることを確認
	// failureLockKey("bad-handle", "10.0.0.1") = "lock:bad-handle:10.0.0.1"
	// failureWindowKey(lockKey) = "failures:lock:bad-handle:10.0.0.1"
	lockWindowKey := "failures:lock:bad-handle:10.0.0.1"
	if stateRepo.counters[lockWindowKey] < 1 {
		t.Errorf("expected failure counter to be incremented, got %d", stateRepo.counters[lockWindowKey])
	}
}

// [AUTH-BE-WA-9] FinishPasskeyAuthentication with provider: ErrAuthStoreUnavailable → ErrInternalError, failure counter NOT incremented.
func TestFinishPasskeyAuthenticationWithProviderDBOutageReturnsInternalError(t *testing.T) {
	t.Parallel()
	_, accountRepo := accountWithOnePasskey(t)
	stateRepo := newStubStateRepo(fixedClock())

	provider := &mockWebAuthnProvider{
		finishLoginFn: func(_ context.Context, _ string, _ application.WebAuthnAssertionCredentialDTO, _ func(context.Context, string) (domain.WebAuthnStoredCredential, error)) (string, uint32, bool, bool, error) {
			return "", 0, false, false, domain.ErrAuthStoreUnavailable
		},
	}
	svc := newTestAuthServiceWithProvider(stateRepo, accountRepo, provider)

	_, err := svc.FinishPasskeyAuthentication(context.Background(), application.FinishPasskeyAuthenticationInput{
		Credential: application.WebAuthnAssertionCredentialDTO{
			ID:       "some-handle",
			Response: application.WebAuthnAssertionResponseDTO{ClientDataJSON: "x"},
		},
		ClientIP: "10.0.0.1",
	})
	if err != application.ErrInternalError {
		t.Fatalf("expected ErrInternalError for DB outage, got %v", err)
	}
	// DB 障害時は failure counter を加算しない（正当ユーザーを誤ってロックしないため）
	lockWindowKey := "failures:lock:some-handle:10.0.0.1"
	if stateRepo.counters[lockWindowKey] != 0 {
		t.Errorf("expected failure counter NOT to be incremented on DB outage, got %d", stateRepo.counters[lockWindowKey])
	}
}

// [AUTH-BE-WA-10] FinishPasskeyAuthentication with provider: signCountUpdated=false skips UpdateWebAuthnCredentialState.
func TestFinishPasskeyAuthenticationSkipsStateUpdateWhenSignCountNotObtained(t *testing.T) {
	t.Parallel()
	account, _ := accountWithOnePasskey(t)
	accountRepo := newTrackingAccountRepo(account)
	stateRepo := newStubStateRepo(fixedClock())

	provider := &mockWebAuthnProvider{
		finishLoginFn: func(_ context.Context, _ string, _ application.WebAuthnAssertionCredentialDTO, _ func(context.Context, string) (domain.WebAuthnStoredCredential, error)) (string, uint32, bool, bool, error) {
			// signCountUpdated = false: updatedCred が nil だったケース
			return "handle-one", 0, false, false, nil
		},
	}
	svc := newTestAuthServiceWithProvider(stateRepo, accountRepo, provider)

	result, err := svc.FinishPasskeyAuthentication(context.Background(), application.FinishPasskeyAuthenticationInput{
		Credential: application.WebAuthnAssertionCredentialDTO{
			ID:       "handle-one",
			Response: application.WebAuthnAssertionResponseDTO{ClientDataJSON: "clientdata"},
		},
		ClientIP: "127.0.0.1",
	})
	if err != nil {
		t.Fatalf("FinishPasskeyAuthentication: %v", err)
	}
	if result.AccountID != account.AccountID() {
		t.Errorf("expected AccountID=%q, got %q", account.AccountID(), result.AccountID)
	}
	// signCountUpdated=false なので UpdateWebAuthnCredentialState は呼ばれないはず
	if accountRepo.updateStateCallCount != 0 {
		t.Errorf("expected UpdateWebAuthnCredentialState NOT to be called when signCountUpdated=false, got %d calls", accountRepo.updateStateCallCount)
	}
}

// [AUTH-BE-S028] UV-less login assertion は拒否される
func TestFinishPasskeyAuthenticationRejectsUVLessAssertion(t *testing.T) {
	t.Parallel()
	_, accountRepo := accountWithOnePasskey(t)
	stateRepo := newStubStateRepo(fixedClock())

	provider := &mockWebAuthnProvider{
		finishLoginFn: func(_ context.Context, _ string, _ application.WebAuthnAssertionCredentialDTO, _ func(context.Context, string) (domain.WebAuthnStoredCredential, error)) (string, uint32, bool, bool, error) {
			// UV 不足をシミュレート: エラーを返す
			return "", 0, false, false, fmt.Errorf("webauthn: user verification required")
		},
	}
	svc := newTestAuthServiceWithProvider(stateRepo, accountRepo, provider)

	_, err := svc.FinishPasskeyAuthentication(context.Background(), application.FinishPasskeyAuthenticationInput{
		Credential: application.WebAuthnAssertionCredentialDTO{
			ID:       "handle-one",
			Response: application.WebAuthnAssertionResponseDTO{ClientDataJSON: "x"},
		},
		ClientIP: "127.0.0.1",
	})
	if err != application.ErrBadRequest {
		t.Fatalf("expected ErrBadRequest for UV-less assertion, got %v", err)
	}

	// session が発行されていないことを確認
	if len(stateRepo.sessions) != 0 {
		t.Fatalf("expected no session issued for UV-less assertion, got %d sessions", len(stateRepo.sessions))
	}
}

// [AUTH-BE-S029] UV-less new-device registration は拒否される
func TestRegisterPasskeyRejectsUVLessAttestation(t *testing.T) {
	t.Parallel()
	stateRepo := newStubStateRepo(fixedClock())
	_, accountRepo := accountWithOnePasskey(t)

	accountID := "01ARZ3NDEKTSV4RRFFQ69G5FAW"
	recoverySession, err := domain.NewRecoverySession("01ARZ3NDEKTSV4RRFFQ69G5FAV", accountID, time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("NewRecoverySession: %v", err)
	}
	stateRepo.recoverySessions[recoverySession.ID()] = recoverySession

	provider := &mockWebAuthnProvider{
		finishRegistrationFn: func(_ context.Context, _ string, _ string, _ application.WebAuthnAttestationCredentialDTO) (string, domain.WebAuthnCredentialData, error) {
			// UV 不足をシミュレート: エラーを返す
			return "", domain.ZeroWebAuthnCredentialData(), fmt.Errorf("webauthn: user verification required")
		},
	}
	svc := newTestAuthServiceWithProvider(stateRepo, accountRepo, provider)

	_, err = svc.RegisterPasskey(context.Background(), application.RegisterPasskeyInput{
		RecoverySession: recoverySession.ID(),
		Credential: application.WebAuthnAttestationCredentialDTO{
			ID:    "new-handle",
			RawID: "new-handle",
			Type:  "public-key",
			Response: application.WebAuthnAttestationResponseDTO{
				ClientDataJSON:    "clientdata",
				AttestationObject: "attestation",
			},
		},
		ClientIP: "127.0.0.1",
	})
	if err != application.ErrBadRequest {
		t.Fatalf("expected ErrBadRequest for UV-less attestation, got %v", err)
	}

	// credential が追加されていないことを確認
	updatedAccount, _ := accountRepo.FindByID(context.Background(), accountID)
	if len(updatedAccount.Credentials()) != 1 {
		t.Fatalf("expected 1 credential (none added), got %d", len(updatedAccount.Credentials()))
	}
}

// [AUTH-BE-S030] recovery token consume はアトミックであり、2 回目は拒否される
func TestRecoveryTokenAtomicConsumeRejectsReplay(t *testing.T) {
	t.Parallel()
	stateRepo := newStubStateRepo(fixedClock())
	_, accountRepo := accountWithOnePasskey(t)
	svc := newTestAuthService(stateRepo, accountRepo)

	// Recovery token を発行
	account, _ := accountRepo.FindByEmail(context.Background(), "user@example.com")
	secret := "opaque-01ARZ3NDEKTSV4RRFFQ69G5FAV" // #nosec G101
	token, err := domain.NewRecoveryToken("01ARZ3NDEKTSV4RRFFQ69G5FAV", account.AccountID(), secret, fixedClock()().Add(30*time.Minute))
	if err != nil {
		t.Fatalf("NewRecoveryToken: %v", err)
	}
	stateRepo.recoveryTokens[secret] = token

	// 1 回目: 成功
	result1, err := svc.ConsumeRecoveryToken(context.Background(), application.ConsumeRecoveryTokenInput{Token: secret, ClientIP: "127.0.0.1"})
	if err != nil {
		t.Fatalf("first consume should succeed, got: %v", err)
	}
	if result1.RecoverySessionID == "" {
		t.Fatal("expected recovery session on first consume")
	}

	// 2 回目: 拒否（アトミック consume により token が既に消費済み）
	_, err = svc.ConsumeRecoveryToken(context.Background(), application.ConsumeRecoveryTokenInput{Token: secret, ClientIP: "127.0.0.1"})
	if err != application.ErrBadRequest {
		t.Fatalf("expected ErrBadRequest for replay consume, got %v", err)
	}
}

// [AUTH-BE-S031] identifier rotation では passkey start budget を回避できない
func TestIdentifierRotationCannotBypassPasskeyStartBudget(t *testing.T) {
	t.Parallel()
	_, accountRepo := accountWithOnePasskey(t)
	stateRepo := newStubStateRepo(fixedClock())
	svc := newTestAuthServiceWithProvider(stateRepo, accountRepo, &mockWebAuthnProvider{})

	clientIP := "192.0.2.10"
	// IP bucket limit (5) を超えるまで異なる identifier で試行する
	for i := 0; i < 6; i++ {
		identifier := fmt.Sprintf("user%d@example.com", i)
		_, err := svc.StartPasskeyAuthentication(context.Background(), application.StartPasskeyAuthenticationInput{
			Identifier: identifier,
			ClientIP:   clientIP,
		})
		if i < 5 && err != nil {
			t.Fatalf("expected success for attempt %d, got %v", i, err)
		}
		if i >= 5 && err != application.ErrBadRequest {
			t.Fatalf("expected ErrBadRequest for throttled attempt %d, got %v", i, err)
		}
	}
}

// [AUTH-BE-S032] OTP brute force は throttle により account takeover 前に抑制される
func TestOtpBruteForceTriggersThrottle(t *testing.T) {
	t.Parallel()
	stateRepo := newStubStateRepo(fixedClock())
	svc := newTestAuthService(stateRepo, newStubAccountRepoWithAccount(emptyAuthAccount()))

	clientIP := "192.0.2.10"
	// IP/email bucket limit (5) を超えるまで無効な OTP を試行する
	for i := 0; i < 7; i++ {
		_, err := svc.StartAddPasskeyByOtp(context.Background(), "user@example.com", fmt.Sprintf("%06d", i), clientIP)
		if i < 5 {
			// スロットル前は ErrOtpExpiredOrConsumed（generic）
			if err != application.ErrOtpExpiredOrConsumed {
				t.Fatalf("expected ErrOtpExpiredOrConsumed for attempt %d, got %v", i, err)
			}
		} else {
			// スロットル後は ErrBadRequest（generic）
			if err != application.ErrBadRequest {
				t.Fatalf("expected ErrBadRequest for throttled attempt %d, got %v", i, err)
			}
		}
	}
}

// hashSecretForTest は AuthService.hashSecret と同じロジックで HMAC-SHA256 ハッシュを計算する。
func hashSecretForTest(secret string) string {
	mac := hmac.New(sha256.New, []byte("test-pepper"))
	mac.Write([]byte(secret))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

// [AUTH-BE-S034] 同じ OTP 値は別アカウントの handoff state を上書きしない
func TestSameOtpValueIsolatedAcrossAccounts(t *testing.T) {
	t.Parallel()
	stateRepo := newStubStateRepo(fixedClock())

	// アカウント A と B を作成
	accountA, err := domain.NewAuthAccount("01ARZ3NDEKTSV4RRFFQ69G5FAW", "user@example.com", "user@example.com", "01ARZ3NDEKTSV4RRFFQ69G5FAV", "handle-a")
	if err != nil {
		t.Fatalf("NewAuthAccount A: %v", err)
	}
	accountB, err := domain.NewAuthAccount("01ARZ3NDEKTSV4RRFFQ69G5FAV", "other@example.com", "other@example.com", "01ARZ3NDEKTSV4RRFFQ69G5FB0", "handle-b")
	if err != nil {
		t.Fatalf("NewAuthAccount B: %v", err)
	}
	accountRepo := &stubAccountRepo{accounts: map[string]domain.AuthAccount{
		accountA.AccountID(): accountA,
		accountB.AccountID(): accountB,
	}}
	svc := newTestAuthServiceWithProvider(stateRepo, accountRepo, &mockWebAuthnProvider{})

	// 同じ OTP 値（"123456"）で両アカウントに handoff を作成
	otpValue := "123456"
	emailHashA := hashSecretForTest("user@example.com")
	emailHashB := hashSecretForTest("other@example.com")
	otpHash := hashSecretForTest(otpValue)

	handoffA, _ := domain.NewDeviceLoginHandoff("01ARZ3NDEKTSV4RRFFQ69G5FC1", accountA.AccountID(), "01ARZ3NDEKTSV4RRFFQ69G5FAY", emailHashA, otpHash, fixedClock()().Add(5*time.Minute))
	handoffB, _ := domain.NewDeviceLoginHandoff("01ARZ3NDEKTSV4RRFFQ69G5FC2", accountB.AccountID(), "01ARZ3NDEKTSV4RRFFQ69G5FAZ", emailHashB, otpHash, fixedClock()().Add(5*time.Minute))
	stateRepo.handoffs[handoffA.ID()] = handoffA
	stateRepo.handoffs[handoffB.ID()] = handoffB

	// アカウント A の email + OTP で検索 → A の handoff が返る（challenge 発行成功を確認）
	_, err = svc.StartAddPasskeyByOtp(context.Background(), "user@example.com", otpValue, "127.0.0.1")
	if err != nil {
		t.Fatalf("StartAddPasskeyByOtp A: %v", err)
	}

	// アカウント B の email + OTP で検索 → B の handoff が返る（challenge 発行成功を確認）
	_, err = svc.StartAddPasskeyByOtp(context.Background(), "other@example.com", otpValue, "127.0.0.2")
	if err != nil {
		t.Fatalf("StartAddPasskeyByOtp B: %v", err)
	}

	// 両方の handoff が独立して存在することを確認（一方が上書きされていない）
	if len(stateRepo.handoffs) != 2 {
		t.Fatalf("expected 2 independent handoffs, got %d", len(stateRepo.handoffs))
	}
}

// [AUTH-BE-S035] handoff finish はアトミックであり、2 回目は拒否される
func TestHandoffFinishAtomicRejectsReplay(t *testing.T) {
	t.Parallel()
	stateRepo := newStubStateRepo(fixedClock())
	_, accountRepo := accountWithOnePasskey(t)
	svc := newTestAuthServiceWithProvider(stateRepo, accountRepo, &mockWebAuthnProvider{})

	accountID := "01ARZ3NDEKTSV4RRFFQ69G5FAW"

	// OTP を発行
	otp, err := svc.IssuePasskeyOtp(context.Background(), accountID, "01ARZ3NDEKTSV4RRFFQ69G5FB1")
	if err != nil {
		t.Fatalf("IssuePasskeyOtp: %v", err)
	}

	// StartAddPasskeyByOtp でチャレンジを取得
	challenge, err := svc.StartAddPasskeyByOtp(context.Background(), "user@example.com", otp, "127.0.0.1")
	if err != nil {
		t.Fatalf("StartAddPasskeyByOtp: %v", err)
	}

	// 1 回目: 成功
	credential := application.WebAuthnAttestationCredentialDTO{
		ID:    "handle-new",
		RawID: "handle-new",
		Type:  "public-key",
		Response: application.WebAuthnAttestationResponseDTO{
			ClientDataJSON: challenge.Challenge,
		},
	}
	if err := svc.FinishAddPasskeyByOtp(context.Background(), "user@example.com", otp, credential, "127.0.0.1"); err != nil {
		t.Fatalf("first finish should succeed, got: %v", err)
	}

	// 2 回目: 拒否（handoff が既に消費済み）
	if err := svc.FinishAddPasskeyByOtp(context.Background(), "user@example.com", otp, credential, "127.0.0.1"); err == nil {
		t.Fatal("expected error for replay finish, got nil")
	} else if err != application.ErrOtpExpiredOrConsumed {
		t.Fatalf("expected ErrOtpExpiredOrConsumed for replay finish, got %v", err)
	}
}

// [AUTH-BE-S035-AUDIT] FinishAddPasskeyByOtp 成功時に audit event が emit される
func TestFinishAddPasskeyByOtpEmitsAuditEvent(t *testing.T) {
	t.Parallel()
	stateRepo := newStubStateRepo(fixedClock())
	_, accountRepo := accountWithOnePasskey(t)
	svc := newTestAuthServiceWithProvider(stateRepo, accountRepo, &mockWebAuthnProvider{})

	accountID := "01ARZ3NDEKTSV4RRFFQ69G5FAW"

	otp, err := svc.IssuePasskeyOtp(context.Background(), accountID, "01ARZ3NDEKTSV4RRFFQ69G5FB1")
	if err != nil {
		t.Fatalf("IssuePasskeyOtp: %v", err)
	}

	challenge, err := svc.StartAddPasskeyByOtp(context.Background(), "user@example.com", otp, "127.0.0.1")
	if err != nil {
		t.Fatalf("StartAddPasskeyByOtp: %v", err)
	}

	notifier := &stubAuditNotifier{}
	svc.UseAuditNotifier(notifier)

	credential := application.WebAuthnAttestationCredentialDTO{
		ID:    "handle-new",
		RawID: "handle-new",
		Type:  "public-key",
		Response: application.WebAuthnAttestationResponseDTO{
			ClientDataJSON: challenge.Challenge,
		},
	}
	if err := svc.FinishAddPasskeyByOtp(context.Background(), "user@example.com", otp, credential, "127.0.0.1"); err != nil {
		t.Fatalf("FinishAddPasskeyByOtp: %v", err)
	}

	if notifier.emitCount != 1 {
		t.Fatalf("expected 1 audit event, got %d", notifier.emitCount)
	}
	if notifier.lastAccountID != accountID {
		t.Errorf("expected AccountID=%q, got %q", accountID, notifier.lastAccountID)
	}
	if notifier.lastPasskeyID == "" {
		t.Error("expected non-empty PasskeyID")
	}
	if notifier.lastRequestID == "" {
		t.Error("expected non-empty RequestID")
	}
}
