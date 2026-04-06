package http

import (
	"context"
	"errors"
	stdhttp "net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"www-template/packages/backend/internal/generated/openapi"
	"www-template/packages/backend/internal/types"
	"www-template/packages/backend/internal/usecases"
)

type Dependencies struct {
	Auth *usecases.AuthService
}

type StrictServer struct {
	auth *usecases.AuthService
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
	return &StrictServer{auth: dependencies.Auth}
}

func (s *StrictServer) GetStatus(ctx context.Context, _ openapi.GetStatusRequestObject) (openapi.GetStatusResponseObject, error) {
	return openapi.GetStatus200JSONResponse{
		Message:   "API status ok",
		Timestamp: time.Now().UTC(),
	}, nil
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

func chooseCredential(primary string, secondary string) string {
	if primary != "" {
		return primary
	}
	return secondary
}

// ─── Multi-passkey management handlers ──────────────────────────────────────

func (s *StrictServer) ListPasskeys(ctx context.Context, _ openapi.ListPasskeysRequestObject) (openapi.ListPasskeysResponseObject, error) {
	session, err := s.auth.AuthorizeSession(ctx, bearerTokenFromContext(ctx))
	if err != nil {
		return openapi.ListPasskeys401JSONResponse{Body: authFailureResponseObject(nextAuthRequestID(), err), Headers: openapi.ListPasskeys401ResponseHeaders{CacheControl: noStoreValue}}, nil
	}

	creds, err := s.auth.ListPasskeys(ctx, session.AccountID)
	if err != nil {
		if errors.Is(err, usecases.ErrInternalError) {
			return openapi.ListPasskeys503JSONResponse{Body: authFailureResponseObject(nextAuthRequestID(), err), Headers: openapi.ListPasskeys503ResponseHeaders{CacheControl: noStoreValue}}, nil
		}
		return openapi.ListPasskeys401JSONResponse{Body: authFailureResponseObject(nextAuthRequestID(), err), Headers: openapi.ListPasskeys401ResponseHeaders{CacheControl: noStoreValue}}, nil
	}

	items := make([]openapi.PasskeyItem, 0, len(creds))
	for _, c := range creds {
		items = append(items, openapi.PasskeyItem{Id: c.ID, Identifier: c.Identifier, CreatedAt: c.CreatedAt})
	}
	requestID := nextAuthRequestID()
	return openapi.ListPasskeys200JSONResponse{
		Body:    openapi.PasskeyListResponse{RequestId: requestID, Passkeys: items},
		Headers: openapi.ListPasskeys200ResponseHeaders{CacheControl: noStoreValue},
	}, nil
}

func (s *StrictServer) StartPasskeyAddition(ctx context.Context, _ openapi.StartPasskeyAdditionRequestObject) (openapi.StartPasskeyAdditionResponseObject, error) {
	session, err := s.auth.AuthorizeSession(ctx, bearerTokenFromContext(ctx))
	if err != nil {
		return openapi.StartPasskeyAddition401JSONResponse{Body: authFailureResponseObject(nextAuthRequestID(), err), Headers: openapi.StartPasskeyAddition401ResponseHeaders{CacheControl: noStoreValue}}, nil
	}

	result, err := s.auth.StartAddPasskey(ctx, session.AccountID)
	if err != nil {
		return openapi.StartPasskeyAddition503JSONResponse{Body: authFailureResponseObject(nextAuthRequestID(), err), Headers: openapi.StartPasskeyAddition503ResponseHeaders{CacheControl: noStoreValue}}, nil
	}

	return openapi.StartPasskeyAddition200JSONResponse{
		Body:    openapi.PasskeyAddStartResponse{RequestId: result.RequestID, Challenge: result.Challenge, RpId: result.WebAuthnRPID},
		Headers: openapi.StartPasskeyAddition200ResponseHeaders{CacheControl: noStoreValue},
	}, nil
}

func (s *StrictServer) FinishPasskeyAddition(ctx context.Context, request openapi.FinishPasskeyAdditionRequestObject) (openapi.FinishPasskeyAdditionResponseObject, error) {
	if request.Body == nil {
		return openapi.FinishPasskeyAddition400JSONResponse{Body: authOperationError(nextAuthRequestID(), "request body is required"), Headers: openapi.FinishPasskeyAddition400ResponseHeaders{CacheControl: noStoreValue}}, nil
	}

	session, err := s.auth.AuthorizeSession(ctx, bearerTokenFromContext(ctx))
	if err != nil {
		return openapi.FinishPasskeyAddition401JSONResponse{Body: authFailureResponseObject(nextAuthRequestID(), err), Headers: openapi.FinishPasskeyAddition401ResponseHeaders{CacheControl: noStoreValue}}, nil
	}

	creds, err := s.auth.FinishAddPasskey(ctx, session.AccountID, request.Body.Credential)
	if err != nil {
		failureRequestID := nextAuthRequestID()
		if errors.Is(err, usecases.ErrInternalError) {
			return openapi.FinishPasskeyAddition503JSONResponse{Body: authFailureResponseObject(failureRequestID, err), Headers: openapi.FinishPasskeyAddition503ResponseHeaders{CacheControl: noStoreValue}}, nil
		}
		return openapi.FinishPasskeyAddition400JSONResponse{Body: authOperationError(failureRequestID, err.Error()), Headers: openapi.FinishPasskeyAddition400ResponseHeaders{CacheControl: noStoreValue}}, nil
	}

	items := make([]openapi.PasskeyItem, 0, len(creds))
	for _, c := range creds {
		items = append(items, openapi.PasskeyItem{Id: c.ID, Identifier: c.Identifier, CreatedAt: c.CreatedAt})
	}
	requestID := nextAuthRequestID()
	return openapi.FinishPasskeyAddition200JSONResponse{
		Body:    openapi.PasskeyListResponse{RequestId: requestID, Passkeys: items},
		Headers: openapi.FinishPasskeyAddition200ResponseHeaders{CacheControl: noStoreValue},
	}, nil
}

func (s *StrictServer) DeletePasskey(ctx context.Context, request openapi.DeletePasskeyRequestObject) (openapi.DeletePasskeyResponseObject, error) {
	session, err := s.auth.AuthorizeSession(ctx, bearerTokenFromContext(ctx))
	if err != nil {
		return openapi.DeletePasskey401JSONResponse{Body: authFailureResponseObject(nextAuthRequestID(), err), Headers: openapi.DeletePasskey401ResponseHeaders{CacheControl: noStoreValue}}, nil
	}

	if err := s.auth.DeletePasskey(ctx, session.AccountID, string(request.Id)); err != nil {
		failureRequestID := nextAuthRequestID()
		if errors.Is(err, usecases.ErrLastPasskeyCannotBeDeleted) {
			return openapi.DeletePasskey409JSONResponse{Body: authOperationError(failureRequestID, err.Error()), Headers: openapi.DeletePasskey409ResponseHeaders{CacheControl: noStoreValue}}, nil
		}
		if errors.Is(err, usecases.ErrInternalError) {
			return openapi.DeletePasskey503JSONResponse{Body: authFailureResponseObject(failureRequestID, err), Headers: openapi.DeletePasskey503ResponseHeaders{CacheControl: noStoreValue}}, nil
		}
		return openapi.DeletePasskey403JSONResponse{Body: authOperationError(failureRequestID, err.Error()), Headers: openapi.DeletePasskey403ResponseHeaders{CacheControl: noStoreValue}}, nil
	}

	return openapi.DeletePasskey204Response{Headers: openapi.DeletePasskey204ResponseHeaders{CacheControl: noStoreValue}}, nil
}

func (s *StrictServer) IssuePasskeyOtp(ctx context.Context, _ openapi.IssuePasskeyOtpRequestObject) (openapi.IssuePasskeyOtpResponseObject, error) {
	session, err := s.auth.AuthorizeSession(ctx, bearerTokenFromContext(ctx))
	if err != nil {
		return openapi.IssuePasskeyOtp401JSONResponse{Body: authFailureResponseObject(nextAuthRequestID(), err), Headers: openapi.IssuePasskeyOtp401ResponseHeaders{CacheControl: noStoreValue}}, nil
	}

	otp, err := s.auth.IssuePasskeyOtp(ctx, session.AccountID)
	if err != nil {
		return openapi.IssuePasskeyOtp503JSONResponse{Body: authFailureResponseObject(nextAuthRequestID(), err), Headers: openapi.IssuePasskeyOtp503ResponseHeaders{CacheControl: noStoreValue}}, nil
	}

	requestID := nextAuthRequestID()
	return openapi.IssuePasskeyOtp200JSONResponse{
		Body:    openapi.PasskeyOtpResponse{RequestId: requestID, Otp: otp},
		Headers: openapi.IssuePasskeyOtp200ResponseHeaders{CacheControl: noStoreValue},
	}, nil
}

func (s *StrictServer) StartPasskeyAdditionByOtp(ctx context.Context, request openapi.StartPasskeyAdditionByOtpRequestObject) (openapi.StartPasskeyAdditionByOtpResponseObject, error) {
	if request.Body == nil {
		return openapi.StartPasskeyAdditionByOtp400JSONResponse{Body: authOperationError(nextAuthRequestID(), "request body is required"), Headers: openapi.StartPasskeyAdditionByOtp400ResponseHeaders{CacheControl: noStoreValue}}, nil
	}

	result, err := s.auth.StartAddPasskeyByOtp(ctx, request.Body.Otp)
	if err != nil {
		failureRequestID := nextAuthRequestID()
		if errors.Is(err, usecases.ErrInternalError) {
			return openapi.StartPasskeyAdditionByOtp503JSONResponse{Body: authFailureResponseObject(failureRequestID, err), Headers: openapi.StartPasskeyAdditionByOtp503ResponseHeaders{CacheControl: noStoreValue}}, nil
		}
		return openapi.StartPasskeyAdditionByOtp400JSONResponse{Body: authOperationError(failureRequestID, err.Error()), Headers: openapi.StartPasskeyAdditionByOtp400ResponseHeaders{CacheControl: noStoreValue}}, nil
	}

	return openapi.StartPasskeyAdditionByOtp200JSONResponse{
		Body:    openapi.PasskeyAddStartResponse{RequestId: result.RequestID, Challenge: result.Challenge, RpId: result.WebAuthnRPID},
		Headers: openapi.StartPasskeyAdditionByOtp200ResponseHeaders{CacheControl: noStoreValue},
	}, nil
}

func (s *StrictServer) FinishPasskeyAdditionByOtp(ctx context.Context, request openapi.FinishPasskeyAdditionByOtpRequestObject) (openapi.FinishPasskeyAdditionByOtpResponseObject, error) {
	if request.Body == nil {
		return openapi.FinishPasskeyAdditionByOtp400JSONResponse{Body: authOperationError(nextAuthRequestID(), "request body is required"), Headers: openapi.FinishPasskeyAdditionByOtp400ResponseHeaders{CacheControl: noStoreValue}}, nil
	}

	if err := s.auth.FinishAddPasskeyByOtp(ctx, request.Body.Otp, request.Body.Credential); err != nil {
		failureRequestID := nextAuthRequestID()
		if errors.Is(err, usecases.ErrInternalError) {
			return openapi.FinishPasskeyAdditionByOtp503JSONResponse{Body: authFailureResponseObject(failureRequestID, err), Headers: openapi.FinishPasskeyAdditionByOtp503ResponseHeaders{CacheControl: noStoreValue}}, nil
		}
		return openapi.FinishPasskeyAdditionByOtp400JSONResponse{Body: authOperationError(failureRequestID, err.Error()), Headers: openapi.FinishPasskeyAdditionByOtp400ResponseHeaders{CacheControl: noStoreValue}}, nil
	}

	return openapi.FinishPasskeyAdditionByOtp200Response{Headers: openapi.FinishPasskeyAdditionByOtp200ResponseHeaders{CacheControl: noStoreValue}}, nil
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
