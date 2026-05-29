package domain

import (
	"errors"
	"regexp"
	"strings"
	"time"
)

var (
	// ErrInvalidAdminAuditOutcome は AdminAuditEvent の outcome が未定義の場合に返すエラーである。
	// 永続化から復元した値が未知の場合、監査結果を誤って成功扱いにしないため fail-closed にする。
	ErrInvalidAdminAuditOutcome = errors.New("invalid admin audit outcome")

	// ErrAdminAuditAlreadyCompleted は AdminAuditEvent を二重に完了しようとした場合に返すエラーである。
	// 監査結果は mutation の最終結果を一度だけ記録するため、再完了を拒否する。
	ErrAdminAuditAlreadyCompleted = errors.New("admin audit event already completed")

	// ErrInvalidAdminAuditCompletedAt は completedAt がゼロ値の場合に返すエラーである。
	// 完了時刻がない audit outcome は時系列監査に使えないため、domain 層で拒否する。
	ErrInvalidAdminAuditCompletedAt = errors.New("invalid admin audit completed timestamp")

	// ErrInvalidAdminAuditStableErrorCode は failed outcome の stable error code が空または不正な場合に返すエラーである。
	// user-facing message ではなく安定分類を保存し、集計・検索・再試行判定を壊さないために検証する。
	ErrInvalidAdminAuditStableErrorCode = errors.New("invalid admin audit stable error code")
)

var stableErrorCodePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_:-]*$`)

// AdminAuditOutcome は Admin mutation audit の完了状態を表す値オブジェクトである。
//
// pending は intent 記録直後、succeeded/failed は mutation 完了後の最終状態である。
// 最終状態から別の最終状態へ遷移することは AdminAuditEvent の method が拒否する。
type AdminAuditOutcome string

const (
	// AdminAuditOutcomePending は mutation intent が記録され、まだ完了していない状態である。
	AdminAuditOutcomePending AdminAuditOutcome = "pending"

	// AdminAuditOutcomeSucceeded は mutation が成功し、完了時刻が記録された最終状態である。
	AdminAuditOutcomeSucceeded AdminAuditOutcome = "succeeded"

	// AdminAuditOutcomeFailed は mutation が失敗し、stable error code と完了時刻が記録された最終状態である。
	AdminAuditOutcomeFailed AdminAuditOutcome = "failed"
)

// Validate は AdminAuditOutcome が既知の値であることを検証する。
//
// 不明な値は ErrInvalidAdminAuditOutcome を返す。
// この method は永続化復元時やテスト fixture の安全確認に利用でき、副作用はない。
func (o AdminAuditOutcome) Validate() error {
	// Step 1: 明示的に許可した outcome だけを受け付ける。
	switch o {
	case AdminAuditOutcomePending, AdminAuditOutcomeSucceeded, AdminAuditOutcomeFailed:
		return nil
	default:
		return ErrInvalidAdminAuditOutcome
	}
}

// StableErrorCode は failed audit outcome に保存する安定したエラー分類である。
//
// 文言や locale に依存する message ではなく、duplicate_email や permission_denied のような
// 機械処理しやすい code を保存するための値オブジェクトとして扱う。
type StableErrorCode string

// NewStableErrorCode は raw code を検証し、StableErrorCode を返す。
//
// raw は trim 後に lowercase へ正規化され、英小文字・数字・underscore・colon・hyphen のみを許可する。
// 空文字や空白を含む値は ErrInvalidAdminAuditStableErrorCode を返す。
func NewStableErrorCode(raw string) (StableErrorCode, error) {
	// Step 1: stable code を監査検索で揺れない lowercase 表現へ正規化する。
	canonical := strings.ToLower(strings.TrimSpace(raw))

	// Step 2: 空文字および許可外文字を拒否し、error message や動的値の混入を防ぐ。
	if !stableErrorCodePattern.MatchString(canonical) {
		return "", ErrInvalidAdminAuditStableErrorCode
	}

	return StableErrorCode(canonical), nil
}

// String は StableErrorCode を永続化・監査検索向けの canonical 文字列へ変換する。
//
// 戻り値は NewStableErrorCode で検証済みの安定 code であり、副作用はない。
func (c StableErrorCode) String() string { return string(c) }

// AdminAuditEvent は Admin mutation の intent と outcome を表す domain object である。
//
// 作成時は pending で始まり、MarkSucceeded または MarkFailed のどちらか一度だけで完了する。
// failed の場合は stable error code、succeeded/failed のどちらも completed timestamp を必須とする。
type AdminAuditEvent struct {
	outcome         AdminAuditOutcome
	stableErrorCode StableErrorCode
	completedAt     *time.Time
}

// NewAdminAuditEvent は pending outcome の AdminAuditEvent を生成する。
//
// Admin account creation use case は mutation 前にこの intent を保存し、mutation 後に
// MarkSucceeded または MarkFailed で一度だけ完了させる。引数はなく、副作用もない。
func NewAdminAuditEvent() AdminAuditEvent {
	// Step 1: intent は必ず pending として開始し、完了情報は空にする。
	return AdminAuditEvent{outcome: AdminAuditOutcomePending}
}

// ReconstituteAdminAuditEvent は永続化済みの outcome 情報から AdminAuditEvent を復元する。
//
// pending は completedAt/errorCode を持たない状態だけを受け付ける。
// succeeded/failed は completedAt を必須とし、failed では stable error code も必須とする。
func ReconstituteAdminAuditEvent(
	outcome AdminAuditOutcome,
	stableErrorCode StableErrorCode,
	completedAt *time.Time,
) (AdminAuditEvent, error) {
	// Step 1: outcome 自体が既知値かを先に検証する。
	if err := outcome.Validate(); err != nil {
		return AdminAuditEvent{}, err
	}

	// Step 2: pending は未完了 intent なので、完了情報が混ざった復元を拒否する。
	if outcome == AdminAuditOutcomePending {
		if stableErrorCode != "" || completedAt != nil {
			return AdminAuditEvent{}, ErrInvalidAdminAuditOutcome
		}
		return AdminAuditEvent{outcome: AdminAuditOutcomePending}, nil
	}

	// Step 3: 完了済み outcome は completedAt を必須にし、監査時系列を保証する。
	if completedAt == nil || completedAt.IsZero() {
		return AdminAuditEvent{}, ErrInvalidAdminAuditCompletedAt
	}

	// Step 4: succeeded は error code を持たない状態だけを受け付ける。
	completedAtCopy := completedAt.UTC()
	if outcome == AdminAuditOutcomeSucceeded {
		if stableErrorCode != "" {
			return AdminAuditEvent{}, ErrInvalidAdminAuditStableErrorCode
		}
		return AdminAuditEvent{outcome: outcome, completedAt: &completedAtCopy}, nil
	}

	// Step 5: failed は stable error code を必須にし、動的 error message の保存を防ぐ。
	validatedCode, err := NewStableErrorCode(stableErrorCode.String())
	if err != nil {
		return AdminAuditEvent{}, err
	}
	return AdminAuditEvent{outcome: outcome, stableErrorCode: validatedCode, completedAt: &completedAtCopy}, nil
}

// MarkSucceeded は pending audit intent を succeeded outcome に遷移させる。
//
// completedAt はゼロ値不可で、UTC に正規化して保持する。
// 既に succeeded/failed の event に対して呼ぶと ErrAdminAuditAlreadyCompleted を返す。
func (e AdminAuditEvent) MarkSucceeded(completedAt time.Time) (AdminAuditEvent, error) {
	// Step 1: audit outcome は一度だけ完了できるため、pending 以外は拒否する。
	if err := e.ensurePending(); err != nil {
		return AdminAuditEvent{}, err
	}

	// Step 2: completedAt がない成功 outcome は監査不能なので拒否する。
	if completedAt.IsZero() {
		return AdminAuditEvent{}, ErrInvalidAdminAuditCompletedAt
	}

	// Step 3: 完了時刻を UTC にそろえ、succeeded の最終状態として返す。
	completedAtCopy := completedAt.UTC()
	e.outcome = AdminAuditOutcomeSucceeded
	e.stableErrorCode = ""
	e.completedAt = &completedAtCopy
	return e, nil
}

// MarkFailed は pending audit intent を failed outcome に遷移させる。
//
// stableErrorCode は NewStableErrorCode と同じ規則で検証され、completedAt はゼロ値不可である。
// 既に完了済みの event に対して呼ぶと ErrAdminAuditAlreadyCompleted を返す。
func (e AdminAuditEvent) MarkFailed(stableErrorCode StableErrorCode, completedAt time.Time) (AdminAuditEvent, error) {
	// Step 1: audit outcome は一度だけ完了できるため、pending 以外は拒否する。
	if err := e.ensurePending(); err != nil {
		return AdminAuditEvent{}, err
	}

	// Step 2: failed outcome は stable error code を必須にする。
	validatedCode, err := NewStableErrorCode(stableErrorCode.String())
	if err != nil {
		return AdminAuditEvent{}, err
	}

	// Step 3: completedAt がない失敗 outcome は監査不能なので拒否する。
	if completedAt.IsZero() {
		return AdminAuditEvent{}, ErrInvalidAdminAuditCompletedAt
	}

	// Step 4: 完了時刻を UTC にそろえ、failed の最終状態として返す。
	completedAtCopy := completedAt.UTC()
	e.outcome = AdminAuditOutcomeFailed
	e.stableErrorCode = validatedCode
	e.completedAt = &completedAtCopy
	return e, nil
}

// Outcome は AdminAuditEvent の現在 outcome を返す。
//
// pending/succeeded/failed のいずれかであり、mutation outcome 判定に使用できる。
func (e AdminAuditEvent) Outcome() AdminAuditOutcome { return e.outcome }

// StableErrorCode は failed outcome に保存された stable error code を返す。
//
// pending/succeeded の場合は空値を返す。戻り値は user-facing message ではない。
func (e AdminAuditEvent) StableErrorCode() StableErrorCode { return e.stableErrorCode }

// CompletedAt は AdminAuditEvent の完了時刻を返す。
//
// pending の場合は nil を返す。完了済みの場合は内部 pointer を直接公開せず、copy の pointer を返す。
func (e AdminAuditEvent) CompletedAt() *time.Time {
	// Step 1: 未完了の場合、完了時刻が存在しないことを nil で表す。
	if e.completedAt == nil {
		return nil
	}

	// Step 2: 呼び出し側が内部状態を書き換えられないよう copy を返す。
	completedAtCopy := *e.completedAt
	return &completedAtCopy
}

// ensurePending は AdminAuditEvent が pending outcome であることを確認する。
//
// pending 以外の場合は ErrAdminAuditAlreadyCompleted を返し、二重完了を防ぐ。
func (e AdminAuditEvent) ensurePending() error {
	// Step 1: pending のみ MarkSucceeded/MarkFailed の開始状態として許可する。
	if e.outcome != AdminAuditOutcomePending {
		return ErrAdminAuditAlreadyCompleted
	}

	return nil
}
