package product

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	stdhttp "net/http"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	application "www-template/packages/backend/internal/application"
	"www-template/packages/backend/internal/generated/openapi"
	"www-template/packages/backend/internal/platform/config"
)

type Dependencies struct {
	Auth            *application.AuthService
	AccountSetting  *application.AccountSettingService
	AccountSnapshot *application.AccountSettingSnapshotService
	TokenService    *application.TokenService
	SessionService  *application.SessionService
}

type StrictServer struct {
	auth                    *application.AuthService
	accountSetting          *application.AccountSettingService
	accountSnapshot         *application.AccountSettingSnapshotService
	tokenService            *application.TokenService
	sessionService          *application.SessionService
	authRefreshCookieMaxAge time.Duration
}

const productRefreshCookieName = "refresh_token"

// NewRouter は設定と依存関係をもとに Gin エンジンを構築する。
// production 環境では trusted proxy、CORS、認証、body limit、observability の
// 各ミドルウェアを適切な順序で登録する。
func NewRouter(cfg config.Config, dependencies Dependencies) *gin.Engine {
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	_ = router.SetTrustedProxies(cfg.TrustedProxyCIDRs)
	router.Use(gin.Recovery())
	// CORS は認証ミドルウェアより前に配置し、OPTIONS preflight が
	// 401 になるのを防ぐ。OTel の trace context header も明示的に許可する。
	router.Use(cors.New(cors.Config{
		AllowCredentials: true,
		AllowHeaders:     []string{"Content-Type", "Authorization", "X-Reauth-Session", "traceparent", "tracestate", "baggage"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowOrigins:     cfg.AllowedOrigins,
		MaxAge:           12 * time.Hour,
	}))
	router.Use(authNoStoreAndBindErrorMiddleware())
	router.Use(otelMiddleware())
	router.Use(appAuthMiddleware(cfg, dependencies.Auth))
	router.Use(refreshOptionalBodyMiddleware())
	if cfg.Auth.AuthBodyLimitBytes > 0 {
		router.Use(authBodyLimitMiddleware(int64(cfg.Auth.AuthBodyLimitBytes)))
	}

	router.GET("/health", func(c *gin.Context) {
		c.JSON(stdhttp.StatusOK, gin.H{
			"status":    "ok",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		})
	})

	strictHandler := openapi.NewStrictHandler(NewStrictServer(cfg, dependencies), nil)
	openapi.RegisterHandlers(router, strictHandler)

	router.NoRoute(func(c *gin.Context) {
		c.JSON(stdhttp.StatusNotFound, gin.H{
			"error": "not found",
			"path":  c.Request.URL.Path,
		})
	})

	return router
}

// authBodyLimitMiddleware は auth 系エンドポイントに対して request body の
// サイズ上限を適用する。limit を超える場合は http.MaxBytesReader が
// http.StatusRequestEntityTooLarge (413) を返す。
func authBodyLimitMiddleware(limit int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		if strings.HasPrefix(path, "/api/v1/auth/") || strings.HasPrefix(path, "/api/v1/passkeys/") || path == "/api/v1/account/settings" {
			c.Request.Body = stdhttp.MaxBytesReader(c.Writer, c.Request.Body, limit)
		}
		c.Next()
	}
}

func refreshOptionalBodyMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Step 1: Product refresh endpoint 以外では request body を変更せず、既存 route の挙動へ副作用を出さない。
		if c.Request.Method != stdhttp.MethodPost || c.Request.URL.Path != "/api/v1/auth/refresh" {
			c.Next()
			return
		}

		// Step 2: Cookie-only refresh client が body を送らない場合だけ、generated strict binding が handler 到達前に 400 を返さないよう空 JSON object を補う。
		if c.Request.ContentLength == 0 {
			c.Request.Body = io.NopCloser(strings.NewReader("{}"))
			c.Request.ContentLength = 2
			c.Request.Header.Set("Content-Type", "application/json")
		}

		// Step 3: 以降の generated handler では body を認証材料にせず、HttpOnly Cookie だけを refreshToken として読む。
		c.Next()
	}
}

// NewStrictServer は generated OpenAPI strict handler に渡す server 実装を構築する。
//
// cfg には HttpOnly refresh Cookie の寿命を決める Auth 設定を渡す。
// dependencies には Auth、Token、Session、AccountSetting の各 use case を渡す。
// nil の依存関係は handler 側で fail-close し、外部へは 503 または認証失敗として返す。
func NewStrictServer(cfg config.Config, dependencies Dependencies) *StrictServer {
	return &StrictServer{
		auth:                    dependencies.Auth,
		accountSetting:          dependencies.AccountSetting,
		accountSnapshot:         dependencies.AccountSnapshot,
		tokenService:            dependencies.TokenService,
		sessionService:          dependencies.SessionService,
		authRefreshCookieMaxAge: cfg.Auth.RefreshTokenTTL,
	}
}

func (s *StrictServer) GetStatus(ctx context.Context, _ openapi.GetStatusRequestObject) (openapi.GetStatusResponseObject, error) {
	return openapi.GetStatus200JSONResponse{
		Message:   "API status ok",
		Timestamp: time.Now().UTC(),
	}, nil
}

// GetAccountSettings は bearer session で認可された AccountID の AccountSetting を返す。
func (s *StrictServer) GetAccountSettings(ctx context.Context, _ openapi.GetAccountSettingsRequestObject) (openapi.GetAccountSettingsResponseObject, error) {
	session, err := s.auth.AuthorizeSession(ctx, bearerTokenFromContext(ctx))
	if err != nil {
		failureRequestID := nextAuthRequestID()
		if errors.Is(err, application.ErrAccountSuspended) {
			return openapi.GetAccountSettings403JSONResponse{Body: authFailureResponseObject(failureRequestID, err), Headers: openapi.GetAccountSettings403ResponseHeaders{CacheControl: noStoreValue}}, nil
		}
		return openapi.GetAccountSettings401JSONResponse{Body: authFailureResponseObject(failureRequestID, err), Headers: openapi.GetAccountSettings401ResponseHeaders{CacheControl: noStoreValue}}, nil
	}
	if s.accountSetting == nil {
		return openapi.GetAccountSettings503JSONResponse{Body: authFailureResponseObject(nextAuthRequestID(), application.ErrInternalError), Headers: openapi.GetAccountSettings503ResponseHeaders{CacheControl: noStoreValue}}, nil
	}
	setting, err := s.accountSetting.Get(ctx, session.AccountID)
	if err != nil {
		return accountSettingGetErrorResponse(err), nil
	}
	return openapi.GetAccountSettings200JSONResponse{Body: mapAccountSettingResponse(setting), Headers: openapi.GetAccountSettings200ResponseHeaders{CacheControl: noStoreValue}}, nil
}

// UpdateAccountSettings は bearer session で認可された AccountID の AccountSetting.locale を更新する。
func (s *StrictServer) UpdateAccountSettings(ctx context.Context, request openapi.UpdateAccountSettingsRequestObject) (openapi.UpdateAccountSettingsResponseObject, error) {
	if request.Body == nil {
		return openapi.UpdateAccountSettings400JSONResponse{Body: authOperationError(nextAuthRequestID(), invalidRequestBodyMessage), Headers: openapi.UpdateAccountSettings400ResponseHeaders{CacheControl: noStoreValue}}, nil
	}
	session, err := s.auth.AuthorizeSession(ctx, bearerTokenFromContext(ctx))
	if err != nil {
		failureRequestID := nextAuthRequestID()
		if errors.Is(err, application.ErrAccountSuspended) {
			return openapi.UpdateAccountSettings403JSONResponse{Body: authFailureResponseObject(failureRequestID, err), Headers: openapi.UpdateAccountSettings403ResponseHeaders{CacheControl: noStoreValue}}, nil
		}
		return openapi.UpdateAccountSettings401JSONResponse{Body: authFailureResponseObject(failureRequestID, err), Headers: openapi.UpdateAccountSettings401ResponseHeaders{CacheControl: noStoreValue}}, nil
	}
	if s.accountSetting == nil {
		return openapi.UpdateAccountSettings503JSONResponse{Body: authFailureResponseObject(nextAuthRequestID(), application.ErrInternalError), Headers: openapi.UpdateAccountSettings503ResponseHeaders{CacheControl: noStoreValue}}, nil
	}
	setting, err := s.accountSetting.Update(ctx, session.AccountID, string(request.Body.Locale))
	if err != nil {
		return accountSettingUpdateErrorResponse(err), nil
	}
	return openapi.UpdateAccountSettings200JSONResponse{Body: mapAccountSettingResponse(setting), Headers: openapi.UpdateAccountSettings200ResponseHeaders{CacheControl: noStoreValue}}, nil
}

func (s *StrictServer) Logout(ctx context.Context, _ openapi.LogoutRequestObject) (openapi.LogoutResponseObject, error) {
	requestID, err := s.auth.Logout(ctx, bearerTokenFromContext(ctx))
	if err != nil {
		failureRequestID := nextAuthRequestID()
		if errors.Is(err, application.ErrInternalError) {
			return openapi.Logout503JSONResponse{Body: authFailureResponseObject(failureRequestID, err), Headers: openapi.Logout503ResponseHeaders{CacheControl: noStoreValue}}, nil
		}
		if errors.Is(err, application.ErrAccountSuspended) {
			return openapi.Logout403JSONResponse{Body: authFailureResponseObject(failureRequestID, err), Headers: openapi.Logout403ResponseHeaders{CacheControl: noStoreValue}}, nil
		}
		return openapi.Logout401JSONResponse{Body: authFailureResponseObject(failureRequestID, err), Headers: openapi.Logout401ResponseHeaders{CacheControl: noStoreValue}}, nil
	}

	clearProductRefreshCookie(ctx)
	return openapi.Logout200JSONResponse{
		Body:    openapi.LogoutResponse{RequestId: requestID, Revoked: openapi.LogoutResponseRevokedTrue},
		Headers: openapi.Logout200ResponseHeaders{CacheControl: noStoreValue},
	}, nil
}

func (s *StrictServer) StartReauthentication(ctx context.Context, request openapi.StartReauthenticationRequestObject) (openapi.StartReauthenticationResponseObject, error) {
	session, err := s.auth.AuthorizeSession(ctx, bearerTokenFromContext(ctx))
	if err != nil {
		failureRequestID := nextAuthRequestID()
		if errors.Is(err, application.ErrAccountSuspended) {
			return openapi.StartReauthentication403JSONResponse{Body: authFailureResponseObject(failureRequestID, err), Headers: openapi.StartReauthentication403ResponseHeaders{CacheControl: noStoreValue}}, nil
		}
		return openapi.StartReauthentication401JSONResponse{Body: authFailureResponseObject(failureRequestID, err), Headers: openapi.StartReauthentication401ResponseHeaders{CacheControl: noStoreValue}}, nil
	}

	if request.Body == nil {
		return openapi.StartReauthentication400JSONResponse{Body: authOperationError(nextAuthRequestID(), nonRevealingAuthRejectMessage), Headers: openapi.StartReauthentication400ResponseHeaders{CacheControl: noStoreValue}}, nil
	}

	result, err := s.auth.StartReauthentication(ctx, application.StartReauthenticationInput{
		AccountID: session.AccountID,
		SessionID: session.SessionID,
		Kind:      string(request.Body.Kind),
		ClientIP:  clientIPFromContext(ctx),
	})
	if err != nil {
		failureRequestID := nextAuthRequestID()
		if errors.Is(err, application.ErrInternalError) {
			return openapi.StartReauthentication503JSONResponse{Body: authFailureResponseObject(failureRequestID, err), Headers: openapi.StartReauthentication503ResponseHeaders{CacheControl: noStoreValue}}, nil
		}
		return openapi.StartReauthentication400JSONResponse{Body: authOperationError(failureRequestID, nonRevealingAuthRejectMessage), Headers: openapi.StartReauthentication400ResponseHeaders{CacheControl: noStoreValue}}, nil
	}

	body := openapi.PasskeyStartResponse{RequestId: result.RequestID, Challenge: result.Challenge, RpId: result.WebAuthnRPID}
	applyWebAuthnLoginOptions(&body, result.WebAuthnOptions)
	return openapi.StartReauthentication200JSONResponse{
		Body:    body,
		Headers: openapi.StartReauthentication200ResponseHeaders{CacheControl: noStoreValue},
	}, nil
}

func (s *StrictServer) FinishReauthentication(ctx context.Context, request openapi.FinishReauthenticationRequestObject) (openapi.FinishReauthenticationResponseObject, error) {
	session, err := s.auth.AuthorizeSession(ctx, bearerTokenFromContext(ctx))
	if err != nil {
		failureRequestID := nextAuthRequestID()
		if errors.Is(err, application.ErrAccountSuspended) {
			return openapi.FinishReauthentication403JSONResponse{Body: authFailureResponseObject(failureRequestID, err), Headers: openapi.FinishReauthentication403ResponseHeaders{CacheControl: noStoreValue}}, nil
		}
		return openapi.FinishReauthentication401JSONResponse{Body: authFailureResponseObject(failureRequestID, err), Headers: openapi.FinishReauthentication401ResponseHeaders{CacheControl: noStoreValue}}, nil
	}

	if request.Body == nil {
		return openapi.FinishReauthentication400JSONResponse{Body: authOperationError(nextAuthRequestID(), invalidRequestBodyMessage), Headers: openapi.FinishReauthentication400ResponseHeaders{CacheControl: noStoreValue}}, nil
	}

	result, err := s.auth.FinishReauthentication(ctx, application.FinishReauthenticationInput{
		AccountID:  session.AccountID,
		SessionID:  session.SessionID,
		Kind:       string(request.Body.Kind),
		Credential: mapAssertionCredentialToDTO(request.Body.Credential),
		ClientIP:   clientIPFromContext(ctx),
	})
	if err != nil {
		failureRequestID := nextAuthRequestID()
		if errors.Is(err, application.ErrInternalError) {
			return openapi.FinishReauthentication503JSONResponse{Body: authFailureResponseObject(failureRequestID, err), Headers: openapi.FinishReauthentication503ResponseHeaders{CacheControl: noStoreValue}}, nil
		}
		return openapi.FinishReauthentication400JSONResponse{Body: authOperationError(failureRequestID, nonRevealingAuthRejectMessage), Headers: openapi.FinishReauthentication400ResponseHeaders{CacheControl: noStoreValue}}, nil
	}

	return openapi.FinishReauthentication200JSONResponse{
		Body: openapi.ReauthenticationSessionResponse{
			RequestId:       result.RequestID,
			ReauthSessionId: result.ReauthSessionID,
			Kind:            openapi.ReauthenticationSessionKind(result.Kind),
			ExpiresAt:       result.ExpiresAt,
		},
		Headers: openapi.FinishReauthentication200ResponseHeaders{CacheControl: noStoreValue},
	}, nil
}

func (s *StrictServer) StartPasskeyAuthentication(ctx context.Context, request openapi.StartPasskeyAuthenticationRequestObject) (openapi.StartPasskeyAuthenticationResponseObject, error) {
	if request.Body == nil {
		return openapi.StartPasskeyAuthentication400JSONResponse{Body: authOperationError(nextAuthRequestID(), nonRevealingAuthRejectMessage), Headers: openapi.StartPasskeyAuthentication400ResponseHeaders{CacheControl: noStoreValue}}, nil
	}

	result, err := s.auth.StartPasskeyAuthentication(ctx, application.StartPasskeyAuthenticationInput{Identifier: request.Body.Identifier, ClientIP: clientIPFromContext(ctx)})
	if err != nil {
		failureRequestID := nextAuthRequestID()
		if errors.Is(err, application.ErrInternalError) {
			return openapi.StartPasskeyAuthentication503JSONResponse{Body: authFailureResponseObject(failureRequestID, err), Headers: openapi.StartPasskeyAuthentication503ResponseHeaders{CacheControl: noStoreValue}}, nil
		}
		return openapi.StartPasskeyAuthentication400JSONResponse{Body: authOperationError(failureRequestID, nonRevealingAuthRejectMessage), Headers: openapi.StartPasskeyAuthentication400ResponseHeaders{CacheControl: noStoreValue}}, nil
	}

	body := openapi.PasskeyStartResponse{RequestId: result.RequestID, Challenge: result.Challenge, RpId: result.WebAuthnRPID}
	applyWebAuthnLoginOptions(&body, result.WebAuthnOptions)
	return openapi.StartPasskeyAuthentication200JSONResponse{
		Body:    body,
		Headers: openapi.StartPasskeyAuthentication200ResponseHeaders{CacheControl: noStoreValue},
	}, nil
}

func (s *StrictServer) FinishPasskeyAuthentication(ctx context.Context, request openapi.FinishPasskeyAuthenticationRequestObject) (openapi.FinishPasskeyAuthenticationResponseObject, error) {
	if request.Body == nil {
		return openapi.FinishPasskeyAuthentication400JSONResponse{Body: authOperationError(nextAuthRequestID(), "request body is required"), Headers: openapi.FinishPasskeyAuthentication400ResponseHeaders{CacheControl: noStoreValue}}, nil
	}

	result, err := s.auth.FinishPasskeyAuthentication(ctx, application.FinishPasskeyAuthenticationInput{Credential: mapAssertionCredentialToDTO(request.Body.Credential), ClientIP: clientIPFromContext(ctx), UserAgent: userAgentFromContext(ctx)})
	if err != nil {
		failureRequestID := nextAuthRequestID()
		if errors.Is(err, application.ErrInternalError) {
			return openapi.FinishPasskeyAuthentication503JSONResponse{Body: authFailureResponseObject(failureRequestID, err), Headers: openapi.FinishPasskeyAuthentication503ResponseHeaders{CacheControl: noStoreValue}}, nil
		}
		if errors.Is(err, application.ErrAccountSuspended) {
			return openapi.FinishPasskeyAuthentication403JSONResponse{Body: authFailureResponseObject(failureRequestID, err), Headers: openapi.FinishPasskeyAuthentication403ResponseHeaders{CacheControl: noStoreValue}}, nil
		}
		return openapi.FinishPasskeyAuthentication400JSONResponse{Body: authOperationError(failureRequestID, err.Error()), Headers: openapi.FinishPasskeyAuthentication400ResponseHeaders{CacheControl: noStoreValue}}, nil
	}

	body := openapi.AuthSessionResponse{
		RequestId:           result.RequestID,
		AccountId:           result.AccountID.String(),
		PasskeyCredentialId: result.PasskeyCredentialID,
		SessionId:           result.SessionID,
		AccessToken:         result.AccessToken,
		ExpiresAt:           result.ExpiresAt,
	}
	// fail-closed: accessToken body と HttpOnly Cookie 用 refreshToken のどちらかが空なら session を発行しない。
	if body.AccessToken == "" || result.RefreshToken == "" {
		return openapi.FinishPasskeyAuthentication503JSONResponse{Body: authFailureResponseObject(nextAuthRequestID(), application.ErrInternalError), Headers: openapi.FinishPasskeyAuthentication503ResponseHeaders{CacheControl: noStoreValue}}, nil
	}
	setProductRefreshCookie(ctx, result.RefreshToken, s.authRefreshCookieMaxAge)
	return openapi.FinishPasskeyAuthentication200JSONResponse{
		Body:    body,
		Headers: openapi.FinishPasskeyAuthentication200ResponseHeaders{CacheControl: noStoreValue},
	}, nil
}

func (s *StrictServer) RequestPasskeyRecovery(ctx context.Context, request openapi.RequestPasskeyRecoveryRequestObject) (openapi.RequestPasskeyRecoveryResponseObject, error) {
	if request.Body == nil {
		return openapi.RequestPasskeyRecovery400JSONResponse{Body: authOperationError(nextAuthRequestID(), "request body is required"), Headers: openapi.RequestPasskeyRecovery400ResponseHeaders{CacheControl: noStoreValue}}, nil
	}

	result, err := s.auth.RequestPasskeyRecovery(ctx, application.RequestPasskeyRecoveryInput{Email: string(request.Body.Email), ClientIP: clientIPFromContext(ctx)})
	if err != nil {
		return openapi.RequestPasskeyRecovery503JSONResponse{Body: authFailureResponseObject(nextAuthRequestID(), err), Headers: openapi.RequestPasskeyRecovery503ResponseHeaders{CacheControl: noStoreValue}}, nil
	}

	return openapi.RequestPasskeyRecovery202JSONResponse{
		Body:    openapi.RecoveryAcceptedResponse{RequestId: result.RequestID, Accepted: true},
		Headers: openapi.RequestPasskeyRecovery202ResponseHeaders{CacheControl: noStoreValue},
	}, nil
}

func (s *StrictServer) ConsumeRecoveryToken(ctx context.Context, request openapi.ConsumeRecoveryTokenRequestObject) (openapi.ConsumeRecoveryTokenResponseObject, error) {
	if request.Body == nil {
		return openapi.ConsumeRecoveryToken400JSONResponse{Body: authOperationError(nextAuthRequestID(), "request body is required"), Headers: openapi.ConsumeRecoveryToken400ResponseHeaders{CacheControl: noStoreValue}}, nil
	}

	result, err := s.auth.ConsumeRecoveryToken(ctx, application.ConsumeRecoveryTokenInput{Token: request.Body.Token, ClientIP: clientIPFromContext(ctx)})
	if err != nil {
		failureRequestID := nextAuthRequestID()
		if errors.Is(err, application.ErrInternalError) {
			return openapi.ConsumeRecoveryToken503JSONResponse{Body: authFailureResponseObject(failureRequestID, err), Headers: openapi.ConsumeRecoveryToken503ResponseHeaders{CacheControl: noStoreValue}}, nil
		}
		return openapi.ConsumeRecoveryToken400JSONResponse{Body: authOperationError(failureRequestID, err.Error()), Headers: openapi.ConsumeRecoveryToken400ResponseHeaders{CacheControl: noStoreValue}}, nil
	}

	return openapi.ConsumeRecoveryToken200JSONResponse{
		Body:    openapi.RecoveryConsumeResponse{RequestId: result.RequestID, RecoveryTokenId: result.RecoveryTokenID, RecoverySessionId: result.RecoverySessionID, RecoverySession: result.RecoverySessionRef, Kind: openapi.TokenKind(result.Kind), ExpiresAt: result.ExpiresAt},
		Headers: openapi.ConsumeRecoveryToken200ResponseHeaders{CacheControl: noStoreValue},
	}, nil
}

func (s *StrictServer) StartPasskeyRegistration(ctx context.Context, request openapi.StartPasskeyRegistrationRequestObject) (openapi.StartPasskeyRegistrationResponseObject, error) {
	if request.Body == nil {
		return openapi.StartPasskeyRegistration400JSONResponse{Body: authOperationError(nextAuthRequestID(), "request body is required"), Headers: openapi.StartPasskeyRegistration400ResponseHeaders{CacheControl: noStoreValue}}, nil
	}

	result, err := s.auth.StartPasskeyRegistration(ctx, application.StartPasskeyRegistrationInput{RecoverySession: string(request.Body.RecoverySession), ClientIP: clientIPFromContext(ctx)})
	if err != nil {
		failureRequestID := nextAuthRequestID()
		if errors.Is(err, application.ErrInternalError) {
			return openapi.StartPasskeyRegistration503JSONResponse{Body: authFailureResponseObject(failureRequestID, err), Headers: openapi.StartPasskeyRegistration503ResponseHeaders{CacheControl: noStoreValue}}, nil
		}
		return openapi.StartPasskeyRegistration400JSONResponse{Body: authOperationError(failureRequestID, nonRevealingAuthRejectMessage), Headers: openapi.StartPasskeyRegistration400ResponseHeaders{CacheControl: noStoreValue}}, nil
	}

	body := openapi.PasskeyAddStartResponse{RequestId: result.RequestID, Challenge: result.Challenge, RpId: result.WebAuthnRPID}
	if err := applyWebAuthnRegistrationOptions(&body, result.WebAuthnOptions); err != nil {
		return openapi.StartPasskeyRegistration503JSONResponse{Body: authFailureResponseObject(nextAuthRequestID(), err), Headers: openapi.StartPasskeyRegistration503ResponseHeaders{CacheControl: noStoreValue}}, nil
	}
	return openapi.StartPasskeyRegistration200JSONResponse{
		Body:    body,
		Headers: openapi.StartPasskeyRegistration200ResponseHeaders{CacheControl: noStoreValue},
	}, nil
}

func (s *StrictServer) RegisterPasskey(ctx context.Context, request openapi.RegisterPasskeyRequestObject) (openapi.RegisterPasskeyResponseObject, error) {
	if request.Body == nil {
		return openapi.RegisterPasskey400JSONResponse{Body: authOperationError(nextAuthRequestID(), "request body is required"), Headers: openapi.RegisterPasskey400ResponseHeaders{CacheControl: noStoreValue}}, nil
	}

	recovery, _ := request.Body.AsRecoveryPasskeyFinishRequest()
	invitation, _ := request.Body.AsInvitationPasskeyFinishRequest()
	credential := chooseAttestationCredential(recovery.Credential, invitation.Credential)
	result, err := s.auth.RegisterPasskey(ctx, application.RegisterPasskeyInput{RecoverySession: string(recovery.RecoverySession), InvitationSession: string(invitation.InvitationSession), Credential: credential, ClientIP: clientIPFromContext(ctx), UserAgent: userAgentFromContext(ctx)})
	if err != nil {
		failureRequestID := nextAuthRequestID()
		if errors.Is(err, application.ErrInternalError) {
			return openapi.RegisterPasskey503JSONResponse{Body: authFailureResponseObject(failureRequestID, err), Headers: openapi.RegisterPasskey503ResponseHeaders{CacheControl: noStoreValue}}, nil
		}
		return openapi.RegisterPasskey400JSONResponse{Body: authOperationError(failureRequestID, nonRevealingAuthRejectMessage), Headers: openapi.RegisterPasskey400ResponseHeaders{CacheControl: noStoreValue}}, nil
	}

	body := openapi.AuthSessionResponse{
		RequestId:           result.RequestID,
		AccountId:           result.AccountID.String(),
		PasskeyCredentialId: result.PasskeyCredentialID,
		SessionId:           result.SessionID,
		AccessToken:         result.AccessToken,
		ExpiresAt:           result.ExpiresAt,
	}
	// fail-closed: accessToken body と HttpOnly Cookie 用 refreshToken のどちらかが空なら session を発行しない。
	if body.AccessToken == "" || result.RefreshToken == "" {
		return openapi.RegisterPasskey503JSONResponse{Body: authFailureResponseObject(nextAuthRequestID(), application.ErrInternalError), Headers: openapi.RegisterPasskey503ResponseHeaders{CacheControl: noStoreValue}}, nil
	}
	setProductRefreshCookie(ctx, result.RefreshToken, s.authRefreshCookieMaxAge)
	return openapi.RegisterPasskey200JSONResponse{
		Body:    body,
		Headers: openapi.RegisterPasskey200ResponseHeaders{CacheControl: noStoreValue},
	}, nil
}

// ─── OpenAPI → usecases DTO mapping helpers ──────────────────────────────────

func mapAssertionCredentialToDTO(c openapi.WebAuthnAssertionCredential) application.WebAuthnAssertionCredentialDTO {
	userHandle := ""
	if c.Response.UserHandle != nil {
		userHandle = *c.Response.UserHandle
	}
	attachment := ""
	if c.AuthenticatorAttachment != nil {
		attachment = *c.AuthenticatorAttachment
	}
	return application.WebAuthnAssertionCredentialDTO{
		ID:                      c.Id,
		RawID:                   c.RawId,
		Type:                    c.Type,
		AuthenticatorAttachment: attachment,
		Response: application.WebAuthnAssertionResponseDTO{
			ClientDataJSON:    c.Response.ClientDataJSON,
			AuthenticatorData: c.Response.AuthenticatorData,
			Signature:         c.Response.Signature,
			UserHandle:        userHandle,
		},
	}
}

func mapAttestationCredentialToDTO(c openapi.WebAuthnAttestationCredential) application.WebAuthnAttestationCredentialDTO {
	attachment := ""
	if c.AuthenticatorAttachment != nil {
		attachment = *c.AuthenticatorAttachment
	}
	var transports []string
	if c.Response.Transports != nil {
		transports = *c.Response.Transports
	}
	return application.WebAuthnAttestationCredentialDTO{
		ID:                      c.Id,
		RawID:                   c.RawId,
		Type:                    c.Type,
		AuthenticatorAttachment: attachment,
		Response: application.WebAuthnAttestationResponseDTO{
			ClientDataJSON:    c.Response.ClientDataJSON,
			AttestationObject: c.Response.AttestationObject,
			Transports:        transports,
		},
	}
}

// chooseAttestationCredential は recovery または invitation の credential を選択する。
// どちらも空（ゼロ値）の場合は空の DTO を返す（usecases 層で検証する）。
func chooseAttestationCredential(primary openapi.WebAuthnAttestationCredential, secondary openapi.WebAuthnAttestationCredential) application.WebAuthnAttestationCredentialDTO {
	if primary.Id != "" {
		return mapAttestationCredentialToDTO(primary)
	}
	return mapAttestationCredentialToDTO(secondary)
}

// ─── Multi-passkey management handlers ──────────────────────────────────────

func (s *StrictServer) ListPasskeys(ctx context.Context, _ openapi.ListPasskeysRequestObject) (openapi.ListPasskeysResponseObject, error) {
	session, err := s.auth.AuthorizeSession(ctx, bearerTokenFromContext(ctx))
	if err != nil {
		failureRequestID := nextAuthRequestID()
		if errors.Is(err, application.ErrAccountSuspended) {
			return openapi.ListPasskeys403JSONResponse{Body: authFailureResponseObject(failureRequestID, err), Headers: openapi.ListPasskeys403ResponseHeaders{CacheControl: noStoreValue}}, nil
		}
		return openapi.ListPasskeys401JSONResponse{Body: authFailureResponseObject(failureRequestID, err), Headers: openapi.ListPasskeys401ResponseHeaders{CacheControl: noStoreValue}}, nil
	}

	creds, err := s.auth.ListPasskeys(ctx, session.AccountID)
	if err != nil {
		if errors.Is(err, application.ErrInternalError) {
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
		failureRequestID := nextAuthRequestID()
		if errors.Is(err, application.ErrAccountSuspended) {
			return openapi.StartPasskeyAddition403JSONResponse{Body: authFailureResponseObject(failureRequestID, err), Headers: openapi.StartPasskeyAddition403ResponseHeaders{CacheControl: noStoreValue}}, nil
		}
		return openapi.StartPasskeyAddition401JSONResponse{Body: authFailureResponseObject(failureRequestID, err), Headers: openapi.StartPasskeyAddition401ResponseHeaders{CacheControl: noStoreValue}}, nil
	}

	result, err := s.auth.StartAddPasskey(ctx, session.AccountID)
	if err != nil {
		return openapi.StartPasskeyAddition503JSONResponse{Body: authFailureResponseObject(nextAuthRequestID(), err), Headers: openapi.StartPasskeyAddition503ResponseHeaders{CacheControl: noStoreValue}}, nil
	}

	body := openapi.PasskeyAddStartResponse{RequestId: result.RequestID, Challenge: result.Challenge, RpId: result.WebAuthnRPID}
	if err := applyWebAuthnRegistrationOptions(&body, result.WebAuthnOptions); err != nil {
		return openapi.StartPasskeyAddition503JSONResponse{Body: authFailureResponseObject(nextAuthRequestID(), err), Headers: openapi.StartPasskeyAddition503ResponseHeaders{CacheControl: noStoreValue}}, nil
	}
	return openapi.StartPasskeyAddition200JSONResponse{
		Body:    body,
		Headers: openapi.StartPasskeyAddition200ResponseHeaders{CacheControl: noStoreValue},
	}, nil
}

func (s *StrictServer) FinishPasskeyAddition(ctx context.Context, request openapi.FinishPasskeyAdditionRequestObject) (openapi.FinishPasskeyAdditionResponseObject, error) {
	if request.Body == nil {
		return openapi.FinishPasskeyAddition400JSONResponse{Body: authOperationError(nextAuthRequestID(), "request body is required"), Headers: openapi.FinishPasskeyAddition400ResponseHeaders{CacheControl: noStoreValue}}, nil
	}

	session, err := s.auth.AuthorizeSession(ctx, bearerTokenFromContext(ctx))
	if err != nil {
		failureRequestID := nextAuthRequestID()
		if errors.Is(err, application.ErrAccountSuspended) {
			return openapi.FinishPasskeyAddition403JSONResponse{Body: authFailureResponseObject(failureRequestID, err), Headers: openapi.FinishPasskeyAddition403ResponseHeaders{CacheControl: noStoreValue}}, nil
		}
		return openapi.FinishPasskeyAddition401JSONResponse{Body: authFailureResponseObject(failureRequestID, err), Headers: openapi.FinishPasskeyAddition401ResponseHeaders{CacheControl: noStoreValue}}, nil
	}

	creds, err := s.auth.FinishAddPasskey(ctx, session.AccountID, mapAttestationCredentialToDTO(request.Body.Credential))
	if err != nil {
		failureRequestID := nextAuthRequestID()
		if errors.Is(err, application.ErrInternalError) {
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
		failureRequestID := nextAuthRequestID()
		if errors.Is(err, application.ErrAccountSuspended) {
			return deletePasskey403Response{body: authFailureResponseObject(failureRequestID, err), headers: openapi.DeletePasskey403ResponseHeaders{CacheControl: noStoreValue}}, nil
		}
		return openapi.DeletePasskey401JSONResponse{Body: authFailureResponseObject(failureRequestID, err), Headers: openapi.DeletePasskey401ResponseHeaders{CacheControl: noStoreValue}}, nil
	}

	// X-Reauth-Session header で提示された再認証セッションを検証・consume する。
	if err := s.auth.VerifyReauthSession(ctx, request.Params.XReauthSession, session.AccountID, session.SessionID, "passkey-delete"); err != nil {
		failureRequestID := nextAuthRequestID()
		if errors.Is(err, application.ErrInternalError) {
			return openapi.DeletePasskey503JSONResponse{Body: authFailureResponseObject(failureRequestID, err), Headers: openapi.DeletePasskey503ResponseHeaders{CacheControl: noStoreValue}}, nil
		}
		return deletePasskey403Response{body: authOperationError(failureRequestID, nonRevealingAuthRejectMessage), headers: openapi.DeletePasskey403ResponseHeaders{CacheControl: noStoreValue}}, nil
	}

	if err := s.auth.DeletePasskey(ctx, session.AccountID, string(request.Id)); err != nil {
		failureRequestID := nextAuthRequestID()
		if errors.Is(err, application.ErrLastPasskeyCannotBeDeleted) {
			return openapi.DeletePasskey409JSONResponse{Body: authOperationError(failureRequestID, err.Error()), Headers: openapi.DeletePasskey409ResponseHeaders{CacheControl: noStoreValue}}, nil
		}
		if errors.Is(err, application.ErrInternalError) {
			return openapi.DeletePasskey503JSONResponse{Body: authFailureResponseObject(failureRequestID, err), Headers: openapi.DeletePasskey503ResponseHeaders{CacheControl: noStoreValue}}, nil
		}
		return deletePasskey403Response{body: authOperationError(failureRequestID, err.Error()), headers: openapi.DeletePasskey403ResponseHeaders{CacheControl: noStoreValue}}, nil
	}

	return openapi.DeletePasskey204Response{Headers: openapi.DeletePasskey204ResponseHeaders{CacheControl: noStoreValue}}, nil
}

func (s *StrictServer) SendDeviceLink(ctx context.Context, request openapi.SendDeviceLinkRequestObject) (openapi.SendDeviceLinkResponseObject, error) {
	session, err := s.auth.AuthorizeSession(ctx, bearerTokenFromContext(ctx))
	if err != nil {
		failureRequestID := nextAuthRequestID()
		if errors.Is(err, application.ErrAccountSuspended) {
			return sendDeviceLink403Response{body: authFailureResponseObject(failureRequestID, err), headers: openapi.SendDeviceLink403ResponseHeaders{CacheControl: noStoreValue}}, nil
		}
		return openapi.SendDeviceLink401JSONResponse{Body: authFailureResponseObject(failureRequestID, err), Headers: openapi.SendDeviceLink401ResponseHeaders{CacheControl: noStoreValue}}, nil
	}

	// X-Reauth-Session header で提示された再認証セッションを検証・consume する。
	if err := s.auth.VerifyReauthSession(ctx, request.Params.XReauthSession, session.AccountID, session.SessionID, "device-link"); err != nil {
		failureRequestID := nextAuthRequestID()
		if errors.Is(err, application.ErrInternalError) {
			return openapi.SendDeviceLink503JSONResponse{Body: authFailureResponseObject(failureRequestID, err), Headers: openapi.SendDeviceLink503ResponseHeaders{CacheControl: noStoreValue}}, nil
		}
		return sendDeviceLink403Response{body: authOperationError(failureRequestID, nonRevealingAuthRejectMessage), headers: openapi.SendDeviceLink403ResponseHeaders{CacheControl: noStoreValue}}, nil
	}

	result, err := s.auth.ExecuteDeviceLink(ctx, session.AccountID, session.SessionID)
	if err != nil {
		return openapi.SendDeviceLink503JSONResponse{Body: authFailureResponseObject(nextAuthRequestID(), err), Headers: openapi.SendDeviceLink503ResponseHeaders{CacheControl: noStoreValue}}, nil
	}

	return openapi.SendDeviceLink200JSONResponse{
		Body:    openapi.DeviceLinkResponse{RequestId: result.RequestID, Issued: openapi.DeviceLinkResponseIssuedTrue},
		Headers: openapi.SendDeviceLink200ResponseHeaders{CacheControl: noStoreValue},
	}, nil
}

// ─── Custom response types for union 403 bodies ─────────────────────────────
// oapi-codegen が union response body に対して構築ヘルパーを生成しないため、
// ハンドラから直接 JSON エンコードできるカスタムレスポンスタイプを定義する。

type deletePasskey403Response struct {
	body    interface{}
	headers openapi.DeletePasskey403ResponseHeaders
}

func (r deletePasskey403Response) VisitDeletePasskeyResponse(w stdhttp.ResponseWriter) error {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", r.headers.CacheControl)
	w.WriteHeader(403)
	return json.NewEncoder(w).Encode(r.body)
}

type sendDeviceLink403Response struct {
	body    interface{}
	headers openapi.SendDeviceLink403ResponseHeaders
}

func (r sendDeviceLink403Response) VisitSendDeviceLinkResponse(w stdhttp.ResponseWriter) error {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", r.headers.CacheControl)
	w.WriteHeader(403)
	return json.NewEncoder(w).Encode(r.body)
}

type revokeSession403Response struct {
	body    interface{}
	headers openapi.RevokeSession403ResponseHeaders
}

func (r revokeSession403Response) VisitRevokeSessionResponse(w stdhttp.ResponseWriter) error {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", r.headers.CacheControl)
	w.WriteHeader(403)
	return json.NewEncoder(w).Encode(r.body)
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

func userAgentFromContext(ctx context.Context) string {
	ginContext, ok := ctx.(*gin.Context)
	if !ok {
		return ""
	}
	return ginContext.Request.UserAgent()
}

func productRefreshCookieValue(ctx context.Context) (string, bool) {
	// Step 1: generated strict handler から渡される context が Gin context でない場合は Cookie を信頼できないため拒否する。
	ginContext, ok := ctx.(*gin.Context)
	if !ok {
		return "", false
	}

	// Step 2: Product refresh Cookie だけを読み取り、body や JavaScript 可読 storage 由来の値を認証材料にしない。
	value, err := ginContext.Cookie(productRefreshCookieName)
	if err != nil || strings.TrimSpace(value) == "" {
		return "", false
	}

	// Step 3: Cookie の平文値は token service に渡す直前だけ保持し、response body には戻さない。
	return value, true
}

func setProductRefreshCookie(ctx context.Context, value string, maxAge time.Duration) {
	// Step 1: generated strict handler から渡される context が Gin context でない場合は header を設定できないため何もしない。
	ginContext, ok := ctx.(*gin.Context)
	if !ok {
		return
	}

	// Step 2: SameSite=Lax を明示し、通常遷移では送信しつつ cross-site POST での送信余地を抑える。
	ginContext.SetSameSite(stdhttp.SameSiteLaxMode)

	// Step 3: refreshToken は HttpOnly/Secure Cookie としてだけ browser に返し、JavaScript から読める body field を作らない。
	ginContext.SetCookie(productRefreshCookieName, value, productRefreshCookieMaxAge(maxAge), "/api/v1/auth", "", true, true)
}

func clearProductRefreshCookie(ctx context.Context) {
	// Step 1: generated strict handler から渡される context が Gin context でない場合は header を設定できないため何もしない。
	ginContext, ok := ctx.(*gin.Context)
	if !ok {
		return
	}

	// Step 2: 発行時と同じ SameSite/Path/HttpOnly/Secure 属性で削除指示を返し、HttpOnly Cookie を server 側から確実に消す。
	ginContext.SetSameSite(stdhttp.SameSiteLaxMode)
	ginContext.SetCookie(productRefreshCookieName, "", -1, "/api/v1/auth", "", true, true)
}

func productRefreshCookieMaxAge(maxAge time.Duration) int {
	// Step 1: TTL 未設定の既存構成では Max-Age を省略する session cookie として扱い、無期限 delete 指示にはしない。
	if maxAge <= 0 {
		return 0
	}

	// Step 2: 1 秒未満の正の TTL でも即時失効しないよう、Cookie Max-Age の最小値を 1 秒に丸める。
	seconds := int(maxAge.Seconds())
	if seconds < 1 {
		return 1
	}

	// Step 3: 秒単位に変換した Max-Age を Gin の SetCookie に渡す。
	return seconds
}

// ─── WebAuthn options helpers ────────────────────────────────────────────────

// webAuthnLoginOptionsJSON は BeginLogin/BeginDiscoverableLogin が返す JSON の
// 関連フィールドを保持する中間構造体。
type webAuthnLoginOptionsJSON struct {
	PublicKey struct {
		AllowCredentials []struct {
			ID         string   `json:"id"`
			Type       string   `json:"type"`
			Transports []string `json:"transports,omitempty"`
		} `json:"allowCredentials,omitempty"`
		Timeout          *int64 `json:"timeout,omitempty"`
		UserVerification string `json:"userVerification,omitempty"`
	} `json:"publicKey"`
}

// webAuthnRegistrationOptionsJSON は BeginRegistration が返す JSON の
// 関連フィールドを保持する中間構造体。
type webAuthnRegistrationOptionsJSON struct {
	PublicKey struct {
		RP struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"rp"`
		User struct {
			ID          any    `json:"id"`
			Name        string `json:"name"`
			DisplayName string `json:"displayName"`
		} `json:"user"`
		PubKeyCredParams []struct {
			Type string `json:"type"`
			Alg  int32  `json:"alg"`
		} `json:"pubKeyCredParams,omitempty"`
		ExcludeCredentials []struct {
			ID         string   `json:"id"`
			Type       string   `json:"type"`
			Transports []string `json:"transports,omitempty"`
		} `json:"excludeCredentials,omitempty"`
		AuthenticatorSelection struct {
			RequireResidentKey *bool  `json:"requireResidentKey,omitempty"`
			ResidentKey        string `json:"residentKey,omitempty"`
			UserVerification   string `json:"userVerification,omitempty"`
		} `json:"authenticatorSelection"`
		Attestation string `json:"attestation,omitempty"`
		Timeout     *int64 `json:"timeout,omitempty"`
	} `json:"publicKey"`
}

// applyWebAuthnLoginOptions は WebAuthnOptions の JSON bytes から
// PasskeyStartResponse の optional フィールドを設定する。
// optionsJSON が nil または parse エラーの場合は何もしない（fail-open）。
func applyWebAuthnLoginOptions(resp *openapi.PasskeyStartResponse, optionsJSON []byte) {
	if len(optionsJSON) == 0 {
		return
	}
	var opts webAuthnLoginOptionsJSON
	if err := json.Unmarshal(optionsJSON, &opts); err != nil {
		return
	}
	pk := opts.PublicKey
	if len(pk.AllowCredentials) > 0 {
		descriptors := make([]openapi.WebAuthnCredentialDescriptor, 0, len(pk.AllowCredentials))
		for _, c := range pk.AllowCredentials {
			d := openapi.WebAuthnCredentialDescriptor{Id: c.ID, Type: c.Type}
			if len(c.Transports) > 0 {
				t := c.Transports
				d.Transports = &t
			}
			descriptors = append(descriptors, d)
		}
		resp.AllowCredentials = &descriptors
	}
	if pk.Timeout != nil {
		resp.Timeout = pk.Timeout
	}
	if pk.UserVerification != "" {
		resp.UserVerification = "required"
	}
}

// buildRegistrationUserEntity は webAuthnRegistrationOptionsJSON の User フィールドから
// openapi.WebAuthnUserEntity を構築する。
// go-webauthn は User.ID を base64url string として JSON 返す。
// 非 string 型や空文字の場合は error を返す（fail-closed）。
func buildRegistrationUserEntity(u struct {
	ID          any    `json:"id"`
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
},
) (openapi.WebAuthnUserEntity, error) {
	userID, ok := u.ID.(string)
	if !ok || userID == "" {
		return openapi.WebAuthnUserEntity{}, errors.New("webauthn options missing required user.id")
	}
	return openapi.WebAuthnUserEntity{Id: userID, Name: u.Name, DisplayName: u.DisplayName}, nil
}

// buildCredentialDescriptors は webAuthnRegistrationOptionsJSON の ExcludeCredentials から
// []openapi.WebAuthnCredentialDescriptor を構築する。
func buildCredentialDescriptors(raw []struct {
	ID         string   `json:"id"`
	Type       string   `json:"type"`
	Transports []string `json:"transports,omitempty"`
},
) []openapi.WebAuthnCredentialDescriptor {
	out := make([]openapi.WebAuthnCredentialDescriptor, 0, len(raw))
	for _, c := range raw {
		d := openapi.WebAuthnCredentialDescriptor{Id: c.ID, Type: c.Type}
		if len(c.Transports) > 0 {
			t := c.Transports
			d.Transports = &t
		}
		out = append(out, d)
	}
	return out
}

// applyWebAuthnRegistrationOptions は WebAuthnOptions の JSON bytes から
// PasskeyAddStartResponse のフィールドを設定する。
// optionsJSON が nil・parse エラー・必須フィールド（user / pubKeyCredParams / authenticatorSelection）欠落の場合は
// error を返す（fail-closed）。
func applyWebAuthnRegistrationOptions(resp *openapi.PasskeyAddStartResponse, optionsJSON []byte) error {
	if len(optionsJSON) == 0 {
		return errors.New("webauthn options are empty")
	}
	var opts webAuthnRegistrationOptionsJSON
	if err := json.Unmarshal(optionsJSON, &opts); err != nil {
		return fmt.Errorf("failed to parse webauthn options: %w", err)
	}
	pk := opts.PublicKey

	if pk.RP.Name == "" {
		return errors.New("webauthn options missing required rp.name")
	}
	resp.RpName = pk.RP.Name

	if pk.User.Name == "" {
		return errors.New("webauthn options missing required user.name")
	}
	if pk.User.DisplayName == "" {
		return errors.New("webauthn options missing required user.displayName")
	}
	userEntity, err := buildRegistrationUserEntity(pk.User)
	if err != nil {
		return err
	}
	resp.User = userEntity

	if len(pk.PubKeyCredParams) == 0 {
		return errors.New("webauthn options missing required pubKeyCredParams")
	}
	params := make([]openapi.WebAuthnCredentialParameter, 0, len(pk.PubKeyCredParams))
	for _, p := range pk.PubKeyCredParams {
		params = append(params, openapi.WebAuthnCredentialParameter{Type: p.Type, Alg: p.Alg})
	}
	resp.PubKeyCredParams = params

	if len(pk.ExcludeCredentials) > 0 {
		descriptors := buildCredentialDescriptors(pk.ExcludeCredentials)
		resp.ExcludeCredentials = &descriptors
	}
	if err := applyRegistrationAuthenticatorSelection(resp, pk.AuthenticatorSelection); err != nil {
		return err
	}
	if pk.Attestation != "" {
		att := pk.Attestation
		resp.Attestation = &att
	}
	if pk.Timeout != nil {
		resp.Timeout = pk.Timeout
	}
	return nil
}

// applyRegistrationAuthenticatorSelection は provider が生成した authenticatorSelection を Product API DTO へ写像する。
func applyRegistrationAuthenticatorSelection(resp *openapi.PasskeyAddStartResponse, selection struct {
	RequireResidentKey *bool  `json:"requireResidentKey,omitempty"`
	ResidentKey        string `json:"residentKey,omitempty"`
	UserVerification   string `json:"userVerification,omitempty"`
}) error {
	// Step 1: discoverable credential でない登録を返すと usernameless login と password manager 保存の前提が崩れるため、欠落や false は fail-closed にする。
	if selection.RequireResidentKey == nil || !*selection.RequireResidentKey {
		return errors.New("webauthn options require discoverable credential")
	}
	if selection.ResidentKey != "required" {
		return errors.New("webauthn options missing required residentKey")
	}

	// Step 2: UV-less credential を登録候補に出さないため、browser hint と server-side 検証条件を一致させる。
	if selection.UserVerification != "required" {
		return errors.New("webauthn options missing required userVerification")
	}

	// Step 3: API contract の literal fields として frontend に渡し、navigator.credentials.create の authenticatorSelection を復元できるようにする。
	resp.RequireResidentKey = true
	resp.ResidentKey = "required"
	resp.UserVerification = openapi.PasskeyAddStartResponseUserVerificationRequired
	return nil
}

// ─── Refresh token & session management handlers ────────────────────────────

func (s *StrictServer) RefreshToken(ctx context.Context, _ openapi.RefreshTokenRequestObject) (openapi.RefreshTokenResponseObject, error) {
	if s.tokenService == nil {
		return openapi.RefreshToken503JSONResponse{Body: authFailureResponseObject(nextAuthRequestID(), application.ErrInternalError), Headers: openapi.RefreshToken503ResponseHeaders{CacheControl: noStoreValue}}, nil
	}
	refreshToken, ok := productRefreshCookieValue(ctx)
	if !ok {
		return openapi.RefreshToken401JSONResponse{Body: authFailureResponseObject(nextAuthRequestID(), application.ErrUnauthenticated), Headers: openapi.RefreshToken401ResponseHeaders{CacheControl: noStoreValue}}, nil
	}

	accessToken, rotatedRefreshToken, accountID, err := s.tokenService.RefreshWithAccountID(ctx, refreshToken, clientIPFromContext(ctx), userAgentFromContext(ctx))
	if err != nil {
		failureRequestID := nextAuthRequestID()
		if errors.Is(err, application.ErrInternalError) {
			return openapi.RefreshToken503JSONResponse{Body: authFailureResponseObject(failureRequestID, err), Headers: openapi.RefreshToken503ResponseHeaders{CacheControl: noStoreValue}}, nil
		}
		if errors.Is(err, application.ErrAccountSuspended) {
			return openapi.RefreshToken403JSONResponse{Body: authFailureResponseObject(failureRequestID, err), Headers: openapi.RefreshToken403ResponseHeaders{CacheControl: noStoreValue}}, nil
		}
		return openapi.RefreshToken401JSONResponse{Body: authFailureResponseObject(failureRequestID, err), Headers: openapi.RefreshToken401ResponseHeaders{CacheControl: noStoreValue}}, nil
	}

	body := openapi.RefreshTokenResponse{
		AccessToken: accessToken,
	}
	if rotatedRefreshToken == "" {
		return openapi.RefreshToken503JSONResponse{Body: authFailureResponseObject(nextAuthRequestID(), application.ErrInternalError), Headers: openapi.RefreshToken503ResponseHeaders{CacheControl: noStoreValue}}, nil
	}
	if s.accountSnapshot == nil {
		return openapi.RefreshToken503JSONResponse{Body: authFailureResponseObject(nextAuthRequestID(), application.ErrInternalError), Headers: openapi.RefreshToken503ResponseHeaders{CacheControl: noStoreValue}}, nil
	}
	// refresh 成功応答は AccountID が確定した後の AccountSetting snapshot を必ず含める。
	// ここで snapshot を省略すると client が stale/default locale で描画するため、取得失敗は fail-closed にする。
	snapshot, snapshotErr := s.accountSnapshot.Load(ctx, accountID)
	if snapshotErr != nil {
		return openapi.RefreshToken503JSONResponse{Body: authFailureResponseObject(nextAuthRequestID(), application.ErrInternalError), Headers: openapi.RefreshToken503ResponseHeaders{CacheControl: noStoreValue}}, nil
	}
	body.AccountSetting = &openapi.AccountSettingSnapshot{Locale: openapi.AccountLocale(snapshot.Locale)}
	setProductRefreshCookie(ctx, rotatedRefreshToken, s.authRefreshCookieMaxAge)

	return openapi.RefreshToken200JSONResponse{
		Body:    body,
		Headers: openapi.RefreshToken200ResponseHeaders{CacheControl: noStoreValue},
	}, nil
}

func mapAccountSettingResponse(setting application.AccountSetting) openapi.AccountSettingResponse {
	return openapi.AccountSettingResponse{RequestId: nextAuthRequestID(), Setting: openapi.AccountSetting{Locale: openapi.AccountLocale(setting.Locale)}}
}

func accountSettingGetErrorResponse(err error) openapi.GetAccountSettingsResponseObject {
	requestID := nextAuthRequestID()
	if errors.Is(err, application.ErrAccountSettingNotFound) || errors.Is(err, application.ErrInvalidAccountSetting) {
		return openapi.GetAccountSettings403JSONResponse{Body: authFailureResponseObject(requestID, application.ErrAccountSuspended), Headers: openapi.GetAccountSettings403ResponseHeaders{CacheControl: noStoreValue}}
	}
	return openapi.GetAccountSettings503JSONResponse{Body: authFailureResponseObject(requestID, application.ErrInternalError), Headers: openapi.GetAccountSettings503ResponseHeaders{CacheControl: noStoreValue}}
}

func accountSettingUpdateErrorResponse(err error) openapi.UpdateAccountSettingsResponseObject {
	requestID := nextAuthRequestID()
	if errors.Is(err, application.ErrInvalidAccountSetting) {
		return openapi.UpdateAccountSettings400JSONResponse{Body: authOperationError(requestID, invalidRequestBodyMessage), Headers: openapi.UpdateAccountSettings400ResponseHeaders{CacheControl: noStoreValue}}
	}
	if errors.Is(err, application.ErrAccountSettingNotFound) {
		return openapi.UpdateAccountSettings403JSONResponse{Body: authFailureResponseObject(requestID, application.ErrAccountSuspended), Headers: openapi.UpdateAccountSettings403ResponseHeaders{CacheControl: noStoreValue}}
	}
	return openapi.UpdateAccountSettings503JSONResponse{Body: authFailureResponseObject(requestID, application.ErrInternalError), Headers: openapi.UpdateAccountSettings503ResponseHeaders{CacheControl: noStoreValue}}
}

func (s *StrictServer) ListSessions(ctx context.Context, _ openapi.ListSessionsRequestObject) (openapi.ListSessionsResponseObject, error) {
	if s.sessionService == nil {
		return openapi.ListSessions503JSONResponse{Body: authFailureResponseObject(nextAuthRequestID(), application.ErrInternalError), Headers: openapi.ListSessions503ResponseHeaders{CacheControl: noStoreValue}}, nil
	}

	session, err := s.auth.AuthorizeSession(ctx, bearerTokenFromContext(ctx))
	if err != nil {
		failureRequestID := nextAuthRequestID()
		if errors.Is(err, application.ErrAccountSuspended) {
			return openapi.ListSessions403JSONResponse{Body: authFailureResponseObject(failureRequestID, err), Headers: openapi.ListSessions403ResponseHeaders{CacheControl: noStoreValue}}, nil
		}
		return openapi.ListSessions401JSONResponse{Body: authFailureResponseObject(failureRequestID, err), Headers: openapi.ListSessions401ResponseHeaders{CacheControl: noStoreValue}}, nil
	}

	sessions, err := s.sessionService.List(ctx, session.AccountID)
	if err != nil {
		if errors.Is(err, application.ErrInternalError) {
			return openapi.ListSessions503JSONResponse{Body: authFailureResponseObject(nextAuthRequestID(), err), Headers: openapi.ListSessions503ResponseHeaders{CacheControl: noStoreValue}}, nil
		}
		return openapi.ListSessions401JSONResponse{Body: authFailureResponseObject(nextAuthRequestID(), err), Headers: openapi.ListSessions401ResponseHeaders{CacheControl: noStoreValue}}, nil
	}

	items := make([]openapi.SessionItem, 0, len(sessions))
	for _, sess := range sessions {
		items = append(items, openapi.SessionItem{
			SessionId:        sess.SessionID,
			DeviceName:       sess.DeviceName,
			LoginAt:          sess.LoginAt,
			LastActiveAt:     sess.LastActiveAt,
			IpHash:           sess.IPHash,
			IsCurrentSession: sess.SessionID == session.SessionID,
		})
	}

	requestID := nextAuthRequestID()
	return openapi.ListSessions200JSONResponse{
		Body: openapi.SessionListResponse{
			RequestId: requestID,
			Sessions:  items,
		},
		Headers: openapi.ListSessions200ResponseHeaders{CacheControl: noStoreValue},
	}, nil
}

func (s *StrictServer) RevokeSession(ctx context.Context, request openapi.RevokeSessionRequestObject) (openapi.RevokeSessionResponseObject, error) {
	if s.sessionService == nil {
		return openapi.RevokeSession503JSONResponse{Body: authFailureResponseObject(nextAuthRequestID(), application.ErrInternalError), Headers: openapi.RevokeSession503ResponseHeaders{CacheControl: noStoreValue}}, nil
	}

	session, err := s.auth.AuthorizeSession(ctx, bearerTokenFromContext(ctx))
	if err != nil {
		failureRequestID := nextAuthRequestID()
		if errors.Is(err, application.ErrAccountSuspended) {
			return revokeSession403Response{body: authFailureResponseObject(failureRequestID, err), headers: openapi.RevokeSession403ResponseHeaders{CacheControl: noStoreValue}}, nil
		}
		return openapi.RevokeSession401JSONResponse{Body: authFailureResponseObject(failureRequestID, err), Headers: openapi.RevokeSession401ResponseHeaders{CacheControl: noStoreValue}}, nil
	}

	if err := s.sessionService.Revoke(ctx, session.AccountID, string(request.Id)); err != nil {
		failureRequestID := nextAuthRequestID()
		if errors.Is(err, application.ErrInternalError) {
			return openapi.RevokeSession503JSONResponse{Body: authFailureResponseObject(failureRequestID, err), Headers: openapi.RevokeSession503ResponseHeaders{CacheControl: noStoreValue}}, nil
		}
		if errors.Is(err, application.ErrBadRequest) {
			return revokeSession403Response{body: authOperationError(failureRequestID, nonRevealingAuthRejectMessage), headers: openapi.RevokeSession403ResponseHeaders{CacheControl: noStoreValue}}, nil
		}
		return openapi.RevokeSession401JSONResponse{Body: authFailureResponseObject(failureRequestID, err), Headers: openapi.RevokeSession401ResponseHeaders{CacheControl: noStoreValue}}, nil
	}

	return openapi.RevokeSession204Response{Headers: openapi.RevokeSession204ResponseHeaders{CacheControl: noStoreValue}}, nil
}

func (s *StrictServer) RevokeOtherSessions(ctx context.Context, _ openapi.RevokeOtherSessionsRequestObject) (openapi.RevokeOtherSessionsResponseObject, error) {
	if s.sessionService == nil {
		return openapi.RevokeOtherSessions503JSONResponse{Body: authFailureResponseObject(nextAuthRequestID(), application.ErrInternalError), Headers: openapi.RevokeOtherSessions503ResponseHeaders{CacheControl: noStoreValue}}, nil
	}

	session, err := s.auth.AuthorizeSession(ctx, bearerTokenFromContext(ctx))
	if err != nil {
		failureRequestID := nextAuthRequestID()
		if errors.Is(err, application.ErrAccountSuspended) {
			return openapi.RevokeOtherSessions403JSONResponse{Body: authFailureResponseObject(failureRequestID, err), Headers: openapi.RevokeOtherSessions403ResponseHeaders{CacheControl: noStoreValue}}, nil
		}
		return openapi.RevokeOtherSessions401JSONResponse{Body: authFailureResponseObject(failureRequestID, err), Headers: openapi.RevokeOtherSessions401ResponseHeaders{CacheControl: noStoreValue}}, nil
	}

	if err := s.sessionService.RevokeOthers(ctx, session.AccountID, session.SessionID); err != nil {
		if errors.Is(err, application.ErrInternalError) {
			return openapi.RevokeOtherSessions503JSONResponse{Body: authFailureResponseObject(nextAuthRequestID(), err), Headers: openapi.RevokeOtherSessions503ResponseHeaders{CacheControl: noStoreValue}}, nil
		}
		return openapi.RevokeOtherSessions401JSONResponse{Body: authFailureResponseObject(nextAuthRequestID(), err), Headers: openapi.RevokeOtherSessions401ResponseHeaders{CacheControl: noStoreValue}}, nil
	}

	return openapi.RevokeOtherSessions204Response{Headers: openapi.RevokeOtherSessions204ResponseHeaders{CacheControl: noStoreValue}}, nil
}
