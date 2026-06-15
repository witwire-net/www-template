package shared

import "github.com/gin-gonic/gin"

// EnableStrictHandlerRequestContextFallback は generated strict handler が受け取る Gin context に request context fallback を有効化する。
//
// 役割:
//   - oapi-codegen の strict handler は `*gin.Context` を `context.Context` として application handler へ渡すため、
//     Gin context の `Value` / `Done` / `Deadline` / `Err` が `Request.Context()` を参照できる状態にする。
//   - otelgin が `Request.Context()` に設定した trace context を strict handler 経由の application call へ伝搬させる。
//   - Product/Admin の route registration と strict handler wiring を維持し、non-strict handler へ移行しない。
//
// 引数:
//   - router: fallback を有効化する Gin engine。nil の場合は何も変更しない。
//
// 副作用:
//   - router.ContextWithFallback を true に設定する。
//
// 使用例:
//
//	router := gin.New()
//	shared.EnableStrictHandlerRequestContextFallback(router)
func EnableStrictHandlerRequestContextFallback(router *gin.Engine) {
	// Step 1: nil router は test fixture の誤用でも panic させず、caller が後続の fail-close 検証へ進めるよう何もしない。
	if router == nil {
		return
	}

	// Step 2: Gin context の context.Context 実装を Request.Context へ委譲し、OTel span と cancellation を strict handler の ctx へ伝える。
	router.ContextWithFallback = true
}
