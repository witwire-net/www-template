package http

import (
	stdhttp "net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"www-template/packages/backend/internal/auth/application"
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

func TestRoutePolicy(t *testing.T) {
	t.Parallel()

	router := newTestRouter(t)
	allowedPublicRoutes := map[string]struct{}{
		"GET /api/v1/status":                       {},
		"POST /api/v1/auth/passkey/start":          {},
		"POST /api/v1/auth/passkey/finish":         {},
		"POST /api/v1/auth/passkey/register/start": {},
		"POST /api/v1/auth/passkey/register":       {},
		"POST /api/v1/auth/recovery":               {},
		"POST /api/v1/auth/recovery/consume":       {},
		"POST /api/v1/auth/refresh":                {},
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

func routePolicyRequestPath(routePath string) string {
	return strings.ReplaceAll(routePath, ":id", "01ARZ3NDEKTSV4RRFFQ69G5FAV")
}

func newTestRouter(t *testing.T) *gin.Engine {
	t.Helper()
	clock := func() time.Time {
		return time.Date(2026, time.March, 21, 0, 0, 0, 0, time.UTC)
	}
	auth := application.NewAuthService(newStubAuthStateRepository(clock), stubAuthAccountRepositoryWithMember(), nil, nil, clock, newSequentialPolicy(), testConfig().AuthRuntime())
	auth.UseWebAuthnProvider(newMockWebAuthnProvider())

	refreshStore := newStubRefreshTokenStore()
	sessionStore := newStubSessionStore()
	tokenService := application.NewTokenService(refreshStore, sessionStore, nil, testConfig().AuthRuntime(), clock, newSequentialPolicy())
	auth.UseTokenService(tokenService)
	sessionService := application.NewSessionService(sessionStore, refreshStore)

	return NewRouter(testConfig(), Dependencies{Auth: auth, TokenService: tokenService, SessionService: sessionService})
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
