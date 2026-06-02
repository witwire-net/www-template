package product

import (
	stdhttp "net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	application "www-template/packages/backend/internal/application/auth"
	"www-template/packages/backend/internal/platform/config"
)

func TestHealthRoute(t *testing.T) {
	t.Parallel()

	router := newTestRouter(t)
	request := httptest.NewRequest(stdhttp.MethodGet, "/health", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != stdhttp.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}
}

func TestAppSurfaceFallsBackToNotFound(t *testing.T) {
	t.Parallel()

	router := newTestRouter(t)
	request := httptest.NewRequest(stdhttp.MethodGet, "/api/v1/unknown", nil)
	challengeRecorder := performJSON(t, router, stdhttp.MethodPost, "/api/v1/auth/passkey/start", map[string]string{"identifier": "member@example.com"}, "")
	challenge := decodeJSONBody(t, challengeRecorder)
	finishRecorder := performJSON(t, router, stdhttp.MethodPost, "/api/v1/auth/passkey/finish", map[string]any{"credential": assertionCredentialJSON("existing-credential", challengeValue(challenge))}, "")
	finishBody := decodeJSONBody(t, finishRecorder)
	request.Header.Set("Authorization", "Bearer "+finishBody["accessToken"].(string))
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != stdhttp.StatusNotFound {
		t.Fatalf("expected status 404, got %d", recorder.Code)
	}
}

func TestAppAuthEndpointRequiresAuthorization(t *testing.T) {
	t.Parallel()

	router := newTestRouter(t)
	request := httptest.NewRequest(stdhttp.MethodPost, "/api/v1/auth/logout", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != stdhttp.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", recorder.Code)
	}
}

// [ADMIN-AUTH-BE-S056] Product host の `/api/v1/auth/passkey/*` は Product account auth flow として完結し、同一 relative path の Admin operator auth と混線しないことを trace する。
func TestAppAuthEndpointSucceedsWithBearerToken(t *testing.T) {
	t.Parallel()

	router := newTestRouter(t)
	challengeRecorder := performJSON(t, router, stdhttp.MethodPost, "/api/v1/auth/passkey/start", map[string]string{"identifier": "member@example.com"}, "")
	if challengeRecorder.Code != stdhttp.StatusOK {
		t.Fatalf("expected status 200, got %d", challengeRecorder.Code)
	}
	challenge := decodeJSONBody(t, challengeRecorder)

	finishRecorder := performJSON(t, router, stdhttp.MethodPost, "/api/v1/auth/passkey/finish", map[string]any{"credential": assertionCredentialJSON("existing-credential", challengeValue(challenge))}, "")
	if finishRecorder.Code != stdhttp.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", finishRecorder.Code, finishRecorder.Body.String())
	}

	finishBody := decodeJSONBody(t, finishRecorder)
	bearer, ok := finishBody["accessToken"].(string)
	if !ok || bearer == "" {
		t.Fatalf("expected accessToken in response, got %v", finishBody["accessToken"])
	}
	logoutRecorder := performJSON(t, router, stdhttp.MethodPost, "/api/v1/auth/logout", nil, bearer)

	if logoutRecorder.Code != stdhttp.StatusOK {
		t.Fatalf("expected status 200, got %d", logoutRecorder.Code)
	}
}

func TestProductHostAdminAuthBoundaryScenarioTitles(t *testing.T) {
	t.Parallel()

	t.Run("[ADMIN-AUTH-BE-S056] Product host keeps passkey auth on Product account flow", func(t *testing.T) {
		// Step 1: Product router の同一 relative path で passkey start/finish を実行し、Admin operator auth handler を必要としない Product flow を作る。
		router := newTestRouter(t)
		challengeRecorder := performJSON(t, router, stdhttp.MethodPost, "/api/v1/auth/passkey/start", map[string]string{"identifier": "member@example.com"}, "")
		if challengeRecorder.Code != stdhttp.StatusOK {
			t.Fatalf("expected Product passkey start status 200, got %d", challengeRecorder.Code)
		}
		challenge := decodeJSONBody(t, challengeRecorder)

		// Step 2: finish response が Product account payload だけを持ち、Admin operator payload や Admin refresh Cookie 名を含まないことを確認する。
		finishRecorder := performJSON(t, router, stdhttp.MethodPost, "/api/v1/auth/passkey/finish", map[string]any{"credential": assertionCredentialJSON("existing-credential", challengeValue(challenge))}, "")
		if finishRecorder.Code != stdhttp.StatusOK {
			t.Fatalf("expected Product passkey finish status 200, got %d body=%s", finishRecorder.Code, finishRecorder.Body.String())
		}
		finishBody := decodeJSONBody(t, finishRecorder)
		if finishBody["account"] == nil || finishBody["operator"] != nil {
			t.Fatalf("expected Product account payload without Admin operator payload, got %#v", finishBody)
		}
		for _, header := range finishRecorder.Header().Values("Set-Cookie") {
			if strings.Contains(header, "admin_refresh_token") {
				t.Fatalf("Product host must not set Admin refresh Cookie, got %q", header)
			}
		}
	})
}

func TestRoutePolicy(t *testing.T) {
	t.Parallel()

	router := newTestRouter(t)
	allowedPublicRoutes := map[string]struct{}{
		"GET /api/v1/status":                                {},
		"POST /api/v1/auth/passkey/start":                   {},
		"POST /api/v1/auth/passkey/finish":                  {},
		"POST /api/v1/auth/passkey/register/start":          {},
		"POST /api/v1/auth/passkey/register":                {},
		"POST /api/v1/auth/recovery":                        {},
		"POST /api/v1/auth/recovery/consume":                {},
		"POST /api/v1/auth/contexts/:authContextId/refresh": {},
	}
	seenPublicRoutes := map[string]struct{}{}

	for _, route := range router.Routes() {
		if route.Path == "/health" || route.Path == "/metrics" {
			continue
		}
		if !strings.HasPrefix(route.Path, "/api/v1/") {
			t.Fatalf("route policy violation: %s %s", route.Method, route.Path)
		}

		routeKey := route.Method + " " + route.Path
		if _, ok := allowedPublicRoutes[routeKey]; ok {
			seenPublicRoutes[routeKey] = struct{}{}
			continue
		}

		request := httptest.NewRequest(route.Method, routePolicyRequestPath(route.Path), strings.NewReader("{}"))
		request.Header.Set("Content-Type", "application/json")
		recorder := httptest.NewRecorder()

		router.ServeHTTP(recorder, request)

		if recorder.Code != stdhttp.StatusUnauthorized {
			t.Fatalf("route policy violation: expected bearer protection for %s, got %d body=%s", routeKey, recorder.Code, recorder.Body.String())
		}
	}

	publicRouteKeys := make([]string, 0, len(allowedPublicRoutes))
	for routeKey := range allowedPublicRoutes {
		publicRouteKeys = append(publicRouteKeys, routeKey)
		if _, ok := seenPublicRoutes[routeKey]; !ok {
			t.Fatalf("route policy violation: missing public route %s", routeKey)
		}
	}
	slices.Sort(publicRouteKeys)
}

// [ADMIN-CONSOLE-BE-S056] TestProductRuntimeDoesNotRegisterAdminOperations は Product runtime の router が Admin 専用 operation を公開しないことを検証する。
// Product と Admin は同じ `/api/v1/*` path 空間を別 origin / 別 binary で使うため、Product 側で Admin 専用 path が 404 になることを route table 境界の証拠にする。
func TestProductRuntimeDoesNotRegisterAdminOperations(t *testing.T) {
	t.Parallel()

	// Step 1: Product router に有効な Product bearer を渡し、認証 middleware で止まらず NoRoute まで到達できる状態にする。
	router := newTestRouter(t)
	bearer := issueProductBearerForRoutePolicy(t, router)
	adminOnlyRoutes := []struct {
		method string
		path   string
	}{
		{method: stdhttp.MethodGet, path: "/api/v1/accounts"},
		{method: stdhttp.MethodPost, path: "/api/v1/accounts"},
		{method: stdhttp.MethodGet, path: "/api/v1/accounts/01ARZ3NDEKTSV4RRFFQ69G5FAV"},
		{method: stdhttp.MethodPost, path: "/api/v1/auth/operator-setup/finish"},
		{method: stdhttp.MethodPost, path: "/api/v1/auth/operator-setup/start"},
		{method: stdhttp.MethodGet, path: "/api/v1/auth/operator/current"},
		{method: stdhttp.MethodPost, path: "/api/v1/auth/operator/logout"},
		{method: stdhttp.MethodPost, path: "/api/v1/auth/operator/refresh"},
	}

	// Step 2: Admin 専用 path が Product router で見つからないことを確認し、Product binary への Admin operation 混入を検出できるようにする。
	for _, route := range adminOnlyRoutes {
		response := performJSON(t, router, route.method, route.path, map[string]string{}, bearer)
		if response.Code != stdhttp.StatusNotFound {
			t.Fatalf("expected Product router to reject Admin route %s %s with 404, got %d body=%s", route.method, route.path, response.Code, response.Body.String())
		}
	}
}

func issueProductBearerForRoutePolicy(t *testing.T, router *gin.Engine) string {
	t.Helper()

	// Step 1: Product passkey start endpoint で challenge を発行し、Product 認証 flow だけで bearer を作る。
	challengeRecorder := performJSON(t, router, stdhttp.MethodPost, "/api/v1/auth/passkey/start", map[string]string{"identifier": "member@example.com"}, "")
	if challengeRecorder.Code != stdhttp.StatusOK {
		t.Fatalf("expected challenge status 200, got %d body=%s", challengeRecorder.Code, challengeRecorder.Body.String())
	}
	challenge := decodeJSONBody(t, challengeRecorder)

	// Step 2: Product passkey finish endpoint で accessToken を取得し、Admin route absence の検証時に認証済み Product request として使う。
	finishRecorder := performJSON(t, router, stdhttp.MethodPost, "/api/v1/auth/passkey/finish", map[string]any{"credential": assertionCredentialJSON("existing-credential", challengeValue(challenge))}, "")
	if finishRecorder.Code != stdhttp.StatusOK {
		t.Fatalf("expected finish status 200, got %d body=%s", finishRecorder.Code, finishRecorder.Body.String())
	}
	finishBody := decodeJSONBody(t, finishRecorder)
	bearer, ok := finishBody["accessToken"].(string)
	if !ok || bearer == "" {
		t.Fatalf("expected accessToken in response, got %v", finishBody["accessToken"])
	}
	return bearer
}

func routePolicyRequestPath(routePath string) string {
	return strings.ReplaceAll(routePath, ":id", "01ARZ3NDEKTSV4RRFFQ69G5FAV")
}

func newTestRouter(t *testing.T) *gin.Engine {
	t.Helper()
	clock := func() time.Time {
		return time.Date(2026, time.March, 21, 0, 0, 0, 0, time.UTC)
	}
	accountRepo := stubAccountAuthRepositoryWithMember()
	lifecycle, contextRefresh := newTestProductAccountLifecycle(t, accountRepo, clock, testConfig().AuthRuntime().RefreshTokenTTL)
	auth := mustNewProductAuthForTest(t, newStubAuthStateRepository(clock), accountRepo, nil, nil, lifecycle, clock, testConfig().AuthRuntime(), application.AuthServiceOptionalPorts{WebAuthn: newMockWebAuthnProvider()})
	sessionService := application.NewProductSessionService(lifecycle)

	return NewRouter(testConfig(), Dependencies{Auth: auth, TokenService: contextRefresh, SessionService: sessionService})
}

func testConfig() config.Config {
	cfg := config.Config{
		AllowedOrigins: []string{"http://localhost:5173", "http://localhost:5174"},
		AppBearerToken: "dev-app-auth",
		Port:           "8080",
	}
	cfg.Auth.JWTSecret = "test-jwt-secret-key-must-be-at-least-32bytes"
	cfg.Auth.RefreshTokenTTL = 0
	cfg.Auth.SessionAbsoluteTTL = 14 * 24 * time.Hour
	cfg.Auth.SessionIdleTTL = 12 * time.Hour
	return cfg
}
