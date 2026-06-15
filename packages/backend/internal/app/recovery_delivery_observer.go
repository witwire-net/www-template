package app

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"log/slog"
	"strings"

	productauth "www-template/packages/backend/internal/application/auth"
	platformobservability "www-template/packages/backend/internal/platform/observability"
)

// slogRecoveryDeliveryObserver は復旧配送イベントを Product runtime の標準 logger と OTel trace へ橋渡しする adapter である。
// application/auth の port 実装としてだけ使い、HTTP response やメール配送そのものには副作用を与えない。
type slogRecoveryDeliveryObserver struct {
	logger       *slog.Logger
	emailHashKey string
}

// asyncRecoveryDeliveryRunner は復旧配送 job を HTTP request path の外で実行する runner である。
// logger は goroutine panic を SigNoz Logs に残すために使い、配送 job の通常イベントは observer が記録する。
type asyncRecoveryDeliveryRunner struct {
	logger *slog.Logger
}

// newSlogRecoveryDeliveryObserver は復旧配送イベントを slog と OTel trace event に流す observer を生成する。
// emailHashKey はメールアドレスを HMAC 化する鍵であり、raw email をログに出さず同一アドレスの相関だけ可能にする。
func newSlogRecoveryDeliveryObserver(logger *slog.Logger, emailHashKey string) *slogRecoveryDeliveryObserver {
	return &slogRecoveryDeliveryObserver{logger: logger, emailHashKey: emailHashKey}
}

// newAsyncRecoveryDeliveryRunner は Product runtime 用の非同期復旧配送 runner を生成する。
// logger は復旧配送 goroutine の panic を観測するために保持する。
func newAsyncRecoveryDeliveryRunner(logger *slog.Logger) asyncRecoveryDeliveryRunner {
	return asyncRecoveryDeliveryRunner{logger: logger}
}

// RunRecoveryDelivery は復旧配送 job を goroutine で実行し、HTTP response timing から SMTP 経路を切り離す。
// request context は再利用され得るため、trace 相関だけを抽出した detached context へ変換してから job に渡す。
func (r asyncRecoveryDeliveryRunner) RunRecoveryDelivery(ctx context.Context, job func(context.Context)) {
	deliveryCtx := platformobservability.DetachedTraceContext(ctx)
	go func() {
		jobCtx, endSpan := platformobservability.StartDetachedSpan(deliveryCtx, "www-template-recovery-delivery", "recovery.delivery")
		defer endSpan()
		defer func() {
			if recovered := recover(); recovered != nil {
				logger := r.logger
				if logger == nil {
					logger = slog.Default()
				}
				// panic 値は復旧 URL などの機密値を含む可能性があるため、値そのものではなく型だけを記録する。
				attrs := []slog.Attr{
					slog.String("event_type", "recovery.delivery.runner_panic"),
					slog.String("panic_type", fmt.Sprintf("%T", recovered)),
				}
				logger.LogAttrs(deliveryCtx, slog.LevelError, "recovery delivery runner panicked", attrs...)
			}
		}()
		job(jobCtx)
	}()
}

// ObserveRecoveryDeliveryEvent は復旧配送イベントを構造化ログと trace event の両方へ記録する。
// 入力 event に raw email が含まれていても、recoveryDeliveryLogAttrs で hash/domain 化してから出力する。
func (o *slogRecoveryDeliveryObserver) ObserveRecoveryDeliveryEvent(ctx context.Context, event productauth.RecoveryDeliveryEvent) {
	logger := o.logger
	if logger == nil {
		logger = slog.Default()
	}
	attrs := o.recoveryDeliveryLogAttrs(event)
	logger.LogAttrs(ctx, recoveryDeliveryLogLevel(event), "recovery delivery event", attrs...)
	o.recordRecoveryDeliveryTraceEvent(ctx, event)
}

// recoveryDeliveryLogAttrs は SigNoz Logs に送る属性を組み立てる。
// recovery URL、token secret、メール本文は event DTO に持たせず、email も hash/domain のみに変換する。
func (o *slogRecoveryDeliveryObserver) recoveryDeliveryLogAttrs(event productauth.RecoveryDeliveryEvent) []slog.Attr {
	attrs := []slog.Attr{slog.String("event_type", string(event.EventType))}
	attrs = appendStringAttr(attrs, "request_id", event.RequestID)
	attrs = appendStringAttr(attrs, "account_id", event.AccountID)
	attrs = appendStringAttr(attrs, "recovery_token_id", event.RecoveryTokenID)
	attrs = appendStringAttr(attrs, "kind", string(event.Kind))
	attrs = appendStringAttr(attrs, "suppressed_reason", event.SuppressedReason)
	attrs = appendStringAttr(attrs, "delivery_stage", event.DeliveryStage)
	attrs = appendStringAttr(attrs, "error_class", event.ErrorClass)
	if event.AccountFoundKnown {
		attrs = append(attrs, slog.Bool("account_found", event.AccountFound))
	}
	if emailHash := recoveryEmailLogHash(event.Email, o.emailHashKey); emailHash != "" {
		attrs = append(attrs, slog.String("email_hash", emailHash))
	}
	if emailDomain := recoveryEmailLogDomain(event.Email); emailDomain != "" {
		attrs = append(attrs, slog.String("email_domain", emailDomain))
	}
	return attrs
}

// recordRecoveryDeliveryTraceEvent は現在の配送 span に復旧配送イベントを追加する。
// 非同期 runner が作成した child span がある場合はその span に、同期テストでは request span にイベントが乗る。
func (o *slogRecoveryDeliveryObserver) recordRecoveryDeliveryTraceEvent(ctx context.Context, event productauth.RecoveryDeliveryEvent) {
	platformobservability.AddTraceEvent(ctx, string(event.EventType), o.recoveryDeliveryLogAttrs(event))
}

// recoveryDeliveryLogLevel は配送失敗だけを Error、それ以外を Info として分類する。
// policy suppression は外部 response として正常な generic accepted のため Info にする。
func recoveryDeliveryLogLevel(event productauth.RecoveryDeliveryEvent) slog.Level {
	switch event.EventType {
	case productauth.RecoveryDeliveryEventFailed, productauth.RecoveryDeliveryEventFailureRecordFailed:
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// recoveryEmailLogHash は raw email を HMAC-SHA256 で不可逆化した相関 ID に変換する。
// 同じ email の再試行は追えるが、ログだけからメールアドレスを復元しにくくする。
func recoveryEmailLogHash(email string, key string) string {
	normalized := strings.ToLower(strings.TrimSpace(email))
	if normalized == "" {
		return ""
	}
	mac := hmac.New(sha256.New, []byte(key))
	_, _ = mac.Write([]byte(normalized))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

// recoveryEmailLogDomain は配送先ドメインだけを小文字で返す。
// SMTP provider 側の問題切り分けに使うが、local-part はログに出さない。
func recoveryEmailLogDomain(email string) string {
	normalized := strings.ToLower(strings.TrimSpace(email))
	_, domain, ok := strings.Cut(normalized, "@")
	if !ok {
		return ""
	}
	return strings.TrimSpace(domain)
}

// appendStringAttr は空文字属性をログから省き、不要な空値で検索性を落とさないための helper である。
func appendStringAttr(attrs []slog.Attr, key string, value string) []slog.Attr {
	if strings.TrimSpace(value) == "" {
		return attrs
	}
	return append(attrs, slog.String(key, value))
}
