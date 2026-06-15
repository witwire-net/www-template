package observability

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

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

	// Step 2: slog 属性を OTel trace 属性へ変換し、同じ key で検索できるようにする。
	span.AddEvent(name, trace.WithAttributes(slogAttrsToTraceAttributes(attrs)...))
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
