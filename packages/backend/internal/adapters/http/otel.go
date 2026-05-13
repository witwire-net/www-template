package http

import (
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

// OTelMiddleware returns the otelgin middleware with service name.
func OTelMiddleware() gin.HandlerFunc {
	return otelgin.Middleware("www-template-api")
}
