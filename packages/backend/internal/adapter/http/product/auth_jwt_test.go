package product

import (
	"context"
	stdhttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	productaccounts "www-template/packages/backend/internal/application/accounts"
	application "www-template/packages/backend/internal/application/auth"
	domain "www-template/packages/backend/internal/domain"
)

// newJWTAuthTestEnv は canonical Product account lifecycle と SessionService を注入したテスト環境を構築する。
func newJWTAuthTestEnv(t *testing.T) authTestEnv {
	t.Helper()
	clock := &mutableClock{current: time.Date(2026, time.March, 21, 0, 0, 0, 0, time.UTC)}
	stateRepo := newStubAuthStateRepository(clock.Now)
	accountRepo := stubAccountAuthRepositoryWithMember()
	return newJWTAuthTestEnvWithRepository(t, clock, stateRepo, accountRepo)
}

// newJWTAuthTestEnvWithRepository は指定された account repository を使用して
// JWT 認証テスト環境を構築する。accountRepo は AuthService と canonical lifecycle の
// 両方に注入されるため、login / bearer 認可 / refresh が同じ account status を参照する。
func newJWTAuthTestEnvWithRepository(t *testing.T, clock *mutableClock, stateRepo *stubAuthStateRepository, accountRepo application.PasskeyAccountRepository) authTestEnv {
	t.Helper()
	return newJWTAuthTestEnvWithAccountSettingRepository(t, clock, stateRepo, accountRepo, newStubHTTPAccountSettingRepository())
}

// newJWTAuthTestEnvWithAccountSettingRepository は refresh/account settings の AccountSetting 読み込み結果を差し替えられる JWT 環境を構築する。
func newJWTAuthTestEnvWithAccountSettingRepository(t *testing.T, clock *mutableClock, stateRepo *stubAuthStateRepository, accountRepo application.PasskeyAccountRepository, accountSettingRepo *stubHTTPAccountSettingRepository) authTestEnv {
	t.Helper()
	invite := &stubInvitationPasskeyRegistrar{}
	sender := &capturingAccountRecoverySender{}
	cfg := testConfig()
	lifecycle, contextRefresh, productRefreshStore, _ := newTestProductAccountLifecycleWithStores(t, accountRepo, clock.Now, cfg.AuthRuntime().RefreshTokenTTL)
	auth := mustNewProductAuthForTest(t, stateRepo, accountRepo, sender, invite, lifecycle, clock.Now, cfg.AuthRuntime(), application.AuthServiceOptionalPorts{WebAuthn: newMockWebAuthnProvider()})
	sessionService := application.NewProductSessionService(lifecycle)

	return authTestEnv{
		router: NewRouter(cfg, Dependencies{
			Auth:            auth,
			AccountSetting:  productaccounts.NewAccountSettingService(accountSettingRepo),
			AccountSnapshot: productaccounts.NewAccountSettingSnapshotService(accountSettingRepo),
			TokenService:    contextRefresh,
			SessionService:  sessionService,
		}),
		stateRepo:           stateRepo,
		sender:              sender,
		invite:              invite,
		auth:                auth,
		now:                 clock.Now,
		advance:             clock.Advance,
		productRefreshStore: productRefreshStore,
	}
}

func newTestProductAccountLifecycle(t *testing.T, accountRepo application.AccountAuthRepository, clock func() time.Time, refreshTokenTTL time.Duration) (*application.AccountSessionService, application.ProductContextRefreshService) {
	t.Helper()
	lifecycle, contextRefresh, _, _ := newTestProductAccountLifecycleWithStores(t, accountRepo, clock, refreshTokenTTL)
	return lifecycle, contextRefresh
}

func newTestProductAccountLifecycleWithStores(t *testing.T, accountRepo application.AccountAuthRepository, clock func() time.Time, refreshTokenTTL time.Duration) (*application.AccountSessionService, application.ProductContextRefreshService, *testProductRefreshSessionStore, *testProductSessionMetadataStore) {
	t.Helper()

	// Step 1: テスト用 signer と in-memory store を canonical Product account lifecycle へ注入し、root legacy TokenService なしで login/refresh/bearer を検証する。
	signer, err := application.NewTokenJSONSignVerifier([]byte(testConfig().AuthRuntime().JWTSecret))
	if err != nil {
		t.Fatalf("create product auth signer: %v", err)
	}
	if refreshTokenTTL <= 0 {
		refreshTokenTTL = 14 * 24 * time.Hour
	}
	refreshStore := newTestProductRefreshSessionStore()
	metadataStore := newTestProductSessionMetadataStore()
	lifecycle, err := application.NewAccountSessionService(application.AccountSessionDependencies{Accounts: accountRepo, RefreshSessions: refreshStore, Sessions: metadataStore, Signer: signer, IDGenerator: newSequentialPolicy(), TokenGenerator: application.NewCryptoOpaqueTokenGenerator(), Clock: clock}, application.AccountSessionConfig{AccessTokenTTL: domain.AccessTokenTTL, RefreshTokenTTL: refreshTokenTTL, RefreshCookieLifetime: refreshTokenTTL})
	if err != nil {
		t.Fatalf("create product account lifecycle: %v", err)
	}
	return lifecycle, application.NewProductContextRefreshService(lifecycle), refreshStore, metadataStore
}

type testProductRefreshSessionStore struct {
	sessions  map[string]domain.AccountRefreshSession
	saveFails bool
}

func newTestProductRefreshSessionStore() *testProductRefreshSessionStore {
	return &testProductRefreshSessionStore{sessions: map[string]domain.AccountRefreshSession{}}
}

func (s *testProductRefreshSessionStore) Save(_ context.Context, session domain.AccountRefreshSession, _ time.Duration) error {
	if s.saveFails {
		return domain.ErrAuthStoreUnavailable
	}
	// Step 1: 平文 refreshToken ではなく domain hash だけを map key として保存する。
	s.sessions[session.TokenHash().String()] = session
	return nil
}

func (s *testProductRefreshSessionStore) Rotate(_ context.Context, tokenHash domain.OpaqueTokenHash, _ time.Duration, build application.RefreshRotationBuilder) (domain.AccountRefreshSession, domain.AccountRefreshSession, error) {
	// Step 1: old hash から session を原子的に取り出すテスト用挙動を再現する。
	consumed, ok := s.sessions[tokenHash.String()]
	if !ok {
		return domain.AccountRefreshSession{}, domain.AccountRefreshSession{}, domain.ErrSessionNotFound
	}
	delete(s.sessions, tokenHash.String())
	next, err := build(consumed)
	if err != nil {
		return consumed, domain.AccountRefreshSession{}, err
	}
	s.sessions[next.TokenHash().String()] = next
	return consumed, next, nil
}

func (s *testProductRefreshSessionStore) RevokeSession(_ context.Context, accountID domain.AccountID, sessionID domain.AccountAuthSessionID, _ time.Time) error {
	// Step 1: 対象 account/session の refresh state だけを削除し、bearer revoke と同じ selector を使う。
	for hash, session := range s.sessions {
		if session.AccountID() == accountID && session.SessionID() == sessionID {
			delete(s.sessions, hash)
		}
	}
	return nil
}

func (s *testProductRefreshSessionStore) RevokeAllForAccount(_ context.Context, accountID domain.AccountID, _ time.Time) error {
	// Step 1: 対象 account の refresh state をすべて削除する。
	for hash, session := range s.sessions {
		if session.AccountID() == accountID {
			delete(s.sessions, hash)
		}
	}
	return nil
}

type testProductSessionMetadataStore struct {
	sessions map[string]application.SessionMetadata
}

func newTestProductSessionMetadataStore() *testProductSessionMetadataStore {
	return &testProductSessionMetadataStore{sessions: map[string]application.SessionMetadata{}}
}

func (s *testProductSessionMetadataStore) Save(_ context.Context, metadata application.SessionMetadata, ttl time.Duration) error {
	// Step 1: Product session selector を key にして bearer validation 用 metadata を保存する。
	s.sessions[metadata.SessionID] = metadata
	_ = ttl
	return nil
}

func (s *testProductSessionMetadataStore) Get(_ context.Context, sessionID domain.AccountAuthSessionID) (application.SessionMetadata, error) {
	// Step 1: accessToken の sid が存在する場合だけ metadata を返す。
	metadata, ok := s.sessions[sessionID.String()]
	if !ok {
		return application.SessionMetadata{}, domain.ErrSessionNotFound
	}
	return metadata, nil
}

func (s *testProductSessionMetadataStore) List(_ context.Context, accountID domain.AccountID) ([]application.SessionMetadata, error) {
	// Step 1: canonical Product session metadata だけを accountID で抽出し、legacy SessionStore fixture に依存しない一覧を返す。
	result := make([]application.SessionMetadata, 0, len(s.sessions))
	for _, metadata := range s.sessions {
		if metadata.AccountID == accountID {
			result = append(result, metadata)
		}
	}
	return result, nil
}

func (s *testProductSessionMetadataStore) Revoke(_ context.Context, accountID domain.AccountID, sessionID domain.AccountAuthSessionID) error {
	// Step 1: owner が一致する session だけを削除し、他 account の session を誤削除しない。
	metadata, ok := s.sessions[sessionID.String()]
	if !ok {
		return domain.ErrSessionNotFound
	}
	if metadata.AccountID != accountID {
		return domain.ErrInvalidToken
	}
	delete(s.sessions, sessionID.String())
	return nil
}

func (s *testProductSessionMetadataStore) RevokeAllForAccount(_ context.Context, accountID domain.AccountID) error {
	// Step 1: 対象 account の bearer metadata をすべて削除する。
	for sessionID, metadata := range s.sessions {
		if metadata.AccountID == accountID {
			delete(s.sessions, sessionID)
		}
	}
	return nil
}

// stubHTTPAccountSettingRepository は HTTP handler テストで AccountSetting の保存値を再現するインメモリ repository である。
type stubHTTPAccountSettingRepository struct {
	settings map[string]domain.AccountSetting
	getErr   error
}

func newStubHTTPAccountSettingRepository() *stubHTTPAccountSettingRepository {
	return &stubHTTPAccountSettingRepository{settings: map[string]domain.AccountSetting{}}
}

func (r *stubHTTPAccountSettingRepository) CreateDefault(_ context.Context, accountID domain.AccountID) (domain.AccountSetting, error) {
	setting, err := domain.NewDefaultAccountSetting(accountID)
	if err != nil {
		return emptyHTTPAccountSettingForTest(), err
	}
	r.settings[accountID.String()] = setting
	return setting, nil
}

func (r *stubHTTPAccountSettingRepository) Get(_ context.Context, accountID domain.AccountID) (domain.AccountSetting, error) {
	if r.getErr != nil {
		return emptyHTTPAccountSettingForTest(), r.getErr
	}
	if setting, ok := r.settings[accountID.String()]; ok {
		return setting, nil
	}
	setting, err := domain.NewDefaultAccountSetting(accountID)
	if err != nil {
		return emptyHTTPAccountSettingForTest(), err
	}
	r.settings[accountID.String()] = setting
	return setting, nil
}

func (r *stubHTTPAccountSettingRepository) UpdateLocale(_ context.Context, accountID domain.AccountID, locale domain.AccountLocale) (domain.AccountSetting, error) {
	setting, err := domain.NewAccountSetting(accountID, locale)
	if err != nil {
		return emptyHTTPAccountSettingForTest(), err
	}
	r.settings[accountID.String()] = setting
	return setting, nil
}

func emptyHTTPAccountSettingForTest() domain.AccountSetting {
	setting, _ := domain.NewDefaultAccountSetting(testAccountID("01ARZ3NDEKTSV4RRFFQ69G5FAV"))
	return setting
}

// newJWTAuthTestEnvWithAccountRepo はテストが account status を途中で変更できるように、
// mutable な stub repository と JWT 認証環境をまとめて返す。
func newJWTAuthTestEnvWithAccountRepo(t *testing.T) (authTestEnv, *stubAccountAuthRepository) {
	t.Helper()
	clock := &mutableClock{current: time.Date(2026, time.March, 21, 0, 0, 0, 0, time.UTC)}
	stateRepo := newStubAuthStateRepository(clock.Now)
	accountRepo := stubAccountAuthRepositoryWithMember()
	return newJWTAuthTestEnvWithRepository(t, clock, stateRepo, accountRepo), accountRepo
}

// loginWithJWT はパスキー認証フローを実行して JWT access token と HttpOnly refresh Cookie 値を返す helper。
func loginWithJWT(t *testing.T, router *gin.Engine, identifier string) (accessToken, refreshToken string) {
	t.Helper()
	return loginWithJWTUsingCredential(t, router, identifier, "existing-credential")
}

func loginWithJWTUsingCredential(t *testing.T, router *gin.Engine, identifier string, credentialHandle string) (accessToken, refreshToken string) {
	t.Helper()
	challenge := startPasskey(t, router, identifier)
	resp := performJSON(t, router, stdhttp.MethodPost, "/api/v1/auth/passkey/finish",
		map[string]any{"credential": assertionCredentialJSON(credentialHandle, challengeValue(challenge))}, "")
	assertStatus(t, resp, stdhttp.StatusOK)
	var body map[string]any
	decodeJSON(t, resp, &body)
	at, _ := body["accessToken"].(string)
	if _, exposed := body["refreshToken"]; exposed {
		t.Fatal("refreshToken must not be exposed in login response body")
	}
	rt := refreshCookieValueFromResponse(t, resp)
	return at, rt
}

func refreshCookieValueFromResponse(t *testing.T, response interface{ Result() *stdhttp.Response }) string {
	t.Helper()
	for _, cookie := range response.Result().Cookies() {
		if cookie.Name == productRefreshCookieName {
			if cookie.Value == "" {
				t.Fatal("expected non-empty refresh cookie value")
			}
			if !cookie.HttpOnly || !cookie.Secure || cookie.SameSite != stdhttp.SameSiteLaxMode {
				t.Fatalf("refresh cookie must be HttpOnly, Secure, SameSite=Lax, got %+v", cookie)
			}
			return cookie.Value
		}
	}
	t.Fatalf("expected %s Set-Cookie header", productRefreshCookieName)
	return ""
}

func performRefreshWithCookie(t *testing.T, router *gin.Engine, refreshToken string) *httptest.ResponseRecorder {
	t.Helper()
	// Step 1: test ID policy が最初の Product session/authContext に使う ULID を path に入れ、context-scoped refresh endpoint を呼ぶ。
	return performJSONWithHeaders(t, router, stdhttp.MethodPost, "/api/v1/auth/contexts/01ARZ3NDEKTSV4RRFFQ69G5FAV/refresh", nil, "", map[string]string{"Cookie": productRefreshCookieName + "=" + refreshToken})
}

func performRefreshWithCookieWithoutDefaultOrigin(t *testing.T, router *gin.Engine, refreshToken string, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	// Step 1: Origin 欠落の fail-close を検証するため、共通 helper の既定 Origin 補完を避けて request を手動構築する。
	request := httptest.NewRequest(stdhttp.MethodPost, "/api/v1/auth/contexts/01ARZ3NDEKTSV4RRFFQ69G5FAV/refresh", strings.NewReader("{}"))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Cookie", productRefreshCookieName+"="+refreshToken)
	request.RemoteAddr = "192.0.2.10:1234"
	for key, value := range headers {
		request.Header.Set(key, value)
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	return recorder
}

func assertRefreshCookieCleared(t *testing.T, response *httptest.ResponseRecorder) {
	t.Helper()
	for _, header := range response.Header().Values("Set-Cookie") {
		if strings.Contains(header, productRefreshCookieName+"=") && strings.Contains(header, "Max-Age=0") && strings.Contains(header, "HttpOnly") && strings.Contains(header, "Secure") {
			return
		}
	}
	t.Fatalf("expected cleared %s Set-Cookie header, got %q", productRefreshCookieName, response.Header().Values("Set-Cookie"))
}

// [AUTH-BE-S001] Passkey finish returns JWT and refresh token Cookie
func TestAuthPasskeyFinishReturnsJWTAndRefreshTokenCookie(t *testing.T) {
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

// [AUTH-BE-S054] suspended account は valid passkey assertion 後も token pair を発行されない。
func TestAuthPasskeyFinishRejectsSuspendedAccount(t *testing.T) {
	t.Parallel()
	env, accountRepo := newJWTAuthTestEnvWithAccountRepo(t)
	revokedAt := env.now()
	accountRepo.account = accountRepo.account.WithStatus("suspended", &revokedAt)

	challenge := startPasskey(t, env.router, "member@example.com")
	response := performJSON(t, env.router, stdhttp.MethodPost, "/api/v1/auth/passkey/finish",
		map[string]any{"credential": assertionCredentialJSON("existing-credential", challengeValue(challenge))}, "")

	assertStatus(t, response, stdhttp.StatusForbidden)
	assertNoStore(t, response)
	assertFailureCode(t, response, "account-suspended")
}

// [AUTH-BE-S043] Refresh endpoint returns new pair
func TestAuthRefreshReturnsNewPair(t *testing.T) {
	t.Parallel()
	env := newJWTAuthTestEnv(t)
	_, rt := loginWithJWT(t, env.router, "member@example.com")

	resp := performRefreshWithCookie(t, env.router, rt)
	assertStatus(t, resp, stdhttp.StatusOK)
	assertNoStore(t, resp)
	var body map[string]any
	decodeJSON(t, resp, &body)
	if body["accessToken"] == "" {
		t.Fatal("expected new accessToken")
	}
	if _, exposed := body["refreshToken"]; exposed {
		t.Fatal("refreshToken must not be exposed in refresh response body")
	}
	_ = refreshCookieValueFromResponse(t, resp)
}

func TestProductCookieSettingFlowOriginAndFetchMetadata(t *testing.T) {
	t.Parallel()

	t.Run("[AUTH-BE-S088] Cookie mode login accepts only exact allowed Origin", func(t *testing.T) {
		env := newJWTAuthTestEnv(t)
		challenge := startPasskey(t, env.router, "member@example.com")

		response := performJSONWithHeaders(t, env.router, stdhttp.MethodPost, "/api/v1/auth/passkey/finish",
			map[string]any{"credential": assertionCredentialJSON("existing-credential", challengeValue(challenge))}, "", map[string]string{productOriginHeader: "https://evil.example.com"})

		assertStatus(t, response, stdhttp.StatusForbidden)
		assertNoStore(t, response)
		assertProductSecurityHeaders(t, response)
		if cookies := response.Header().Values("Set-Cookie"); len(cookies) != 0 {
			t.Fatalf("expected disallowed Origin to avoid Set-Cookie, got %q", cookies)
		}
	})

	t.Run("[AUTH-BE-S088] Cookie mode refresh rejects disallowed Origin before rotation", func(t *testing.T) {
		env := newJWTAuthTestEnv(t)
		_, refreshToken := loginWithJWT(t, env.router, "member@example.com")
		storedBefore := len(env.productRefreshStore.sessions)

		response := performJSONWithHeaders(t, env.router, stdhttp.MethodPost, "/api/v1/auth/contexts/01ARZ3NDEKTSV4RRFFQ69G5FAV/refresh", nil, "", map[string]string{
			"Cookie":            productRefreshCookieName + "=" + refreshToken,
			productOriginHeader: productTestAllowedOrigin + "/unexpected-path",
		})

		assertStatus(t, response, stdhttp.StatusForbidden)
		assertNoStore(t, response)
		if len(env.productRefreshStore.sessions) != storedBefore {
			t.Fatalf("expected disallowed Origin to stop refresh rotation, before=%d after=%d", storedBefore, len(env.productRefreshStore.sessions))
		}
		if cookies := response.Header().Values("Set-Cookie"); len(cookies) != 0 {
			t.Fatalf("expected disallowed Origin to avoid Set-Cookie, got %q", cookies)
		}
	})

	t.Run("[AUTH-BE-S089] Cookie mode refresh rejects cross-site Fetch Metadata", func(t *testing.T) {
		env := newJWTAuthTestEnv(t)
		_, refreshToken := loginWithJWT(t, env.router, "member@example.com")
		storedBefore := len(env.productRefreshStore.sessions)

		response := performJSONWithHeaders(t, env.router, stdhttp.MethodPost, "/api/v1/auth/contexts/01ARZ3NDEKTSV4RRFFQ69G5FAV/refresh", nil, "", map[string]string{
			"Cookie":               productRefreshCookieName + "=" + refreshToken,
			productOriginHeader:    productTestAllowedOrigin,
			productFetchSiteHeader: "cross-site",
		})

		assertStatus(t, response, stdhttp.StatusForbidden)
		assertNoStore(t, response)
		assertProductSecurityHeaders(t, response)
		if len(env.productRefreshStore.sessions) != storedBefore {
			t.Fatalf("expected cross-site Fetch Metadata to stop refresh rotation, before=%d after=%d", storedBefore, len(env.productRefreshStore.sessions))
		}
		if cookies := response.Header().Values("Set-Cookie"); len(cookies) != 0 {
			t.Fatalf("expected cross-site Fetch Metadata to avoid Set-Cookie, got %q", cookies)
		}
	})

	t.Run("[AUTH-BE-S089] Cookie mode refresh without Fetch Metadata still requires allowed Origin", func(t *testing.T) {
		env := newJWTAuthTestEnv(t)
		_, refreshToken := loginWithJWT(t, env.router, "member@example.com")
		storedBefore := len(env.productRefreshStore.sessions)

		response := performRefreshWithCookieWithoutDefaultOrigin(t, env.router, refreshToken, nil)

		assertStatus(t, response, stdhttp.StatusForbidden)
		assertNoStore(t, response)
		if len(env.productRefreshStore.sessions) != storedBefore {
			t.Fatalf("expected missing Origin to stop refresh rotation, before=%d after=%d", storedBefore, len(env.productRefreshStore.sessions))
		}
	})

	t.Run("[AUTH-BE-S088] CORS credential policy is limited to allowed Product origins", func(t *testing.T) {
		env := newJWTAuthTestEnv(t)
		request := httptest.NewRequest(stdhttp.MethodOptions, "/api/v1/auth/passkey/finish", nil)
		request.Header.Set(productOriginHeader, productTestAllowedOrigin)
		request.Header.Set("Access-Control-Request-Method", stdhttp.MethodPost)
		recorder := httptest.NewRecorder()

		env.router.ServeHTTP(recorder, request)

		if recorder.Header().Get("Access-Control-Allow-Origin") != productTestAllowedOrigin {
			t.Fatalf("expected allowed CORS origin, got %q", recorder.Header().Get("Access-Control-Allow-Origin"))
		}
		if recorder.Header().Get("Access-Control-Allow-Credentials") != "true" {
			t.Fatalf("expected credentialed CORS policy, got %q", recorder.Header().Get("Access-Control-Allow-Credentials"))
		}
	})
}

// [AUTH-BE-S058] suspended account の refresh は rotation と新 token pair 発行を拒否する。
func TestAuthRefreshRejectsSuspendedAccountWithoutRotation(t *testing.T) {
	t.Parallel()
	env, accountRepo := newJWTAuthTestEnvWithAccountRepo(t)
	_, refreshToken := loginWithJWT(t, env.router, "member@example.com")
	revokedAt := env.now().Add(time.Second)
	accountRepo.account = accountRepo.account.WithStatus("suspended", &revokedAt)

	response := performRefreshWithCookie(t, env.router, refreshToken)

	assertStatus(t, response, stdhttp.StatusForbidden)
	assertNoStore(t, response)
	assertFailureCode(t, response, "account-suspended")
	if len(env.productRefreshStore.sessions) != 0 {
		t.Fatalf("expected refresh token family to be revoked without new rotation, got %d tokens", len(env.productRefreshStore.sessions))
	}
}

// [AUTH-BE-S044] Rotation failure revokes family
func TestAuthRefreshReuseRejectsOldToken(t *testing.T) {
	t.Parallel()
	env := newJWTAuthTestEnv(t)
	_, rt := loginWithJWT(t, env.router, "member@example.com")

	// 1回目のリフレッシュ
	resp1 := performRefreshWithCookie(t, env.router, rt)
	assertStatus(t, resp1, stdhttp.StatusOK)

	// 同じトークンで再試行 → 拒否
	resp2 := performRefreshWithCookie(t, env.router, rt)
	assertStatus(t, resp2, stdhttp.StatusUnauthorized)
	assertNoStore(t, resp2)
}

// [AUTH-BE-S045] Invalid refresh token rejected
func TestAuthRefreshInvalidTokenRejected(t *testing.T) {
	t.Parallel()
	env := newJWTAuthTestEnv(t)
	resp := performRefreshWithCookie(t, env.router, "invalid-token")
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
	assertRefreshCookieCleared(t, logoutResp)

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
	accountRepo := newMultiAccountStubAccountAuthRepository(
		stubAccountAuthRepositoryWithMember(),
		stubAccountAuthRepositoryWithAccount("01ARZ3NDEKTSV4RRFFQ69G5FB1", "other@example.com", "other-credential"),
	)
	cfg := testConfig()
	lifecycle, contextRefresh := newTestProductAccountLifecycle(t, accountRepo, clock.Now, cfg.AuthRuntime().RefreshTokenTTL)
	auth := mustNewProductAuthForTest(t, stateRepo, accountRepo, nil, nil, lifecycle, clock.Now, cfg.AuthRuntime(), application.AuthServiceOptionalPorts{WebAuthn: newMockWebAuthnProvider()})

	accountSettingRepo := newStubHTTPAccountSettingRepository()
	router := NewRouter(cfg, Dependencies{
		Auth:            auth,
		AccountSetting:  productaccounts.NewAccountSettingService(accountSettingRepo),
		AccountSnapshot: productaccounts.NewAccountSettingSnapshotService(accountSettingRepo),
		TokenService:    contextRefresh,
	})

	at1, _ := loginWithJWT(t, router, "member@example.com")
	at2, _ := loginWithJWTUsingCredential(t, router, "other@example.com", "other-credential")

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

// [LOCALIZATION-BE-S001] Account settings endpoint は認証済み account の既定 locale を返す。
func TestAccountSettingsGetReturnsDefaultLocale(t *testing.T) {
	t.Parallel()
	env := newJWTAuthTestEnv(t)
	accessToken, _ := loginWithJWT(t, env.router, "member@example.com")

	response := performJSON(t, env.router, stdhttp.MethodGet, "/api/v1/account/settings", nil, accessToken)

	assertStatus(t, response, stdhttp.StatusOK)
	assertNoStore(t, response)
	body := decodeJSONBody(t, response)
	setting, ok := body["setting"].(map[string]any)
	if !ok {
		t.Fatalf("expected setting object, got %#v", body["setting"])
	}
	if setting["locale"] != "ja" {
		t.Fatalf("expected default locale ja, got %#v", setting["locale"])
	}
}

// [LOCALIZATION-BE-S002] Account settings endpoint は認証済み account の locale 更新を保存する。
func TestAccountSettingsPatchUpdatesLocale(t *testing.T) {
	t.Parallel()
	env := newJWTAuthTestEnv(t)
	accessToken, _ := loginWithJWT(t, env.router, "member@example.com")

	patchResponse := performJSON(t, env.router, stdhttp.MethodPatch, "/api/v1/account/settings", map[string]any{"locale": "en"}, accessToken)
	assertStatus(t, patchResponse, stdhttp.StatusOK)
	assertNoStore(t, patchResponse)
	patchBody := decodeJSONBody(t, patchResponse)
	patchSetting, ok := patchBody["setting"].(map[string]any)
	if !ok {
		t.Fatalf("expected patch setting object, got %#v", patchBody["setting"])
	}
	if patchSetting["locale"] != "en" {
		t.Fatalf("expected patch response locale en, got %#v", patchSetting["locale"])
	}

	getResponse := performJSON(t, env.router, stdhttp.MethodGet, "/api/v1/account/settings", nil, accessToken)
	assertStatus(t, getResponse, stdhttp.StatusOK)
	body := decodeJSONBody(t, getResponse)
	setting, ok := body["setting"].(map[string]any)
	if !ok {
		t.Fatalf("expected setting object, got %#v", body["setting"])
	}
	if setting["locale"] != "en" {
		t.Fatalf("expected updated locale en, got %#v", setting["locale"])
	}
}

// [LOCALIZATION-BE-S003] Account settings endpoint は未対応 locale を拒否し、保存値を変更しない。
func TestAccountSettingsPatchRejectsUnsupportedLocale(t *testing.T) {
	t.Parallel()
	env := newJWTAuthTestEnv(t)
	accessToken, _ := loginWithJWT(t, env.router, "member@example.com")

	response := performJSON(t, env.router, stdhttp.MethodPatch, "/api/v1/account/settings", map[string]any{"locale": "fr"}, accessToken)

	assertStatus(t, response, stdhttp.StatusBadRequest)
	assertNoStore(t, response)

	getResponse := performJSON(t, env.router, stdhttp.MethodGet, "/api/v1/account/settings", nil, accessToken)
	assertStatus(t, getResponse, stdhttp.StatusOK)
	body := decodeJSONBody(t, getResponse)
	setting, ok := body["setting"].(map[string]any)
	if !ok {
		t.Fatalf("expected setting object, got %#v", body["setting"])
	}
	if setting["locale"] != "ja" {
		t.Fatalf("expected locale to remain ja, got %#v", setting["locale"])
	}
}

// [LOCALIZATION-BE-S004] Account settings endpoint は未認証 request を拒否する。
func TestAccountSettingsRejectsUnauthenticatedRequest(t *testing.T) {
	t.Parallel()
	env := newJWTAuthTestEnv(t)

	response := performJSON(t, env.router, stdhttp.MethodGet, "/api/v1/account/settings", nil, "")

	assertStatus(t, response, stdhttp.StatusUnauthorized)
	assertNoStore(t, response)
}

// [LOCALIZATION-BE-S013] Refresh response は確定済み AccountID から読み込んだ AccountSetting snapshot を含む。
func TestAuthRefreshReturnsAccountSettingSnapshot(t *testing.T) {
	t.Parallel()
	env := newJWTAuthTestEnv(t)
	accessToken, refreshToken := loginWithJWT(t, env.router, "member@example.com")
	patchResponse := performJSON(t, env.router, stdhttp.MethodPatch, "/api/v1/account/settings", map[string]any{"locale": "en"}, accessToken)
	assertStatus(t, patchResponse, stdhttp.StatusOK)

	refreshResponse := performRefreshWithCookie(t, env.router, refreshToken)

	assertStatus(t, refreshResponse, stdhttp.StatusOK)
	body := decodeJSONBody(t, refreshResponse)
	snapshot, ok := body["accountSetting"].(map[string]any)
	if !ok {
		t.Fatalf("expected accountSetting snapshot, got %#v", body["accountSetting"])
	}
	if snapshot["locale"] != "en" {
		t.Fatalf("expected refresh snapshot locale en, got %#v", snapshot["locale"])
	}
}

// [LOCALIZATION-BE-S013] Refresh response は AccountSetting snapshot を取得できない場合に省略せず fail-closed する。
func TestAuthRefreshFailsClosedWhenAccountSettingSnapshotUnavailable(t *testing.T) {
	t.Parallel()
	clock := &mutableClock{current: time.Date(2026, time.March, 21, 0, 0, 0, 0, time.UTC)}
	stateRepo := newStubAuthStateRepository(clock.Now)
	accountRepo := stubAccountAuthRepositoryWithMember()
	accountSettingRepo := newStubHTTPAccountSettingRepository()
	accountSettingRepo.getErr = domain.ErrAccountSettingNotFound
	env := newJWTAuthTestEnvWithAccountSettingRepository(t, clock, stateRepo, accountRepo, accountSettingRepo)
	_, refreshToken := loginWithJWT(t, env.router, "member@example.com")

	refreshResponse := performRefreshWithCookie(t, env.router, refreshToken)

	assertStatus(t, refreshResponse, stdhttp.StatusServiceUnavailable)
	assertNoStore(t, refreshResponse)
	assertFailureCode(t, refreshResponse, "internal-error")
}

// [AUTH-BE-S055] [AUTH-BE-S059] suspended account の既存 bearer access token は 403 の stable failure で拒否される。
func TestAuthBearerRejectsSuspendedAccount(t *testing.T) {
	t.Parallel()
	env, accountRepo := newJWTAuthTestEnvWithAccountRepo(t)
	accessToken, _ := loginWithJWT(t, env.router, "member@example.com")
	revokedAt := env.now().Add(time.Second)
	accountRepo.account = accountRepo.account.WithStatus("suspended", &revokedAt)

	response := performJSON(t, env.router, stdhttp.MethodGet, "/api/v1/sessions", nil, accessToken)

	assertStatus(t, response, stdhttp.StatusForbidden)
	assertNoStore(t, response)
	assertFailureCode(t, response, "account-suspended")
}

// [AUTH-BE-S056] session_revoked_after 以前に発行された bearer access token は拒否される。
func TestAuthBearerRejectsSessionRevokedAfterOldSession(t *testing.T) {
	t.Parallel()
	env, accountRepo := newJWTAuthTestEnvWithAccountRepo(t)
	accessToken, _ := loginWithJWT(t, env.router, "member@example.com")
	revokedAt := env.now()
	accountRepo.account = accountRepo.account.WithStatus("active", &revokedAt)

	response := performJSON(t, env.router, stdhttp.MethodGet, "/api/v1/sessions", nil, accessToken)

	assertStatus(t, response, stdhttp.StatusForbidden)
	assertNoStore(t, response)
	assertFailureCode(t, response, "account-suspended")
}

// [AUTH-BE-S057] restored account は suspend 前 session では復帰できず、再ログインのみ許可される。
func TestAuthRestoredAccountRejectsPreSuspendSession(t *testing.T) {
	t.Parallel()
	env, accountRepo := newJWTAuthTestEnvWithAccountRepo(t)
	oldAccessToken, _ := loginWithJWT(t, env.router, "member@example.com")
	revokedAt := env.now()
	accountRepo.account = accountRepo.account.WithStatus("active", &revokedAt)

	oldResponse := performJSON(t, env.router, stdhttp.MethodGet, "/api/v1/sessions", nil, oldAccessToken)
	assertStatus(t, oldResponse, stdhttp.StatusForbidden)
	assertFailureCode(t, oldResponse, "account-suspended")

	env.advance(time.Second)
	newAccessToken, _ := loginWithJWT(t, env.router, "member@example.com")
	newResponse := performJSON(t, env.router, stdhttp.MethodGet, "/api/v1/sessions", nil, newAccessToken)
	assertStatus(t, newResponse, stdhttp.StatusOK)
	assertNoStore(t, newResponse)
}

// [AUTH-BE-S057] restored account は同一秒内でも session_revoked_after 後の再ログインなら成功する。
func TestAuthRestoredAccountAllowsSameSecondPostSuspendLogin(t *testing.T) {
	t.Parallel()
	env, accountRepo := newJWTAuthTestEnvWithAccountRepo(t)
	revokedAt := env.now().Add(500 * time.Millisecond)
	accountRepo.account = accountRepo.account.WithStatus("active", &revokedAt)

	env.advance(800 * time.Millisecond)
	newAccessToken, _ := loginWithJWT(t, env.router, "member@example.com")
	newResponse := performJSON(t, env.router, stdhttp.MethodGet, "/api/v1/sessions", nil, newAccessToken)

	assertStatus(t, newResponse, stdhttp.StatusOK)
	assertNoStore(t, newResponse)
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
	accountRepo := newMultiAccountStubAccountAuthRepository(
		stubAccountAuthRepositoryWithMember(),
		stubAccountAuthRepositoryWithAccount("01ARZ3NDEKTSV4RRFFQ69G5FB1", "other@example.com", "other-credential"),
	)
	cfg := testConfig()
	lifecycle, contextRefresh := newTestProductAccountLifecycle(t, accountRepo, clock.Now, cfg.AuthRuntime().RefreshTokenTTL)
	auth := mustNewProductAuthForTest(t, stateRepo, accountRepo, nil, nil, lifecycle, clock.Now, cfg.AuthRuntime(), application.AuthServiceOptionalPorts{WebAuthn: newMockWebAuthnProvider()})
	sessionService := application.NewProductSessionService(lifecycle)

	router := NewRouter(cfg, Dependencies{Auth: auth, TokenService: contextRefresh, SessionService: sessionService})

	at1, _ := loginWithJWT(t, router, "member@example.com")
	listResp := performJSON(t, router, stdhttp.MethodGet, "/api/v1/sessions", nil, at1)
	var listBody map[string]any
	decodeJSON(t, listResp, &listBody)
	sessions := listBody["sessions"].([]any)
	session := sessions[0].(map[string]any)
	targetSessionID := session["sessionId"].(string)
	at2, _ := loginWithJWTUsingCredential(t, router, "other@example.com", "other-credential")

	// member のセッションを other で削除しようとする → 403
	resp := performJSON(t, router, stdhttp.MethodDelete, "/api/v1/sessions/"+targetSessionID, nil, at2)
	assertStatus(t, resp, stdhttp.StatusForbidden)
}

// [AUTH-BE-S038] Unset refresh token TTL is unlimited
func TestAuthRefreshTokenTTLUnsetIsUnlimited(t *testing.T) {
	t.Parallel()
	env := newJWTAuthTestEnv(t)
	_, rt := loginWithJWT(t, env.router, "member@example.com")

	// 時間を進めても TTL が未設定なので有効
	env.advance(48 * time.Hour)
	resp := performRefreshWithCookie(t, env.router, rt)
	assertStatus(t, resp, stdhttp.StatusOK)
}

// [AUTH-BE-S039] 24h+ TTL is applied correctly
func TestAuthRefreshTokenTTL24hApplied(t *testing.T) {
	t.Parallel()
	clock := &mutableClock{current: time.Date(2026, time.March, 21, 0, 0, 0, 0, time.UTC)}
	stateRepo := newStubAuthStateRepository(clock.Now)
	accountRepo := stubAccountAuthRepositoryWithMember()
	cfg := testConfig()
	cfg.Auth.RefreshTokenTTL = 48 * time.Hour
	lifecycle, contextRefresh := newTestProductAccountLifecycle(t, accountRepo, clock.Now, cfg.AuthRuntime().RefreshTokenTTL)
	auth := mustNewProductAuthForTest(t, stateRepo, accountRepo, nil, nil, lifecycle, clock.Now, cfg.AuthRuntime(), application.AuthServiceOptionalPorts{WebAuthn: newMockWebAuthnProvider()})

	accountSettingRepo := newStubHTTPAccountSettingRepository()
	router := NewRouter(cfg, Dependencies{
		Auth:            auth,
		AccountSetting:  productaccounts.NewAccountSettingService(accountSettingRepo),
		AccountSnapshot: productaccounts.NewAccountSettingSnapshotService(accountSettingRepo),
		TokenService:    contextRefresh,
	})

	_, rt := loginWithJWT(t, router, "member@example.com")

	// 48時間以内は有効
	clock.Advance(47 * time.Hour)
	resp := performRefreshWithCookie(t, router, rt)
	assertStatus(t, resp, stdhttp.StatusOK)

	// 48時間を超えると失効
	clock.Advance(2 * time.Hour)
	resp2 := performRefreshWithCookie(t, router, rt)
	assertStatus(t, resp2, stdhttp.StatusUnauthorized)
}

// [AUTH-BE-S010] Auth state store unavailable is internal-error
func TestAuthJWTStoreUnavailableReturnsInternalError(t *testing.T) {
	t.Parallel()
	t.Run("[AUTH-BE-S010] auth state store unavailable fails closed with internal-error", func(t *testing.T) {
		clock := &mutableClock{current: time.Date(2026, time.March, 21, 0, 0, 0, 0, time.UTC)}
		lifecycle, contextRefresh := newTestProductAccountLifecycle(t, stubAccountAuthRepositoryWithMember(), clock.Now, testConfig().AuthRuntime().RefreshTokenTTL)
		auth := mustNewProductAuthForTest(t, failingAuthStateRepository{}, stubAccountAuthRepositoryWithMember(), nil, nil, lifecycle, clock.Now, testConfig().AuthRuntime(), application.AuthServiceOptionalPorts{WebAuthn: newMockWebAuthnProvider()})

		router := NewRouter(testConfig(), Dependencies{Auth: auth, TokenService: contextRefresh})

		resp := performJSON(t, router, stdhttp.MethodPost, "/api/v1/auth/passkey/start", map[string]string{"identifier": "member@example.com"}, "")
		assertStatus(t, resp, stdhttp.StatusServiceUnavailable)
		assertNoStore(t, resp)
		assertFailureCode(t, resp, "internal-error")
	})
}

// [AUTH-BE-S010] Token issuance failure returns 503 fail-close
func TestAuthPasskeyFinishTokenIssuanceFailureReturns503(t *testing.T) {
	t.Parallel()
	t.Run("[AUTH-BE-S010] passkey finish token issuance failure fails closed", func(t *testing.T) {
		env := newJWTAuthTestEnv(t)
		env.productRefreshStore.saveFails = true

		challenge := startPasskey(t, env.router, "member@example.com")
		resp := performJSON(t, env.router, stdhttp.MethodPost, "/api/v1/auth/passkey/finish",
			map[string]any{"credential": assertionCredentialJSON("existing-credential", challengeValue(challenge))}, "")
		assertStatus(t, resp, stdhttp.StatusServiceUnavailable)
		assertFailureCode(t, resp, "internal-error")
	})
}

// [AUTH-BE-S010] Register passkey token issuance failure returns 503 fail-close
func TestAuthRegisterPasskeyTokenIssuanceFailureReturns503(t *testing.T) {
	t.Parallel()
	t.Run("[AUTH-BE-S010] passkey registration token issuance failure fails closed", func(t *testing.T) {
		env := newJWTAuthTestEnv(t)
		recoverySession := consumeRecoverySession(t, env)
		env.productRefreshStore.saveFails = true

		resp := performJSON(t, env.router, stdhttp.MethodPost, "/api/v1/auth/passkey/register",
			map[string]any{"recovery_session": recoverySession, "credential": attestationCredentialJSON("new-credential", "")}, "")
		assertStatus(t, resp, stdhttp.StatusServiceUnavailable)
		assertFailureCode(t, resp, "internal-error")
	})
}

// stubAccountAuthRepositoryWithAccount は指定パラメータで account stub を生成する。
func stubAccountAuthRepositoryWithAccount(accountID, email, credentialHandle string) *stubAccountAuthRepository {
	account, _ := domain.NewAccountAuth(testAccountID(accountID), email, email, "01ARZ3NDEKTSV4RRFFQ69G5FB0", credentialHandle)
	return &stubAccountAuthRepository{account: account}
}
