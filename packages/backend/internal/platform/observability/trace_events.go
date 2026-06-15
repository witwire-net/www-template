package observability

import (
	"context"
	"log/slog"
	"strings"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

var traceEventRecorderForTesting traceEventRecorderState

type traceEventRecorderState struct {
	mu      sync.Mutex
	enabled bool
	events  []RecordedTraceEvent
}

// DetachedTraceContext は request context から trace 相関だけを取り出した background context を返す。
//
// 役割:
//   - Gin context など request 完了後に再利用され得る context を goroutine へ持ち出さない。
//   - trace_id は保持し、非同期 job のログと span を元 request と相関できるようにする。
//
// 引数:
//   - ctx: request span を含む可能性がある context。
//
// 戻り値:
//   - context.Context: request cancellation から切り離され、SpanContext だけを保持した background context。
//
// 使用例:
//
//	detached := DetachedTraceContext(requestCtx)
//	jobCtx, endSpan := StartDetachedSpan(detached, "worker", "recovery.delivery")
//	defer endSpan()
func DetachedTraceContext(ctx context.Context) context.Context {
	// Step 1: context に含まれる SpanContext だけを読み取り、request context 自体を保持しない。
	spanContext := trace.SpanContextFromContext(ctx)
	if !spanContext.IsValid() {
		// Step 2: trace が無い呼び出しでも job は実行できるよう background context を返す。
		return context.Background()
	}

	// Step 3: 非同期 span の parent として使える SpanContext だけを新しい context に移す。
	return trace.ContextWithSpanContext(context.Background(), spanContext)
}

// StartDetachedSpan は detached context 上で非同期処理用 span を開始する。
//
// 役割:
//   - request span が終了した後でも、同じ trace に属する child span として非同期処理を記録する。
//   - 戻り値の shutdown 関数は caller が defer で呼び、span を必ず閉じる。
//
// 引数:
//   - ctx: DetachedTraceContext で生成した context、または trace 相関を含む context。
//   - tracerName: OTel tracer 名。空の場合は www-template を使う。
//   - spanName: 開始する span 名。空の場合は background.job を使う。
//
// 戻り値:
//   - context.Context: 開始した span を含む context。
//   - func(): span を終了する関数。caller は必ず defer で呼び出す。
//
// 使用例:
//
//	ctx, endSpan := StartDetachedSpan(detached, "www-template-recovery-delivery", "recovery.delivery")
//	defer endSpan()
func StartDetachedSpan(ctx context.Context, tracerName string, spanName string) (context.Context, func()) {
	// Step 1: tracer/span 名を空にしないことで SigNoz 上の service graph と span 検索を安定させる。
	if strings.TrimSpace(tracerName) == "" {
		tracerName = "www-template"
	}
	if strings.TrimSpace(spanName) == "" {
		spanName = "background.job"
	}

	// Step 2: global tracer provider から span を開始し、InitTracer 済み runtime では OTLP に送信されるようにする。
	spanCtx, span := otel.Tracer(tracerName).Start(ctx, spanName)
	return spanCtx, func() {
		span.End()
	}
}

// AddTraceEvent は現在の span に slog.Attr 由来の属性付き event を追加する。
//
// 役割:
//   - log と trace event の属性名を揃え、SigNoz で request_id / error_class から同じ事象を追えるようにする。
//   - span が recording でない場合は何もせず、通常処理へ副作用を出さない。
//
// 引数:
//   - ctx: event を追加する対象 span を含む context。
//   - name: trace event 名。復旧配送では event_type と同じ安定文字列を渡す。
//   - attrs: trace event 属性へ変換する slog.Attr。secret や本文を含めないこと。
//
// 副作用:
//   - recording span がある場合だけ、その span に event を追加する。
//
// 使用例:
//
//	AddTraceEvent(ctx, "recovery.delivery.failed", []slog.Attr{slog.String("request_id", requestID)})
func AddTraceEvent(ctx context.Context, name string, attrs []slog.Attr) {
	// Step 1: 現在の context に recording span がなければ trace event 追加を省略する。
	span := trace.SpanFromContext(ctx)
	if !span.IsRecording() {
		return
	}

	// Step 2: test recorder が有効な場合は、OTel 型を使わず AddTraceEvent 境界で同じ event を検査できるよう控える。
	recordTraceEventForTesting(name, attrs)

	// Step 3: slog 属性を OTel trace 属性へ変換し、同じ key で検索できるようにする。
	span.AddEvent(name, trace.WithAttributes(slogAttrsToTraceAttributes(attrs)...))
}

// InstallTraceEventRecorderForTesting は AddTraceEvent が受け取った event を OTel 型なしで収集する test recorder を有効化する。
//
// 役割:
//   - adapter test が OTel SDK の exporter/test package を import せず、platform/observability 境界で trace event を検証できるようにする。
//   - recording span がある場合だけ AddTraceEvent が recorder にも書くため、実運用の「span が無ければ no-op」という挙動を保つ。
//   - cleanup 関数で以前の recorder 状態を復元し、他テストの event 記録へ干渉しないようにする。
//
// 戻り値:
//   - func(): test cleanup で呼ぶ復元関数。
//   - func() []RecordedTraceEvent: 現在までに AddTraceEvent が受け取った event の snapshot を返す reader。
//
// 使用例:
//
//	restoreRecorder, events := observability.InstallTraceEventRecorderForTesting()
//	t.Cleanup(restoreRecorder)
func InstallTraceEventRecorderForTesting() (func(), func() []RecordedTraceEvent) {
	// Step 1: 既存 recorder 状態を保存し、cleanup 後にテスト間の global state を完全に戻せるようにする。
	traceEventRecorderForTesting.mu.Lock()
	previousEnabled := traceEventRecorderForTesting.enabled
	previousEvents := append([]RecordedTraceEvent(nil), traceEventRecorderForTesting.events...)
	traceEventRecorderForTesting.enabled = true
	traceEventRecorderForTesting.events = nil
	traceEventRecorderForTesting.mu.Unlock()

	// Step 2: cleanup は保存済み状態へ復元する。events slice は copy 済みなので呼び出し元の read 結果に影響されない。
	restore := func() {
		traceEventRecorderForTesting.mu.Lock()
		traceEventRecorderForTesting.enabled = previousEnabled
		traceEventRecorderForTesting.events = previousEvents
		traceEventRecorderForTesting.mu.Unlock()
	}

	// Step 3: reader は snapshot を返し、呼び出し側が slice を変更しても recorder 内部状態を壊せないようにする。
	readEvents := func() []RecordedTraceEvent {
		traceEventRecorderForTesting.mu.Lock()
		defer traceEventRecorderForTesting.mu.Unlock()
		return append([]RecordedTraceEvent(nil), traceEventRecorderForTesting.events...)
	}

	return restore, readEvents
}

func recordTraceEventForTesting(name string, attrs []slog.Attr) {
	// Step 1: recorder 無効時は lock 内で即 return し、production/runtime の通常 path に event 保持を追加しない。
	traceEventRecorderForTesting.mu.Lock()
	defer traceEventRecorderForTesting.mu.Unlock()
	if !traceEventRecorderForTesting.enabled {
		return
	}

	// Step 2: slog.Attr を安全な文字列表現へ解決し、adapter tests が OTel 型なしで属性検査できる形にする。
	converted := make(map[string]string, len(attrs))
	for _, attr := range attrs {
		converted[attr.Key] = attr.Value.Resolve().String()
	}
	traceEventRecorderForTesting.events = append(traceEventRecorderForTesting.events, RecordedTraceEvent{Name: name, Attributes: converted})
}

func slogAttrsToTraceAttributes(attrs []slog.Attr) []attribute.KeyValue {
	// Step 1: 入力 attr 数に合わせて容量を確保し、復旧配送 event の hot path で余計な再割当を避ける。
	otelAttrs := make([]attribute.KeyValue, 0, len(attrs))
	for _, attr := range attrs {
		// Step 2: attr.Value.Resolve により LogValuer を展開し、実際に記録される値へ正規化する。
		value := attr.Value.Resolve()
		switch value.Kind() {
		case slog.KindString:
			otelAttrs = append(otelAttrs, attribute.String(attr.Key, value.String()))
		case slog.KindBool:
			otelAttrs = append(otelAttrs, attribute.Bool(attr.Key, value.Bool()))
		case slog.KindInt64:
			otelAttrs = append(otelAttrs, attribute.Int64(attr.Key, value.Int64()))
		case slog.KindUint64:
			otelAttrs = append(otelAttrs, attribute.String(attr.Key, value.String()))
		case slog.KindFloat64:
			otelAttrs = append(otelAttrs, attribute.Float64(attr.Key, value.Float64()))
		case slog.KindDuration:
			otelAttrs = append(otelAttrs, attribute.String(attr.Key, value.Duration().String()))
		case slog.KindTime:
			otelAttrs = append(otelAttrs, attribute.String(attr.Key, value.Time().Format(time.RFC3339Nano)))
		default:
			otelAttrs = append(otelAttrs, attribute.String(attr.Key, value.String()))
		}
	}
	return otelAttrs
}
