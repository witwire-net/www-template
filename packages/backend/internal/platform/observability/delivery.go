package observability

import (
	"context"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

// StartSMTPDeliverySpan は SMTP 配送処理を表す child span を開始する。
//
// 役割:
//   - mailer adapter が OTel 型を直接 import せずに、request trace 配下へ `smtp.send` span を残せるようにする。
//   - span 属性は delivery_stage / error_class などの分類値だけに限定し、宛先、本文、SMTP password、raw error を記録しない。
//
// 引数:
//   - ctx: request span または detached delivery span を含む context。
//
// 戻り値:
//   - context.Context: 開始した `smtp.send` span を含む context。
//   - func(string, string): SMTP 処理終了時に呼ぶ終了関数。第 1 引数は delivery stage、第 2 引数は error class。
//
// 使用例:
//
//	spanCtx, endSpan := observability.StartSMTPDeliverySpan(ctx)
//	defer endSpan("smtp", "none")
func StartSMTPDeliverySpan(ctx context.Context) (context.Context, func(string, string)) {
	// Step 1: platform observability の tracer 名に閉じ、adapter 層が OTel SDK や attribute 型を直接扱わない境界を作る。
	spanCtx, span := otel.Tracer("www-template-mailer").Start(ctx, "smtp.send")
	return spanCtx, func(stage string, class string) {
		// Step 2: 空値を安定分類へ正規化し、SigNoz 上で検索キーが欠落しないようにする。
		normalizedStage := strings.TrimSpace(stage)
		if normalizedStage == "" {
			normalizedStage = "smtp"
		}
		normalizedClass := strings.TrimSpace(class)
		if normalizedClass == "" {
			normalizedClass = "none"
		}

		// Step 3: 属性は分類値だけに限定し、recipient / message / raw SMTP error などの機微情報を API 形状上受け取らない。
		span.SetAttributes(
			attribute.String("messaging.system", "smtp"),
			attribute.String("delivery_stage", normalizedStage),
			attribute.String("error_class", normalizedClass),
		)
		// Step 4: caller が defer で一度だけ呼ぶ前提で span を終了する。
		span.End()
	}
}
