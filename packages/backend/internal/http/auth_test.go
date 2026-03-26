package http

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

	"www-template/packages/backend/internal/domain"
	"www-template/packages/backend/internal/types"
	"www-template/packages/backend/internal/usecases"
)

var ulidRegex = regexp.MustCompile(`^[0-9A-HJKMNP-TV-Z]{26}$`)

func TestAuthPasskeyFinishIssuesSession(t *testing.T) {
	t.Parallel()
	env := newAuthTestEnv(t)
	challenge := startPasskey(t, env.router, "member@example.com")
	response := performJSON(t, env.router, stdhttp.MethodPost, "/api/v1/auth/passkey/finish", map[string]string{"credential": credentialEnvelope("existing-credential", challengeValue(challenge))}, "")
	assertStatus(t, response, stdhttp.StatusOK)
	assertNoStore(t, response)

	var session map[string]any
	decodeJSON(t, response, &session)
	assertULIDField(t, session, "accountId")
	assertULIDField(t, session, "passkeyCredentialId")
	assertULIDField(t, session, "sessionId")

	appResponse := performJSON(t, env.router, stdhttp.MethodPost, "/api/v1/app/auth/logout", nil, session["sessionToken"].(string))
	assertStatus(t, appResponse, stdhttp.StatusOK)
}

func TestAuthInactiveSessionRejected(t *testing.T) {
	t.Parallel()
	env := newAuthTestEnv(t)
	challenge := startPasskey(t, env.router, "member@example.com")
	finishResponse := performJSON(t, env.router, stdhttp.MethodPost, "/api/v1/auth/passkey/finish", map[string]string{"credential": credentialEnvelope("existing-credential", challengeValue(challenge))}, "")
	assertStatus(t, finishResponse, stdhttp.StatusOK)
	assertNoStore(t, finishResponse)
	var session map[string]any
	decodeJSON(t, finishResponse, &session)
	env.advance(15 * 24 * time.Hour)

	response := performJSON(t, env.router, stdhttp.MethodPost, "/api/v1/app/auth/logout", nil, session["sessionToken"].(string))
	assertStatus(t, response, stdhttp.StatusUnauthorized)
	assertNoStore(t, response)
	assertFailureCode(t, response, "session-expired")
}

func TestAuthLogoutRevokesSession(t *testing.T) {
	t.Parallel()
	env := newAuthTestEnv(t)
	challenge := startPasskey(t, env.router, "member@example.com")
	finishResponse := performJSON(t, env.router, stdhttp.MethodPost, "/api/v1/auth/passkey/finish", map[string]string{"credential": credentialEnvelope("existing-credential", challengeValue(challenge))}, "")
	assertStatus(t, finishResponse, stdhttp.StatusOK)
	assertNoStore(t, finishResponse)
	var session map[string]any
	decodeJSON(t, finishResponse, &session)

	logoutResponse := performJSON(t, env.router, stdhttp.MethodPost, "/api/v1/app/auth/logout", nil, session["sessionToken"].(string))
	assertStatus(t, logoutResponse, stdhttp.StatusOK)
	assertNoStore(t, logoutResponse)

	appResponse := performJSON(t, env.router, stdhttp.MethodPost, "/api/v1/app/auth/logout", nil, session["sessionToken"].(string))
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
	response := performJSON(t, env.router, stdhttp.MethodPost, "/api/v1/app/auth/logout", nil, "")
	assertStatus(t, response, stdhttp.StatusUnauthorized)
	assertNoStore(t, response)
	assertFailureCode(t, response, "unauthenticated")
}

func TestAuthStoreOutageFailsClosed(t *testing.T) {
	t.Parallel()
	env := newFailClosedAuthEnv(t)
	response := performJSON(t, env.router, stdhttp.MethodPost, "/api/v1/app/auth/logout", nil, "opaque-token")
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
	accountRepo := stubAuthAccountRepositoryWithMember()
	cfg := testConfig()
	cfg.Auth.WebAuthnRPID = "www-template"
	auth := usecases.NewAuthService(stateRepo, accountRepo, &capturingAccountRecoverySender{}, &stubInvitationPasskeyRegistrar{}, clock.Now, newSequentialPolicy(), cfg.AuthRuntime())
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

func TestAuthFailureResponsesIssueDistinctRequestIDs(t *testing.T) {
	t.Parallel()
	env := newAuthTestEnv(t)
	first := performJSON(t, env.router, stdhttp.MethodPost, "/api/v1/app/auth/logout", nil, "")
	second := performJSON(t, env.router, stdhttp.MethodPost, "/api/v1/app/auth/logout", nil, "")
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
	if !ulidRegex.MatchString(recorded.AccountID()) {
		t.Fatalf("expected account id ULID, got %q", recorded.AccountID())
	}
}

func TestAuthRecoverySendFailurePersistsOriginalTokenExpiryAUTHBES011(t *testing.T) {
	t.Parallel()
	clock := &mutableClock{current: time.Date(2026, time.March, 21, 0, 0, 0, 0, time.UTC)}
	stateRepo := newStubAuthStateRepository(clock.Now)
	accountRepo := stubAuthAccountRepositoryWithMember()
	sender := advancingFailingAccountRecoverySender{advance: func() { clock.Advance(2 * time.Minute) }, err: errors.New("smtp delayed failure")}
	auth := usecases.NewAuthService(stateRepo, accountRepo, sender, &stubInvitationPasskeyRegistrar{}, clock.Now, newSequentialPolicy(), testConfig().AuthRuntime())
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
	response := performJSON(t, env.router, stdhttp.MethodPost, "/api/v1/auth/passkey/register", map[string]string{"recovery_session": recoverySession, "credential": "new-credential"}, "")
	assertStatus(t, response, stdhttp.StatusOK)
	assertNoStore(t, response)
	var body map[string]any
	decodeJSON(t, response, &body)
	assertULIDField(t, body, "accountId")
	assertULIDField(t, body, "passkeyCredentialId")
}

func TestAuthInviteOnlyCannotRegisterRecovery(t *testing.T) {
	t.Parallel()
	env := newAuthTestEnv(t)
	response := performJSON(t, env.router, stdhttp.MethodPost, "/api/v1/auth/passkey/register", map[string]string{"credential": "invite-only", "invitation_session": "01ARZ3NDEKTSV4RRFFQ69G5FC1"}, "")
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
	cases := []map[string]string{
		{"credential": "only-credential"},
		{"credential": "both", "invitation_session": "01ARZ3NDEKTSV4RRFFQ69G5FC2", "recovery_session": "01ARZ3NDEKTSV4RRFFQ69G5FC3"},
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

type authTestEnv struct {
	router    *gin.Engine
	stateRepo *stubAuthStateRepository
	sender    *capturingAccountRecoverySender
	invite    *stubInvitationPasskeyRegistrar
	now       func() time.Time
	advance   func(time.Duration)
}

func newAuthTestEnv(t *testing.T) authTestEnv {
	t.Helper()
	return newAuthTestEnvWithSender(t, &capturingAccountRecoverySender{})
}

func newAuthTestEnvWithSender(t *testing.T, sender usecases.AccountRecoverySender) authTestEnv {
	t.Helper()
	clock := &mutableClock{current: time.Date(2026, time.March, 21, 0, 0, 0, 0, time.UTC)}
	stateRepo := newStubAuthStateRepository(clock.Now)
	accountRepo := stubAuthAccountRepositoryWithMember()
	invite := &stubInvitationPasskeyRegistrar{}
	auth := usecases.NewAuthService(stateRepo, accountRepo, sender, invite, clock.Now, newSequentialPolicy(), testConfig().AuthRuntime())
	capturingSender, _ := sender.(*capturingAccountRecoverySender)
	return authTestEnv{
		router:    NewRouter(testConfig(), Dependencies{Auth: auth}),
		stateRepo: stateRepo,
		sender:    capturingSender,
		invite:    invite,
		now:       clock.Now,
		advance:   clock.Advance,
	}
}

func newFailClosedAuthEnv(t *testing.T) authTestEnv {
	t.Helper()
	clock := &mutableClock{current: time.Date(2026, time.March, 21, 0, 0, 0, 0, time.UTC)}
	auth := usecases.NewAuthService(failingAuthStateRepository{}, stubAuthAccountRepositoryWithMember(), nil, nil, clock.Now, newSequentialPolicy(), testConfig().AuthRuntime())
	return authTestEnv{router: NewRouter(testConfig(), Dependencies{Auth: auth})}
}

type mutableClock struct{ current time.Time }

func (c *mutableClock) Now() time.Time          { return c.current }
func (c *mutableClock) Advance(d time.Duration) { c.current = c.current.Add(d) }

type capturingAccountRecoverySender struct{ lastDelivery usecases.RecoveryDelivery }

func (m *capturingAccountRecoverySender) SendAccountRecovery(_ context.Context, delivery usecases.RecoveryDelivery) error {
	m.lastDelivery = delivery
	return nil
}

type failingAccountRecoverySender struct{ err error }

func (m failingAccountRecoverySender) SendAccountRecovery(_ context.Context, _ usecases.RecoveryDelivery) error {
	return m.err
}

type advancingFailingAccountRecoverySender struct {
	advance func()
	err     error
}

func (m advancingFailingAccountRecoverySender) SendAccountRecovery(_ context.Context, _ usecases.RecoveryDelivery) error {
	if m.advance != nil {
		m.advance()
	}
	return m.err
}

type sequentialPolicy struct{ next int }

func newSequentialPolicy() types.AuthIDPolicy {
	seq := &sequentialPolicy{}
	ids := []string{
		"01ARZ3NDEKTSV4RRFFQ69G5FAV", "01ARZ3NDEKTSV4RRFFQ69G5FAW", "01ARZ3NDEKTSV4RRFFQ69G5FAX", "01ARZ3NDEKTSV4RRFFQ69G5FAY",
		"01ARZ3NDEKTSV4RRFFQ69G5FAZ", "01ARZ3NDEKTSV4RRFFQ69G5FB1", "01ARZ3NDEKTSV4RRFFQ69G5FB2", "01ARZ3NDEKTSV4RRFFQ69G5FB3",
		"01ARZ3NDEKTSV4RRFFQ69G5FB4", "01ARZ3NDEKTSV4RRFFQ69G5FB5", "01ARZ3NDEKTSV4RRFFQ69G5FB6", "01ARZ3NDEKTSV4RRFFQ69G5FB7",
		"01ARZ3NDEKTSV4RRFFQ69G5FB8", "01ARZ3NDEKTSV4RRFFQ69G5FB9", "01ARZ3NDEKTSV4RRFFQ69G5FBA", "01ARZ3NDEKTSV4RRFFQ69G5FBB",
	}
	return types.AuthIDPolicy{
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
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	return recorder
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

func credentialEnvelope(credential string, challenge string) string {
	return credential + "::" + challenge
}

type stubAuthStateRepository struct {
	challenges              map[string]domain.AuthChallenge
	sessions                map[string]domain.Session
	recoveryTokens          map[string]domain.RecoveryToken
	recoveryFailures        map[string]domain.RecoveryDeliveryFailure
	lastRecoveryFailureTTL  time.Duration
	recoverySessions        map[string]domain.RecoverySession
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
		sessions:         map[string]domain.Session{},
		recoveryTokens:   map[string]domain.RecoveryToken{},
		recoveryFailures: map[string]domain.RecoveryDeliveryFailure{},
		recoverySessions: map[string]domain.RecoverySession{},
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

func (r *stubAuthStateRepository) SaveSession(_ context.Context, session domain.Session, _ time.Duration) error {
	r.sessions[session.Token()] = session
	return nil
}

func (r *stubAuthStateRepository) RefreshSession(_ context.Context, session domain.Session, _ time.Duration) error {
	r.sessions[session.Token()] = session
	return nil
}

func (r *stubAuthStateRepository) GetSessionByToken(_ context.Context, token string) (domain.Session, error) {
	session, ok := r.sessions[token]
	if !ok {
		return emptySessionForTest(), domain.ErrSessionNotFound
	}
	return session, nil
}

func (r *stubAuthStateRepository) RevokeSession(_ context.Context, session domain.Session, _ time.Duration) error {
	r.sessions[session.Token()] = session
	return nil
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

func (r *stubAuthStateRepository) GetRecoveryTokenBySecret(_ context.Context, secret string) (domain.RecoveryToken, error) {
	token, ok := r.recoveryTokens[secret]
	if !ok {
		return emptyRecoveryTokenForTest(), domain.ErrRecoveryTokenNotFound
	}
	return token, nil
}

func (r *stubAuthStateRepository) ConsumeRecoveryToken(_ context.Context, token domain.RecoveryToken) error {
	r.recoveryTokens[token.Secret()] = token
	return nil
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
	input  usecases.InvitationPasskeyRegistrationInput
}

func (r *stubInvitationPasskeyRegistrar) RegisterInvitationPasskey(_ context.Context, input usecases.InvitationPasskeyRegistrationInput) (usecases.AuthSession, error) {
	r.called = true
	r.input = input
	return usecases.AuthSession{}, usecases.ErrBadRequest
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

type stubAuthAccountRepository struct {
	account domain.AuthAccount
}

func stubAuthAccountRepositoryWithMember() *stubAuthAccountRepository {
	account, _ := domain.NewAuthAccount("01ARZ3NDEKTSV4RRFFQ69G5FAV", "member@example.com", "member@example.com", "01ARZ3NDEKTSV4RRFFQ69G5FB0", "existing-credential")
	return &stubAuthAccountRepository{account: account}
}

func (r *stubAuthAccountRepository) FindByIdentifier(_ context.Context, identifier string) (domain.AuthAccount, error) {
	if identifier != r.account.Identifier() {
		return emptyAuthAccountForTest(), domain.ErrAuthAccountNotFound
	}
	return r.account, nil
}

func (r *stubAuthAccountRepository) FindByCredential(_ context.Context, credential string) (domain.AuthAccount, error) {
	if credential != r.account.CredentialHandle() {
		return emptyAuthAccountForTest(), domain.ErrAuthAccountNotFound
	}
	return r.account, nil
}

func (r *stubAuthAccountRepository) FindByEmail(_ context.Context, email string) (domain.AuthAccount, error) {
	if email != r.account.Email() {
		return emptyAuthAccountForTest(), domain.ErrAuthAccountNotFound
	}
	return r.account, nil
}

func (r *stubAuthAccountRepository) ReplacePasskey(_ context.Context, accountID string, passkeyCredentialID string, credential string) (domain.AuthAccount, error) {
	if accountID != r.account.AccountID() {
		return emptyAuthAccountForTest(), domain.ErrAuthAccountNotFound
	}
	updated, err := domain.NewAuthAccount(r.account.AccountID(), r.account.Identifier(), r.account.Email(), passkeyCredentialID, credential)
	if err != nil {
		return emptyAuthAccountForTest(), err
	}
	r.account = updated
	return updated, nil
}

func emptyChallengeForTest() domain.AuthChallenge {
	challenge, _ := domain.NewAuthChallenge("01ARZ3NDEKTSV4RRFFQ69G5FAV", "placeholder", "placeholder", time.Unix(0, 0).UTC())
	return challenge
}

func emptySessionForTest() domain.Session {
	session, _ := domain.NewSession("01ARZ3NDEKTSV4RRFFQ69G5FAV", "01ARZ3NDEKTSV4RRFFQ69G5FAW", "01ARZ3NDEKTSV4RRFFQ69G5FAX", "placeholder", time.Unix(1, 0).UTC(), time.Unix(2, 0).UTC())
	return session
}

func emptyRecoveryTokenForTest() domain.RecoveryToken {
	token, _ := domain.NewRecoveryToken("01ARZ3NDEKTSV4RRFFQ69G5FAV", "01ARZ3NDEKTSV4RRFFQ69G5FAW", "placeholder", time.Unix(1, 0).UTC())
	return token
}

func emptyRecoverySessionForTest() domain.RecoverySession {
	session, _ := domain.NewRecoverySession("01ARZ3NDEKTSV4RRFFQ69G5FAV", "01ARZ3NDEKTSV4RRFFQ69G5FAW", time.Unix(1, 0).UTC())
	return session
}

func emptyAuthAccountForTest() domain.AuthAccount {
	account, _ := domain.NewAuthAccount("01ARZ3NDEKTSV4RRFFQ69G5FAV", "placeholder@example.com", "placeholder@example.com", "01ARZ3NDEKTSV4RRFFQ69G5FAW", "placeholder")
	return account
}

type failingAuthStateRepository struct{}

func (failingAuthStateRepository) SaveChallenge(context.Context, domain.AuthChallenge, time.Duration) error {
	return domain.ErrAuthStoreUnavailable
}
func (failingAuthStateRepository) ConsumeChallenge(context.Context, string) (domain.AuthChallenge, error) {
	return emptyChallengeForTest(), domain.ErrAuthStoreUnavailable
}
func (failingAuthStateRepository) SaveSession(context.Context, domain.Session, time.Duration) error {
	return domain.ErrAuthStoreUnavailable
}
func (failingAuthStateRepository) RefreshSession(context.Context, domain.Session, time.Duration) error {
	return domain.ErrAuthStoreUnavailable
}
func (failingAuthStateRepository) GetSessionByToken(context.Context, string) (domain.Session, error) {
	return emptySessionForTest(), domain.ErrAuthStoreUnavailable
}
func (failingAuthStateRepository) RevokeSession(context.Context, domain.Session, time.Duration) error {
	return domain.ErrAuthStoreUnavailable
}
func (failingAuthStateRepository) IssueRecoveryToken(context.Context, domain.RecoveryToken, time.Duration) error {
	return domain.ErrAuthStoreUnavailable
}
func (failingAuthStateRepository) SaveRecoveryDeliveryFailure(context.Context, domain.RecoveryDeliveryFailure, time.Duration) error {
	return domain.ErrAuthStoreUnavailable
}
func (failingAuthStateRepository) GetRecoveryTokenBySecret(context.Context, string) (domain.RecoveryToken, error) {
	return emptyRecoveryTokenForTest(), domain.ErrAuthStoreUnavailable
}
func (failingAuthStateRepository) ConsumeRecoveryToken(context.Context, domain.RecoveryToken) error {
	return domain.ErrAuthStoreUnavailable
}
func (failingAuthStateRepository) SaveRecoverySession(context.Context, domain.RecoverySession, time.Duration) error {
	return domain.ErrAuthStoreUnavailable
}
func (failingAuthStateRepository) GetRecoverySession(context.Context, string) (domain.RecoverySession, error) {
	return emptyRecoverySessionForTest(), domain.ErrAuthStoreUnavailable
}
func (failingAuthStateRepository) ConsumeRecoverySession(context.Context, domain.RecoverySession) error {
	return domain.ErrAuthStoreUnavailable
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
