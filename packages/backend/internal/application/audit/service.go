package audit

import (
	"context"
	"errors"
	"time"

	domain "www-template/packages/backend/internal/domain"
)

var (
	// ErrAuditInternal は Admin audit 境界で repository、clock、または復元済み state が不正な場合に返す application error である。
	// adapter は内部詳細や永続化 error を露出せず、fail-close な 5xx 系応答へ変換する。
	ErrAuditInternal = errors.New("admin audit internal")

	// ErrAuditBadRequest は audit outcome を domain rule で完了できない入力または状態に対して返す application error である。
	// 二重完了、不正 outcome、不正 stable error code などを handler に domain 型を露出せず伝えるために使う。
	ErrAuditBadRequest = errors.New("admin audit bad request")
)

// Repository は Admin mutation audit の intent 記録と outcome 完了を永続化 adapter へ委譲する port である。
//
// 役割:
//   - application use case が concrete DB/GORM/generated 型に依存しないよう、primitive DTO だけを境界に置く。
//   - RecordAuditIntent は mutation 前に pending intent を永続化し、失敗時に後続 mutation を開始させない。
//   - CompleteAudit は domain.AdminAuditEvent で確定した succeeded/failed outcome だけを保存する。
//
// 引数:
//   - ctx: request deadline と cancellation を repository adapter へ伝播する。
//   - record: intent または completion の application DTO。domain entity や adapter 型を含めない。
//
// 戻り値:
//   - Record: 永続化後の audit snapshot。audit ID や outcome を後続処理で参照する。
//   - error: repository が intent または outcome を保存できない場合に返す。use case は ErrAuditInternal に写像する。
type Repository interface {
	RecordAuditIntent(ctx context.Context, record IntentRecord) (Record, error)
	FindAudit(ctx context.Context, auditID string) (Record, error)
	CompleteAudit(ctx context.Context, record CompletionRecord) (Record, error)
}

// Projector は完了済み Admin audit event を検索基盤へ投影する port である。
//
// 役割:
//   - Go backend から Admin audit event を OpenSearch などの read/search index へ送る。
//   - projection failure は mutation の commit 後に扱われるため、呼び出し元 use case は成功済み mutation を取り消さない。
//   - 実装は Admin audit namespace だけを使い、Product domain namespace へ書き込んではならない。
type Projector interface {
	ProjectAdminAuditEvent(ctx context.Context, record ProjectionRecord) error
}

// ProjectionFailureObserver は audit projection の失敗を観測可能にする port である。
//
// 役割:
//   - application 層から logger/metric 具象実装を直接 import せず、warning log や retry marker への記録を外側へ委譲する。
//   - projection failure の error を受け取るが、その error を mutation response の失敗には変換しない。
//   - constructor で必須依存として扱い、failure が沈黙しない構成を保証する。
type ProjectionFailureObserver interface {
	ObserveAdminAuditProjectionFailure(ctx context.Context, auditID string, err error)
}

// AuditService は Admin mutation の intent 作成と success/failed outcome 完了を集約する application use case である。
//
// 役割:
//   - handler や account creation use case が AdminAuditEvent の transition rule を重複実装しないようにする。
//   - mutation 前の pending intent 記録と、mutation 後の succeeded/failed 完了を同じ境界にまとめる。
//   - Product audit や Product auth application を import せず、Admin 専用 audit path を維持する。
//
// 使用例:
//
//	audit, err := NewAuditService(repository, clock)
//	if err != nil {
//		return err
//	}
//	intent, err := audit.RecordMutationIntent(ctx, input)
//	if err != nil {
//		return err
//	}
//	_, err = audit.CompleteMutationSucceeded(ctx, CompletionInput{AuditID: intent.AuditID})
type AuditService struct {
	audits Repository
	clock  func() time.Time
}

type adminAuditCompletionKind string

const (
	adminAuditCompletionSucceeded adminAuditCompletionKind = "succeeded"
	adminAuditCompletionFailed    adminAuditCompletionKind = "failed"
)

// IntentInput は mutation 開始前に記録する audit intent の入力 DTO である。
//
// 役割:
//   - operator、action、target、request correlation を application 境界の primitive として受け取る。
//   - handler や後続 account creation use case が domain.AdminAuditEvent を直接組み立てないようにする。
//   - DetailsJSON は adapter が検証済み JSON 文字列を渡すための将来拡張フィールドであり、空文字を許容する。
type IntentInput struct {
	OperatorID  string
	Action      string
	TargetType  string
	TargetID    string
	RequestID   string
	DetailsJSON string
}

// IntentRecord は repository へ渡す pending audit intent の保存 DTO である。
//
// 役割:
//   - domain.NewAdminAuditEvent が決めた pending outcome を primitive 文字列として永続化境界へ渡す。
//   - OccurredAt は注入 clock 由来の UTC 時刻で、mutation intent の発生時刻として保存される。
//   - StableErrorCode と CompletedAt は intent 時点では空にし、outcome 完了時だけ設定する。
type IntentRecord struct {
	OperatorID      string
	Action          string
	TargetType      string
	TargetID        string
	RequestID       string
	DetailsJSON     string
	Outcome         string
	StableErrorCode string
	OccurredAt      time.Time
	CompletedAt     *time.Time
}

// CompletionInput は succeeded outcome へ完了する audit event の入力 DTO である。
//
// AuditID は RecordMutationIntent が返した永続化済み audit event の識別子である。
// 戻り値は repository が更新後に返す Record で、失敗時は application error を返す。
type CompletionInput struct {
	AuditID string
}

// FailureInput は failed outcome へ完了する audit event の入力 DTO である。
//
// AuditID は pending intent の識別子であり、StableErrorCode は domain.NewStableErrorCode と MarkFailed により検証・正規化される。
// user-facing message や動的 error text は StableErrorCode に入れず、duplicate_email などの安定分類だけを渡す。
type FailureInput struct {
	AuditID         string
	StableErrorCode string
}

// CompletionRecord は repository へ渡す completed audit outcome の保存 DTO である。
//
// 役割:
//   - domain.AdminAuditEvent の MarkSucceeded/MarkFailed が返した最終 outcome だけを保存境界へ渡す。
//   - succeeded では StableErrorCode が空、failed では canonical stable error code が入る。
//   - CompletedAt は必ず non-nil で、domain method が UTC 正規化した完了時刻を保持する。
type CompletionRecord struct {
	AuditID         string
	Outcome         string
	StableErrorCode string
	CompletedAt     time.Time
}

// Record は repository から application use case へ返す audit snapshot DTO である。
//
// 役割:
//   - audit ID と現在 outcome を primitive として保持し、domain entity を public API に露出しない。
//   - FindAudit の戻り値として ReconstituteAdminAuditEvent に必要な outcome/error/completedAt を提供する。
//   - account creation use case はこの DTO の AuditID を mutation correlation に利用できる。
type Record struct {
	AuditID         string
	OperatorID      string
	Action          string
	TargetType      string
	TargetID        string
	RequestID       string
	DetailsJSON     string
	Outcome         string
	StableErrorCode string
	OccurredAt      time.Time
	CompletedAt     *time.Time
}

// ProjectionRecord は Admin audit event を検索 index へ投影するための application DTO である。
//
// 役割:
//   - repository や HTTP/generated 型を含めず、OpenSearch document に必要な primitive だけを保持する。
//   - AuditID/OperatorID/Action/Target/Outcome/OccurredAt は Admin audit の検索・調査に必要な安定属性である。
//   - DetailsJSON は保存済み JSON 文字列をそのまま渡し、projector 実装が必要に応じて document 化する。
type ProjectionRecord struct {
	AuditID         string
	OperatorID      string
	Action          string
	TargetType      string
	TargetID        string
	RequestID       string
	DetailsJSON     string
	Outcome         string
	StableErrorCode string
	OccurredAt      time.Time
	CompletedAt     *time.Time
}

// NewAuditService は Admin audit use case を生成する。
//
// 引数:
//   - audits: audit intent/outcome を保存する repository port。nil は fail-close のため拒否する。
//   - clock: intent/outcome timestamp を生成する注入 clock。time.Now を application から直接呼ばないため必須にする。
//
// 戻り値:
//   - *AuditService: 検証済み依存を保持する use case service。
//   - error: 必須依存が欠ける場合は ErrAuditInternal。
func NewAuditService(audits Repository, clock func() time.Time) (*AuditService, error) {
	// Step 1: 永続化 port または clock が欠けると未監査 mutation や非決定的時刻になり得るため、構築時に拒否する。
	if audits == nil || clock == nil {
		return nil, ErrAuditInternal
	}

	// Step 2: 検証済み依存だけを保持し、以後の use case method が同じ Admin audit 境界を共有する。
	return &AuditService{audits: audits, clock: clock}, nil
}

// RecordMutationIntent は Admin mutation 開始前に pending audit intent を記録する。
//
// ctx は repository へ deadline/cancellation を伝播する。
// input は operator/action/target/request correlation の primitive DTO で、Product audit 情報を混入させない。
// 戻り値は repository が保存した Record であり、保存失敗時は ErrAuditInternal を返して mutation 開始を止める。
func (s *AuditService) RecordMutationIntent(ctx context.Context, input IntentInput) (Record, error) {
	// Step 1: 呼び出し元 context が既に中断済みなら、永続化と後続 mutation を開始しない。
	if err := ctx.Err(); err != nil {
		return Record{}, ErrAuditInternal
	}

	// Step 2: intent の初期 outcome は concrete domain object に委譲し、handler/application 側で pending 文字列を手組みしない。
	event := domain.NewAdminAuditEvent()

	// Step 3: 注入 clock の時刻を UTC にそろえ、監査時系列の保存 DTO を組み立てる。
	intent := IntentRecord{
		OperatorID:  input.OperatorID,
		Action:      input.Action,
		TargetType:  input.TargetType,
		TargetID:    input.TargetID,
		RequestID:   input.RequestID,
		DetailsJSON: input.DetailsJSON,
		Outcome:     string(event.Outcome()),
		OccurredAt:  s.clock().UTC(),
	}

	// Step 4: pending intent の永続化に失敗した場合は mutation を開始させないため、抽象 application error へ写像する。
	stored, err := s.audits.RecordAuditIntent(ctx, intent)
	if err != nil {
		return Record{}, ErrAuditInternal
	}

	// Step 5: 後続 mutation が audit ID を correlation として使えるよう、保存済み snapshot を返す。
	return stored, nil
}

// CompleteMutationSucceeded は pending audit event を succeeded outcome へ完了する。
//
// ctx は repository へ deadline/cancellation を伝播する。
// input.AuditID は RecordMutationIntent が返した audit event を指定する。
// 戻り値は更新後の Record で、二重完了や不正な復元 state は ErrAuditBadRequest、保存失敗は ErrAuditInternal を返す。
func (s *AuditService) CompleteMutationSucceeded(ctx context.Context, input CompletionInput) (Record, error) {
	// Step 1: 共通 completion path に succeeded marker を渡し、transition 自体は domain.AdminAuditEvent.MarkSucceeded に委譲する。
	return s.completeAudit(ctx, input.AuditID, adminAuditCompletionSucceeded, "")
}

// BuildMutationSucceededCompletion は外側の transaction 境界で保存する succeeded audit completion DTO を生成する。
//
// ctx を持つ CompleteMutationSucceeded と異なり、この method は repository I/O を実行しない。
// Account 作成 repository のように mutation 本体と audit outcome を同一 DB transaction へ含める場合に使う。
// outcome transition と completed timestamp 検証は domain.AdminAuditEvent.MarkSucceeded に委譲する。
func (s *AuditService) BuildMutationSucceededCompletion(input CompletionInput) (CompletionRecord, error) {
	// Step 1: 新規 intent と同じ pending domain event を作り、成功 outcome の構築を domain rule に委譲する。
	completed, err := domain.NewAdminAuditEvent().MarkSucceeded(s.clock().UTC())
	if err != nil {
		return CompletionRecord{}, mapAdminAuditDomainError(err)
	}

	// Step 2: repository transaction が保存できる primitive completion DTO へ変換する。
	return completionRecordFromEvent(input.AuditID, completed)
}

// CompleteMutationFailed は pending audit event を failed outcome へ完了する。
//
// ctx は repository へ deadline/cancellation を伝播する。
// input.StableErrorCode は domain.AdminAuditEvent.MarkFailed が検証し、canonical code として保存される。
// 戻り値は更新後の Record で、stable error code が不正な場合は ErrAuditBadRequest を返す。
func (s *AuditService) CompleteMutationFailed(ctx context.Context, input FailureInput) (Record, error) {
	// Step 1: 共通 completion path に raw stable error code を渡し、failed transition と code 検証は domain method に委譲する。
	return s.completeAudit(ctx, input.AuditID, adminAuditCompletionFailed, input.StableErrorCode)
}

func (s *AuditService) completeAudit(
	ctx context.Context,
	auditID string,
	kind adminAuditCompletionKind,
	stableErrorCode string,
) (Record, error) {
	// Step 1: 呼び出し元 context が中断済みなら、完了更新を開始せず抽象 application error を返す。
	if err := ctx.Err(); err != nil {
		return Record{}, ErrAuditInternal
	}

	// Step 2: 永続化済み snapshot を取得し、現在 outcome を concrete domain object として復元する。
	current, err := s.audits.FindAudit(ctx, auditID)
	if err != nil {
		return Record{}, ErrAuditInternal
	}
	event, err := reconstituteAdminAuditEvent(current)
	if err != nil {
		return Record{}, err
	}

	// Step 3: 呼び出し元 method が指定した completion 種別で success/failure を選び、transition rule は domain method へ委譲する。
	completed, err := completeAdminAuditEvent(event, kind, stableErrorCode, s.clock().UTC())
	if err != nil {
		return Record{}, err
	}

	// Step 4: domain が保証した completedAt を repository DTO に変換する。
	completion, err := completionRecordFromEvent(auditID, completed)
	if err != nil {
		return Record{}, err
	}

	// Step 5: 完了済み outcome を保存し、保存失敗は pending reconciliation 対象として残すため内部 error に写像する。
	stored, err := s.audits.CompleteAudit(ctx, completion)
	if err != nil {
		return Record{}, ErrAuditInternal
	}

	return stored, nil
}

func reconstituteAdminAuditEvent(record Record) (domain.AdminAuditEvent, error) {
	// Step 1: repository snapshot の primitive outcome を domain enum に戻し、unknown outcome を fail-closed にする。
	event, err := domain.ReconstituteAdminAuditEvent(
		domain.AdminAuditOutcome(record.Outcome),
		domain.StableErrorCode(record.StableErrorCode),
		record.CompletedAt,
	)
	if err != nil {
		return emptyAdminAuditEvent(), mapAdminAuditDomainError(err)
	}

	return event, nil
}

func completeAdminAuditEvent(
	event domain.AdminAuditEvent,
	kind adminAuditCompletionKind,
	stableErrorCode string,
	completedAt time.Time,
) (domain.AdminAuditEvent, error) {
	// Step 1: failure API から来た completion は、stable error code が空でも必ず MarkFailed に通して domain rule で拒否させる。
	if kind == adminAuditCompletionFailed {
		completed, err := event.MarkFailed(domain.StableErrorCode(stableErrorCode), completedAt)
		if err != nil {
			return emptyAdminAuditEvent(), mapAdminAuditDomainError(err)
		}
		return completed, nil
	}

	// Step 2: success API から来た completion だけを succeeded outcome として扱い、二重完了拒否は MarkSucceeded に任せる。
	completed, err := event.MarkSucceeded(completedAt)
	if err != nil {
		return emptyAdminAuditEvent(), mapAdminAuditDomainError(err)
	}
	return completed, nil
}

func emptyAdminAuditEvent() domain.AdminAuditEvent {
	// Step 1: guardrail が禁止する domain composite literal を使わず、error return 用のゼロ値を作る。
	var event domain.AdminAuditEvent
	return event
}

func completionRecordFromEvent(auditID string, event domain.AdminAuditEvent) (CompletionRecord, error) {
	// Step 1: domain method 完了後の completedAt を取得し、nil は不整合として内部 error にする。
	completedAt := event.CompletedAt()
	if completedAt == nil {
		return CompletionRecord{}, ErrAuditInternal
	}

	// Step 2: repository が保存する primitive DTO へ outcome、stable code、完了時刻を変換する。
	return CompletionRecord{
		AuditID:         auditID,
		Outcome:         string(event.Outcome()),
		StableErrorCode: event.StableErrorCode().String(),
		CompletedAt:     *completedAt,
	}, nil
}

func mapAdminAuditDomainError(err error) error {
	// Step 1: domain が拒否した outcome/code/timestamp/transition は、呼び出し側入力または永続化 state の不正として bad request に集約する。
	if errors.Is(err, domain.ErrInvalidAdminAuditOutcome) ||
		errors.Is(err, domain.ErrAdminAuditAlreadyCompleted) ||
		errors.Is(err, domain.ErrInvalidAdminAuditCompletedAt) ||
		errors.Is(err, domain.ErrInvalidAdminAuditStableErrorCode) {
		return ErrAuditBadRequest
	}

	// Step 2: 想定外 error は詳細を露出せず内部 error として扱う。
	return ErrAuditInternal
}
