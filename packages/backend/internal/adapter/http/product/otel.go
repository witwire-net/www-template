package product

import (
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

// otelMiddleware は Product API の Gin router に OpenTelemetry middleware を接続する。
// service 名は Product binary の既存 telemetry 名を維持し、router 登録順の副作用だけを持つ。
func otelMiddleware() gin.HandlerFunc {
	return otelgin.Middleware("www-template-api")
}
