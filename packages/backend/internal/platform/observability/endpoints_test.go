package observability

import (
	"context"
	"net"
	"strings"
	"testing"
)

func TestResolveOTLPEndpointsKeepsOTLPEnabledWithDefaultEndpoint(t *testing.T) {
	t.Parallel()

	// Step 1: endpoint がすべて空の設定を解決し、空設定が no-op ではなく既定 OTLP gRPC endpoint へ解決されることを確認する。
	endpoints := ResolveOTLPEndpoints("", "", "")

	// Step 2: traces/metrics/logs がすべて同じ既定 endpoint を使い、ローカル観測機能を無効化しないことを固定する。
	if endpoints.Traces != defaultOTLPGRPCEndpoint || endpoints.Metrics != defaultOTLPGRPCEndpoint || endpoints.Logs != defaultOTLPGRPCEndpoint {
		t.Fatalf("expected default OTLP endpoint for every signal, got %#v", endpoints)
	}
}

func TestResolveOTLPEndpointsUsesSignalSpecificOverrides(t *testing.T) {
	t.Parallel()

	// Step 1: common endpoint と signal 専用 endpoint を混在させ、traces/logs だけが専用値へ向く設定を作る。
	endpoints := ResolveOTLPEndpoints(
		"signoz-otel-collector:4317",
		"trace-collector:4317",
		"log-collector:4317",
	)

	// Step 2: metrics は現行 config の共通 endpoint を使い、traces/logs は個別 endpoint を優先することを確認する。
	if endpoints.Traces != "trace-collector:4317" || endpoints.Metrics != "signoz-otel-collector:4317" || endpoints.Logs != "log-collector:4317" {
		t.Fatalf("expected signal-specific endpoint resolution, got %#v", endpoints)
	}
}

func TestVerifyOTLPEndpointsAcceptsListeningEndpoint(t *testing.T) {
	t.Parallel()

	// Step 1: テスト専用の TCP listener を起動し、OTLP collector が gRPC port を listen している状態を再現する。
	listener := listenLocalTCP(t)
	defer func() {
		_ = listener.Close()
	}()

	// Step 2: traces/metrics/logs が同じ listener を共有する設定で、重複 endpoint でも検証が成功することを確認する。
	endpoint := listener.Addr().String()
	if err := VerifyOTLPEndpoints(context.Background(), OTLPEndpoints{Traces: endpoint, Metrics: endpoint, Logs: endpoint}); err != nil {
		t.Fatalf("expected listening OTLP endpoint to pass verification: %v", err)
	}
}

func TestVerifyOTLPEndpointsRejectsClosedEndpoint(t *testing.T) {
	t.Parallel()

	// Step 1: 一度確保した TCP address を閉じ、collector が gRPC port を listen していない状態を作る。
	listener := listenLocalTCP(t)
	endpoint := listener.Addr().String()
	if err := listener.Close(); err != nil {
		t.Fatalf("close test listener: %v", err)
	}

	// Step 2: 閉じた endpoint へ到達確認し、runtime が exporter retry に進む前に起動エラーへできることを確認する。
	err := VerifyOTLPEndpoints(context.Background(), OTLPEndpoints{Traces: endpoint, Metrics: endpoint, Logs: endpoint})
	if err == nil {
		t.Fatal("expected closed OTLP endpoint to fail verification")
	}

	// Step 3: error message に signal 名と endpoint が含まれ、利用者が collector の問題として特定できることを確認する。
	message := err.Error()
	if !strings.Contains(message, "traces") || !strings.Contains(message, endpoint) {
		t.Fatalf("expected endpoint verification error to include signal and endpoint, got %q", message)
	}
}

func listenLocalTCP(t *testing.T) net.Listener {
	t.Helper()

	// Step 1: OS に空き port を割り当てさせ、他テストや開発サーバーと衝突しない TCP listener を作る。
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen on local tcp port: %v", err)
	}

	return listener
}
