package product

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	stdhttp "net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	application "www-template/packages/backend/internal/application/auth"
	domain "www-template/packages/backend/internal/domain"
	"www-template/packages/backend/internal/platform/config"

	"www-template/packages/backend/internal/platform/id"
)

var ulidRegex = regexp.MustCompile(`^[0-9A-HJKMNP-TV-Z]{26}$`)

const productTestAllowedOrigin = "http://localhost:5173"

func TestAuthPasskeyFinishIssuesSession(t *testing.T) {
	t.Parallel()
	env := newAuthTestEnv(t)
	challenge := startPasskey(t, env.router, "member@example.com")
	response := performJSON(t, env.router, stdhttp.MethodPost, "/api/v1/auth/passkey/finish", map[string]any{"credential": assertionCredentialJSON("existing-credential", challengeValue(challenge))}, "")
	assertStatus(t, response, stdhttp.StatusOK)
	assertNoStore(t, response)

	var session map[string]any
	decodeJSON(t, response, &session)
	assertProductAccountSubject(t, session, true)
	assertULIDField(t, session, "sessionId")

	appResponse := performJSON(t, env.router, stdhttp.MethodPost, "/api/v1/auth/logout", nil, session["accessToken"].(string))
	assertStatus(t, appResponse, stdhttp.StatusOK)
}

func TestAuthInactiveSessionRejected(t *testing.T) {
	t.Parallel()
	env := newAuthTestEnv(t)
	challenge := startPasskey(t, env.router, "member@example.com")
	finishResponse := performJSON(t, env.router, stdhttp.MethodPost, "/api/v1/auth/passkey/finish", map[string]any{"credential": assertionCredentialJSON("existing-credential", challengeValue(challenge))}, "")
	assertStatus(t, finishResponse, stdhttp.StatusOK)
	assertNoStore(t, finishResponse)
	var session map[string]any
	decodeJSON(t, finishResponse, &session)
	env.advance(15 * 24 * time.Hour)

	response := performJSON(t, env.router, stdhttp.MethodPost, "/api/v1/auth/logout", nil, session["accessToken"].(string))
	assertStatus(t, response, stdhttp.StatusUnauthorized)
	assertNoStore(t, response)
	assertFailureCode(t, response, "session-expired")
}

func TestAuthLogoutRevokesSession(t *testing.T) {
	t.Parallel()
	env := newAuthTestEnv(t)
	challenge := startPasskey(t, env.router, "member@example.com")
	finishResponse := performJSON(t, env.router, stdhttp.MethodPost, "/api/v1/auth/passkey/finish", map[string]any{"credential": assertionCredentialJSON("existing-credential", challengeValue(challenge))}, "")
	assertStatus(t, finishResponse, stdhttp.StatusOK)
	assertNoStore(t, finishResponse)
	var session map[string]any
	decodeJSON(t, finishResponse, &session)

	logoutResponse := performJSON(t, env.router, stdhttp.MethodPost, "/api/v1/auth/logout", nil, session["accessToken"].(string))
	assertStatus(t, logoutResponse, stdhttp.StatusOK)
	assertNoStore(t, logoutResponse)

	appResponse := performJSON(t, env.router, stdhttp.MethodPost, "/api/v1/auth/logout", nil, session["accessToken"].(string))
	assertStatus(t, appResponse, stdhttp.StatusUnauthorized)
	assertFailureCode(t, appResponse, "session-expired")
}

func TestAuthRecoveryRequestGenericAccepted(t *testing.T) {
	t.Parallel()
	env := newAuthTestEnv(t)
	existing := performJSON(t, env.router, stdhttp.MethodPost, "/api/v1/auth/recovery", map[string]string{"email": "member@example.com"}, "")
	missing := performJSON(t, env.router, stdhttp.MethodPost, "/api/v1/auth/recovery", map[string]string{"email": "missing@example.com"}, "")
	assertStatus(t, existing, stdhttp.StatusAccepted)
	assertStatus(t, missing, stdhttp.StatusAccepted)
	assertNoStore(t, existing)
	assertNoStore(t, missing)
	assertAcceptedBody(t, existing)
	assertAcceptedBody(t, missing)
}

func TestAuthMissingSessionUnauthenticated(t *testing.T) {
	t.Parallel()
	env := newAuthTestEnv(t)
	response := performJSON(t, env.router, stdhttp.MethodPost, "/api/v1/auth/logout", nil, "")
	assertStatus(t, response, stdhttp.StatusUnauthorized)
	assertNoStore(t, response)
	assertFailureCode(t, response, "unauthenticated")
}

func TestAuthStoreOutageFailsClosed(t *testing.T) {
	t.Parallel()
	// 正常な環境でログインし JWT を発行する
	env := newAuthTestEnv(t)
	token := loginWithPasskey(t, env.router, "member@example.com")

	// セッションストア障害時の fail-closed を検証するため、canonical lifecycle が利用不能な Auth を構築する。
	auth := mustNewProductAuthForTest(t, newStubAuthStateRepository(env.now), stubAccountAuthRepositoryWithMember(), nil, nil, unavailableProductAccountLifecycle{}, env.now, testConfig().AuthRuntime(), application.AuthServiceOptionalPorts{})
	router := NewRouter(testConfig(), Dependencies{Auth: auth})

	response := performJSON(t, router, stdhttp.MethodPost, "/api/v1/auth/logout", nil, token)
	assertStatus(t, response, stdhttp.StatusServiceUnavailable)
	assertNoStore(t, response)
	assertFailureCode(t, response, "internal-error")
}

func TestAuthRecoveryThrottleStaysGeneric(t *testing.T) {
	t.Parallel()
	env := newAuthTestEnv(t)
	for range 4 {
		response := performJSON(t, env.router, stdhttp.MethodPost, "/api/v1/auth/recovery", map[string]string{"email": "member@example.com"}, "")
		assertStatus(t, response, stdhttp.StatusAccepted)
		assertNoStore(t, response)
		assertAcceptedBody(t, response)
	}
}

func TestAuthPasskeyStartUsesConfiguredWebAuthnRPID(t *testing.T) {
	t.Parallel()
	clock := &mutableClock{current: time.Date(2026, time.March, 21, 0, 0, 0, 0, time.UTC)}
	stateRepo := newStubAuthStateRepository(clock.Now)
	accountRepo := stubAccountAuthRepositoryWithMember()
	cfg := testConfig()
	cfg.Auth.WebAuthnRPID = "www-template"
	auth := mustNewProductAuthForTest(t, stateRepo, accountRepo, &capturingAccountRecoverySender{}, &stubInvitationPasskeyRegistrar{}, nil, clock.Now, cfg.AuthRuntime(), application.AuthServiceOptionalPorts{WebAuthn: newMockWebAuthnProvider()})
	router := NewRouter(cfg, Dependencies{Auth: auth})

	response := performJSON(t, router, stdhttp.MethodPost, "/api/v1/auth/passkey/start", map[string]string{"identifier": "member@example.com"}, "")

	assertStatus(t, response, stdhttp.StatusOK)
	assertNoStore(t, response)
	var body map[string]any
	decodeJSON(t, response, &body)
	if body["rpId"] != "www-template" {
		t.Fatalf("expected rpId www-template, got %#v", body["rpId"])
	}
}

func TestAuthPasskeyStartThrottleRejectsNonRevealing(t *testing.T) {
	t.Parallel()
	t.Run("[AUTH-BE-S013] passkey/start throttle rejects without extra challenge issuance", func(t *testing.T) {
		env := newAuthTestEnv(t)
		for range 5 {
			response := performJSON(t, env.router, stdhttp.MethodPost, "/api/v1/auth/passkey/start", map[string]string{"identifier": "member@example.com"}, "")
			assertStatus(t, response, stdhttp.StatusOK)
			assertNoStore(t, response)
		}
		issuedBeforeReject := len(env.stateRepo.challenges)

		response := performJSON(t, env.router, stdhttp.MethodPost, "/api/v1/auth/passkey/start", map[string]string{"identifier": "member@example.com"}, "")

		assertStatus(t, response, stdhttp.StatusBadRequest)
		assertNoStore(t, response)
		var body map[string]any
		decodeJSON(t, response, &body)
		assertULIDField(t, body, "requestId")
		if body["error"] != nonRevealingAuthRejectMessage {
			t.Fatalf("expected generic throttle reject message, got %#v", body["error"])
		}
		if _, ok := body["challenge"]; ok {
			t.Fatalf("expected throttled response to omit challenge, got %#v", body["challenge"])
		}
		if len(env.stateRepo.challenges) != issuedBeforeReject {
			t.Fatalf("expected no additional challenge issuance on throttle, before=%d after=%d", issuedBeforeReject, len(env.stateRepo.challenges))
		}
		if strings.Contains(response.Body.String(), "internal-error") {
			t.Fatalf("expected throttle reject to avoid internal-error classification, got %s", response.Body.String())
		}
	})
}

func TestAuthInvalidJSONReturnsTypedNoStoreError(t *testing.T) {
	t.Parallel()
	env := newAuthTestEnv(t)
	for _, path := range []string{
		"/api/v1/auth/passkey/start",
		"/api/v1/auth/passkey/finish",
		"/api/v1/auth/passkey/register",
		"/api/v1/auth/recovery",
		"/api/v1/auth/recovery/consume",
	} {
		response := performRawJSON(t, env.router, stdhttp.MethodPost, path, []byte(`{"invalid":`), "")
		assertStatus(t, response, stdhttp.StatusBadRequest)
		assertNoStore(t, response)
		var body map[string]any
		decodeJSON(t, response, &body)
		assertULIDField(t, body, "requestId")
		if body["error"] != invalidRequestBodyMessage {
			t.Fatalf("expected invalid request body message for %s, got %#v", path, body["error"])
		}
	}
}

func TestPasskeyAppEndpointsInvalidJSONReturnsTypedNoStoreError(t *testing.T) {
	t.Parallel()
	env := newAuthTestEnv(t)
	token := loginWithPasskey(t, env.router, "member@example.com")

	for _, tc := range []struct {
		method string
		path   string
	}{
		{stdhttp.MethodPost, "/api/v1/passkeys/finish"},
	} {
		response := performRawJSON(t, env.router, tc.method, tc.path, []byte(`{"invalid":`), token)
		assertStatus(t, response, stdhttp.StatusBadRequest)
		assertNoStore(t, response)
		var body map[string]any
		decodeJSON(t, response, &body)
		assertULIDField(t, body, "requestId")
		if body["error"] != invalidRequestBodyMessage {
			t.Fatalf("expected invalid request body message for %s, got %#v", tc.path, body["error"])
		}
	}
}

func TestAuthFailureResponsesIssueDistinctRequestIDs(t *testing.T) {
	t.Parallel()
	env := newAuthTestEnv(t)
	first := performJSON(t, env.router, stdhttp.MethodPost, "/api/v1/auth/logout", nil, "")
	second := performJSON(t, env.router, stdhttp.MethodPost, "/api/v1/auth/logout", nil, "")
	assertStatus(t, first, stdhttp.StatusUnauthorized)
	assertStatus(t, second, stdhttp.StatusUnauthorized)
	assertNoStore(t, first)
	assertNoStore(t, second)
	firstBody := decodeJSONBody(t, first)
	secondBody := decodeJSONBody(t, second)
	assertULIDField(t, firstBody, "requestId")
	assertULIDField(t, secondBody, "requestId")
	if firstBody["requestId"] == secondBody["requestId"] {
		t.Fatalf("expected distinct request ids, got %q", firstBody["requestId"])
	}
}

func TestAuthRecoverySendFailureStaysGenericAndRecordsRetryExpectationAUTHBES011(t *testing.T) {
	t.Parallel()
	env := newAuthTestEnvWithSender(t, failingAccountRecoverySender{err: errors.New("smtp rejected message")})

	response := performJSON(t, env.router, stdhttp.MethodPost, "/api/v1/auth/recovery", map[string]string{"email": "member@example.com"}, "")

	assertStatus(t, response, stdhttp.StatusAccepted)
	assertNoStore(t, response)
	assertAcceptedBody(t, response)

	if len(env.stateRepo.recoveryFailures) != 1 {
		t.Fatalf("expected one recovery delivery failure record, got %d", len(env.stateRepo.recoveryFailures))
	}
	var recorded domain.RecoveryDeliveryFailure
	for _, failure := range env.stateRepo.recoveryFailures {
		recorded = failure
	}
	if recorded.Email() != "member@example.com" {
		t.Fatalf("expected failure record email member@example.com, got %q", recorded.Email())
	}
	if recorded.LastError() != "smtp rejected message" {
		t.Fatalf("expected failure record last error, got %q", recorded.LastError())
	}
	if !recorded.FailedAt().Equal(env.now()) {
		t.Fatalf("expected failedAt %s, got %s", env.now(), recorded.FailedAt())
	}
	if !recorded.RetryAfter().Equal(recorded.FailedAt()) {
		t.Fatalf("expected retryAfter to equal failedAt, got %s and %s", recorded.RetryAfter(), recorded.FailedAt())
	}
	if !recorded.ExpiresAt().Equal(env.now().Add(testConfig().AuthRuntime().RecoveryTokenTTL)) {
		t.Fatalf("expected expiresAt %s, got %s", env.now().Add(testConfig().AuthRuntime().RecoveryTokenTTL), recorded.ExpiresAt())
	}
	if env.stateRepo.lastRecoveryFailureTTL != testConfig().AuthRuntime().RecoveryTokenTTL {
		t.Fatalf("expected retry ttl %s, got %s", testConfig().AuthRuntime().RecoveryTokenTTL, env.stateRepo.lastRecoveryFailureTTL)
	}
	if !ulidRegex.MatchString(recorded.RequestID()) {
		t.Fatalf("expected request id ULID, got %q", recorded.RequestID())
	}
	if !ulidRegex.MatchString(recorded.RecoveryTokenID()) {
		t.Fatalf("expected recovery token id ULID, got %q", recorded.RecoveryTokenID())
	}
	if !ulidRegex.MatchString(recorded.AccountID().String()) {
		t.Fatalf("expected account id ULID, got %q", recorded.AccountID())
	}
}

func TestAuthRecoverySendFailurePersistsOriginalTokenExpiryAUTHBES011(t *testing.T) {
	t.Parallel()
	clock := &mutableClock{current: time.Date(2026, time.March, 21, 0, 0, 0, 0, time.UTC)}
	stateRepo := newStubAuthStateRepository(clock.Now)
	accountRepo := stubAccountAuthRepositoryWithMember()
	sender := advancingFailingAccountRecoverySender{advance: func() { clock.Advance(2 * time.Minute) }, err: errors.New("smtp delayed failure")}
	auth := mustNewProductAuthForTest(t, stateRepo, accountRepo, sender, &stubInvitationPasskeyRegistrar{}, nil, clock.Now, testConfig().AuthRuntime(), application.AuthServiceOptionalPorts{WebAuthn: newMockWebAuthnProvider()})
	router := NewRouter(testConfig(), Dependencies{Auth: auth})

	response := performJSON(t, router, stdhttp.MethodPost, "/api/v1/auth/recovery", map[string]string{"email": "member@example.com"}, "")

	assertStatus(t, response, stdhttp.StatusAccepted)
	assertNoStore(t, response)
	assertAcceptedBody(t, response)

	var recorded domain.RecoveryDeliveryFailure
	for _, failure := range stateRepo.recoveryFailures {
		recorded = failure
	}
	expectedExpiresAt := time.Date(2026, time.March, 21, 0, 30, 0, 0, time.UTC)
	if !recorded.ExpiresAt().Equal(expectedExpiresAt) {
		t.Fatalf("expected original token expiry %s, got %s", expectedExpiresAt, recorded.ExpiresAt())
	}
	expectedTTL := 28 * time.Minute
	if stateRepo.lastRecoveryFailureTTL != expectedTTL {
		t.Fatalf("expected remaining retry ttl %s, got %s", expectedTTL, stateRepo.lastRecoveryFailureTTL)
	}
	if !recorded.FailedAt().Equal(time.Date(2026, time.March, 21, 0, 2, 0, 0, time.UTC)) {
		t.Fatalf("expected delayed failedAt, got %s", recorded.FailedAt())
	}
}

func TestAuthValidRecoveryTokenYieldsRecoverySession(t *testing.T) {
	t.Parallel()
	env := newAuthTestEnv(t)
	performJSON(t, env.router, stdhttp.MethodPost, "/api/v1/auth/recovery", map[string]string{"email": "member@example.com"}, "")
	response := performJSON(t, env.router, stdhttp.MethodPost, "/api/v1/auth/recovery/consume", map[string]string{"token": deliveryToken(t, env.sender.lastDelivery.RecoveryURL)}, "")
	assertStatus(t, response, stdhttp.StatusOK)
	assertNoStore(t, response)
	var body map[string]any
	decodeJSON(t, response, &body)
	assertULIDField(t, body, "recoveryTokenId")
	assertULIDField(t, body, "recoverySessionId")
	assertULIDField(t, body, "recovery_session")
}

func TestAuthInvalidRecoveryTokenRejected(t *testing.T) {
	t.Parallel()
	env := newAuthTestEnv(t)
	response := performJSON(t, env.router, stdhttp.MethodPost, "/api/v1/auth/recovery/consume", map[string]string{"token": "opaque-invalid"}, "")
	assertStatus(t, response, stdhttp.StatusBadRequest)
	assertNoStore(t, response)
}

func TestAuthRecoveryRegisterExistingAccountOnly(t *testing.T) {
	t.Parallel()
	env := newAuthTestEnv(t)
	recoverySession := consumeRecoverySession(t, env)
	response := performJSON(t, env.router, stdhttp.MethodPost, "/api/v1/auth/passkey/register", map[string]any{"recovery_session": recoverySession, "credential": attestationCredentialJSON("new-credential", "")}, "")
	assertStatus(t, response, stdhttp.StatusOK)
	assertNoStore(t, response)
	var body map[string]any
	decodeJSON(t, response, &body)
	assertProductAccountSubject(t, body, true)
}

func TestAuthInviteOnlyCannotRegisterRecovery(t *testing.T) {
	t.Parallel()
	env := newAuthTestEnv(t)
	response := performJSON(t, env.router, stdhttp.MethodPost, "/api/v1/auth/passkey/register", map[string]any{"credential": attestationCredentialJSON("invite-only", ""), "invitation_session": "01ARZ3NDEKTSV4RRFFQ69G5FC1"}, "")
	assertStatus(t, response, stdhttp.StatusBadRequest)
	assertNoStore(t, response)
	if !env.invite.called {
		t.Fatal("expected invite selector seam to be invoked")
	}
	if env.stateRepo.getRecoverySessionCalls != 0 {
		t.Fatalf("expected no recovery session lookup, got %d", env.stateRepo.getRecoverySessionCalls)
	}
}

func TestAuthRegisterRejectsAmbiguousSelectors(t *testing.T) {
	t.Parallel()
	env := newAuthTestEnv(t)
	cases := []map[string]any{
		{"credential": attestationCredentialJSON("only-credential", "")},
		{"credential": attestationCredentialJSON("both", ""), "invitation_session": "01ARZ3NDEKTSV4RRFFQ69G5FC2", "recovery_session": "01ARZ3NDEKTSV4RRFFQ69G5FC3"},
	}
	for _, payload := range cases {
		response := performJSON(t, env.router, stdhttp.MethodPost, "/api/v1/auth/passkey/register", payload, "")
		assertStatus(t, response, stdhttp.StatusBadRequest)
		assertNoStore(t, response)
	}
}

func TestAuthRepeatedFailuresEnterTemporaryLock(t *testing.T) {
	t.Parallel()
	env := newAuthTestEnv(t)
	for range 11 {
		response := performJSON(t, env.router, stdhttp.MethodPost, "/api/v1/auth/recovery/consume", map[string]string{"token": "opaque-invalid"}, "")
		assertStatus(t, response, stdhttp.StatusBadRequest)
		assertNoStore(t, response)
	}
	lock, ok, err := env.stateRepo.GetLock(context.Background(), "lock:opaque-invalid:192.0.2.10")
	if err != nil {
		t.Fatalf("get lock: %v", err)
	}
	if !ok || !lock.Active(env.now()) {
		t.Fatal("expected temporary lock to be active")
	}
}

func mustNewProductAuthForTest(t *testing.T, stateRepo application.AuthStateRepository, accountRepo application.PasskeyAccountRepository, sender application.AccountRecoverySender, invite application.InvitationPasskeyRegistrar, lifecycle application.ProductAccountLifecycle, clock func() time.Time, cfg config.AuthConfig, optional application.AuthServiceOptionalPorts) *application.AuthService {
	t.Helper()
	// Step 1: テストで直接使わない配送 port も constructor 必須依存として明示し、production と同じ fail-close 構成を保つ。
	if sender == nil {
		sender = &capturingAccountRecoverySender{}
	}
	if invite == nil {
		invite = &stubInvitationPasskeyRegistrar{}
	}
	// Step 2: caller が canonical lifecycle を指定しない場合だけ、テスト用 Product account lifecycle を構築する。
	if lifecycle == nil {
		createdLifecycle, _ := newTestProductAccountLifecycle(t, accountRepo, clock, cfg.RefreshTokenTTL)
		lifecycle = createdLifecycle
	}
	// Step 3: constructor-time DI を使い、Use* mutator による後付け状態変更をテストにも残さない。
	auth, err := application.NewAuthService(application.AuthServiceDependencies{StateRepo: stateRepo, AccountRepo: accountRepo, RecoverySender: sender, InvitationRegistrar: invite, AccountLifecycle: lifecycle, Clock: clock, Policy: newSequentialPolicy()}, optional, cfg)
	if err != nil {
		t.Fatalf("new auth service: %v", err)
	}
	return auth
}

type unavailableProductAccountLifecycle struct{}

func (unavailableProductAccountLifecycle) IssueAccountSession(context.Context, application.IssueAccountSessionInput) (application.AccountSessionResult, error) {
	return application.AccountSessionResult{}, application.ErrAccountAuthUnavailable
}

func (unavailableProductAccountLifecycle) AuthorizeAccountSession(context.Context, string) (application.ValidatedSession, error) {
	return application.ValidatedSession{}, application.ErrAccountAuthUnavailable
}

func (unavailableProductAccountLifecycle) ListAccountSessions(context.Context, domain.AccountID) ([]application.SessionMetadata, error) {
	return nil, application.ErrAccountAuthUnavailable
}

func (unavailableProductAccountLifecycle) RevokeAccountSession(context.Context, application.RevokeAccountSessionInput) error {
	return application.ErrAccountAuthUnavailable
}

func (unavailableProductAccountLifecycle) RevokeOtherAccountSessions(context.Context, domain.AccountID, string) error {
	return application.ErrAccountAuthUnavailable
}

func (unavailableProductAccountLifecycle) RevokeAllAccountSessions(context.Context, domain.AccountID) error {
	return application.ErrAccountAuthUnavailable
}

func (unavailableProductAccountLifecycle) RefreshAccountSession(context.Context, application.RefreshAccountSessionInput) (application.AccountRefreshResult, error) {
	return application.AccountRefreshResult{}, application.ErrAccountAuthUnavailable
}

type authTestEnv struct {
	router              *gin.Engine
	stateRepo           *stubAuthStateRepository
	sender              *capturingAccountRecoverySender
	invite              *stubInvitationPasskeyRegistrar
	auth                *application.AuthService
	now                 func() time.Time
	advance             func(time.Duration)
	productRefreshStore *testProductRefreshSessionStore
}

// saveTestReauthSession はテスト用の再認証セッションを stateRepo に保存する。
func saveTestReauthSession(t *testing.T, stateRepo *stubAuthStateRepository, accountID domain.AccountID, sessionID string, kind string, now func() time.Time) {
	t.Helper()
	session, err := domain.NewReauthenticationSession(
		"01ARZ3NDEKTSV4RRFFQ69G5FAY", accountID, sessionID, kind,
		"01ARZ3NDEKTSV4RRFFQ69G5FAZ", now().Add(5*time.Minute),
	)
	if err != nil {
		t.Fatalf("new reauth session: %v", err)
	}
	if err := stateRepo.SaveReauthenticationSession(context.Background(), session, 5*time.Minute); err != nil {
		t.Fatalf("save test reauth session: %v", err)
	}
}

func newAuthTestEnv(t *testing.T) authTestEnv {
	t.Helper()
	return newAuthTestEnvWithSender(t, &capturingAccountRecoverySender{})
}

func newAuthTestEnvWithSender(t *testing.T, sender application.AccountRecoverySender) authTestEnv {
	t.Helper()
	clock := &mutableClock{current: time.Date(2026, time.March, 21, 0, 0, 0, 0, time.UTC)}
	stateRepo := newStubAuthStateRepository(clock.Now)
	accountRepo := stubAccountAuthRepositoryWithMember()
	invite := &stubInvitationPasskeyRegistrar{}
	lifecycle, contextRefresh := newTestProductAccountLifecycle(t, accountRepo, clock.Now, testConfig().AuthRuntime().RefreshTokenTTL)
	auth := mustNewProductAuthForTest(t, stateRepo, accountRepo, sender, invite, lifecycle, clock.Now, testConfig().AuthRuntime(), application.AuthServiceOptionalPorts{WebAuthn: newMockWebAuthnProvider()})
	sessionService := application.NewProductSessionService(lifecycle)

	capturingSender, _ := sender.(*capturingAccountRecoverySender)
	return authTestEnv{
		router:    NewRouter(testConfig(), Dependencies{Auth: auth, TokenService: contextRefresh, SessionService: sessionService}),
		stateRepo: stateRepo,
		sender:    capturingSender,
		invite:    invite,
		auth:      auth,
		now:       clock.Now,
		advance:   clock.Advance,
	}
}

type mutableClock struct{ current time.Time }

func (c *mutableClock) Now() time.Time          { return c.current }
func (c *mutableClock) Advance(d time.Duration) { c.current = c.current.Add(d) }

type capturingAccountRecoverySender struct{ lastDelivery application.RecoveryDelivery }

func (m *capturingAccountRecoverySender) SendAccountRecovery(_ context.Context, delivery application.RecoveryDelivery) error {
	m.lastDelivery = delivery
	return nil
}

type failingAccountRecoverySender struct{ err error }

func (m failingAccountRecoverySender) SendAccountRecovery(_ context.Context, _ application.RecoveryDelivery) error {
	return m.err
}

type advancingFailingAccountRecoverySender struct {
	advance func()
	err     error
}

func (m advancingFailingAccountRecoverySender) SendAccountRecovery(_ context.Context, _ application.RecoveryDelivery) error {
	if m.advance != nil {
		m.advance()
	}
	return m.err
}

type sequentialPolicy struct{ next int }

func newSequentialPolicy() id.AuthIDPolicy {
	seq := &sequentialPolicy{}
	ids := []string{
		"01ARZ3NDEKTSV4RRFFQ69G5FAV", "01ARZ3NDEKTSV4RRFFQ69G5FAW", "01ARZ3NDEKTSV4RRFFQ69G5FAX", "01ARZ3NDEKTSV4RRFFQ69G5FAY",
		"01ARZ3NDEKTSV4RRFFQ69G5FAZ", "01ARZ3NDEKTSV4RRFFQ69G5FB1", "01ARZ3NDEKTSV4RRFFQ69G5FB2", "01ARZ3NDEKTSV4RRFFQ69G5FB3",
		"01ARZ3NDEKTSV4RRFFQ69G5FB4", "01ARZ3NDEKTSV4RRFFQ69G5FB5", "01ARZ3NDEKTSV4RRFFQ69G5FB6", "01ARZ3NDEKTSV4RRFFQ69G5FB7",
		"01ARZ3NDEKTSV4RRFFQ69G5FB8", "01ARZ3NDEKTSV4RRFFQ69G5FB9", "01ARZ3NDEKTSV4RRFFQ69G5FBA", "01ARZ3NDEKTSV4RRFFQ69G5FBB",
	}
	return id.AuthIDPolicy{
		New:      func() string { value := ids[seq.next]; seq.next++; return value },
		Validate: domain.ValidateAuthID,
	}
}

func startPasskey(t *testing.T, router *gin.Engine, identifier string) map[string]any {
	t.Helper()
	response := performJSON(t, router, stdhttp.MethodPost, "/api/v1/auth/passkey/start", map[string]string{"identifier": identifier}, "")
	assertStatus(t, response, stdhttp.StatusOK)
	assertNoStore(t, response)
	var body map[string]any
	decodeJSON(t, response, &body)
	assertULIDField(t, body, "requestId")
	return body
}

func consumeRecoverySession(t *testing.T, env authTestEnv) string {
	t.Helper()
	performJSON(t, env.router, stdhttp.MethodPost, "/api/v1/auth/recovery", map[string]string{"email": "member@example.com"}, "")
	response := performJSON(t, env.router, stdhttp.MethodPost, "/api/v1/auth/recovery/consume", map[string]string{"token": deliveryToken(t, env.sender.lastDelivery.RecoveryURL)}, "")
	var body map[string]any
	decodeJSON(t, response, &body)
	return body["recovery_session"].(string)
}

func performJSON(t *testing.T, router *gin.Engine, method string, path string, body any, bearer string) *httptest.ResponseRecorder {
	t.Helper()
	// Step 1: Product session 発行系の test request は現行 contract の credentialMode を明示し、Cookie mode の response shape を検証する。
	body = withDefaultCookieCredentialMode(path, body)
	var payload []byte
	if body != nil {
		var err error
		payload, err = json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
	}
	request := httptest.NewRequest(method, path, bytes.NewReader(payload))
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	request.RemoteAddr = "192.0.2.10:1234"
	if bearer != "" {
		request.Header.Set("Authorization", "Bearer "+bearer)
	}
	if productTestRequestNeedsDefaultOrigin(method, path, body) {
		request.Header.Set(productOriginHeader, productTestAllowedOrigin)
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	return recorder
}

func performJSONWithHeaders(t *testing.T, router *gin.Engine, method string, path string, body any, bearer string, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	// Step 1: header 付き helper でも session 発行系 request の credentialMode を固定し、generated DTO の必須 field 欠落で test intent が崩れないようにする。
	body = withDefaultCookieCredentialMode(path, body)
	var payload []byte
	if body != nil {
		var err error
		payload, err = json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
	}
	request := httptest.NewRequest(method, path, bytes.NewReader(payload))
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	request.RemoteAddr = "192.0.2.10:1234"
	if bearer != "" {
		request.Header.Set("Authorization", "Bearer "+bearer)
	}
	if productTestRequestNeedsDefaultOrigin(method, path, body) && !productTestHeadersContainOrigin(headers) {
		request.Header.Set(productOriginHeader, productTestAllowedOrigin)
	}
	for k, v := range headers {
		request.Header.Set(k, v)
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	return recorder
}

func productTestHeadersContainOrigin(headers map[string]string) bool {
	// Step 1: test が Origin の有無や値を明示的に検証する場合は、既定 Origin を補わず意図した header 状態を保つ。
	for key := range headers {
		if strings.EqualFold(key, productOriginHeader) {
			return true
		}
	}
	return false
}

func productTestRequestNeedsDefaultOrigin(method string, path string, body any) bool {
	// Step 1: Cookie を設定・rotation する可能性がある POST だけに allowed Origin を補い、既存の non-browser / bearer-only test へ副作用を出さない。
	if method != stdhttp.MethodPost {
		return false
	}
	if path == "/api/v1/auth/passkey/finish" || path == "/api/v1/auth/passkey/register" {
		return productTestBodyCredentialMode(body) == "cookie"
	}
	if isProductContextRefreshPath(path) {
		return productTestBodyCredentialMode(body) != "bearer"
	}
	return false
}

func productTestBodyCredentialMode(body any) string {
	// Step 1: helper が扱う map body だけから credentialMode を読み取り、未知 body 型では Cookie mode 補完済みとみなして安全側に倒す。
	payload, ok := body.(map[string]any)
	if !ok {
		return ""
	}
	mode, _ := payload["credentialMode"].(string)
	return mode
}

func withDefaultCookieCredentialMode(path string, body any) any {
	// Step 1: passkey login/register finish 以外の body は変更せず、対象 test の入力だけを contract に合わせる。
	if path != "/api/v1/auth/passkey/finish" && path != "/api/v1/auth/passkey/register" {
		return body
	}
	payload, ok := body.(map[string]any)
	if !ok {
		return body
	}
	if _, ok := payload["credentialMode"]; ok {
		return body
	}

	// Step 2: 元 map を破壊せず shallow copy へ cookie mode を追加し、他 test case の再利用 body に副作用を残さない。
	copyPayload := make(map[string]any, len(payload)+1)
	for key, value := range payload {
		copyPayload[key] = value
	}
	copyPayload["credentialMode"] = "cookie"
	return copyPayload
}

func performRawJSON(t *testing.T, router *gin.Engine, method string, path string, body []byte, bearer string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(method, path, bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request.RemoteAddr = "192.0.2.10:1234"
	if bearer != "" {
		request.Header.Set("Authorization", "Bearer "+bearer)
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	return recorder
}

func assertStatus(t *testing.T, response *httptest.ResponseRecorder, expected int) {
	t.Helper()
	if response.Code != expected {
		t.Fatalf("expected status %d, got %d body=%s", expected, response.Code, response.Body.String())
	}
}

func assertNoStore(t *testing.T, response *httptest.ResponseRecorder) {
	t.Helper()
	if response.Header().Get("Cache-Control") != noStoreValue {
		t.Fatalf("expected Cache-Control no-store, got %q", response.Header().Get("Cache-Control"))
	}
}

func assertProductSecurityHeaders(t *testing.T, response *httptest.ResponseRecorder) {
	t.Helper()
	expected := map[string]string{
		"Content-Security-Policy":   productSecurityCSP,
		"Strict-Transport-Security": productSecurityHSTS,
		"Referrer-Policy":           productSecurityReferrerPolicy,
		"X-Content-Type-Options":    "nosniff",
		"X-Frame-Options":           "DENY",
	}
	for name, value := range expected {
		if actual := response.Header().Get(name); actual != value {
			t.Fatalf("expected %s %q, got %q", name, value, actual)
		}
	}
}

func assertFailureCode(t *testing.T, response *httptest.ResponseRecorder, expected string) {
	t.Helper()
	var body map[string]any
	decodeJSON(t, response, &body)
	if body["error"] != expected {
		t.Fatalf("expected error %q, got %#v", expected, body["error"])
	}
}

func assertAcceptedBody(t *testing.T, response *httptest.ResponseRecorder) {
	t.Helper()
	var body map[string]any
	decodeJSON(t, response, &body)
	if body["accepted"] != true {
		t.Fatalf("expected accepted=true, got %#v", body["accepted"])
	}
	assertULIDField(t, body, "requestId")
}

func assertULIDField(t *testing.T, body map[string]any, field string) {
	t.Helper()
	value, ok := body[field].(string)
	if !ok || !ulidRegex.MatchString(value) {
		t.Fatalf("expected %s to be ULID, got %#v", field, body[field])
	}
}

func assertProductAccountSubject(t *testing.T, body map[string]any, requirePasskey bool) {
	t.Helper()

	// Step 1: Product auth response は account subject payload に accountId/passkeyCredentialId を集約するため、top-level に戻っていないことを確認する。
	account, ok := body["account"].(map[string]any)
	if !ok {
		t.Fatalf("expected account subject payload, got %#v", body["account"])
	}
	if _, ok := body["operator"]; ok {
		t.Fatalf("expected Product response not to contain operator subject payload, got %#v", body["operator"])
	}
	assertULIDField(t, account, "accountId")
	if requirePasskey {
		assertULIDField(t, account, "passkeyCredentialId")
	}
}

func decodeJSON(t *testing.T, response *httptest.ResponseRecorder, target any) {
	t.Helper()
	if err := json.Unmarshal(response.Body.Bytes(), target); err != nil {
		t.Fatalf("decode json: %v body=%s", err, response.Body.String())
	}
}

func decodeJSONBody(t *testing.T, response *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var body map[string]any
	decodeJSON(t, response, &body)
	return body
}

func deliveryToken(t *testing.T, url string) string {
	t.Helper()
	parts := strings.Split(url, "token=")
	if len(parts) != 2 {
		t.Fatalf("unexpected recovery url: %s", url)
	}
	return parts[1]
}

func challengeValue(challenge map[string]any) string {
	value, _ := challenge["challenge"].(string)
	return value
}

// assertionCredentialJSON は WebAuthn login credential の JSON 表現を返す。
// ID に credential handle を、Response.ClientDataJSON に challenge 値を格納する。
// HTTP テスト用 mock provider（mockWebAuthnProvider）がこの構造を解釈する。
func assertionCredentialJSON(credentialHandle string, challengeVal string) map[string]any {
	return map[string]any{
		"id":    credentialHandle,
		"rawId": credentialHandle,
		"type":  "public-key",
		"response": map[string]any{
			"clientDataJSON":    challengeVal,
			"authenticatorData": "",
			"signature":         "",
		},
	}
}

// attestationCredentialJSON は WebAuthn registration credential の JSON 表現を返す。
// ID に credential handle を、Response.ClientDataJSON に challenge 値を格納する。
// HTTP テスト用 mock provider（mockWebAuthnProvider）がこの構造を解釈する。
func attestationCredentialJSON(credentialHandle string, challengeVal string) map[string]any {
	return map[string]any{
		"id":    credentialHandle,
		"rawId": credentialHandle,
		"type":  "public-key",
		"response": map[string]any{
			"clientDataJSON":    challengeVal,
			"attestationObject": "",
		},
	}
}

type stubAuthStateRepository struct {
	challenges              map[string]domain.AuthChallenge
	recoveryTokens          map[string]domain.RecoveryToken
	recoveryFailures        map[string]domain.RecoveryDeliveryFailure
	lastRecoveryFailureTTL  time.Duration
	recoverySessions        map[string]domain.RecoverySession
	reauthSessions          map[string]domain.ReauthenticationSession
	counters                map[string]stubCounter
	locks                   map[string]time.Time
	clock                   func() time.Time
	getRecoverySessionCalls int
}

type stubCounter struct {
	count     int
	expiresAt time.Time
}

func newStubAuthStateRepository(clock func() time.Time) *stubAuthStateRepository {
	return &stubAuthStateRepository{
		challenges:       map[string]domain.AuthChallenge{},
		recoveryTokens:   map[string]domain.RecoveryToken{},
		recoveryFailures: map[string]domain.RecoveryDeliveryFailure{},
		recoverySessions: map[string]domain.RecoverySession{},
		reauthSessions:   map[string]domain.ReauthenticationSession{},
		counters:         map[string]stubCounter{},
		locks:            map[string]time.Time{},
		clock:            clock,
	}
}

func (r *stubAuthStateRepository) SaveChallenge(_ context.Context, challenge domain.AuthChallenge, _ time.Duration) error {
	r.challenges[challenge.Challenge()] = challenge
	return nil
}

func (r *stubAuthStateRepository) ConsumeChallenge(_ context.Context, secret string) (domain.AuthChallenge, error) {
	challenge, ok := r.challenges[secret]
	if !ok {
		return emptyChallengeForTest(), domain.ErrChallengeNotFound
	}
	delete(r.challenges, secret)
	return challenge, nil
}

func (r *stubAuthStateRepository) IssueRecoveryToken(_ context.Context, token domain.RecoveryToken, _ time.Duration) error {
	r.recoveryTokens[token.Secret()] = token
	return nil
}

func (r *stubAuthStateRepository) SaveRecoveryDeliveryFailure(_ context.Context, failure domain.RecoveryDeliveryFailure, ttl time.Duration) error {
	r.recoveryFailures[failure.RequestID()] = failure
	r.lastRecoveryFailureTTL = ttl
	return nil
}

func (r *stubAuthStateRepository) ConsumeRecoveryTokenAtomic(_ context.Context, tokenID string, secret string) (domain.RecoveryToken, error) {
	token, ok := r.recoveryTokens[secret]
	if !ok {
		return emptyRecoveryTokenForTest(), domain.ErrRecoveryTokenNotFound
	}
	delete(r.recoveryTokens, secret)
	return token, nil
}

func (r *stubAuthStateRepository) SaveRecoverySession(_ context.Context, session domain.RecoverySession, _ time.Duration) error {
	r.recoverySessions[session.ID()] = session
	return nil
}

func (r *stubAuthStateRepository) GetRecoverySession(_ context.Context, id string) (domain.RecoverySession, error) {
	r.getRecoverySessionCalls++
	session, ok := r.recoverySessions[id]
	if !ok {
		return emptyRecoverySessionForTest(), domain.ErrRecoverySessionNotFound
	}
	return session, nil
}

type stubInvitationPasskeyRegistrar struct {
	called bool
	input  application.InvitationPasskeyRegistrationInput
}

func (r *stubInvitationPasskeyRegistrar) RegisterInvitationPasskey(_ context.Context, input application.InvitationPasskeyRegistrationInput) (application.AuthSession, error) {
	r.called = true
	r.input = input
	return application.AuthSession{}, application.ErrBadRequest
}

func (r *stubAuthStateRepository) ConsumeRecoverySession(_ context.Context, session domain.RecoverySession) error {
	r.recoverySessions[session.ID()] = session
	return nil
}

func (r *stubAuthStateRepository) IncrementThrottle(_ context.Context, key string, ttl time.Duration) (int, error) {
	record, ok := r.counters[key]
	now := r.clock()
	if !ok || now.After(record.expiresAt) {
		record = stubCounter{expiresAt: now.Add(ttl)}
	}
	record.count++
	r.counters[key] = record
	return record.count, nil
}

func (r *stubAuthStateRepository) SetLock(_ context.Context, key string, until time.Time, _ time.Duration) error {
	r.locks[key] = until
	return nil
}

func (r *stubAuthStateRepository) GetLock(_ context.Context, key string) (domain.AuthLock, bool, error) {
	until, ok := r.locks[key]
	if !ok {
		return domain.NewAuthLock(time.Time{}), false, nil
	}
	return domain.NewAuthLock(until), true, nil
}

func (r *stubAuthStateRepository) SaveReauthenticationSession(_ context.Context, session domain.ReauthenticationSession, _ time.Duration) error {
	if r.reauthSessions == nil {
		r.reauthSessions = map[string]domain.ReauthenticationSession{}
	}
	r.reauthSessions[session.ID()] = session
	return nil
}

func (r *stubAuthStateRepository) ConsumeReauthenticationSession(_ context.Context, reauthID string) (domain.ReauthenticationSession, error) {
	if r.reauthSessions == nil {
		return emptyReauthenticationSessionForTest(), domain.ErrReauthSessionNotFound
	}
	session, ok := r.reauthSessions[reauthID]
	if !ok {
		return emptyReauthenticationSessionForTest(), domain.ErrReauthSessionNotFound
	}
	delete(r.reauthSessions, reauthID)
	return session, nil
}

type stubAccountAuthRepository struct {
	account domain.AccountAuth
}

func stubAccountAuthRepositoryWithMember() *stubAccountAuthRepository {
	account, _ := domain.NewAccountAuth(testAccountID("01ARZ3NDEKTSV4RRFFQ69G5FAV"), "member@example.com", "member@example.com", "01ARZ3NDEKTSV4RRFFQ69G5FB0", "existing-credential")
	return &stubAccountAuthRepository{account: account}
}

func (r *stubAccountAuthRepository) FindByID(_ context.Context, accountID domain.AccountID) (domain.AccountAuth, error) {
	if accountID != r.account.AccountID() {
		return emptyAccountAuthForTest(), domain.ErrAccountAuthNotFound
	}
	return r.account, nil
}
func (r *stubAccountAuthRepository) FindByIdentifier(_ context.Context, identifier string) (domain.AccountAuth, error) {
	if identifier != r.account.Identifier() {
		return emptyAccountAuthForTest(), domain.ErrAccountAuthNotFound
	}
	return r.account, nil
}

func (r *stubAccountAuthRepository) FindByCredential(_ context.Context, credential string) (domain.AccountAuth, error) {
	if credential != firstCredentialHandleForTest(r.account) {
		return emptyAccountAuthForTest(), domain.ErrAccountAuthNotFound
	}
	return r.account, nil
}

func (r *stubAccountAuthRepository) FindByEmail(_ context.Context, email string) (domain.AccountAuth, error) {
	if email != r.account.Email() {
		return emptyAccountAuthForTest(), domain.ErrAccountAuthNotFound
	}
	return r.account, nil
}

func (r *stubAccountAuthRepository) AddPasskey(_ context.Context, accountID domain.AccountID, passkeyCredentialID string, credential string, _ domain.WebAuthnCredentialData) (domain.AccountAuth, error) {
	if accountID != r.account.AccountID() {
		return emptyAccountAuthForTest(), domain.ErrAccountAuthNotFound
	}
	newCred, err := domain.NewPasskeyCredential(passkeyCredentialID, accountID, r.account.Identifier(), credential, time.Now().UTC())
	if err != nil {
		return emptyAccountAuthForTest(), err
	}
	updated, err := domain.NewAccountAuthWithCredentials(
		r.account.AccountID(),
		r.account.Identifier(),
		r.account.Email(),
		append(r.account.Credentials(), newCred),
	)
	if err != nil {
		return emptyAccountAuthForTest(), err
	}
	r.account = updated
	return updated, nil
}

func (r *stubAccountAuthRepository) ListPasskeys(_ context.Context, accountID domain.AccountID) ([]domain.PasskeyCredential, error) {
	if accountID != r.account.AccountID() {
		return nil, domain.ErrAccountAuthNotFound
	}
	return r.account.Credentials(), nil
}

func (r *stubAccountAuthRepository) DeletePasskeyByID(_ context.Context, accountID domain.AccountID, credentialID string) error {
	if accountID != r.account.AccountID() {
		return domain.ErrAccountAuthNotFound
	}
	creds := r.account.Credentials()
	remaining := make([]domain.PasskeyCredential, 0, len(creds))
	found := false
	for _, c := range creds {
		if c.ID() == credentialID {
			found = true
			continue
		}
		remaining = append(remaining, c)
	}
	if !found {
		return domain.ErrAccountAuthNotFound
	}
	updated, err := domain.NewAccountAuthWithCredentials(r.account.AccountID(), r.account.Identifier(), r.account.Email(), remaining)
	if err != nil {
		return err
	}
	r.account = updated
	return nil
}

func (r *stubAccountAuthRepository) FindWebAuthnCredential(_ context.Context, handle string) (domain.WebAuthnStoredCredential, error) {
	for _, c := range r.account.Credentials() {
		if c.CredentialHandle() == handle {
			return domain.ReconstituteWebAuthnStoredCredential(handle, nil, 0, nil, false, false, nil), nil
		}
	}
	return domain.ZeroWebAuthnStoredCredential(), domain.ErrAccountAuthNotFound
}

func (r *stubAccountAuthRepository) UpdateWebAuthnCredentialState(_ context.Context, _ string, _ uint32, _ bool) error {
	return nil
}

func emptyChallengeForTest() domain.AuthChallenge {
	challenge, _ := domain.NewAuthChallenge("01ARZ3NDEKTSV4RRFFQ69G5FAV", "placeholder", "placeholder", time.Unix(0, 0).UTC())
	return challenge
}

func emptyRecoveryTokenForTest() domain.RecoveryToken {
	token, _ := domain.NewRecoveryToken("01ARZ3NDEKTSV4RRFFQ69G5FAV", testAccountID("01ARZ3NDEKTSV4RRFFQ69G5FAW"), "placeholder", domain.TokenKindRecovery, time.Unix(1, 0).UTC())
	return token
}
func emptyRecoverySessionForTest() domain.RecoverySession {
	session, _ := domain.NewRecoverySession("01ARZ3NDEKTSV4RRFFQ69G5FAV", testAccountID("01ARZ3NDEKTSV4RRFFQ69G5FAW"), domain.TokenKindRecovery, time.Unix(1, 0).UTC())
	return session
}
func emptyReauthenticationSessionForTest() domain.ReauthenticationSession {
	session, _ := domain.NewReauthenticationSession(
		"01ARZ3NDEKTSV4RRFFQ69G5FAV", testAccountID("01ARZ3NDEKTSV4RRFFQ69G5FAW"), "01ARZ3NDEKTSV4RRFFQ69G5FAX",
		"device-link", "01ARZ3NDEKTSV4RRFFQ69G5FAY", time.Unix(1, 0).UTC(),
	)
	return session
}

func emptyAccountAuthForTest() domain.AccountAuth {
	account, _ := domain.NewAccountAuth(testAccountID("01ARZ3NDEKTSV4RRFFQ69G5FAV"), "placeholder@example.com", "placeholder@example.com", "01ARZ3NDEKTSV4RRFFQ69G5FAW", "placeholder")
	return account
}

func firstCredentialHandleForTest(account domain.AccountAuth) string {
	credentials := account.Credentials()
	if len(credentials) == 0 {
		return ""
	}
	return credentials[0].CredentialHandle()
}

type failingAuthStateRepository struct{}

func (failingAuthStateRepository) SaveChallenge(context.Context, domain.AuthChallenge, time.Duration) error {
	return domain.ErrAuthStoreUnavailable
}
func (failingAuthStateRepository) ConsumeChallenge(context.Context, string) (domain.AuthChallenge, error) {
	return emptyChallengeForTest(), domain.ErrAuthStoreUnavailable
}
func (failingAuthStateRepository) IssueRecoveryToken(context.Context, domain.RecoveryToken, time.Duration) error {
	return domain.ErrAuthStoreUnavailable
}
func (failingAuthStateRepository) SaveRecoveryDeliveryFailure(context.Context, domain.RecoveryDeliveryFailure, time.Duration) error {
	return domain.ErrAuthStoreUnavailable
}
func (failingAuthStateRepository) ConsumeRecoveryTokenAtomic(context.Context, string, string) (domain.RecoveryToken, error) {
	return emptyRecoveryTokenForTest(), domain.ErrAuthStoreUnavailable
}
func (failingAuthStateRepository) SaveRecoverySession(context.Context, domain.RecoverySession, time.Duration) error {
	return domain.ErrAuthStoreUnavailable
}
func (failingAuthStateRepository) GetRecoverySession(context.Context, string) (domain.RecoverySession, error) {
	return emptyRecoverySessionForTest(), domain.ErrAuthStoreUnavailable
}
func (failingAuthStateRepository) IncrementThrottle(context.Context, string, time.Duration) (int, error) {
	return 0, domain.ErrAuthStoreUnavailable
}
func (failingAuthStateRepository) SetLock(context.Context, string, time.Time, time.Duration) error {
	return domain.ErrAuthStoreUnavailable
}
func (failingAuthStateRepository) GetLock(context.Context, string) (domain.AuthLock, bool, error) {
	return domain.NewAuthLock(time.Time{}), false, domain.ErrAuthStoreUnavailable
}
func (failingAuthStateRepository) ConsumeRecoverySession(context.Context, domain.RecoverySession) error {
	return domain.ErrAuthStoreUnavailable
}
func (failingAuthStateRepository) SaveReauthenticationSession(context.Context, domain.ReauthenticationSession, time.Duration) error {
	return domain.ErrAuthStoreUnavailable
}
func (failingAuthStateRepository) ConsumeReauthenticationSession(context.Context, string) (domain.ReauthenticationSession, error) {
	return emptyReauthenticationSessionForTest(), domain.ErrAuthStoreUnavailable
}

// ─── Multi-passkey management integration tests ─────────────────────────────

// loginWithPasskey はパスキー認証フローを実行して JWT アクセストークンを返す helper。
func loginWithPasskey(t *testing.T, router *gin.Engine, identifier string) string {
	t.Helper()
	challenge := startPasskey(t, router, identifier)
	resp := performJSON(t, router, stdhttp.MethodPost, "/api/v1/auth/passkey/finish",
		map[string]any{"credential": assertionCredentialJSON("existing-credential", challengeValue(challenge))}, "")
	assertStatus(t, resp, stdhttp.StatusOK)
	var session map[string]any
	decodeJSON(t, resp, &session)
	token, _ := session["accessToken"].(string)
	return token
}

// multiAccountStubAccountAuthRepository は 2 アカウント以上を保持できる stub。
type multiAccountStubAccountAuthRepository struct {
	accounts map[string]*stubAccountAuthRepository
}

func newMultiAccountStubAccountAuthRepository(accounts ...*stubAccountAuthRepository) *multiAccountStubAccountAuthRepository {
	m := &multiAccountStubAccountAuthRepository{accounts: map[string]*stubAccountAuthRepository{}}
	for _, a := range accounts {
		m.accounts[a.account.AccountID().String()] = a
	}
	return m
}

func (m *multiAccountStubAccountAuthRepository) repoByID(id domain.AccountID) (*stubAccountAuthRepository, bool) {
	r, ok := m.accounts[id.String()]
	return r, ok
}

func (m *multiAccountStubAccountAuthRepository) repoByIdentifier(id string) (*stubAccountAuthRepository, bool) {
	for _, r := range m.accounts {
		if r.account.Identifier() == id {
			return r, true
		}
	}
	return nil, false
}

func (m *multiAccountStubAccountAuthRepository) repoByCredentialHandle(handle string) (*stubAccountAuthRepository, bool) {
	for _, r := range m.accounts {
		for _, cred := range r.account.Credentials() {
			if cred.CredentialHandle() == handle {
				return r, true
			}
		}
	}
	return nil, false
}

func (m *multiAccountStubAccountAuthRepository) FindByID(ctx context.Context, accountID domain.AccountID) (domain.AccountAuth, error) {
	r, ok := m.repoByID(accountID)
	if !ok {
		return emptyAccountAuthForTest(), domain.ErrAccountAuthNotFound
	}
	return r.FindByID(ctx, accountID)
}
func (m *multiAccountStubAccountAuthRepository) FindByIdentifier(ctx context.Context, identifier string) (domain.AccountAuth, error) {
	r, ok := m.repoByIdentifier(identifier)
	if !ok {
		return emptyAccountAuthForTest(), domain.ErrAccountAuthNotFound
	}
	return r.FindByIdentifier(ctx, identifier)
}

func (m *multiAccountStubAccountAuthRepository) FindByCredential(ctx context.Context, credential string) (domain.AccountAuth, error) {
	r, ok := m.repoByCredentialHandle(credential)
	if !ok {
		return emptyAccountAuthForTest(), domain.ErrAccountAuthNotFound
	}
	return r.FindByCredential(ctx, credential)
}

func (m *multiAccountStubAccountAuthRepository) FindByEmail(ctx context.Context, email string) (domain.AccountAuth, error) {
	for _, r := range m.accounts {
		if r.account.Email() == email {
			return r.FindByEmail(ctx, email)
		}
	}
	return emptyAccountAuthForTest(), domain.ErrAccountAuthNotFound
}

func (m *multiAccountStubAccountAuthRepository) AddPasskey(ctx context.Context, accountID domain.AccountID, passkeyCredentialID string, credential string, credData domain.WebAuthnCredentialData) (domain.AccountAuth, error) {
	r, ok := m.repoByID(accountID)
	if !ok {
		return emptyAccountAuthForTest(), domain.ErrAccountAuthNotFound
	}
	return r.AddPasskey(ctx, accountID, passkeyCredentialID, credential, credData)
}

func (m *multiAccountStubAccountAuthRepository) ListPasskeys(ctx context.Context, accountID domain.AccountID) ([]domain.PasskeyCredential, error) {
	r, ok := m.repoByID(accountID)
	if !ok {
		return nil, domain.ErrAccountAuthNotFound
	}
	return r.ListPasskeys(ctx, accountID)
}

func (m *multiAccountStubAccountAuthRepository) DeletePasskeyByID(ctx context.Context, accountID domain.AccountID, credentialID string) error {
	r, ok := m.repoByID(accountID)
	if !ok {
		return domain.ErrAccountAuthNotFound
	}
	return r.DeletePasskeyByID(ctx, accountID, credentialID)
}

func (m *multiAccountStubAccountAuthRepository) FindWebAuthnCredential(ctx context.Context, handle string) (domain.WebAuthnStoredCredential, error) {
	r, ok := m.repoByCredentialHandle(handle)
	if !ok {
		return domain.ZeroWebAuthnStoredCredential(), domain.ErrAccountAuthNotFound
	}
	return r.FindWebAuthnCredential(ctx, handle)
}

func (m *multiAccountStubAccountAuthRepository) UpdateWebAuthnCredentialState(_ context.Context, _ string, _ uint32, _ bool) error {
	return nil
}

// stubAccountAuthRepositoryWithTwoCredentials は 2 件の credential を持つ account stub を生成する。
func stubAccountAuthRepositoryWithTwoCredentials(accountID domain.AccountID, identifier string, email string, cred1ID string, cred1Handle string, cred2ID string, cred2Handle string) *stubAccountAuthRepository {
	now := time.Date(2026, time.March, 21, 0, 0, 0, 0, time.UTC)
	c1, _ := domain.NewPasskeyCredential(cred1ID, accountID, identifier, cred1Handle, now)
	c2, _ := domain.NewPasskeyCredential(cred2ID, accountID, identifier, cred2Handle, now.Add(time.Minute))
	account, _ := domain.NewAccountAuthWithCredentials(accountID, identifier, email, []domain.PasskeyCredential{c1, c2})
	return &stubAccountAuthRepository{account: account}
}

// [AUTH-BE-S014] GET /api/v1/passkeys が登録済みパスキー一覧を返す
func TestListPasskeysReturnsCredentials(t *testing.T) {
	t.Parallel()
	env := newAuthTestEnv(t)
	token := loginWithPasskey(t, env.router, "member@example.com")

	resp := performJSON(t, env.router, stdhttp.MethodGet, "/api/v1/passkeys", nil, token)

	assertStatus(t, resp, stdhttp.StatusOK)
	assertNoStore(t, resp)
	var body map[string]any
	decodeJSON(t, resp, &body)
	assertULIDField(t, body, "requestId")
	passkeys, ok := body["passkeys"].([]any)
	if !ok {
		t.Fatalf("expected passkeys array, got %#v", body["passkeys"])
	}
	if len(passkeys) != 1 {
		t.Fatalf("expected 1 passkey, got %d", len(passkeys))
	}
}

// [AUTH-BE-S015] パスキー追加後に既存パスキーが保持される
func TestFinishPasskeyAdditionPreservesExistingPasskeys(t *testing.T) {
	t.Parallel()
	env := newAuthTestEnv(t)
	token := loginWithPasskey(t, env.router, "member@example.com")

	// start: チャレンジを取得
	startResp := performJSON(t, env.router, stdhttp.MethodPost, "/api/v1/passkeys/start", nil, token)
	assertStatus(t, startResp, stdhttp.StatusOK)
	var startBody map[string]any
	decodeJSON(t, startResp, &startBody)

	// finish: 新しい credential を追加
	finishResp := performJSON(t, env.router, stdhttp.MethodPost, "/api/v1/passkeys/finish",
		map[string]any{"credential": attestationCredentialJSON("new-credential", challengeValue(startBody))},
		token)
	assertStatus(t, finishResp, stdhttp.StatusOK)
	assertNoStore(t, finishResp)
	var finishBody map[string]any
	decodeJSON(t, finishResp, &finishBody)
	passkeys, ok := finishBody["passkeys"].([]any)
	if !ok {
		t.Fatalf("expected passkeys array, got %#v", finishBody["passkeys"])
	}
	if len(passkeys) != 2 {
		t.Fatalf("expected 2 passkeys after addition, got %d", len(passkeys))
	}
}

// [AUTH-BE-S016] 最終 1 件の削除が 409 を返す
func TestDeleteLastPasskeyReturns409(t *testing.T) {
	t.Parallel()
	env := newAuthTestEnv(t)
	token := loginWithPasskey(t, env.router, "member@example.com")
	session, _ := env.auth.AuthorizeSession(context.Background(), token)
	saveTestReauthSession(t, env.stateRepo, session.AccountID, session.SessionID, "passkey-delete", env.now)

	// existing-credential の passkeyCredentialID は "01ARZ3NDEKTSV4RRFFQ69G5FB0"
	resp := performJSONWithHeaders(t, env.router, stdhttp.MethodDelete, "/api/v1/passkeys/01ARZ3NDEKTSV4RRFFQ69G5FB0", nil, token, map[string]string{"X-Reauth-Session": "01ARZ3NDEKTSV4RRFFQ69G5FAY"})

	assertStatus(t, resp, stdhttp.StatusConflict)
	assertNoStore(t, resp)
}

// [AUTH-BE-S017] 2 件中 1 件の削除が正しく動作する
func TestDeleteOneOfTwoPasskeysSucceeds(t *testing.T) {
	t.Parallel()
	clock := &mutableClock{current: time.Date(2026, time.March, 21, 0, 0, 0, 0, time.UTC)}
	stateRepo := newStubAuthStateRepository(clock.Now)
	accountRepo := stubAccountAuthRepositoryWithTwoCredentials(
		"01ARZ3NDEKTSV4RRFFQ69G5FAV", "member@example.com", "member@example.com",
		"01ARZ3NDEKTSV4RRFFQ69G5FB0", "existing-credential",
		"01ARZ3NDEKTSV4RRFFQ69G5FB1", "second-credential",
	)
	lifecycle, tokenService := newTestProductAccountLifecycle(t, accountRepo, clock.Now, testConfig().AuthRuntime().RefreshTokenTTL)
	auth := mustNewProductAuthForTest(t, stateRepo, accountRepo, &capturingAccountRecoverySender{}, &stubInvitationPasskeyRegistrar{}, lifecycle, clock.Now, testConfig().AuthRuntime(), application.AuthServiceOptionalPorts{WebAuthn: newMockWebAuthnProvider()})
	sessionService := application.NewProductSessionService(lifecycle)
	router := NewRouter(testConfig(), Dependencies{Auth: auth, TokenService: tokenService, SessionService: sessionService})

	token := loginWithPasskey(t, router, "member@example.com")
	session, _ := auth.AuthorizeSession(context.Background(), token)
	saveTestReauthSession(t, stateRepo, session.AccountID, session.SessionID, "passkey-delete", clock.Now)

	resp := performJSONWithHeaders(t, router, stdhttp.MethodDelete, "/api/v1/passkeys/01ARZ3NDEKTSV4RRFFQ69G5FB1", nil, token, map[string]string{"X-Reauth-Session": "01ARZ3NDEKTSV4RRFFQ69G5FAY"})

	assertStatus(t, resp, stdhttp.StatusNoContent)
	assertNoStore(t, resp)

	// 残り 1 件を確認
	listResp := performJSON(t, router, stdhttp.MethodGet, "/api/v1/passkeys", nil, token)
	assertStatus(t, listResp, stdhttp.StatusOK)
	var listBody map[string]any
	decodeJSON(t, listResp, &listBody)
	passkeys, _ := listBody["passkeys"].([]any)
	if len(passkeys) != 1 {
		t.Fatalf("expected 1 passkey after deletion, got %d", len(passkeys))
	}
}

// [AUTH-BE-S018] 他アカウントのパスキー削除が 403 を返す
func TestDeleteOtherAccountPasskeyReturns403(t *testing.T) {
	t.Parallel()
	clock := &mutableClock{current: time.Date(2026, time.March, 21, 0, 0, 0, 0, time.UTC)}
	stateRepo := newStubAuthStateRepository(clock.Now)

	// account1: member@example.com (2 credentials で削除可能)
	account1 := stubAccountAuthRepositoryWithTwoCredentials(
		"01ARZ3NDEKTSV4RRFFQ69G5FAV", "member@example.com", "member@example.com",
		"01ARZ3NDEKTSV4RRFFQ69G5FB0", "existing-credential",
		"01ARZ3NDEKTSV4RRFFQ69G5FB1", "second-credential",
	)
	// account2: other@example.com (1 credential)
	account2Cred, _ := domain.NewPasskeyCredential("01ARZ3NDEKTSV4RRFFQ69G5FBC", "01ARZ3NDEKTSV4RRFFQ69G5FBD", "other@example.com", "other-credential", time.Date(2026, time.March, 21, 0, 0, 0, 0, time.UTC))
	account2Account, _ := domain.NewAccountAuthWithCredentials(testAccountID("01ARZ3NDEKTSV4RRFFQ69G5FBD"), "other@example.com", "other@example.com", []domain.PasskeyCredential{account2Cred})
	account2 := &stubAccountAuthRepository{account: account2Account}

	accountRepo := newMultiAccountStubAccountAuthRepository(account1, account2)
	lifecycle, tokenService := newTestProductAccountLifecycle(t, accountRepo, clock.Now, testConfig().AuthRuntime().RefreshTokenTTL)
	auth := mustNewProductAuthForTest(t, stateRepo, accountRepo, &capturingAccountRecoverySender{}, &stubInvitationPasskeyRegistrar{}, lifecycle, clock.Now, testConfig().AuthRuntime(), application.AuthServiceOptionalPorts{WebAuthn: newMockWebAuthnProvider()})
	sessionService := application.NewProductSessionService(lifecycle)
	router := NewRouter(testConfig(), Dependencies{Auth: auth, TokenService: tokenService, SessionService: sessionService})

	// account1 でログイン
	token := loginWithPasskey(t, router, "member@example.com")
	session, _ := auth.AuthorizeSession(context.Background(), token)
	saveTestReauthSession(t, stateRepo, session.AccountID, session.SessionID, "passkey-delete", clock.Now)

	// account2 の credential を account1 セッションで削除しようとする
	resp := performJSONWithHeaders(t, router, stdhttp.MethodDelete, "/api/v1/passkeys/01ARZ3NDEKTSV4RRFFQ69G5FBC", nil, token, map[string]string{"X-Reauth-Session": "01ARZ3NDEKTSV4RRFFQ69G5FAY"})

	assertStatus(t, resp, stdhttp.StatusForbidden)
	assertNoStore(t, resp)
}

// [AUTH-BE-S019] 未認証リクエストが 401 を返す
func TestPasskeyManagementUnauthenticatedReturns401(t *testing.T) {
	t.Parallel()
	env := newAuthTestEnv(t)

	for _, tc := range []struct {
		method string
		path   string
		body   any
	}{
		{stdhttp.MethodGet, "/api/v1/passkeys", nil},
		{stdhttp.MethodPost, "/api/v1/passkeys/start", nil},
		{stdhttp.MethodPost, "/api/v1/passkeys/finish", map[string]any{"credential": attestationCredentialJSON("x", "y")}},
		{stdhttp.MethodDelete, "/api/v1/passkeys/01ARZ3NDEKTSV4RRFFQ69G5FB0", nil},
	} {
		resp := performJSON(t, env.router, tc.method, tc.path, tc.body, "")
		assertStatus(t, resp, stdhttp.StatusUnauthorized)
		assertNoStore(t, resp)
	}
}

// [AUTH-BE-S020] パスキー追加後に既存パスキーが保持される（回帰）
func TestFinishPasskeyAdditionRetainsExistingOnList(t *testing.T) {
	t.Parallel()
	env := newAuthTestEnv(t)
	token := loginWithPasskey(t, env.router, "member@example.com")

	startResp := performJSON(t, env.router, stdhttp.MethodPost, "/api/v1/passkeys/start", nil, token)
	assertStatus(t, startResp, stdhttp.StatusOK)
	var startBody map[string]any
	decodeJSON(t, startResp, &startBody)

	performJSON(t, env.router, stdhttp.MethodPost, "/api/v1/passkeys/finish",
		map[string]any{"credential": attestationCredentialJSON("reg-credential", challengeValue(startBody))},
		token)

	listResp := performJSON(t, env.router, stdhttp.MethodGet, "/api/v1/passkeys", nil, token)
	assertStatus(t, listResp, stdhttp.StatusOK)
	var listBody map[string]any
	decodeJSON(t, listResp, &listBody)
	passkeys, _ := listBody["passkeys"].([]any)
	if len(passkeys) != 2 {
		t.Fatalf("expected 2 passkeys, got %d", len(passkeys))
	}
}

// assertPasskeyAddStartCreationFields は PasskeyAddStartResponse の
// rpName / user / pubKeyCredParams が正しく設定されていることを検証する共通ヘルパー。
func assertPasskeyAddStartCreationFields(t *testing.T, body map[string]any) {
	t.Helper()
	assertNonEmptyStringField(t, body, "rpName")
	user, ok := body["user"].(map[string]any)
	if !ok || user == nil {
		t.Fatalf("expected user object, got %v", body["user"])
	}
	assertNonEmptyStringField(t, user, "id")
	assertNonEmptyStringField(t, user, "name")
	assertNonEmptyStringField(t, user, "displayName")
	params, ok := body["pubKeyCredParams"].([]any)
	if !ok || len(params) == 0 {
		t.Fatalf("expected non-empty pubKeyCredParams, got %v", body["pubKeyCredParams"])
	}
	for i, p := range params {
		assertCredentialParam(t, i, p)
	}
	if body["residentKey"] != "required" || body["requireResidentKey"] != true || body["userVerification"] != "required" {
		t.Fatalf("expected discoverable credential selection fields, got %#v", body)
	}
}

func assertNonEmptyStringField(t *testing.T, m map[string]any, key string) {
	t.Helper()
	if m[key] == "" || m[key] == nil {
		t.Errorf("expected non-empty %s, got %v", key, m[key])
	}
}

func assertCredentialParam(t *testing.T, idx int, p any) {
	t.Helper()
	param, ok := p.(map[string]any)
	if !ok {
		t.Fatalf("pubKeyCredParams[%d] is not an object: %v", idx, p)
	}
	if param["type"] == "" || param["type"] == nil {
		t.Errorf("pubKeyCredParams[%d].type is empty", idx)
	}
	if param["alg"] == nil {
		t.Errorf("pubKeyCredParams[%d].alg is nil", idx)
	}
}

// [AUTH-BE-S025] StartPasskeyRegistration (recovery path) が rpName / user / pubKeyCredParams を返す
func TestStartPasskeyRegistrationReturnsWebAuthnCreationFields(t *testing.T) {
	t.Parallel()
	env := newAuthTestEnv(t)
	recoverySession := consumeRecoverySession(t, env)

	resp := performJSON(t, env.router, stdhttp.MethodPost, "/api/v1/auth/passkey/register/start",
		map[string]any{"recovery_session": recoverySession}, "")
	assertStatus(t, resp, stdhttp.StatusOK)
	assertNoStore(t, resp)

	var body map[string]any
	decodeJSON(t, resp, &body)
	assertULIDField(t, body, "requestId")
	assertPasskeyAddStartCreationFields(t, body)
}

// BeginRegistration が必須フィールドを返さない場合、register/start は 503 を返す
func TestStartPasskeyRegistrationIncompleteOptionsReturns503(t *testing.T) {
	t.Parallel()
	clock := &mutableClock{current: time.Date(2026, time.March, 21, 0, 0, 0, 0, time.UTC)}
	stateRepo := newStubAuthStateRepository(clock.Now)
	accountRepo := stubAccountAuthRepositoryWithMember()
	invite := &stubInvitationPasskeyRegistrar{}
	sender := &capturingAccountRecoverySender{}
	auth := mustNewProductAuthForTest(t, stateRepo, accountRepo, sender, invite, nil, clock.Now, testConfig().AuthRuntime(), application.AuthServiceOptionalPorts{WebAuthn: newMockWebAuthnProviderWithIncompleteOptions()})
	router := NewRouter(testConfig(), Dependencies{Auth: auth})
	env := authTestEnv{router: router, stateRepo: stateRepo, sender: sender, invite: invite, now: clock.Now, advance: clock.Advance}

	recoverySession := consumeRecoverySession(t, env)
	resp := performJSON(t, router, stdhttp.MethodPost, "/api/v1/auth/passkey/register/start",
		map[string]any{"recovery_session": recoverySession}, "")
	assertStatus(t, resp, stdhttp.StatusServiceUnavailable)
	assertNoStore(t, resp)
}

// StartPasskeyAddition (bearer) が rpName / user / pubKeyCredParams を返す
func TestStartPasskeyAdditionReturnsWebAuthnCreationFields(t *testing.T) {
	t.Parallel()
	env := newAuthTestEnv(t)
	token := loginWithPasskey(t, env.router, "member@example.com")

	resp := performJSON(t, env.router, stdhttp.MethodPost, "/api/v1/passkeys/start", nil, token)
	assertStatus(t, resp, stdhttp.StatusOK)
	assertNoStore(t, resp)

	var body map[string]any
	decodeJSON(t, resp, &body)
	assertULIDField(t, body, "requestId")
	assertPasskeyAddStartCreationFields(t, body)
}

// newAuthEnvWithCustomProvider は指定した WebAuthn provider を使う authTestEnv を構築する。
func newAuthEnvWithCustomProvider(t *testing.T, provider application.WebAuthnProvider) authTestEnv {
	t.Helper()
	clock := &mutableClock{current: time.Date(2026, time.March, 21, 0, 0, 0, 0, time.UTC)}
	stateRepo := newStubAuthStateRepository(clock.Now)
	accountRepo := stubAccountAuthRepositoryWithMember()
	invite := &stubInvitationPasskeyRegistrar{}
	sender := &capturingAccountRecoverySender{}
	auth := mustNewProductAuthForTest(t, stateRepo, accountRepo, sender, invite, nil, clock.Now, testConfig().AuthRuntime(), application.AuthServiceOptionalPorts{WebAuthn: provider})
	router := NewRouter(testConfig(), Dependencies{Auth: auth})
	return authTestEnv{router: router, stateRepo: stateRepo, sender: sender, invite: invite, now: clock.Now, advance: clock.Advance}
}

// user.name が空の場合、register/start は 503 を返す
func TestStartPasskeyRegistrationMissingUserNameReturns503(t *testing.T) {
	t.Parallel()
	// user.name を空にした options
	const opts = `{"publicKey":{"rp":{"id":"localhost","name":"Test RP"},"user":{"id":"dXNlcmlk","name":"","displayName":"Test User"},"challenge":"__KEY__","pubKeyCredParams":[{"type":"public-key","alg":-7}]}}`
	env := newAuthEnvWithCustomProvider(t, newMockWebAuthnProviderWithOptions(opts))
	recoverySession := consumeRecoverySession(t, env)
	resp := performJSON(t, env.router, stdhttp.MethodPost, "/api/v1/auth/passkey/register/start",
		map[string]any{"recovery_session": recoverySession}, "")
	assertStatus(t, resp, stdhttp.StatusServiceUnavailable)
	assertNoStore(t, resp)
}

// user.displayName が空の場合、register/start は 503 を返す
func TestStartPasskeyRegistrationMissingDisplayNameReturns503(t *testing.T) {
	t.Parallel()
	// user.displayName を空にした options
	const opts = `{"publicKey":{"rp":{"id":"localhost","name":"Test RP"},"user":{"id":"dXNlcmlk","name":"testuser","displayName":""},"challenge":"__KEY__","pubKeyCredParams":[{"type":"public-key","alg":-7}]}}`
	env := newAuthEnvWithCustomProvider(t, newMockWebAuthnProviderWithOptions(opts))
	recoverySession := consumeRecoverySession(t, env)
	resp := performJSON(t, env.router, stdhttp.MethodPost, "/api/v1/auth/passkey/register/start",
		map[string]any{"recovery_session": recoverySession}, "")
	assertStatus(t, resp, stdhttp.StatusServiceUnavailable)
	assertNoStore(t, resp)
}

// [AUTH-BE-S026] oversized public auth request は state mutation 前に拒否される
func TestOversizedPublicAuthRequestRejectedBeforeStateMutation(t *testing.T) {
	t.Parallel()
	clock := &mutableClock{current: time.Date(2026, time.March, 21, 0, 0, 0, 0, time.UTC)}
	stateRepo := newStubAuthStateRepository(clock.Now)
	accountRepo := stubAccountAuthRepositoryWithMember()
	auth := mustNewProductAuthForTest(t, stateRepo, accountRepo, &capturingAccountRecoverySender{}, &stubInvitationPasskeyRegistrar{}, nil, clock.Now, testConfig().AuthRuntime(), application.AuthServiceOptionalPorts{WebAuthn: newMockWebAuthnProvider()})

	cfg := testConfig()
	cfg.Auth.AuthBodyLimitBytes = 10 // 10 バイト上限
	router := NewRouter(cfg, Dependencies{Auth: auth})

	challengesBefore := len(stateRepo.challenges)

	// 10 バイトを超える JSON body を送信する
	resp := performJSON(t, router, stdhttp.MethodPost, "/api/v1/auth/passkey/start", map[string]string{"identifier": "member@example.com"}, "")

	// MaxBytesReader 超過により binding エラーとなり 400 が返る
	if resp.Code != stdhttp.StatusBadRequest {
		t.Fatalf("expected status 400 for oversized body, got %d body=%s", resp.Code, resp.Body.String())
	}
	assertNoStore(t, resp)
	if len(stateRepo.challenges) != challengesBefore {
		t.Fatalf("expected no challenge issuance for oversized request, before=%d after=%d", challengesBefore, len(stateRepo.challenges))
	}
}

// [AUTH-BE-S037] bearer-only passkey deletion は再認証セッションなしで拒否される
// OpenAPI generated code は必須ヘッダー欠落時に 400 を返す（handler 到達前に検証される）。
func TestDeletePasskeyWithoutReauthSessionRejected(t *testing.T) {
	t.Parallel()
	clock := &mutableClock{current: time.Date(2026, time.March, 21, 0, 0, 0, 0, time.UTC)}
	stateRepo := newStubAuthStateRepository(clock.Now)
	accountRepo := stubAccountAuthRepositoryWithTwoCredentials(
		"01ARZ3NDEKTSV4RRFFQ69G5FAV", "member@example.com", "member@example.com",
		"01ARZ3NDEKTSV4RRFFQ69G5FB0", "existing-credential",
		"01ARZ3NDEKTSV4RRFFQ69G5FB1", "second-credential",
	)
	lifecycle, tokenService := newTestProductAccountLifecycle(t, accountRepo, clock.Now, testConfig().AuthRuntime().RefreshTokenTTL)
	auth := mustNewProductAuthForTest(t, stateRepo, accountRepo, &capturingAccountRecoverySender{}, &stubInvitationPasskeyRegistrar{}, lifecycle, clock.Now, testConfig().AuthRuntime(), application.AuthServiceOptionalPorts{WebAuthn: newMockWebAuthnProvider()})
	sessionService := application.NewProductSessionService(lifecycle)
	router := NewRouter(testConfig(), Dependencies{Auth: auth, TokenService: tokenService, SessionService: sessionService})

	token := loginWithPasskey(t, router, "member@example.com")

	// X-Reauth-Session なしで削除を試みる
	resp := performJSON(t, router, stdhttp.MethodDelete, "/api/v1/passkeys/01ARZ3NDEKTSV4RRFFQ69G5FB1", nil, token)

	// OpenAPI ミドルウェアが必須ヘッダー欠落を 400 で拒否する
	assertStatus(t, resp, stdhttp.StatusBadRequest)
	assertNoStore(t, resp)
}
