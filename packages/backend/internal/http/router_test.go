package http

import (
	"context"
	"encoding/json"
	stdhttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"witwire.net/www-template/packages/backend/internal/domain"
	"witwire.net/www-template/packages/backend/internal/types"
	"witwire.net/www-template/packages/backend/internal/usecases"
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

func TestProfilesRoutes(t *testing.T) {
	t.Parallel()

	router := newTestRouter(t)

	createRequest := httptest.NewRequest(stdhttp.MethodPost, "/api/v1/profiles", strings.NewReader(`{"name":"Ada","email":"ada@example.com"}`))
	createRequest.Header.Set("Content-Type", "application/json")
	createRecorder := httptest.NewRecorder()
	router.ServeHTTP(createRecorder, createRequest)
	if createRecorder.Code != stdhttp.StatusCreated {
		t.Fatalf("expected status 201, got %d", createRecorder.Code)
	}

	listRequest := httptest.NewRequest(stdhttp.MethodGet, "/api/v1/profiles", nil)
	listRecorder := httptest.NewRecorder()
	router.ServeHTTP(listRecorder, listRequest)
	if listRecorder.Code != stdhttp.StatusOK {
		t.Fatalf("expected status 200, got %d", listRecorder.Code)
	}

	var listed []map[string]any
	if err := json.Unmarshal(listRecorder.Body.Bytes(), &listed); err != nil {
		t.Fatalf("unmarshal list response: %v", err)
	}
	if len(listed) != 1 {
		t.Fatalf("expected 1 profile, got %d", len(listed))
	}

	getRequest := httptest.NewRequest(stdhttp.MethodGet, "/api/v1/profiles/1", nil)
	getRecorder := httptest.NewRecorder()
	router.ServeHTTP(getRecorder, getRequest)
	if getRecorder.Code != stdhttp.StatusOK {
		t.Fatalf("expected status 200, got %d", getRecorder.Code)
	}
}

func TestAppSurfaceFallsBackToNotFound(t *testing.T) {
	t.Parallel()

	router := newTestRouter(t)
	request := httptest.NewRequest(stdhttp.MethodGet, "/api/v1/app/unknown", nil)
	request.Header.Set("Authorization", testConfig().AppAuthorizationValue())
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != stdhttp.StatusNotFound {
		t.Fatalf("expected status 404, got %d", recorder.Code)
	}
}

func TestAppProfilesRequireAuthorization(t *testing.T) {
	t.Parallel()

	router := newTestRouter(t)
	request := httptest.NewRequest(stdhttp.MethodGet, "/api/v1/app/profiles", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != stdhttp.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", recorder.Code)
	}
}

func TestAppProfilesSucceedWithBearerToken(t *testing.T) {
	t.Parallel()

	router := newTestRouter(t)
	createRequest := httptest.NewRequest(stdhttp.MethodPost, "/api/v1/profiles", strings.NewReader(`{"name":"Ada","email":"ada@example.com"}`))
	createRequest.Header.Set("Content-Type", "application/json")
	createRecorder := httptest.NewRecorder()
	router.ServeHTTP(createRecorder, createRequest)

	listRequest := httptest.NewRequest(stdhttp.MethodGet, "/api/v1/app/profiles", nil)
	listRequest.Header.Set("Authorization", testConfig().AppAuthorizationValue())
	listRecorder := httptest.NewRecorder()
	router.ServeHTTP(listRecorder, listRequest)

	if listRecorder.Code != stdhttp.StatusOK {
		t.Fatalf("expected status 200, got %d", listRecorder.Code)
	}
}

func TestAppProfileByIDRequiresBearerToken(t *testing.T) {
	t.Parallel()

	router := newTestRouter(t)
	createRequest := httptest.NewRequest(stdhttp.MethodPost, "/api/v1/profiles", strings.NewReader(`{"name":"Ada","email":"ada@example.com"}`))
	createRequest.Header.Set("Content-Type", "application/json")
	createRecorder := httptest.NewRecorder()
	router.ServeHTTP(createRecorder, createRequest)

	request := httptest.NewRequest(stdhttp.MethodGet, "/api/v1/app/profiles/1", nil)
	request.Header.Set("Authorization", testConfig().AppAuthorizationValue())
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != stdhttp.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}
}

func TestRoutePolicy(t *testing.T) {
	t.Parallel()

	router := newTestRouter(t)
	allowedPublicRoutes := map[string]struct{}{
		"/api/v1/status":       {},
		"/api/v1/profiles":     {},
		"/api/v1/profiles/:id": {},
	}

	for _, route := range router.Routes() {
		if route.Path == "/health" {
			continue
		}

		if strings.HasPrefix(route.Path, "/api/v1/app") {
			continue
		}

		if _, ok := allowedPublicRoutes[route.Path]; ok {
			continue
		}

		t.Fatalf("route policy violation: %s %s", route.Method, route.Path)
	}
}

func newTestRouter(t *testing.T) *gin.Engine {
	t.Helper()

	repository := &stubProfileRepository{profiles: make(map[int64]domain.Profile), nextID: 1}
	profiles := usecases.NewProfilesService(repository, func() time.Time {
		return time.Now().UTC()
	})

	return NewRouter(
		testConfig(),
		Dependencies{Profiles: profiles},
	)
}

func testConfig() types.Config {
	return types.Config{
		AllowedOrigins: []string{"http://localhost:5173"},
		AppBearerToken: "dev-app-auth",
		Port:           "8080",
		ProfileStore:   "memory",
	}
}

type stubProfileRepository struct {
	nextID   int64
	profiles map[int64]domain.Profile
}

func (r *stubProfileRepository) Create(_ context.Context, input domain.CreateProfileInput) (domain.Profile, error) {
	profile, err := domain.NewProfile(r.nextID, time.Now().UTC(), input)
	if err != nil {
		var empty domain.Profile
		return empty, err
	}

	r.profiles[profile.ID()] = profile
	r.nextID++
	return profile, nil
}

func (r *stubProfileRepository) GetByID(_ context.Context, id int64) (domain.Profile, error) {
	profile, ok := r.profiles[id]
	if !ok {
		var empty domain.Profile
		return empty, domain.ErrProfileNotFound
	}

	return profile, nil
}

func (r *stubProfileRepository) List(context.Context) ([]domain.Profile, error) {
	profiles := make([]domain.Profile, 0, len(r.profiles))
	for id := int64(1); id < r.nextID; id++ {
		profile, ok := r.profiles[id]
		if ok {
			profiles = append(profiles, profile)
		}
	}

	return profiles, nil
}
