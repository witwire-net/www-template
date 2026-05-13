package http

import (
	"crypto/rand"
	"errors"
	stdhttp "net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"www-template/packages/backend/internal/auth/application"
	"www-template/packages/backend/internal/generated/openapi"
	"www-template/packages/backend/internal/platform/config"
	"www-template/packages/backend/internal/platform/id"
)

const noStoreValue = "no-store"
const nonRevealingAuthRejectMessage = "request rejected"
const invalidRequestBodyMessage = "invalid request body"
const fallbackAuthRequestID = "01ARZ3NDEKTSV4RRFFQ69G5FAV"

func appAuthMiddleware(cfg config.Config, auth *application.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !strings.HasPrefix(c.Request.URL.Path, "/api/v1/") ||
			strings.HasPrefix(c.Request.URL.Path, "/api/v1/auth/") {
			c.Next()
			return
		}

		token := bearerToken(c.GetHeader("Authorization"))
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

func bearerToken(header string) string {
	trimmed := strings.TrimSpace(header)
	if !strings.HasPrefix(trimmed, "Bearer ") {
		return ""
	}

	return strings.TrimSpace(strings.TrimPrefix(trimmed, "Bearer "))
}

func requestIP(c *gin.Context) string {
	if ip := strings.TrimSpace(c.ClientIP()); ip != "" {
		return ip
	}

	return "unknown"
}

func writeAuthFailure(c *gin.Context, err error) {
	requestID := nextAuthRequestID()
	status := stdhttp.StatusUnauthorized
	classification := openapi.Unauthenticated

	switch {
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

func fallbackRequestID() string {
	return fallbackAuthRequestID
}

func authFailureResponseObject(requestID string, err error) openapi.AuthFailureResponse {
	classification := openapi.Unauthenticated
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
		strings.HasPrefix(path, "/api/v1/passkeys")
}
