package http

import (
	"context"
	"errors"
	stdhttp "net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	openapi_types "github.com/oapi-codegen/runtime/types"

	"witwire.net/www-template/packages/backend/internal/generated/openapi"
	"witwire.net/www-template/packages/backend/internal/types"
	"witwire.net/www-template/packages/backend/internal/usecases"
)

type Dependencies struct {
	Profiles *usecases.ProfilesService
}

type StrictServer struct {
	profiles *usecases.ProfilesService
}

func NewRouter(cfg types.Config, dependencies Dependencies) *gin.Engine {
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(appAuthMiddleware(cfg))
	router.Use(cors.New(cors.Config{
		AllowCredentials: true,
		AllowHeaders:     []string{"Content-Type", "Authorization"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowOrigins:     cfg.AllowedOrigins,
		MaxAge:           12 * time.Hour,
	}))

	router.GET("/health", func(c *gin.Context) {
		c.JSON(stdhttp.StatusOK, gin.H{
			"status":    "ok",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		})
	})

	strictHandler := openapi.NewStrictHandler(NewStrictServer(dependencies), nil)
	openapi.RegisterHandlers(router, strictHandler)

	router.NoRoute(func(c *gin.Context) {
		c.JSON(stdhttp.StatusNotFound, gin.H{
			"error": "not found",
			"path":  c.Request.URL.Path,
		})
	})

	return router
}

func NewStrictServer(dependencies Dependencies) *StrictServer {
	return &StrictServer{profiles: dependencies.Profiles}
}

func (s *StrictServer) GetStatus(ctx context.Context, _ openapi.GetStatusRequestObject) (openapi.GetStatusResponseObject, error) {
	status := s.profiles.GetStatus(ctx)
	return openapi.GetStatus200JSONResponse{
		Message:   status.Message,
		Timestamp: status.Timestamp,
	}, nil
}

func (s *StrictServer) ListProfiles(ctx context.Context, _ openapi.ListProfilesRequestObject) (openapi.ListProfilesResponseObject, error) {
	profiles, err := s.profiles.ListProfiles(ctx)
	if err != nil {
		return nil, err
	}

	response := make([]openapi.Profile, 0, len(profiles))
	for _, profile := range profiles {
		response = append(response, toOpenAPIProfile(profile))
	}

	return openapi.ListProfiles200JSONResponse(response), nil
}

func (s *StrictServer) CreateProfile(ctx context.Context, request openapi.CreateProfileRequestObject) (openapi.CreateProfileResponseObject, error) {
	if request.Body == nil {
		return openapi.CreateProfile400JSONResponse{Error: "request body is required"}, nil
	}

	createdProfile, err := s.profiles.CreateProfile(ctx, usecases.CreateProfileInput{
		Email: string(request.Body.Email),
		Name:  request.Body.Name,
	})
	if err != nil {
		if errors.Is(err, usecases.ErrInvalidProfileEmail) || errors.Is(err, usecases.ErrInvalidProfileName) {
			return openapi.CreateProfile400JSONResponse{Error: err.Error()}, nil
		}

		return nil, err
	}

	return openapi.CreateProfile201JSONResponse(toOpenAPIProfile(createdProfile)), nil
}

func (s *StrictServer) GetProfile(ctx context.Context, request openapi.GetProfileRequestObject) (openapi.GetProfileResponseObject, error) {
	profile, err := s.profiles.GetProfile(ctx, request.Id)
	if err != nil {
		if errors.Is(err, usecases.ErrProfileNotFound) {
			return openapi.GetProfile404JSONResponse{Error: err.Error()}, nil
		}

		return nil, err
	}

	return openapi.GetProfile200JSONResponse(toOpenAPIProfile(profile)), nil
}

func (s *StrictServer) ListAppProfiles(ctx context.Context, _ openapi.ListAppProfilesRequestObject) (openapi.ListAppProfilesResponseObject, error) {
	profiles, err := s.profiles.ListProfiles(ctx)
	if err != nil {
		return nil, err
	}

	response := make([]openapi.Profile, 0, len(profiles))
	for _, profile := range profiles {
		response = append(response, toOpenAPIProfile(profile))
	}

	return openapi.ListAppProfiles200JSONResponse(response), nil
}

func (s *StrictServer) GetAppProfile(ctx context.Context, request openapi.GetAppProfileRequestObject) (openapi.GetAppProfileResponseObject, error) {
	profile, err := s.profiles.GetProfile(ctx, request.Id)
	if err != nil {
		if errors.Is(err, usecases.ErrProfileNotFound) {
			return openapi.GetAppProfile404JSONResponse{Error: err.Error()}, nil
		}

		return nil, err
	}

	return openapi.GetAppProfile200JSONResponse(toOpenAPIProfile(profile)), nil
}

func toOpenAPIProfile(profile usecases.Profile) openapi.Profile {
	return openapi.Profile{
		CreatedAt: profile.CreatedAt,
		Email:     openapi_types.Email(profile.Email),
		Name:      profile.Name,
		Id:        profile.ID,
	}
}
