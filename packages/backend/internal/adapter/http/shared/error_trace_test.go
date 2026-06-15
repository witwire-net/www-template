package shared

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	platformobservability "www-template/packages/backend/internal/platform/observability"
)

// [HTTP-OBS-ERROR-1] 4xx response は raw body を出さず、分類値と request_id だけを trace event に残す。
func TestErrorTraceMiddlewareRecordsSafeTraceEventForClientError(t *testing.T) {
	restoreTracer := platformobservability.InstallLocalTracerProviderForTesting()
	restoreRecorder, readEvents := platformobservability.InstallTraceEventRecorderForTesting()
	t.Cleanup(func() {
		restoreRecorder()
		_ = restoreTracer(context.Background())
	})

	// Step 1: request span を明示的に作り、otelgin を使わない shared middleware 単体 test でも trace event を記録できる状態にする。
	spanCtx, endSpan := platformobservability.StartDetachedSpan(context.Background(), "http-error-trace-test", "request")

	// Step 2: error body には secret 風文字列を含めるが、middleware は requestId 以外を trace 属性へ載せないことを検証する。
	router := gin.New()
	router.Use(ErrorTraceMiddleware("product"))
	router.POST("/api/v1/auth/recovery", func(c *gin.Context) {
		c.JSON(http.StatusBadRequest, gin.H{"requestId": "01HXHTTPERRORTRACE0000000001", "error": "secret@example.com must not leak"})
	})
	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/recovery", strings.NewReader(`{}`)).WithContext(spanCtx)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)
	endSpan()

	if response.Code != http.StatusBadRequest {
		t.Fatalf("response status = %d, want %d", response.Code, http.StatusBadRequest)
	}
	event := findRecordedTraceEvent(t, readEvents(), errorTraceEventType)
	attrs := event.Attributes
	if attrs["event_type"] != errorTraceEventType || attrs["http.surface"] != "product" || attrs["http.status_code"] != "400" {
		t.Fatalf("unexpected error trace attrs: %#v", attrs)
	}
	if attrs["error_source"] != "validation" || attrs["error_class"] != "bad_request" {
		t.Fatalf("unexpected error classification attrs: %#v", attrs)
	}
	if attrs["request_id"] != "01HXHTTPERRORTRACE0000000001" {
		t.Fatalf("request_id attr = %q", attrs["request_id"])
	}
	serializedAttrs := traceEventAttrsString(attrs)
	if strings.Contains(serializedAttrs, "secret@example.com") || strings.Contains(serializedAttrs, "must not leak") || attrs["raw_response_logged"] != "false" {
		t.Fatalf("error trace attrs leaked raw response: %s", serializedAttrs)
	}
}

// [HTTP-OBS-ERROR-2] 2xx response は error trace event を残さない。
func TestErrorTraceMiddlewareSkipsSuccessfulResponse(t *testing.T) {
	restoreTracer := platformobservability.InstallLocalTracerProviderForTesting()
	restoreRecorder, readEvents := platformobservability.InstallTraceEventRecorderForTesting()
	t.Cleanup(func() {
		restoreRecorder()
		_ = restoreTracer(context.Background())
	})
	spanCtx, endSpan := platformobservability.StartDetachedSpan(context.Background(), "http-error-trace-test", "request")

	// Step 1: 成功 response を流し、middleware が error event を追加しないことを確認する。
	router := gin.New()
	router.Use(ErrorTraceMiddleware("admin"))
	router.GET("/api/v1/status", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	request := httptest.NewRequest(http.MethodGet, "/api/v1/status", nil).WithContext(spanCtx)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)
	endSpan()

	if response.Code != http.StatusOK {
		t.Fatalf("response status = %d, want %d", response.Code, http.StatusOK)
	}
	for _, event := range readEvents() {
		if event.Name == errorTraceEventType {
			t.Fatalf("unexpected error trace event for successful response: %#v", event)
		}
	}
}

func findRecordedTraceEvent(t *testing.T, events []platformobservability.RecordedTraceEvent, name string) platformobservability.RecordedTraceEvent {
	t.Helper()
	for _, event := range events {
		if event.Name == name {
			return event
		}
	}
	t.Fatalf("trace event %q not found in %#v", name, events)
	return platformobservability.RecordedTraceEvent{}
}

func traceEventAttrsString(attrs map[string]string) string {
	// Step 1: map の順序に依存せず、漏えい検査用に key=value を単純連結する。
	parts := make([]string, 0, len(attrs))
	for key, value := range attrs {
		parts = append(parts, key+"="+value)
	}
	return strings.Join(parts, " ")
}
