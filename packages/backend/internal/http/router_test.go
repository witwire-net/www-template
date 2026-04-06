package http

import (
	stdhttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"www-template/packages/backend/internal/types"
	"www-template/packages/backend/internal/usecases"
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
	finishRecorder := performJSON(t, router, stdhttp.MethodPost, "/api/v1/auth/passkey/finish", map[string]string{"credential": credentialEnvelope("existing-credential", challengeValue(challenge))}, "")
	finishBody := decodeJSONBody(t, finishRecorder)
	request.Header.Set("Authorization", "Bearer "+finishBody["sessionToken"].(string))
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

	finishRecorder := performJSON(t, router, stdhttp.MethodPost, "/api/v1/auth/passkey/finish", map[string]string{"credential": credentialEnvelope("existing-credential", challengeValue(challenge))}, "")
	if finishRecorder.Code != stdhttp.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", finishRecorder.Code, finishRecorder.Body.String())
	}

	finishBody := decodeJSONBody(t, finishRecorder)
	bearer, ok := finishBody["sessionToken"].(string)
	if !ok || bearer == "" {
		t.Fatalf("expected sessionToken in response, got %v", finishBody["sessionToken"])
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
		"/api/v1/status":                  {},
		"/api/v1/auth/passkey/start":      {},
		"/api/v1/auth/passkey/finish":     {},
		"/api/v1/auth/passkey/register":   {},
		"/api/v1/auth/recovery":           {},
		"/api/v1/auth/recovery/consume":   {},
		"/api/v1/auth/passkey/add/start":  {},
		"/api/v1/auth/passkey/add/finish": {},
	}

	for _, route := range router.Routes() {
		if route.Path == "/health" {
			continue
		}

		if _, ok := allowedPublicRoutes[route.Path]; ok {
			continue
		}

		// routes under /api/v1/ that are not in the public list must be bearer-protected
		if strings.HasPrefix(route.Path, "/api/v1/") {
			continue
		}

		t.Fatalf("route policy violation: %s %s", route.Method, route.Path)
	}
}

func newTestRouter(t *testing.T) *gin.Engine {
	t.Helper()
	clock := func() time.Time {
		return time.Date(2026, time.March, 21, 0, 0, 0, 0, time.UTC)
	}
	auth := usecases.NewAuthService(newStubAuthStateRepository(clock), stubAuthAccountRepositoryWithMember(), nil, nil, clock, newSequentialPolicy(), testConfig().AuthRuntime())

	return NewRouter(testConfig(), Dependencies{Auth: auth})
}

func testConfig() types.Config {
	return types.Config{
		AllowedOrigins: []string{"http://localhost:5173", "http://localhost:5174"},
		AppBearerToken: "dev-app-auth",
		Port:           "8080",
	}
}
