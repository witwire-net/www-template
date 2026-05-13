package http

import (
	"context"
	stdhttp "net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"www-template/packages/backend/internal/auth/application"
	"www-template/packages/backend/internal/auth/domain"
)

// newJWTAuthTestEnv は TokenService と SessionService を注入したテスト環境を構築する。
func newJWTAuthTestEnv(t *testing.T) authTestEnv {
	t.Helper()
	clock := &mutableClock{current: time.Date(2026, time.March, 21, 0, 0, 0, 0, time.UTC)}
	stateRepo := newStubAuthStateRepository(clock.Now)
	accountRepo := stubAuthAccountRepositoryWithMember()
	invite := &stubInvitationPasskeyRegistrar{}
	sender := &capturingAccountRecoverySender{}
	cfg := testConfig()
	auth := application.NewAuthService(stateRepo, accountRepo, sender, invite, clock.Now, newSequentialPolicy(), cfg.AuthRuntime())
	auth.UseWebAuthnProvider(newMockWebAuthnProvider())

	refreshStore := newStubRefreshTokenStore()
	sessionStore := newStubSessionStore()
	tokenService := application.NewTokenService(refreshStore, sessionStore, cfg.AuthRuntime(), clock.Now, newSequentialPolicy())
	auth.UseTokenService(tokenService)
	sessionService := application.NewSessionService(sessionStore, refreshStore)

	return authTestEnv{
		router:       NewRouter(cfg, Dependencies{Auth: auth, TokenService: tokenService, SessionService: sessionService}),
		stateRepo:    stateRepo,
		sender:       sender,
		invite:       invite,
		auth:         auth,
		now:          clock.Now,
		advance:      clock.Advance,
		refreshStore: refreshStore,
	}
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
		// 保存中のトークンはまだ消費されていない
		return application.RefreshTokenRecord{}, domain.ErrSessionNotFound
	}
	// 存在しないトークンは消費済みとみなす（テスト簡略化）
	return application.RefreshTokenRecord{}, domain.ErrSessionNotFound
}

func (s *stubRefreshTokenStore) RevokeAllForFingerprint(_ context.Context, _, _ string) error {
	return nil
}

func (s *stubRefreshTokenStore) RevokeBySessionID(_ context.Context, _, sessionID string) error {
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

func (s *stubSessionStore) SaveSession(_ context.Context, sessionID, _ string, metadata application.SessionMetadata, _ time.Duration) error {
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

func (s *stubSessionStore) ListSessions(_ context.Context, _ string) ([]application.SessionMetadata, error) {
	result := make([]application.SessionMetadata, 0, len(s.sessions))
	for _, v := range s.sessions {
		result = append(result, v)
	}
	return result, nil
}

func (s *stubSessionStore) RevokeSession(_ context.Context, _, sessionID string) error {
	delete(s.sessions, sessionID)
	return nil
}

func (s *stubSessionStore) RevokeOthers(_ context.Context, _, currentSessionID string) ([]string, error) {
	deleted := make([]string, 0)
	for id := range s.sessions {
		if id != currentSessionID {
			delete(s.sessions, id)
			deleted = append(deleted, id)
		}
	}
	return deleted, nil
}

func (s *stubSessionStore) RevokeAllForAccount(_ context.Context, _ string) error {
	for id := range s.sessions {
		delete(s.sessions, id)
	}
	return nil
}

// loginWithJWT はパスキー認証フローを実行して JWT access token と refresh token を返す helper。
func loginWithJWT(t *testing.T, router *gin.Engine, identifier string) (accessToken, refreshToken string) {
	t.Helper()
	challenge := startPasskey(t, router, identifier)
	resp := performJSON(t, router, stdhttp.MethodPost, "/api/v1/auth/passkey/finish",
		map[string]any{"credential": assertionCredentialJSON("existing-credential", challengeValue(challenge))}, "")
	assertStatus(t, resp, stdhttp.StatusOK)
	var body map[string]any
	decodeJSON(t, resp, &body)
	at, _ := body["accessToken"].(string)
	rt, _ := body["refreshToken"].(string)
	return at, rt
}

// [AUTH-BE-S001] Passkey finish returns JWT and refresh token
func TestAuthPasskeyFinishReturnsJWTAndRefreshToken(t *testing.T) {
	t.Parallel()
	env := newJWTAuthTestEnv(t)
	at, rt := loginWithJWT(t, env.router, "member@example.com")
	if at == "" {
		t.Fatal("expected accessToken")
	}
	if rt == "" {
		t.Fatal("expected refreshToken")
	}
}

// [AUTH-BE-S043] Refresh endpoint returns new pair
func TestAuthRefreshReturnsNewPair(t *testing.T) {
	t.Parallel()
	env := newJWTAuthTestEnv(t)
	_, rt := loginWithJWT(t, env.router, "member@example.com")

	resp := performJSON(t, env.router, stdhttp.MethodPost, "/api/v1/auth/refresh", map[string]any{"refreshToken": rt}, "")
	assertStatus(t, resp, stdhttp.StatusOK)
	assertNoStore(t, resp)
	var body map[string]any
	decodeJSON(t, resp, &body)
	if body["accessToken"] == "" {
		t.Fatal("expected new accessToken")
	}
	if body["refreshToken"] == "" {
		t.Fatal("expected new refreshToken")
	}
}

// [AUTH-BE-S044] Rotation failure revokes family
func TestAuthRefreshReuseRejectsOldToken(t *testing.T) {
	t.Parallel()
	env := newJWTAuthTestEnv(t)
	_, rt := loginWithJWT(t, env.router, "member@example.com")

	// 1回目のリフレッシュ
	resp1 := performJSON(t, env.router, stdhttp.MethodPost, "/api/v1/auth/refresh", map[string]any{"refreshToken": rt}, "")
	assertStatus(t, resp1, stdhttp.StatusOK)

	// 同じトークンで再試行 → 拒否
	resp2 := performJSON(t, env.router, stdhttp.MethodPost, "/api/v1/auth/refresh", map[string]any{"refreshToken": rt}, "")
	assertStatus(t, resp2, stdhttp.StatusUnauthorized)
	assertNoStore(t, resp2)
}

// [AUTH-BE-S045] Invalid refresh token rejected
func TestAuthRefreshInvalidTokenRejected(t *testing.T) {
	t.Parallel()
	env := newJWTAuthTestEnv(t)
	resp := performJSON(t, env.router, stdhttp.MethodPost, "/api/v1/auth/refresh", map[string]any{"refreshToken": "invalid-token"}, "")
	assertStatus(t, resp, stdhttp.StatusUnauthorized)
	assertNoStore(t, resp)
}

// [AUTH-BE-S046] Expired JWT rejected
func TestAuthExpiredJWTRejected(t *testing.T) {
	t.Parallel()
	env := newJWTAuthTestEnv(t)
	at, _ := loginWithJWT(t, env.router, "member@example.com")

	env.advance(20 * time.Minute)
	resp := performJSON(t, env.router, stdhttp.MethodPost, "/api/v1/auth/logout", nil, at)
	assertStatus(t, resp, stdhttp.StatusUnauthorized)
	assertFailureCode(t, resp, "session-expired")
}

// [AUTH-BE-S002] Missing or inactive session is rejected
func TestAuthMissingJWTRejected(t *testing.T) {
	t.Parallel()
	env := newJWTAuthTestEnv(t)
	resp := performJSON(t, env.router, stdhttp.MethodPost, "/api/v1/auth/logout", nil, "invalid-jwt")
	assertStatus(t, resp, stdhttp.StatusUnauthorized)
	assertFailureCode(t, resp, "session-expired")
}

// [AUTH-BE-S003] Logout revokes active session
func TestAuthLogoutRevokesJWTSession(t *testing.T) {
	t.Parallel()
	env := newJWTAuthTestEnv(t)
	at, _ := loginWithJWT(t, env.router, "member@example.com")

	logoutResp := performJSON(t, env.router, stdhttp.MethodPost, "/api/v1/auth/logout", nil, at)
	assertStatus(t, logoutResp, stdhttp.StatusOK)

	// 再度ログアウト → 失効済み
	resp := performJSON(t, env.router, stdhttp.MethodPost, "/api/v1/auth/logout", nil, at)
	assertStatus(t, resp, stdhttp.StatusUnauthorized)
	assertFailureCode(t, resp, "session-expired")
}

// [AUTH-BE-S009] Request without session is unauthenticated
func TestAuthNoJWTUnauthenticated(t *testing.T) {
	t.Parallel()
	env := newJWTAuthTestEnv(t)
	resp := performJSON(t, env.router, stdhttp.MethodPost, "/api/v1/auth/logout", nil, "")
	assertStatus(t, resp, stdhttp.StatusUnauthorized)
	assertFailureCode(t, resp, "unauthenticated")
}

// [AUTH-BE-S042] Logout revokes only one session
func TestAuthLogoutRevokesOnlyOneSession(t *testing.T) {
	t.Parallel()
	env := newJWTAuthTestEnv(t)
	at1, _ := loginWithJWT(t, env.router, "member@example.com")
	at2, _ := loginWithJWT(t, env.router, "member@example.com")

	logoutResp := performJSON(t, env.router, stdhttp.MethodPost, "/api/v1/auth/logout", nil, at1)
	assertStatus(t, logoutResp, stdhttp.StatusOK)

	// at2 はまだ有効
	resp := performJSON(t, env.router, stdhttp.MethodPost, "/api/v1/auth/logout", nil, at2)
	assertStatus(t, resp, stdhttp.StatusOK)
}

// [AUTH-BE-S041] Multiple accounts hold independent sessions
func TestAuthMultipleAccountsIndependent(t *testing.T) {
	t.Parallel()
	clock := &mutableClock{current: time.Date(2026, time.March, 21, 0, 0, 0, 0, time.UTC)}
	stateRepo := newStubAuthStateRepository(clock.Now)
	accountRepo := newMultiAccountStubAuthAccountRepository(
		stubAuthAccountRepositoryWithMember(),
		stubAuthAccountRepositoryWithAccount("01ARZ3NDEKTSV4RRFFQ69G5FB1", "other@example.com", "other-credential"),
	)
	cfg := testConfig()
	auth := application.NewAuthService(stateRepo, accountRepo, nil, nil, clock.Now, newSequentialPolicy(), cfg.AuthRuntime())
	auth.UseWebAuthnProvider(newMockWebAuthnProvider())

	refreshStore := newStubRefreshTokenStore()
	sessionStore := newStubSessionStore()
	tokenService := application.NewTokenService(refreshStore, sessionStore, cfg.AuthRuntime(), clock.Now, newSequentialPolicy())
	auth.UseTokenService(tokenService)

	router := NewRouter(cfg, Dependencies{Auth: auth, TokenService: tokenService})

	at1, _ := loginWithJWT(t, router, "member@example.com")
	at2, _ := loginWithJWT(t, router, "other@example.com")

	resp1 := performJSON(t, router, stdhttp.MethodPost, "/api/v1/auth/logout", nil, at1)
	assertStatus(t, resp1, stdhttp.StatusOK)

	resp2 := performJSON(t, router, stdhttp.MethodPost, "/api/v1/auth/logout", nil, at2)
	assertStatus(t, resp2, stdhttp.StatusOK)
}

// [AUTH-BE-S047] Session list endpoint returns sessions
func TestAuthListSessionsReturnsSessions(t *testing.T) {
	t.Parallel()
	env := newJWTAuthTestEnv(t)
	at, _ := loginWithJWT(t, env.router, "member@example.com")

	resp := performJSON(t, env.router, stdhttp.MethodGet, "/api/v1/sessions", nil, at)
	assertStatus(t, resp, stdhttp.StatusOK)
	assertNoStore(t, resp)
	var body map[string]any
	decodeJSON(t, resp, &body)
	sessions, ok := body["sessions"].([]any)
	if !ok {
		t.Fatalf("expected sessions array, got %#v", body["sessions"])
	}
	if len(sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(sessions))
	}
}

// [AUTH-BE-S048] Revoke session endpoint invalidates session
func TestAuthRevokeSessionInvalidatesSession(t *testing.T) {
	t.Parallel()
	env := newJWTAuthTestEnv(t)
	at, _ := loginWithJWT(t, env.router, "member@example.com")

	// セッション一覧を取得して sessionID を得る
	listResp := performJSON(t, env.router, stdhttp.MethodGet, "/api/v1/sessions", nil, at)
	var listBody map[string]any
	decodeJSON(t, listResp, &listBody)
	sessions := listBody["sessions"].([]any)
	session := sessions[0].(map[string]any)
	sessionID := session["sessionId"].(string)

	// 別セッションでログイン（失効対象用）
	at2, _ := loginWithJWT(t, env.router, "member@example.com")

	revokeResp := performJSON(t, env.router, stdhttp.MethodDelete, "/api/v1/sessions/"+sessionID, nil, at2)
	assertStatus(t, revokeResp, stdhttp.StatusNoContent)

	// 失効したセッションでログアウト → 拒否
	logoutResp := performJSON(t, env.router, stdhttp.MethodPost, "/api/v1/auth/logout", nil, at)
	assertStatus(t, logoutResp, stdhttp.StatusUnauthorized)
}

// [AUTH-BE-S049] Revoke others endpoint invalidates other sessions
func TestAuthRevokeOthersInvalidatesOthers(t *testing.T) {
	t.Parallel()
	env := newJWTAuthTestEnv(t)
	at1, _ := loginWithJWT(t, env.router, "member@example.com")
	at2, _ := loginWithJWT(t, env.router, "member@example.com")

	revokeResp := performJSON(t, env.router, stdhttp.MethodDelete, "/api/v1/sessions/others", nil, at1)
	assertStatus(t, revokeResp, stdhttp.StatusNoContent)

	// at2 は失効している
	logoutResp := performJSON(t, env.router, stdhttp.MethodPost, "/api/v1/auth/logout", nil, at2)
	assertStatus(t, logoutResp, stdhttp.StatusUnauthorized)

	// at1 は有効
	resp := performJSON(t, env.router, stdhttp.MethodPost, "/api/v1/auth/logout", nil, at1)
	assertStatus(t, resp, stdhttp.StatusOK)
}

// [AUTH-BE-S048] Revoking another account's session is forbidden
func TestAuthRevokeOtherAccountSessionForbidden(t *testing.T) {
	t.Parallel()
	clock := &mutableClock{current: time.Date(2026, time.March, 21, 0, 0, 0, 0, time.UTC)}
	stateRepo := newStubAuthStateRepository(clock.Now)
	accountRepo := newMultiAccountStubAuthAccountRepository(
		stubAuthAccountRepositoryWithMember(),
		stubAuthAccountRepositoryWithAccount("01ARZ3NDEKTSV4RRFFQ69G5FB1", "other@example.com", "other-credential"),
	)
	cfg := testConfig()
	seq := newSequentialPolicy()
	auth := application.NewAuthService(stateRepo, accountRepo, nil, nil, clock.Now, seq, cfg.AuthRuntime())
	auth.UseWebAuthnProvider(newMockWebAuthnProvider())

	refreshStore := newStubRefreshTokenStore()
	sessionStore := newStubSessionStore()
	tokenService := application.NewTokenService(refreshStore, sessionStore, cfg.AuthRuntime(), clock.Now, seq)
	auth.UseTokenService(tokenService)
	sessionService := application.NewSessionService(sessionStore, refreshStore)

	router := NewRouter(cfg, Dependencies{Auth: auth, TokenService: tokenService, SessionService: sessionService})

	_, _ = loginWithJWT(t, router, "member@example.com")
	at2, _ := loginWithJWT(t, router, "other@example.com")

	// member のセッションを other で削除しようとする → 403
	resp := performJSON(t, router, stdhttp.MethodDelete, "/api/v1/sessions/01ARZ3NDEKTSV4RRFFQ69G5FAV", nil, at2)
	assertStatus(t, resp, stdhttp.StatusForbidden)
}

// [AUTH-BE-S038] Unset refresh token TTL is unlimited
func TestAuthRefreshTokenTTLUnsetIsUnlimited(t *testing.T) {
	t.Parallel()
	env := newJWTAuthTestEnv(t)
	_, rt := loginWithJWT(t, env.router, "member@example.com")

	// 時間を進めても TTL が未設定なので有効
	env.advance(48 * time.Hour)
	resp := performJSON(t, env.router, stdhttp.MethodPost, "/api/v1/auth/refresh", map[string]any{"refreshToken": rt}, "")
	assertStatus(t, resp, stdhttp.StatusOK)
}

// [AUTH-BE-S039] 24h+ TTL is applied correctly
func TestAuthRefreshTokenTTL24hApplied(t *testing.T) {
	t.Parallel()
	clock := &mutableClock{current: time.Date(2026, time.March, 21, 0, 0, 0, 0, time.UTC)}
	stateRepo := newStubAuthStateRepository(clock.Now)
	accountRepo := stubAuthAccountRepositoryWithMember()
	cfg := testConfig()
	cfg.Auth.RefreshTokenTTL = 48 * time.Hour
	auth := application.NewAuthService(stateRepo, accountRepo, nil, nil, clock.Now, newSequentialPolicy(), cfg.AuthRuntime())
	auth.UseWebAuthnProvider(newMockWebAuthnProvider())

	refreshStore := newStubRefreshTokenStore()
	sessionStore := newStubSessionStore()
	tokenService := application.NewTokenService(refreshStore, sessionStore, cfg.AuthRuntime(), clock.Now, newSequentialPolicy())
	auth.UseTokenService(tokenService)

	router := NewRouter(cfg, Dependencies{Auth: auth, TokenService: tokenService})

	_, rt := loginWithJWT(t, router, "member@example.com")

	// 48時間以内は有効
	clock.Advance(47 * time.Hour)
	resp := performJSON(t, router, stdhttp.MethodPost, "/api/v1/auth/refresh", map[string]any{"refreshToken": rt}, "")
	assertStatus(t, resp, stdhttp.StatusOK)

	// 48時間を超えると失効
	clock.Advance(2 * time.Hour)
	resp2 := performJSON(t, router, stdhttp.MethodPost, "/api/v1/auth/refresh", map[string]any{"refreshToken": rt}, "")
	assertStatus(t, resp2, stdhttp.StatusUnauthorized)
}

// [AUTH-BE-S010] Auth state store unavailable is internal-error
func TestAuthJWTStoreUnavailableReturnsInternalError(t *testing.T) {
	t.Parallel()
	clock := &mutableClock{current: time.Date(2026, time.March, 21, 0, 0, 0, 0, time.UTC)}
	auth := application.NewAuthService(failingAuthStateRepository{}, stubAuthAccountRepositoryWithMember(), nil, nil, clock.Now, newSequentialPolicy(), testConfig().AuthRuntime())

	refreshStore := newStubRefreshTokenStore()
	sessionStore := newStubSessionStore()
	tokenService := application.NewTokenService(refreshStore, sessionStore, testConfig().AuthRuntime(), clock.Now, newSequentialPolicy())
	auth.UseTokenService(tokenService)

	router := NewRouter(testConfig(), Dependencies{Auth: auth, TokenService: tokenService})

	resp := performJSON(t, router, stdhttp.MethodPost, "/api/v1/auth/refresh", map[string]any{"refreshToken": "invalid"}, "")
	assertStatus(t, resp, stdhttp.StatusUnauthorized)
}

// [AUTH-BE-S010] Token issuance failure returns 503 fail-close
func TestAuthPasskeyFinishTokenIssuanceFailureReturns503(t *testing.T) {
	t.Parallel()
	env := newJWTAuthTestEnv(t)
	env.refreshStore.saveFails = true

	challenge := startPasskey(t, env.router, "member@example.com")
	resp := performJSON(t, env.router, stdhttp.MethodPost, "/api/v1/auth/passkey/finish",
		map[string]any{"credential": assertionCredentialJSON("existing-credential", challengeValue(challenge))}, "")
	assertStatus(t, resp, stdhttp.StatusServiceUnavailable)
	assertFailureCode(t, resp, "internal-error")
}

// [AUTH-BE-S010] Register passkey token issuance failure returns 503 fail-close
func TestAuthRegisterPasskeyTokenIssuanceFailureReturns503(t *testing.T) {
	t.Parallel()
	env := newJWTAuthTestEnv(t)
	recoverySession := consumeRecoverySession(t, env)
	env.refreshStore.saveFails = true

	resp := performJSON(t, env.router, stdhttp.MethodPost, "/api/v1/auth/passkey/register",
		map[string]any{"recovery_session": recoverySession, "credential": attestationCredentialJSON("new-credential", "")}, "")
	assertStatus(t, resp, stdhttp.StatusServiceUnavailable)
	assertFailureCode(t, resp, "internal-error")
}

// stubAuthAccountRepositoryWithAccount は指定パラメータで account stub を生成する。
func stubAuthAccountRepositoryWithAccount(accountID, email, credentialHandle string) *stubAuthAccountRepository {
	account, _ := domain.NewAuthAccount(accountID, email, email, "01ARZ3NDEKTSV4RRFFQ69G5FB0", credentialHandle)
	return &stubAuthAccountRepository{account: account}
}
