package http

import (
	"context"
	"errors"
	stdhttp "net/http"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	openapi_types "github.com/oapi-codegen/runtime/types"

	"www-template/packages/backend/internal/generated/openapi"
	"www-template/packages/backend/internal/types"
	"www-template/packages/backend/internal/usecases"
)

type Dependencies struct {
	Auth     *usecases.AuthService
	Profiles *usecases.ProfilesService
}

type StrictServer struct {
	auth     *usecases.AuthService
	profiles *usecases.ProfilesService
}

func NewRouter(cfg types.Config, dependencies Dependencies) *gin.Engine {
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(authNoStoreAndBindErrorMiddleware())
	router.Use(appAuthMiddleware(cfg, dependencies.Auth))
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
	return &StrictServer{auth: dependencies.Auth, profiles: dependencies.Profiles}
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

func (s *StrictServer) Logout(ctx context.Context, _ openapi.LogoutRequestObject) (openapi.LogoutResponseObject, error) {
	requestID, err := s.auth.Logout(ctx, bearerTokenFromContext(ctx))
	if err != nil {
		failureRequestID := nextAuthRequestID()
		if errors.Is(err, usecases.ErrInternalError) {
			return openapi.Logout503JSONResponse{Body: authFailureResponseObject(failureRequestID, err), Headers: openapi.Logout503ResponseHeaders{CacheControl: noStoreValue}}, nil
		}
		return openapi.Logout401JSONResponse{Body: authFailureResponseObject(failureRequestID, err), Headers: openapi.Logout401ResponseHeaders{CacheControl: noStoreValue}}, nil
	}

	return openapi.Logout200JSONResponse{
		Body:    openapi.LogoutResponse{RequestId: requestID, Revoked: openapi.LogoutResponseRevokedTrue},
		Headers: openapi.Logout200ResponseHeaders{CacheControl: noStoreValue},
	}, nil
}

func (s *StrictServer) StartPasskeyAuthentication(ctx context.Context, request openapi.StartPasskeyAuthenticationRequestObject) (openapi.StartPasskeyAuthenticationResponseObject, error) {
	if request.Body == nil {
		return openapi.StartPasskeyAuthentication400JSONResponse{Body: authOperationError(nextAuthRequestID(), nonRevealingAuthRejectMessage), Headers: openapi.StartPasskeyAuthentication400ResponseHeaders{CacheControl: noStoreValue}}, nil
	}

	result, err := s.auth.StartPasskeyAuthentication(ctx, usecases.StartPasskeyAuthenticationInput{Identifier: request.Body.Identifier, ClientIP: clientIPFromContext(ctx)})
	if err != nil {
		failureRequestID := nextAuthRequestID()
		if errors.Is(err, usecases.ErrInternalError) {
			return openapi.StartPasskeyAuthentication503JSONResponse{Body: authFailureResponseObject(failureRequestID, err), Headers: openapi.StartPasskeyAuthentication503ResponseHeaders{CacheControl: noStoreValue}}, nil
		}
		return openapi.StartPasskeyAuthentication400JSONResponse{Body: authOperationError(failureRequestID, nonRevealingAuthRejectMessage), Headers: openapi.StartPasskeyAuthentication400ResponseHeaders{CacheControl: noStoreValue}}, nil
	}

	return openapi.StartPasskeyAuthentication200JSONResponse{
		Body:    openapi.PasskeyStartResponse{RequestId: result.RequestID, Challenge: result.Challenge, RpId: result.WebAuthnRPID},
		Headers: openapi.StartPasskeyAuthentication200ResponseHeaders{CacheControl: noStoreValue},
	}, nil
}

func (s *StrictServer) FinishPasskeyAuthentication(ctx context.Context, request openapi.FinishPasskeyAuthenticationRequestObject) (openapi.FinishPasskeyAuthenticationResponseObject, error) {
	if request.Body == nil {
		return openapi.FinishPasskeyAuthentication400JSONResponse{Body: authOperationError(nextAuthRequestID(), "request body is required"), Headers: openapi.FinishPasskeyAuthentication400ResponseHeaders{CacheControl: noStoreValue}}, nil
	}

	result, err := s.auth.FinishPasskeyAuthentication(ctx, usecases.FinishPasskeyAuthenticationInput{Credential: request.Body.Credential, ClientIP: clientIPFromContext(ctx)})
	if err != nil {
		failureRequestID := nextAuthRequestID()
		if errors.Is(err, usecases.ErrInternalError) {
			return openapi.FinishPasskeyAuthentication503JSONResponse{Body: authFailureResponseObject(failureRequestID, err), Headers: openapi.FinishPasskeyAuthentication503ResponseHeaders{CacheControl: noStoreValue}}, nil
		}
		return openapi.FinishPasskeyAuthentication400JSONResponse{Body: authOperationError(failureRequestID, err.Error()), Headers: openapi.FinishPasskeyAuthentication400ResponseHeaders{CacheControl: noStoreValue}}, nil
	}

	return openapi.FinishPasskeyAuthentication200JSONResponse{
		Body: openapi.AuthSessionResponse{
			RequestId:           result.RequestID,
			AccountId:           result.AccountID,
			PasskeyCredentialId: result.PasskeyCredentialID,
			SessionId:           result.SessionID,
			SessionToken:        result.SessionToken,
			ExpiresAt:           result.ExpiresAt,
		},
		Headers: openapi.FinishPasskeyAuthentication200ResponseHeaders{CacheControl: noStoreValue},
	}, nil
}

func (s *StrictServer) RequestPasskeyRecovery(ctx context.Context, request openapi.RequestPasskeyRecoveryRequestObject) (openapi.RequestPasskeyRecoveryResponseObject, error) {
	if request.Body == nil {
		return openapi.RequestPasskeyRecovery400JSONResponse{Body: authOperationError(nextAuthRequestID(), "request body is required"), Headers: openapi.RequestPasskeyRecovery400ResponseHeaders{CacheControl: noStoreValue}}, nil
	}

	result, err := s.auth.RequestPasskeyRecovery(ctx, usecases.RequestPasskeyRecoveryInput{Email: string(request.Body.Email), ClientIP: clientIPFromContext(ctx)})
	if err != nil {
		return openapi.RequestPasskeyRecovery503JSONResponse{Body: authFailureResponseObject(nextAuthRequestID(), err), Headers: openapi.RequestPasskeyRecovery503ResponseHeaders{CacheControl: noStoreValue}}, nil
	}

	return openapi.RequestPasskeyRecovery202JSONResponse{
		Body:    openapi.RecoveryAcceptedResponse{RequestId: result.RequestID, Accepted: openapi.RecoveryAcceptedResponseAcceptedTrue},
		Headers: openapi.RequestPasskeyRecovery202ResponseHeaders{CacheControl: noStoreValue},
	}, nil
}

func (s *StrictServer) ConsumeRecoveryToken(ctx context.Context, request openapi.ConsumeRecoveryTokenRequestObject) (openapi.ConsumeRecoveryTokenResponseObject, error) {
	if request.Body == nil {
		return openapi.ConsumeRecoveryToken400JSONResponse{Body: authOperationError(nextAuthRequestID(), "request body is required"), Headers: openapi.ConsumeRecoveryToken400ResponseHeaders{CacheControl: noStoreValue}}, nil
	}

	result, err := s.auth.ConsumeRecoveryToken(ctx, usecases.ConsumeRecoveryTokenInput{Token: request.Body.Token, ClientIP: clientIPFromContext(ctx)})
	if err != nil {
		failureRequestID := nextAuthRequestID()
		if errors.Is(err, usecases.ErrInternalError) {
			return openapi.ConsumeRecoveryToken503JSONResponse{Body: authFailureResponseObject(failureRequestID, err), Headers: openapi.ConsumeRecoveryToken503ResponseHeaders{CacheControl: noStoreValue}}, nil
		}
		return openapi.ConsumeRecoveryToken400JSONResponse{Body: authOperationError(failureRequestID, err.Error()), Headers: openapi.ConsumeRecoveryToken400ResponseHeaders{CacheControl: noStoreValue}}, nil
	}

	return openapi.ConsumeRecoveryToken200JSONResponse{
		Body:    openapi.RecoveryConsumeResponse{RequestId: result.RequestID, RecoveryTokenId: result.RecoveryTokenID, RecoverySessionId: result.RecoverySessionID, RecoverySession: result.RecoverySessionRef, ExpiresAt: result.ExpiresAt},
		Headers: openapi.ConsumeRecoveryToken200ResponseHeaders{CacheControl: noStoreValue},
	}, nil
}

func (s *StrictServer) RegisterPasskey(ctx context.Context, request openapi.RegisterPasskeyRequestObject) (openapi.RegisterPasskeyResponseObject, error) {
	if request.Body == nil {
		return openapi.RegisterPasskey400JSONResponse{Body: authOperationError(nextAuthRequestID(), "request body is required"), Headers: openapi.RegisterPasskey400ResponseHeaders{CacheControl: noStoreValue}}, nil
	}

	recovery, _ := request.Body.AsRecoveryPasskeyRegisterRequest()
	invitation, _ := request.Body.AsInvitationPasskeyRegisterRequest()
	result, err := s.auth.RegisterPasskey(ctx, usecases.RegisterPasskeyInput{RecoverySession: recovery.RecoverySession, InvitationSession: invitation.InvitationSession, Credential: chooseCredential(recovery.Credential, invitation.Credential), ClientIP: clientIPFromContext(ctx)})
	if err != nil {
		failureRequestID := nextAuthRequestID()
		if errors.Is(err, usecases.ErrInternalError) {
			return openapi.RegisterPasskey503JSONResponse{Body: authFailureResponseObject(failureRequestID, err), Headers: openapi.RegisterPasskey503ResponseHeaders{CacheControl: noStoreValue}}, nil
		}
		return openapi.RegisterPasskey400JSONResponse{Body: authOperationError(failureRequestID, err.Error()), Headers: openapi.RegisterPasskey400ResponseHeaders{CacheControl: noStoreValue}}, nil
	}

	return openapi.RegisterPasskey200JSONResponse{
		Body:    openapi.AuthSessionResponse{RequestId: result.RequestID, AccountId: result.AccountID, PasskeyCredentialId: result.PasskeyCredentialID, SessionId: result.SessionID, SessionToken: result.SessionToken, ExpiresAt: result.ExpiresAt},
		Headers: openapi.RegisterPasskey200ResponseHeaders{CacheControl: noStoreValue},
	}, nil
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

func chooseCredential(primary string, secondary string) string {
	if strings.TrimSpace(primary) != "" {
		return primary
	}
	return secondary
}

func bearerTokenFromContext(ctx context.Context) string {
	ginContext, ok := ctx.(*gin.Context)
	if !ok {
		return ""
	}
	return bearerToken(ginContext.GetHeader("Authorization"))
}

func clientIPFromContext(ctx context.Context) string {
	ginContext, ok := ctx.(*gin.Context)
	if !ok {
		return "unknown"
	}
	return requestIP(ginContext)
}
