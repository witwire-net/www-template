package application_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	application "www-template/packages/backend/internal/application"
	domain "www-template/packages/backend/internal/domain"

	"www-template/packages/backend/internal/platform/config"
	"www-template/packages/backend/internal/platform/id"
)

// ─── stubs ───────────────────────────────────────────────────────────────────

type stubStateRepo struct {
	challenges       map[string]domain.AuthChallenge
	recoveryTokens   map[string]domain.RecoveryToken
	recoverySessions map[string]domain.RecoverySession
	counters         map[string]int
	locks            map[string]time.Time
	clock            func() time.Time
}

// stubRefreshTokenStore はテスト用のインメモリ RefreshTokenStore。
type stubRefreshTokenStore struct {
	data      map[string]application.RefreshTokenRecord
	saveFails bool
}

func newStubRefreshTokenStore() *stubRefreshTokenStore {
	return &stubRefreshTokenStore{data: make(map[string]application.RefreshTokenRecord)}
}

func (s *stubRefreshTokenStore) Save(_ context.Context, hash string, record application.RefreshTokenRecord, _ time.Duration) error {
	if s.saveFails {
		return domain.ErrAuthStoreUnavailable
	}
	s.data[hash] = record
	return nil
}

func (s *stubRefreshTokenStore) Consume(_ context.Context, hash string) (application.RefreshTokenRecord, error) {
	record, ok := s.data[hash]
	if !ok {
		return application.RefreshTokenRecord{}, domain.ErrSessionNotFound
	}
	delete(s.data, hash)
	return record, nil
}

func (s *stubRefreshTokenStore) GetConsumed(_ context.Context, hash string) (application.RefreshTokenRecord, error) {
	_, ok := s.data[hash]
	if ok {
		return application.RefreshTokenRecord{}, domain.ErrSessionNotFound
	}
	return application.RefreshTokenRecord{}, domain.ErrSessionNotFound
}

func (s *stubRefreshTokenStore) RevokeAllForFingerprint(_ context.Context, _ domain.AccountID, _ string) error {
	return nil
}

func (s *stubRefreshTokenStore) RevokeBySessionID(_ context.Context, _ domain.AccountID, sessionID string) error {
	for h, r := range s.data {
		if r.SessionID == sessionID {
			delete(s.data, h)
		}
	}
	return nil
}

// stubSessionStore はテスト用のインメモリ SessionStore。
type stubSessionStore struct {
	sessions map[string]application.SessionMetadata
}

func newStubSessionStore() *stubSessionStore {
	return &stubSessionStore{sessions: make(map[string]application.SessionMetadata)}
}

func (s *stubSessionStore) SaveSession(_ context.Context, sessionID string, _ domain.AccountID, metadata application.SessionMetadata, _ time.Duration) error {
	s.sessions[sessionID] = metadata
	return nil
}

func (s *stubSessionStore) GetSession(_ context.Context, sessionID string) (application.SessionMetadata, error) {
	sess, ok := s.sessions[sessionID]
	if !ok {
		return application.SessionMetadata{}, domain.ErrSessionNotFound
	}
	return sess, nil
}

func (s *stubSessionStore) ListSessions(_ context.Context, _ domain.AccountID) ([]application.SessionMetadata, error) {
	result := make([]application.SessionMetadata, 0, len(s.sessions))
	for _, v := range s.sessions {
		result = append(result, v)
	}
	return result, nil
}

func (s *stubSessionStore) RevokeSession(_ context.Context, _ domain.AccountID, sessionID string) error {
	delete(s.sessions, sessionID)
	return nil
}

func (s *stubSessionStore) RevokeOthers(_ context.Context, _ domain.AccountID, currentSessionID string) ([]string, error) {
	deleted := make([]string, 0)
	for id := range s.sessions {
		if id != currentSessionID {
			delete(s.sessions, id)
			deleted = append(deleted, id)
		}
	}
	return deleted, nil
}

func (s *stubSessionStore) RevokeAllForAccount(_ context.Context, _ domain.AccountID) error {
	s.sessions = make(map[string]application.SessionMetadata)
	return nil
}

func newStubStateRepo(clock func() time.Time) *stubStateRepo {
	return &stubStateRepo{
		challenges:       map[string]domain.AuthChallenge{},
		recoveryTokens:   map[string]domain.RecoveryToken{},
		recoverySessions: map[string]domain.RecoverySession{},
		counters:         map[string]int{},
		locks:            map[string]time.Time{},
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
func (r *stubStateRepo) IssueRecoveryToken(_ context.Context, t domain.RecoveryToken, _ time.Duration) error {
	r.recoveryTokens[t.Secret()] = t
	return nil
}
func (r *stubStateRepo) SaveRecoveryDeliveryFailure(_ context.Context, _ domain.RecoveryDeliveryFailure, _ time.Duration) error {
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
func (r *stubStateRepo) SaveReauthenticationSession(_ context.Context, _ domain.ReauthenticationSession, _ time.Duration) error {
	return nil
}

func (r *stubStateRepo) ConsumeReauthenticationSession(_ context.Context, _ string) (domain.ReauthenticationSession, error) {
	session, _ := domain.NewReauthenticationSession(
		"01ARZ3NDEKTSV4RRFFQ69G5FAV", testAccountID("01ARZ3NDEKTSV4RRFFQ69G5FAW"), "01ARZ3NDEKTSV4RRFFQ69G5FAX",
		"device-link", "01ARZ3NDEKTSV4RRFFQ69G5FAY", time.Unix(1, 0).UTC(),
	)
	return session, domain.ErrReauthSessionNotFound
}

// stubAccountRepo は AccountAuthRepository の in-memory スタブ。
type stubAccountRepo struct {
	accounts map[string]domain.AccountAuth // keyed by accountID
}

func newStubAccountRepoWithAccount(account domain.AccountAuth) *stubAccountRepo {
	return &stubAccountRepo{accounts: map[string]domain.AccountAuth{account.AccountID().String(): account}}
}

func (r *stubAccountRepo) FindByID(_ context.Context, accountID domain.AccountID) (domain.AccountAuth, error) {
	a, ok := r.accounts[accountID.String()]
	if !ok {
		return emptyAccountAuth(), domain.ErrAccountAuthNotFound
	}
	return a, nil
}
func (r *stubAccountRepo) FindByIdentifier(_ context.Context, identifier string) (domain.AccountAuth, error) {
	for _, a := range r.accounts {
		if a.Identifier() == identifier {
			return a, nil
		}
	}
	return emptyAccountAuth(), domain.ErrAccountAuthNotFound
}
func (r *stubAccountRepo) FindByCredential(_ context.Context, handle string) (domain.AccountAuth, error) {
	for _, a := range r.accounts {
		for _, c := range a.Credentials() {
			if c.CredentialHandle() == handle {
				return a, nil
			}
		}
	}
	return emptyAccountAuth(), domain.ErrAccountAuthNotFound
}
func (r *stubAccountRepo) FindByEmail(_ context.Context, email string) (domain.AccountAuth, error) {
	for _, a := range r.accounts {
		if a.Email() == email {
			return a, nil
		}
	}
	return emptyAccountAuth(), domain.ErrAccountAuthNotFound
}
func (r *stubAccountRepo) AddPasskey(_ context.Context, accountID domain.AccountID, credentialID string, handle string, _ domain.WebAuthnCredentialData) (domain.AccountAuth, error) {
	a, ok := r.accounts[accountID.String()]
	if !ok {
		return emptyAccountAuth(), domain.ErrAccountAuthNotFound
	}
	newCred, err := domain.NewPasskeyCredential(credentialID, accountID, a.Identifier(), handle, time.Time{})
	if err != nil {
		return emptyAccountAuth(), err
	}
	updated, err := domain.NewAccountAuthWithCredentials(a.AccountID(), a.Identifier(), a.Email(), append(a.Credentials(), newCred))
	if err != nil {
		return emptyAccountAuth(), err
	}
	r.accounts[accountID.String()] = updated
	return updated, nil
}
func (r *stubAccountRepo) ListPasskeys(_ context.Context, accountID domain.AccountID) ([]domain.PasskeyCredential, error) {
	a, ok := r.accounts[accountID.String()]
	if !ok {
		return nil, domain.ErrAccountAuthNotFound
	}
	return a.Credentials(), nil
}
func (r *stubAccountRepo) DeletePasskeyByID(_ context.Context, accountID domain.AccountID, credentialID string) error {
	a, ok := r.accounts[accountID.String()]
	if !ok {
		return domain.ErrAccountAuthNotFound
	}
	creds := a.Credentials()
	for _, c := range creds {
		if c.ID() == credentialID {
			return nil
		}
	}
	return domain.ErrAccountAuthNotFound
}

func (r *stubAccountRepo) FindWebAuthnCredential(_ context.Context, handle string) (domain.WebAuthnStoredCredential, error) {
	for _, a := range r.accounts {
		for _, c := range a.Credentials() {
			if c.CredentialHandle() == handle {
				return domain.ReconstitueWebAuthnStoredCredential(handle, nil, 0, nil, false, false, nil), nil
			}
		}
	}
	return domain.ZeroWebAuthnStoredCredential(), domain.ErrAccountAuthNotFound
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

func newTrackingAccountRepo(account domain.AccountAuth) *trackingAccountRepo {
	return &trackingAccountRepo{
		stubAccountRepo: stubAccountRepo{accounts: map[string]domain.AccountAuth{account.AccountID().String(): account}},
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
func emptyRecoveryToken() domain.RecoveryToken {
	t, _ := domain.NewRecoveryToken("01ARZ3NDEKTSV4RRFFQ69G5FAV", testAccountID("01ARZ3NDEKTSV4RRFFQ69G5FAW"), "placeholder", domain.TokenKindRecovery, time.Unix(1, 0).UTC())
	return t
}
func emptyRecoverySession() domain.RecoverySession {
	s, _ := domain.NewRecoverySession("01ARZ3NDEKTSV4RRFFQ69G5FAV", testAccountID("01ARZ3NDEKTSV4RRFFQ69G5FAW"), domain.TokenKindRecovery, time.Unix(1, 0).UTC())
	return s
}
func emptyAccountAuth() domain.AccountAuth {
	a, _ := domain.NewAccountAuth(testAccountID("01ARZ3NDEKTSV4RRFFQ69G5FAV"), "placeholder@example.com", "placeholder@example.com", "01ARZ3NDEKTSV4RRFFQ69G5FAW", "placeholder")
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

func accountWithTwoPasskeys(t *testing.T) (domain.AccountAuth, *stubAccountRepo) {
	t.Helper()
	cred1, err := domain.NewPasskeyCredential(
		"01ARZ3NDEKTSV4RRFFQ69G5FAV",
		testAccountID("01ARZ3NDEKTSV4RRFFQ69G5FAW"),
		"user@example.com",
		"handle-one",
		time.Time{},
	)
	if err != nil {
		t.Fatalf("NewPasskeyCredential cred1: %v", err)
	}
	cred2, err := domain.NewPasskeyCredential(
		"01ARZ3NDEKTSV4RRFFQ69G5FAX",
		testAccountID("01ARZ3NDEKTSV4RRFFQ69G5FAW"),
		"user@example.com",
		"handle-two",
		time.Time{},
	)
	if err != nil {
		t.Fatalf("NewPasskeyCredential cred2: %v", err)
	}
	account, err := domain.NewAccountAuthWithCredentials(
		testAccountID("01ARZ3NDEKTSV4RRFFQ69G5FAW"),
		"user@example.com",
		"user@example.com",
		[]domain.PasskeyCredential{cred1, cred2},
	)
	if err != nil {
		t.Fatalf("NewAccountAuthWithCredentials: %v", err)
	}
	return account, newStubAccountRepoWithAccount(account)
}

func accountWithOnePasskey(t *testing.T) (domain.AccountAuth, *stubAccountRepo) {
	t.Helper()
	account, err := domain.NewAccountAuth(
		testAccountID("01ARZ3NDEKTSV4RRFFQ69G5FAW"),
		"user@example.com",
		"user@example.com",
		"01ARZ3NDEKTSV4RRFFQ69G5FAV",
		"handle-one",
	)
	if err != nil {
		t.Fatalf("NewAccountAuth: %v", err)
	}
	return account, newStubAccountRepoWithAccount(account)
}

func newTestAuthService(stateRepo application.AuthStateRepository, accountRepo application.AccountAuthRepository) *application.AuthService {
	cfg := config.AuthConfig{
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
		SecretHashKey:                   "test-pepper",
		PasskeyStartThrottleWindow:      time.Minute,
		FailureLockThreshold:            10,
		FailureLockDuration:             15 * time.Minute,
		FailureLockWindow:               time.Minute,
		WebAuthnRPID:                    "example.com",
		AccountRecoveryURLBase:          "https://example.com/recover",
		JWTSecret:                       "test-jwt-secret-key-must-be-at-least-32bytes",
	}
	auth := application.NewAuthService(stateRepo, accountRepo, nil, nil, fixedClock(), newSeqPolicy(), cfg)
	refreshStore := newStubRefreshTokenStore()
	sessionStore := newStubSessionStore()
	tokenService := application.NewTokenService(refreshStore, sessionStore, nil, cfg, fixedClock(), newSeqPolicy())
	auth.UseTokenService(tokenService)
	return auth
}

// ─── MockWebAuthnProvider ─────────────────────────────────────────────────────

// mockWebAuthnProvider は WebAuthnProvider interface のテスト用スタブ。
// テストごとに BeginLogin/FinishLogin/BeginRegistration/FinishRegistration の
// 戻り値・エラーをカスタマイズできる。
type mockWebAuthnProvider struct {
	beginLoginFn         func(ctx context.Context, identifier string) (string, []byte, error)
	finishLoginFn        func(ctx context.Context, challengeKey string, credential application.WebAuthnAssertionCredentialDTO, lookupCredential func(ctx context.Context, handle string) (domain.WebAuthnStoredCredential, error)) (string, uint32, bool, bool, error)
	beginRegistrationFn  func(ctx context.Context, accountID domain.AccountID) (string, []byte, error)
	finishRegistrationFn func(ctx context.Context, challengeKey string, accountID domain.AccountID, credential application.WebAuthnAttestationCredentialDTO) (string, domain.WebAuthnCredentialData, error)
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

func (m *mockWebAuthnProvider) BeginRegistration(ctx context.Context, accountID domain.AccountID) (string, []byte, error) {
	if m.beginRegistrationFn != nil {
		return m.beginRegistrationFn(ctx, accountID)
	}
	return "challenge-key-reg", []byte(`{"challenge":"challenge-key-reg"}`), nil
}

func (m *mockWebAuthnProvider) FinishRegistration(ctx context.Context, challengeKey string, accountID domain.AccountID, credential application.WebAuthnAttestationCredentialDTO) (string, domain.WebAuthnCredentialData, error) {
	if m.finishRegistrationFn != nil {
		return m.finishRegistrationFn(ctx, challengeKey, accountID, credential)
	}
	return credential.ID, domain.ZeroWebAuthnCredentialData(), nil
}

// newTestAuthServiceWithProvider は WebAuthn provider を注入した AuthService を返す。
func newTestAuthServiceWithProvider(stateRepo application.AuthStateRepository, accountRepo application.AccountAuthRepository, provider application.WebAuthnProvider) *application.AuthService {
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
	if result.AccessToken == "" {
		t.Error("expected non-empty AccessToken")
	}
	if result.RefreshToken == "" {
		t.Error("expected non-empty RefreshToken")
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
	accountID := testAccountID("01ARZ3NDEKTSV4RRFFQ69G5FAW")
	recoverySession, err := domain.NewRecoverySession("01ARZ3NDEKTSV4RRFFQ69G5FAV", accountID, domain.TokenKindRecovery, time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("NewRecoverySession: %v", err)
	}
	stateRepo.recoverySessions[recoverySession.ID()] = recoverySession

	provider := &mockWebAuthnProvider{
		beginRegistrationFn: func(_ context.Context, _ domain.AccountID) (string, []byte, error) {
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

	accountID := testAccountID("01ARZ3NDEKTSV4RRFFQ69G5FAW")
	recoverySession, err := domain.NewRecoverySession("01ARZ3NDEKTSV4RRFFQ69G5FAV", accountID, domain.TokenKindRecovery, time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("NewRecoverySession: %v", err)
	}
	stateRepo.recoverySessions[recoverySession.ID()] = recoverySession

	provider := &mockWebAuthnProvider{
		finishRegistrationFn: func(_ context.Context, _ string, _ domain.AccountID, cred application.WebAuthnAttestationCredentialDTO) (string, domain.WebAuthnCredentialData, error) {
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
		beginRegistrationFn: func(_ context.Context, _ domain.AccountID) (string, []byte, error) {
			return "ck-add-001", []byte(`{"publicKey":{"challenge":"ck-add-001"}}`), nil
		},
	}
	svc := newTestAuthServiceWithProvider(stateRepo, accountRepo, provider)

	ch, err := svc.StartAddPasskey(context.Background(), testAccountID("01ARZ3NDEKTSV4RRFFQ69G5FAW"))
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
		finishRegistrationFn: func(_ context.Context, _ string, _ domain.AccountID, _ application.WebAuthnAttestationCredentialDTO) (string, domain.WebAuthnCredentialData, error) {
			return "handle-added", domain.ZeroWebAuthnCredentialData(), nil
		},
	}
	svc := newTestAuthServiceWithProvider(stateRepo, accountRepo, provider)

	creds, err := svc.FinishAddPasskey(context.Background(), testAccountID("01ARZ3NDEKTSV4RRFFQ69G5FAW"), application.WebAuthnAttestationCredentialDTO{
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

	accountID := testAccountID("01ARZ3NDEKTSV4RRFFQ69G5FAW")
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
// 指定した場合、適切なエラー（domain.ErrAccountAuthNotFound）を返す（cross-account deletion rejection）。
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
	accountB, err := domain.NewAccountAuthWithCredentials(
		testAccountID("01ARZ3NDEKTSV4RRFFQ69G5FB2"),
		"other@example.com",
		"other@example.com",
		[]domain.PasskeyCredential{accountBCred},
	)
	if err != nil {
		t.Fatalf("NewAccountAuthWithCredentials accountB: %v", err)
	}

	// 両アカウントを含む repo
	accountRepo := &stubAccountRepo{accounts: map[string]domain.AccountAuth{
		accountA.AccountID().String(): accountA,
		accountB.AccountID().String(): accountB,
	}}
	svc := newTestAuthService(newStubStateRepo(fixedClock()), accountRepo)

	// account A の accountID で account B の credentialID を削除しようとする
	err = svc.DeletePasskey(context.Background(), accountA.AccountID(), accountBCred.ID())
	if err == nil {
		t.Fatal("expected error for cross-account credential deletion, got nil")
	}
	// account A には account B の credentialID が存在しないので ErrAccountAuthNotFound を返す
	if err != domain.ErrAccountAuthNotFound {
		t.Fatalf("expected domain.ErrAccountAuthNotFound, got %v", err)
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

}

// [AUTH-BE-S029] UV-less new-device registration は拒否される
func TestRegisterPasskeyRejectsUVLessAttestation(t *testing.T) {
	t.Parallel()
	stateRepo := newStubStateRepo(fixedClock())
	_, accountRepo := accountWithOnePasskey(t)

	accountID := testAccountID("01ARZ3NDEKTSV4RRFFQ69G5FAW")
	recoverySession, err := domain.NewRecoverySession("01ARZ3NDEKTSV4RRFFQ69G5FAV", accountID, domain.TokenKindRecovery, time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("NewRecoverySession: %v", err)
	}
	stateRepo.recoverySessions[recoverySession.ID()] = recoverySession

	provider := &mockWebAuthnProvider{
		finishRegistrationFn: func(_ context.Context, _ string, _ domain.AccountID, _ application.WebAuthnAttestationCredentialDTO) (string, domain.WebAuthnCredentialData, error) {
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

	// Recovery token を発行（新しい "tokenID.plainSecret" 形式）
	account, _ := accountRepo.FindByEmail(context.Background(), "user@example.com")
	tokenID := "01ARZ3NDEKTSV4RRFFQ69G5FAV"                    // #nosec G101 -- test ULID, not a secret
	plainSecret := "test-random-secret-b64url-encoded-32bytes" // #nosec G101
	urlToken := tokenID + "." + plainSecret                    // #nosec G101
	token, err := domain.NewRecoveryToken(tokenID, account.AccountID(), plainSecret, domain.TokenKindRecovery, fixedClock()().Add(30*time.Minute))
	if err != nil {
		t.Fatalf("NewRecoveryToken: %v", err)
	}
	stateRepo.recoveryTokens[plainSecret] = token

	// 1 回目: 成功
	result1, err := svc.ConsumeRecoveryToken(context.Background(), application.ConsumeRecoveryTokenInput{Token: urlToken, ClientIP: "127.0.0.1"})
	if err != nil {
		t.Fatalf("first consume should succeed, got: %v", err)
	}
	if result1.RecoverySessionID == "" {
		t.Fatal("expected recovery session on first consume")
	}

	// 2 回目: 拒否（アトミック consume により token が既に消費済み）
	_, err = svc.ConsumeRecoveryToken(context.Background(), application.ConsumeRecoveryTokenInput{Token: urlToken, ClientIP: "127.0.0.1"})
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
