package product

import (
	"context"
	"crypto/rand"
	"errors"
	stdhttp "net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	sharedhttp "www-template/packages/backend/internal/adapter/http/shared"
	application "www-template/packages/backend/internal/application/auth"
	"www-template/packages/backend/internal/generated/openapi"
	"www-template/packages/backend/internal/platform/config"
	"www-template/packages/backend/internal/platform/id"
)

const noStoreValue = "no-store"
const nonRevealingAuthRejectMessage = "request rejected"
const invalidRequestBodyMessage = "invalid request body"
const fallbackAuthRequestID = "01ARZ3NDEKTSV4RRFFQ69G5FAV"
const productOriginHeader = "Origin"
const productFetchSiteHeader = "Sec-Fetch-Site"
const productSecurityCSP = sharedhttp.SecurityCSP
const productSecurityHSTS = sharedhttp.SecurityHSTS
const productSecurityReferrerPolicy = sharedhttp.SecurityReferrerPolicy

func appAuthMiddleware(cfg config.Config, auth application.ProductAuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !strings.HasPrefix(c.Request.URL.Path, "/api/v1/") ||
			strings.HasPrefix(c.Request.URL.Path, "/api/v1/auth/") {
			c.Next()
			return
		}

		token := sharedhttp.BearerToken(c.GetHeader("Authorization"))
		if auth == nil {
			if strings.TrimSpace(c.GetHeader("Authorization")) == cfg.AppAuthorizationValue() {
				c.Next()
				return
			}
			writeAuthFailure(c, application.ErrUnauthenticated)
			return
		}
		_, err := auth.AuthorizeSession(c.Request.Context(), token)
		if err == nil {
			c.Next()
			return
		}

		writeAuthFailure(c, err)
	}
}

func authNoStoreAndBindErrorMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if isNoStoreAuthPath(c.Request.URL.Path) {
			c.Header("Cache-Control", noStoreValue)
		}

		c.Next()

		if !isNoStoreAuthPath(c.Request.URL.Path) {
			return
		}
		if c.Writer.Status() != stdhttp.StatusBadRequest || c.Writer.Size() > 0 || len(c.Errors) == 0 {
			return
		}

		c.JSON(stdhttp.StatusBadRequest, authOperationError(nextAuthRequestID(), invalidRequestBodyMessage))
	}
}

func productSecurityHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Step 1: Product API response だけに browser hardening header を付与し、health check や static ではない route へ不要な副作用を出さない。
		if isProductAPIPath(c.Request.URL.Path) {
			applyProductSecurityHeaders(c)
		}

		// Step 2: handler / auth middleware が status と body を決められるよう、header baseline だけを先に固定して処理を継続する。
		c.Next()
	}
}

func applyProductSecurityHeaders(c *gin.Context) {
	// Step 1: Product API JSON response を clickjacking、MIME sniffing、referer leakage から守る共通 header は shared helper concept に委譲する。
	sharedhttp.ApplyBrowserSecurityHeaders(c, "")
}

func writeAuthFailure(c *gin.Context, err error) {
	requestID := nextAuthRequestID()
	status := stdhttp.StatusUnauthorized
	classification := openapi.Unauthenticated

	switch {
	case errors.Is(err, application.ErrAccountSuspended):
		status = stdhttp.StatusForbidden
		classification = openapi.AccountSuspended
	case errors.Is(err, application.ErrSessionExpired):
		classification = openapi.SessionExpired
	case errors.Is(err, application.ErrInternalError):
		status = stdhttp.StatusServiceUnavailable
		classification = openapi.InternalError
	}

	response := openapi.AuthFailureResponse{
		Error:     classification,
		RequestId: requestID,
	}
	c.Header("Cache-Control", noStoreValue)
	c.AbortWithStatusJSON(status, response)
}

func productCookieSettingRequestAccepted(ctx context.Context, allowedOrigins []string) bool {
	// Step 1: generated strict handler 以外から呼ばれた場合は HTTP header を検証できないため、Cookie 発行・rotation を fail-close する。
	ginContext, ok := ctx.(*gin.Context)
	if !ok {
		return false
	}

	// Step 2: Cookie mode は browser credential を伴うため、Origin を allowlist の canonical origin と完全一致させる。
	if !sharedhttp.OriginAllowed(allowedOrigins, ginContext.GetHeader(productOriginHeader)) {
		return false
	}

	// Step 3: Fetch Metadata が cross-site を示す request は、SameSite=Lax だけに頼らず server 側で拒否する。
	return sharedhttp.FetchMetadataAccepted(ginContext.GetHeader(productFetchSiteHeader))
}

func fallbackRequestID() string {
	return fallbackAuthRequestID
}

func authFailureResponseObject(requestID string, err error) openapi.AuthFailureResponse {
	classification := openapi.Unauthenticated
	if errors.Is(err, application.ErrAccountSuspended) {
		classification = openapi.AccountSuspended
	}
	if errors.Is(err, application.ErrSessionExpired) {
		classification = openapi.SessionExpired
	}
	if errors.Is(err, application.ErrInternalError) {
		classification = openapi.InternalError
	}

	return openapi.AuthFailureResponse{RequestId: requestID, Error: classification}
}

func authOperationError(requestID string, message string) openapi.AuthOperationErrorResponse {
	return openapi.AuthOperationErrorResponse{RequestId: requestID, Error: message}
}

func nextAuthRequestID() string {
	id, err := id.NewULID(time.Now().UTC(), rand.Reader)
	if err != nil {
		return fallbackRequestID()
	}

	return id
}

func isNoStoreAuthPath(path string) bool {
	return strings.HasPrefix(path, "/api/v1/auth/") ||
		path == "/api/v1/auth/logout" ||
		strings.HasPrefix(path, "/api/v1/passkeys") ||
		path == "/api/v1/account/settings"
}

func isProductAPIPath(path string) bool {
	// Step 1: Product/OpenAPI の公開境界に合わせ、browser hardening header の対象を `/api/v1/*` に限定する。
	return strings.HasPrefix(path, "/api/v1/")
}
