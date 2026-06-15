package auth

import (
	"context"

	domain "www-template/packages/backend/internal/domain"
)

// RecoveryDeliveryEventType は復旧配送 flow で発生した内部観測イベントの種類である。
//
// 役割:
//   - Product 復旧 API の外部 response を generic に保ったまま、内部運用では配送原因を追跡できるようにする。
//   - 値は SigNoz Logs / trace event の event_type として利用されるため、安定した文字列として扱う。
type RecoveryDeliveryEventType string

const (
	// RecoveryDeliveryEventRequestAccepted は復旧 request を受け付けた直後のイベントである。
	RecoveryDeliveryEventRequestAccepted RecoveryDeliveryEventType = "recovery.request.accepted"
	// RecoveryDeliveryEventAccountLookupCompleted は account lookup が完了し、存在有無が内部的に確定したイベントである。
	RecoveryDeliveryEventAccountLookupCompleted RecoveryDeliveryEventType = "recovery.account_lookup.completed"
	// RecoveryDeliveryEventSuppressed は列挙防止や throttle により配送を行わず generic accepted に畳んだイベントである。
	RecoveryDeliveryEventSuppressed RecoveryDeliveryEventType = "recovery.delivery.suppressed"
	// RecoveryDeliveryEventTokenIssued は配送用 recovery token を発行し、保存に成功したイベントである。
	RecoveryDeliveryEventTokenIssued RecoveryDeliveryEventType = "recovery.delivery.token_issued"
	// RecoveryDeliveryEventSendStarted は SMTP 配送処理の開始直前イベントである。
	RecoveryDeliveryEventSendStarted RecoveryDeliveryEventType = "recovery.delivery.send_started"
	// RecoveryDeliveryEventSMTPSucceeded は SMTP server が message を受理したイベントである。
	RecoveryDeliveryEventSMTPSucceeded RecoveryDeliveryEventType = "recovery.delivery.smtp_accepted"
	// RecoveryDeliveryEventFailed は復旧配送 flow 内部で失敗が発生したイベントである。
	RecoveryDeliveryEventFailed RecoveryDeliveryEventType = "recovery.delivery.failed"
	// RecoveryDeliveryEventFailureRecordSaved は配送失敗 record を保存できたイベントである。
	RecoveryDeliveryEventFailureRecordSaved RecoveryDeliveryEventType = "recovery.delivery.failure_record_saved"
	// RecoveryDeliveryEventFailureRecordFailed は配送失敗 record の保存自体が失敗したイベントである。
	RecoveryDeliveryEventFailureRecordFailed RecoveryDeliveryEventType = "recovery.delivery.failure_record_failed"
)

// RecoveryDeliveryEvent は復旧配送 flow の内部観測に必要な安全な event DTO である。
//
// 役割:
//   - AuthService から observability 実装へ、request_id / account_id / token_id / error 分類などの追跡情報を渡す。
//   - Email は observer 側で hash/domain 化するための入力であり、そのままログへ出してはならない。
//   - AccountFoundKnown=false の場合、AccountFound は未確定値として無視する。
//
// エラーケース:
//   - この DTO 自体は validation を行わない。observer は空値を省略して安全に記録する。
type RecoveryDeliveryEvent struct {
	EventType         RecoveryDeliveryEventType
	RequestID         string
	RecoveryTokenID   string
	AccountID         string
	Email             string
	Kind              domain.TokenKind
	AccountFound      bool
	AccountFoundKnown bool
	SuppressedReason  string
	DeliveryStage     string
	ErrorClass        string
}

// RecoveryDeliveryObserver は復旧配送 flow の内部イベントを記録するための port である。
//
// 役割:
//   - application 層を SigNoz / slog / OTel の具象実装から分離する。
//   - Product runtime はこの port に実装を注入し、復旧メール未達の原因を request_id 単位で追跡可能にする。
type RecoveryDeliveryObserver interface {
	// ObserveRecoveryDeliveryEvent は復旧配送 flow の内部イベントを記録する。
	// ctx は request trace context を伝搬するために使い、event はログへ出す属性の元データである。
	ObserveRecoveryDeliveryEvent(ctx context.Context, event RecoveryDeliveryEvent)
}

// RecoveryDeliveryRunner は復旧配送 job を request path から分離して実行するための port である。
//
// 役割:
//   - Product runtime では account lookup / token 発行 / SMTP 送信を HTTP response 前に同期実行せず、外部 timing から account existence を推測しにくくする。
//   - テストでは nil にして同期実行へ倒し、既存の deterministic な検証を維持できるようにする。
type RecoveryDeliveryRunner interface {
	// RunRecoveryDelivery は復旧配送 job を実行する。
	// ctx は request trace の相関情報を含み、job へ渡す context は実装側が cancellation 方針を決定する。
	RunRecoveryDelivery(ctx context.Context, job func(context.Context))
}
