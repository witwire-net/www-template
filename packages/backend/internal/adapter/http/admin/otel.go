package admin

import (
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

// otelMiddleware は Admin API の Gin router に OpenTelemetry middleware を接続する。
//
// Product router と同じ構造で OTel tracing を Admin API にも適用し、Admin と Product で
// observability の非対称を解消する。service 名は Admin binary 専用の telemetry 名を使用する。
func otelMiddleware() gin.HandlerFunc {
	return otelgin.Middleware("www-template-admin-api")
}
