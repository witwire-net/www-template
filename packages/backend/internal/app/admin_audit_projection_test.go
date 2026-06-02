package app

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	auditapplication "www-template/packages/backend/internal/application/audit"
	"www-template/packages/backend/internal/platform/config"
)

// [ADMIN-CONSOLE-BE-S085] Admin audit projection は Admin audit prefix の月次 index にだけ書き込む。
func TestAdminAuditOpenSearchProjectorIndexesAdminAuditPrefix(t *testing.T) {
	t.Parallel()

	// Step 1: httptest server で OpenSearch の document API だけを模倣し、実ネットワークなしで index path を捕捉する。
	var requestPath string
	var requestBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestPath = r.URL.Path
		requestBody = readProjectionRequestBody(t, r)
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	// Step 2: Product prefix と異なる Admin audit prefix を設定し、projector が Product namespace を使わないことを確認する。
	projector := mustNewAdminAuditOpenSearchProjector(t, server.URL)
	err := projector.ProjectAdminAuditEvent(context.Background(), adminAuditProjectionTestRecord())
	if err != nil {
		t.Fatalf("project admin audit event: %v", err)
	}
	if requestPath != "/admin-audit-2026.05/_doc/audit-1" {
		t.Fatalf("expected Admin audit index path, got %q", requestPath)
	}
	if strings.Contains(requestPath, "product-domain") || !strings.Contains(requestBody, `"outcome":"succeeded"`) {
		t.Fatalf("unexpected projection request path/body: path=%q body=%s", requestPath, requestBody)
	}
}

// [ADMIN-CONSOLE-BE-S087] OpenSearch indexing failure は warning observer で観測される。
func TestAdminAuditProjectionWarningObserverLogsFailure(t *testing.T) {
	t.Parallel()

	// Step 1: slog の出力先を buffer に差し替え、warning log が audit ID と stable event_type を含むことを検証する。
	var output bytes.Buffer
	observer := &adminAuditProjectionWarningObserver{logger: slog.New(slog.NewTextHandler(&output, &slog.HandlerOptions{Level: slog.LevelWarn}))}

	observer.ObserveAdminAuditProjectionFailure(context.Background(), "audit-1", assertProjectionFailureError{})

	logLine := output.String()
	if !strings.Contains(logLine, "admin audit OpenSearch projection failed") || !strings.Contains(logLine, "admin_audit.opensearch_projection_failed") || !strings.Contains(logLine, "audit-1") {
		t.Fatalf("warning log does not contain projection failure evidence: %s", logLine)
	}
}

func mustNewAdminAuditOpenSearchProjector(t *testing.T, serverURL string) auditapplication.Projector {
	t.Helper()

	// Step 1: constructor の namespace collision validation を通る最小設定を作り、テストごとに OpenSearch endpoint だけ差し替える。
	projector, err := NewAdminAuditOpenSearchProjector(config.OpenSearchConfig{URL: serverURL, AdminAuditIndexPrefix: "admin-audit", ProductIndexPrefix: "product-domain"})
	if err != nil {
		t.Fatalf("new admin audit opensearch projector: %v", err)
	}
	return projector
}

func adminAuditProjectionTestRecord() auditapplication.ProjectionRecord {
	// Step 1: 月次 index 変換が deterministic になるよう UTC の固定日時を使う。
	return auditapplication.ProjectionRecord{AuditID: "audit-1", OperatorID: "operator-1", Action: "accounts:create", TargetType: "account", TargetID: "account-1", RequestID: "req-1", Outcome: "succeeded", OccurredAt: time.Date(2026, 5, 17, 10, 0, 0, 0, time.UTC)}
}

func readProjectionRequestBody(t *testing.T, r *http.Request) string {
	t.Helper()

	// Step 1: test server 側で request body を読み切り、JSON document の主要属性を assertion できる文字列にする。
	buffer := new(bytes.Buffer)
	if _, err := buffer.ReadFrom(r.Body); err != nil {
		t.Fatalf("read projection request body: %v", err)
	}
	return buffer.String()
}

type assertProjectionFailureError struct{}

func (assertProjectionFailureError) Error() string {
	// Step 1: warning observer test で secret を含まない固定 error message を使う。
	return "opensearch down"
}
