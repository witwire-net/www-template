package mailer

import "strings"

// deliveryError は mailer adapter 内で発生した配送失敗を、安全な stage/class に分類する内部 error である。
// raw error は Unwrap で保持するが、AuthService 側のログには DeliveryErrorStage / DeliveryErrorClass のみを使う。
type deliveryError struct {
	stage string
	class string
	err   error
}

// newDeliveryError は SMTP や template 失敗を分類済み error として生成する。
// stage は失敗した処理段階、class は SigNoz に出す安全な分類、err は原因追跡用の元 error である。
func newDeliveryError(stage string, class string, err error) error {
	return &deliveryError{stage: stage, class: class, err: err}
}

// Error は安全な分類名だけを返す。
// raw SMTP error、宛先、メール本文、password などが error 文字列経由でログや一時 record に混入しないよう、元 error の文字列は返さない。
func (e *deliveryError) Error() string {
	// Step 1: class が空の異常な deliveryError でも、空 error 文字列ではなく安全な汎用分類へ倒す。
	class := strings.TrimSpace(e.class)
	if class == "" {
		return "delivery_failed"
	}

	// Step 2: 元 error は Unwrap で辿れるよう保持しつつ、Error 文字列には分類だけを出す。
	return class
}

// Unwrap は errors.As / errors.Is が元 error を辿れるようにする。
func (e *deliveryError) Unwrap() error {
	return e.err
}

// DeliveryErrorStage は失敗した処理段階を返す。
// AuthService はこの値を delivery_stage として記録し、SMTP のどこで止まったかを追跡する。
func (e *deliveryError) DeliveryErrorStage() string {
	return e.stage
}

// DeliveryErrorClass はログへ出してよい安定エラー分類を返す。
// raw SMTP error やメール本文ではなく、この分類だけを SigNoz の error_class に使う。
func (e *deliveryError) DeliveryErrorClass() string {
	return e.class
}
