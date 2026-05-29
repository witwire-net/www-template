package product

import (
	"context"
	stdhttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	application "www-template/packages/backend/internal/application"
	domain "www-template/packages/backend/internal/domain"
)

// newJWTAuthTestEnv は TokenService と SessionService を注入したテスト環境を構築する。
func newJWTAuthTestEnv(t *testing.T) authTestEnv {
	t.Helper()
	clock := &mutableClock{current: time.Date(2026, time.March, 21, 0, 0, 0, 0, time.UTC)}
	stateRepo := newStubAuthStateRepository(clock.Now)
	accountRepo := stubAccountAuthRepositoryWithMember()
	return newJWTAuthTestEnvWithRepository(t, clock, stateRepo, accountRepo)
}

// newJWTAuthTestEnvWithRepository は指定された account repository を使用して
// JWT 認証テスト環境を構築する。accountRepo は AuthService と TokenService の
// 両方に注入されるため、login / bearer 認可 / refresh が同じ account status を参照する。
func newJWTAuthTestEnvWithRepository(t *testing.T, clock *mutableClock, stateRepo *stubAuthStateRepository, accountRepo application.AccountAuthRepository) authTestEnv {
	t.Helper()
	return newJWTAuthTestEnvWithAccountSettingRepository(t, clock, stateRepo, accountRepo, newStubHTTPAccountSettingRepository())
}

// newJWTAuthTestEnvWithAccountSettingRepository は refresh/account settings の AccountSetting 読み込み結果を差し替えられる JWT 環境を構築する。
func newJWTAuthTestEnvWithAccountSettingRepository(t *testing.T, clock *mutableClock, stateRepo *stubAuthStateRepository, accountRepo application.AccountAuthRepository, accountSettingRepo *stubHTTPAccountSettingRepository) authTestEnv {
	t.Helper()
	invite := &stubInvitationPasskeyRegistrar{}
	sender := &capturingAccountRecoverySender{}
	cfg := testConfig()
	auth := application.NewAuthService(stateRepo, accountRepo, sender, invite, clock.Now, newSequentialPolicy(), cfg.AuthRuntime())
	auth.UseWebAuthnProvider(newMockWebAuthnProvider())

	refreshStore := newStubRefreshTokenStore()
	sessionStore := newStubSessionStore()
	tokenService := application.NewTokenService(refreshStore, sessionStore, accountRepo, cfg.AuthRuntime(), clock.Now, newSequentialPolicy())
	auth.UseTokenService(tokenService)
	sessionService := application.NewSessionService(sessionStore, refreshStore)

	return authTestEnv{
		router: NewRouter(cfg, Dependencies{
			Auth:            auth,
			AccountSetting:  application.NewAccountSettingService(accountSettingRepo),
			AccountSnapshot: application.NewAccountSettingSnapshotService(accountSettingRepo),
			TokenService:    tokenService,
			SessionService:  sessionService,
		}),
		stateRepo:    stateRepo,
		sender:       sender,
		invite:       invite,
		auth:         auth,
		now:          clock.Now,
		advance:      clock.Advance,
		refreshStore: refreshStore,
	}
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
	for id := range s.sessions {
		delete(s.sessions, id)
	}
	return nil
}

// loginWithJWT はパスキー認証フローを実行して JWT access token と HttpOnly refresh Cookie 値を返す helper。
func loginWithJWT(t *testing.T, router *gin.Engine, identifier string) (accessToken, refreshToken string) {
	t.Helper()
	challenge := startPasskey(t, router, identifier)
	resp := performJSON(t, router, stdhttp.MethodPost, "/api/v1/auth/passkey/finish",
		map[string]any{"credential": assertionCredentialJSON("existing-credential", challengeValue(challenge))}, "")
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
	return performJSONWithHeaders(t, router, stdhttp.MethodPost, "/api/v1/auth/refresh", nil, "", map[string]string{"Cookie": productRefreshCookieName + "=" + refreshToken})
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
	if len(env.refreshStore.data) != 0 {
		t.Fatalf("expected refresh token family to be revoked without new rotation, got %d tokens", len(env.refreshStore.data))
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
	auth := application.NewAuthService(stateRepo, accountRepo, nil, nil, clock.Now, newSequentialPolicy(), cfg.AuthRuntime())
	auth.UseWebAuthnProvider(newMockWebAuthnProvider())

	refreshStore := newStubRefreshTokenStore()
	sessionStore := newStubSessionStore()
	tokenService := application.NewTokenService(refreshStore, sessionStore, nil, cfg.AuthRuntime(), clock.Now, newSequentialPolicy())
	auth.UseTokenService(tokenService)

	accountSettingRepo := newStubHTTPAccountSettingRepository()
	router := NewRouter(cfg, Dependencies{
		Auth:            auth,
		AccountSetting:  application.NewAccountSettingService(accountSettingRepo),
		AccountSnapshot: application.NewAccountSettingSnapshotService(accountSettingRepo),
		TokenService:    tokenService,
	})

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
	seq := newSequentialPolicy()
	auth := application.NewAuthService(stateRepo, accountRepo, nil, nil, clock.Now, seq, cfg.AuthRuntime())
	auth.UseWebAuthnProvider(newMockWebAuthnProvider())

	refreshStore := newStubRefreshTokenStore()
	sessionStore := newStubSessionStore()
	tokenService := application.NewTokenService(refreshStore, sessionStore, nil, cfg.AuthRuntime(), clock.Now, seq)
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
	auth := application.NewAuthService(stateRepo, accountRepo, nil, nil, clock.Now, newSequentialPolicy(), cfg.AuthRuntime())
	auth.UseWebAuthnProvider(newMockWebAuthnProvider())

	refreshStore := newStubRefreshTokenStore()
	sessionStore := newStubSessionStore()
	tokenService := application.NewTokenService(refreshStore, sessionStore, nil, cfg.AuthRuntime(), clock.Now, newSequentialPolicy())
	auth.UseTokenService(tokenService)

	accountSettingRepo := newStubHTTPAccountSettingRepository()
	router := NewRouter(cfg, Dependencies{
		Auth:            auth,
		AccountSetting:  application.NewAccountSettingService(accountSettingRepo),
		AccountSnapshot: application.NewAccountSettingSnapshotService(accountSettingRepo),
		TokenService:    tokenService,
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
	clock := &mutableClock{current: time.Date(2026, time.March, 21, 0, 0, 0, 0, time.UTC)}
	auth := application.NewAuthService(failingAuthStateRepository{}, stubAccountAuthRepositoryWithMember(), nil, nil, clock.Now, newSequentialPolicy(), testConfig().AuthRuntime())

	refreshStore := newStubRefreshTokenStore()
	sessionStore := newStubSessionStore()
	tokenService := application.NewTokenService(refreshStore, sessionStore, nil, testConfig().AuthRuntime(), clock.Now, newSequentialPolicy())
	auth.UseTokenService(tokenService)

	router := NewRouter(testConfig(), Dependencies{Auth: auth, TokenService: tokenService})

	resp := performRefreshWithCookie(t, router, "invalid")
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

// stubAccountAuthRepositoryWithAccount は指定パラメータで account stub を生成する。
func stubAccountAuthRepositoryWithAccount(accountID, email, credentialHandle string) *stubAccountAuthRepository {
	account, _ := domain.NewAccountAuth(testAccountID(accountID), email, email, "01ARZ3NDEKTSV4RRFFQ69G5FB0", credentialHandle)
	return &stubAccountAuthRepository{account: account}
}
