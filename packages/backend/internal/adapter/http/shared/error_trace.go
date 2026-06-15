package shared

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	platformobservability "www-template/packages/backend/internal/platform/observability"
)

const errorTraceEventType = "http.error.response"
const errorTraceBodyCaptureLimit = 4096

// ErrorTraceMiddleware は HTTP error response を現在の request span へ安全な trace event として記録する。
//
// 役割:
//   - Product/Admin の generated strict handler、middleware、NoRoute、panic recovery から返る 4xx/5xx を一律に trace event 化する。
//   - response body 全体や raw error message は trace に載せず、上限付きに一時捕捉した JSON から安全な requestId だけを抽出する。
//   - OTel 型は platform/observability に閉じ込め、HTTP adapter は slog.Attr ベースの境界だけを使う。
//
// 引数:
//   - surface: `product` または `admin` など、どの API surface の error response かを示す安全な分類値。
//
// 戻り値:
//   - gin.HandlerFunc: router.Use で OTel middleware の後、認証/handler middleware の前に登録する middleware。
//
// 副作用:
//   - 4xx/5xx response または downstream panic 時に、現在の trace span へ `http.error.response` event を追加する。
func ErrorTraceMiddleware(surface string) gin.HandlerFunc {
	// Step 1: surface は middleware 生成時に正規化し、各 request で空文字や任意文字列を属性化しないようにする。
	normalizedSurface := normalizeErrorTraceSurface(surface)
	return func(c *gin.Context) {
		// Step 2: response writer を薄く wrap し、client へ body をそのまま流しながら requestId 抽出用の先頭数 KB だけを保持する。
		writer := &errorTraceResponseWriter{ResponseWriter: c.Writer}
		c.Writer = writer

		// Step 3: downstream panic は gin.Recovery に再送しつつ、recovery が body を書く前に panic 分類だけを trace event へ残す。
		defer func() {
			if recovered := recover(); recovered != nil {
				recordHTTPErrorTraceEvent(c, writer, normalizedSurface, http.StatusInternalServerError, "panic", "panic")
				panic(recovered)
			}
		}()

		// Step 4: 後続 middleware / generated handler を実行し、確定した status code から error response だけを観測する。
		c.Next()
		statusCode := c.Writer.Status()
		if statusCode < http.StatusBadRequest {
			return
		}
		errorSource, errorClass := classifyHTTPErrorStatus(statusCode)
		recordHTTPErrorTraceEvent(c, writer, normalizedSurface, statusCode, errorSource, errorClass)
	}
}

type errorTraceResponseWriter struct {
	gin.ResponseWriter
	body bytes.Buffer
}

func (w *errorTraceResponseWriter) Write(data []byte) (int, error) {
	// Step 1: body は client へ必ずそのまま流し、観測用 buffer は requestId 抽出に必要な上限分だけに制限する。
	w.capture(data)
	return w.ResponseWriter.Write(data)
}

func (w *errorTraceResponseWriter) WriteString(data string) (int, error) {
	// Step 1: Gin の JSON writer が string 経路を使う場合も同じ上限付き buffer に捕捉する。
	w.capture([]byte(data))
	return w.ResponseWriter.WriteString(data)
}

func (w *errorTraceResponseWriter) capture(data []byte) {
	// Step 1: capture limit を超えた response body は保持せず、大きな error body が memory を圧迫しないようにする。
	remaining := errorTraceBodyCaptureLimit - w.body.Len()
	if remaining <= 0 {
		return
	}

	// Step 2: 上限を超える chunk は requestId が先頭付近にある場合だけ拾える範囲に切り詰める。
	if len(data) > remaining {
		data = data[:remaining]
	}
	_, _ = w.body.Write(data)
}

func recordHTTPErrorTraceEvent(c *gin.Context, writer *errorTraceResponseWriter, surface string, statusCode int, errorSource string, errorClass string) {
	// Step 1: route は Gin の template だけを使い、`/api/v1/accounts/:id` の実 ID などを trace 属性へ出さない。
	route := c.FullPath()
	if strings.TrimSpace(route) == "" {
		route = "unmatched"
	}

	// Step 2: method と route template から運用検索用の operation_id を作る。query/body は含めない。
	method := strings.TrimSpace(c.Request.Method)
	operationID := method + " " + route

	// Step 3: 属性は status/source/class/requestId などの安全な分類値だけに限定し、response body や raw error は載せない。
	attrs := []slog.Attr{
		slog.String("event_type", errorTraceEventType),
		slog.String("http.surface", surface),
		slog.String("http.method", method),
		slog.String("http.route", route),
		slog.String("operation_id", operationID),
		slog.Int("http.status_code", statusCode),
		slog.String("error_source", errorSource),
		slog.String("error_class", errorClass),
		slog.Bool("raw_response_logged", false),
	}
	if requestID := requestIDFromCapturedErrorBody(writer.body.Bytes()); requestID != "" {
		attrs = append(attrs, slog.String("request_id", requestID))
	}

	// Step 4: request.Context に入っている otelgin の server span へ event を追加し、span が無い test/utility 呼び出しでは no-op にする。
	platformobservability.AddTraceEvent(c.Request.Context(), errorTraceEventType, attrs)
}

func classifyHTTPErrorStatus(statusCode int) (string, string) {
	// Step 1: 代表的な HTTP status を、業務原因を漏らさない粗い error_source / error_class に正規化する。
	switch statusCode {
	case http.StatusBadRequest, http.StatusRequestEntityTooLarge:
		return "validation", httpErrorClass(statusCode)
	case http.StatusUnauthorized:
		return "auth", "unauthenticated"
	case http.StatusForbidden:
		return "auth", "forbidden"
	case http.StatusNotFound:
		return "routing", "not_found"
	case http.StatusConflict:
		return "conflict", "conflict"
	case http.StatusTooManyRequests:
		return "rate_limit", "rate_limited"
	case http.StatusInternalServerError:
		return "internal", "internal_error"
	case http.StatusServiceUnavailable:
		return "internal", "service_unavailable"
	default:
		if statusCode >= http.StatusInternalServerError {
			return "internal", "server_error"
		}
		return "client", "client_error"
	}
}

func httpErrorClass(statusCode int) string {
	// Step 1: validation 系の中でも payload size は request body 制限として切り分ける。
	if statusCode == http.StatusRequestEntityTooLarge {
		return "payload_too_large"
	}
	return "bad_request"
}

func requestIDFromCapturedErrorBody(body []byte) string {
	// Step 1: 空 body や非 JSON body は requestId なしとして扱い、body 文字列自体は trace へ渡さない。
	if len(body) == 0 {
		return ""
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return ""
	}

	// Step 2: Product/Admin generated response の camelCase と、将来の snake_case の両方から安全な ID だけを拾う。
	for _, key := range []string{"requestId", "request_id"} {
		value, ok := payload[key].(string)
		if !ok {
			continue
		}
		requestID := strings.TrimSpace(value)
		if isSafeTraceIdentifier(requestID) {
			return requestID
		}
	}
	return ""
}

func isSafeTraceIdentifier(value string) bool {
	// Step 1: request_id は検索キーなので短い ASCII token だけを許可し、body 経由の任意文字列を trace 属性にしない。
	if value == "" || len(value) > 128 {
		return false
	}
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' {
			continue
		}
		return false
	}
	return true
}

func normalizeErrorTraceSurface(surface string) string {
	// Step 1: surface は固定分類値に限定し、caller の誤設定で任意文字列が trace facet に増殖しないようにする。
	switch strings.TrimSpace(surface) {
	case "product":
		return "product"
	case "admin":
		return "admin"
	default:
		return "unknown"
	}
}
