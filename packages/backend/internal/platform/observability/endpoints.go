package observability

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"
)

const (
	defaultOTLPGRPCEndpoint = "localhost:4317"
	otlpEndpointDialTimeout = 2 * time.Second
)

// OTLPEndpoints は backend runtime が利用する OTLP gRPC endpoint を用途ごとに保持する。
//
// 役割:
//   - Traces は trace exporter が送信する OTLP gRPC endpoint である。
//   - Metrics は metric exporter が送信する OTLP gRPC endpoint である。
//   - Logs は log exporter が送信する OTLP gRPC endpoint である。
//   - 各値は host:port 形式を想定し、空の場合は ResolveOTLPEndpoints が共通 default へ正規化する。
//
// エラーケース:
//   - この型自体は検証を行わない。到達性は VerifyOTLPEndpoints で検証する。
//
// 使用例:
//
//	endpoints := ResolveOTLPEndpoints("signoz-otel-collector:4317", "", "")
//	if err := VerifyOTLPEndpoints(ctx, endpoints); err != nil {
//		return err
//	}
type OTLPEndpoints struct {
	// Traces は trace exporter が利用する OTLP gRPC endpoint である。
	Traces string
	// Metrics は metric exporter が利用する OTLP gRPC endpoint である。
	Metrics string
	// Logs は log exporter が利用する OTLP gRPC endpoint である。
	Logs string
}

// ResolveOTLPEndpoints は common/traces/logs の設定値から実際に exporter が使う endpoint を決定する。
//
// 役割:
//   - commonEndpoint は metrics の endpoint として使い、traces/logs の fallback としても使う。
//   - tracesEndpoint が空でなければ traces だけに優先適用する。
//   - logsEndpoint が空でなければ logs だけに優先適用する。
//   - すべて空の場合も OTLP 機能を無効化せず、localhost:4317 を既定値として返す。
//
// 引数:
//   - commonEndpoint: OTEL_EXPORTER_OTLP_ENDPOINT 相当の共通 OTLP gRPC endpoint。
//   - tracesEndpoint: OTEL_EXPORTER_OTLP_TRACES_ENDPOINT 相当の traces 専用 endpoint。
//   - logsEndpoint: OTEL_EXPORTER_OTLP_LOGS_ENDPOINT 相当の logs 専用 endpoint。
//
// 戻り値:
//   - OTLPEndpoints: traces/metrics/logs の各 exporter が利用する正規化済み endpoint。
//
// エラーケース:
//   - この関数は到達性検証を行わず、文字列解決だけを行う。接続不能は VerifyOTLPEndpoints で検出する。
//
// 使用例:
//
//	endpoints := ResolveOTLPEndpoints("signoz-otel-collector:4317", "", "")
func ResolveOTLPEndpoints(commonEndpoint, tracesEndpoint, logsEndpoint string) OTLPEndpoints {
	// Step 1: 共通 endpoint を正規化し、空設定でも OTLP を無効化せず既定の gRPC endpoint へ寄せる。
	common := resolveOTLPEndpoint(commonEndpoint)

	// Step 2: traces 専用 endpoint が設定されていればそれを優先し、未設定なら共通 endpoint を使う。
	traces := strings.TrimSpace(tracesEndpoint)
	if traces == "" {
		traces = common
	}

	// Step 3: logs 専用 endpoint が設定されていればそれを優先し、未設定なら共通 endpoint を使う。
	logs := strings.TrimSpace(logsEndpoint)
	if logs == "" {
		logs = common
	}

	// Step 4: metrics 専用 endpoint は現行 config に存在しないため、共通 endpoint を metrics endpoint として返す。
	return OTLPEndpoints{Traces: traces, Metrics: common, Logs: logs}
}

// VerifyOTLPEndpoints は runtime 起動前に OTLP gRPC endpoint の TCP 到達性を検証する。
//
// 役割:
//   - traces/metrics/logs の exporter が送信に使う endpoint を起動前に検証する。
//   - 同じ endpoint が複数用途で使われる場合は 1 回だけ dial し、起動時間を不要に増やさない。
//   - collector が listen していない状態を runtime 起動前に fail-fast し、実行中の exporter retry ログを発生させない。
//
// 引数:
//   - ctx: dial timeout と呼び出し元の cancellation を伝搬する context。
//   - endpoints: ResolveOTLPEndpoints で解決した traces/metrics/logs endpoint。
//
// 戻り値:
//   - error: いずれかの endpoint が TCP 接続を受けられない場合、用途名と endpoint を含む error を返す。
//
// エラーケース:
//   - collector が未起動、停止中、または 4317 を listen していない場合は error を返す。
//   - ctx がキャンセルまたは deadline 超過した場合も error を返す。
//
// 使用例:
//
//	if err := VerifyOTLPEndpoints(ctx, endpoints); err != nil {
//		return err
//	}
func VerifyOTLPEndpoints(ctx context.Context, endpoints OTLPEndpoints) error {
	// Step 1: endpoint と用途名を同じ順序で扱い、エラー時にどの exporter の前提が壊れているか分かるようにする。
	candidates := []struct {
		name     string
		endpoint string
	}{
		{name: "traces", endpoint: endpoints.Traces},
		{name: "metrics", endpoint: endpoints.Metrics},
		{name: "logs", endpoint: endpoints.Logs},
	}

	// Step 2: 同一 endpoint を複数 exporter が共有する場合、重複 dial を避けるため確認済み endpoint を記録する。
	verified := map[string]struct{}{}
	for _, candidate := range candidates {
		// Step 3: 念のため各 endpoint を正規化し、空文字が渡されても OTLP 既定 endpoint として検証する。
		endpoint := resolveOTLPEndpoint(candidate.endpoint)
		if _, ok := verified[endpoint]; ok {
			continue
		}

		// Step 4: exporter 初期化前に TCP 接続できることを確認し、collector 未起動を明確な起動エラーへ変換する。
		if err := verifyOTLPEndpoint(ctx, endpoint); err != nil {
			return fmt.Errorf("verify otlp %s endpoint %q: %w", candidate.name, endpoint, err)
		}
		verified[endpoint] = struct{}{}
	}

	return nil
}

func resolveOTLPEndpoint(endpoint string) string {
	// Step 1: TOML/env 由来の余分な空白を取り除き、接続先比較と dial の揺れをなくす。
	normalized := strings.TrimSpace(endpoint)
	if normalized == "" {
		// Step 2: 空 endpoint は OTLP 無効化ではなく、既定の gRPC endpoint へ解決して観測機能を維持する。
		return defaultOTLPGRPCEndpoint
	}

	return normalized
}

func verifyOTLPEndpoint(ctx context.Context, endpoint string) error {
	// Step 1: 呼び出し元 context を尊重しつつ、collector 未起動時に長時間ブロックしないため短い dial deadline を設ける。
	dialCtx, cancel := context.WithTimeout(ctx, otlpEndpointDialTimeout)
	defer cancel()

	// Step 2: gRPC の詳細 protocol ではなく TCP listen の有無を確認し、collector receiver が接続を受けられる前提だけを検証する。
	dialer := net.Dialer{}
	conn, err := dialer.DialContext(dialCtx, "tcp", endpoint)
	if err != nil {
		return err
	}

	// Step 3: 検証用の短命接続を即座に閉じ、実際の exporter 接続は OTel SDK に任せる。
	if err := conn.Close(); err != nil {
		return fmt.Errorf("close otlp readiness connection: %w", err)
	}

	return nil
}
