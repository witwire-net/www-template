package http

import (
	stdhttp "net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"witwire.net/www-template/packages/backend/internal/types"
)

func appAuthMiddleware(cfg types.Config) gin.HandlerFunc {
	expectedAuthorization := cfg.AppAuthorizationValue()

	return func(c *gin.Context) {
		if !strings.HasPrefix(c.Request.URL.Path, "/api/v1/app") {
			c.Next()
			return
		}

		if strings.TrimSpace(c.GetHeader("Authorization")) != expectedAuthorization {
			c.AbortWithStatusJSON(stdhttp.StatusUnauthorized, gin.H{
				"error": "missing or invalid bearer token",
			})
			return
		}

		c.Next()
	}
}
