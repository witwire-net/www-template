package shared

import (
	stdhttp "net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestSharedHTTPAuthHelpersOwnCommonSecurityAndCredentialRules(t *testing.T) {
	t.Parallel()

	t.Run("[AUTH-BE-S099] shared helper extracts bearer and rejects non bearer schemes", func(t *testing.T) {
		// Step 1: Product/Admin の Authorization parsing が同じ helper を使う前提として、Bearer 形式だけを token として抽出する。
		if got := BearerToken(" Bearer access-token "); got != "access-token" {
			t.Fatalf("expected bearer token extraction, got %q", got)
		}

		// Step 2: Basic や空白値は protected route 側で unauthenticated にできるよう空文字へ正規化する。
		if got := BearerToken("Basic access-token"); got != "" {
			t.Fatalf("expected non bearer scheme to be rejected, got %q", got)
		}
	})

	t.Run("[AUTH-BE-S099] shared helper normalizes Origin without path based wildcard matching", func(t *testing.T) {
		// Step 1: 許可 origin と request Origin を canonical 形式で完全一致させ、大小文字と末尾 slash の揺れだけを吸収する。
		if !OriginAllowed([]string{"https://admin.example.com/"}, "HTTPS://ADMIN.EXAMPLE.COM") {
			t.Fatalf("expected canonical origin match")
		}

		// Step 2: path/query/fragment 付き Origin は allowlist に似ていても拒否し、Cookie-setting flow の曖昧一致を防ぐ。
		if OriginAllowed([]string{"https://admin.example.com"}, "https://admin.example.com/path") {
			t.Fatalf("expected origin with path to be rejected")
		}
	})

	t.Run("[AUTH-BE-S099] shared helper rejects cross-site fetch metadata", func(t *testing.T) {
		// Step 1: cross-site は SameSite や Bearer の有無に頼らず、shared helper の時点で拒否する。
		if FetchMetadataAccepted("cross-site") {
			t.Fatalf("expected cross-site Fetch Metadata to be rejected")
		}

		// Step 2: Fetch Metadata 欠落時は Origin allowlist を主境界にするため、legacy client 用に true を返す。
		if !FetchMetadataAccepted("") {
			t.Fatalf("expected missing Fetch Metadata to defer to Origin validation")
		}
	})

	t.Run("[AUTH-BE-S099] shared helper applies common no-store security headers", func(t *testing.T) {
		// Step 1: Gin の test context を作り、Product/Admin の middleware と同じ header 書き込み対象を準備する。
		writer := httptest.NewRecorder()
		context, _ := gin.CreateTestContext(writer)
		context.Request = httptest.NewRequest(stdhttp.MethodGet, "/api/v1/auth/operator/current", nil)

		// Step 2: shared helper から no-store と browser hardening header を付与し、surface ごとの差分が出ないことを固定する。
		ApplyBrowserSecurityHeaders(context, "no-store")
		if got := writer.Header().Get("Cache-Control"); got != "no-store" {
			t.Fatalf("expected no-store header, got %q", got)
		}
		if got := writer.Header().Get("X-Frame-Options"); got != "DENY" {
			t.Fatalf("expected frame deny header, got %q", got)
		}
	})
}
