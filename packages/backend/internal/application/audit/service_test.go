package audit

import (
	"context"
	"errors"
	"testing"
	"time"
)

var adminAuditUseCaseNow = time.Date(2026, 5, 26, 12, 0, 0, 0, time.FixedZone("JST", 9*60*60))

func TestAdminAuditUseCaseRecordsPendingIntentBeforeMutation(t *testing.T) {
	t.Parallel()

	// Step 1: repository fake と注入 clock だけで audit use case を作り、DB 実装なしで application boundary を検証する。
	repository := newFakeRepository()
	service := mustNewAuditService(t, repository)

	// Step 2: account creation use case 予定の入力に近い operator/action/target correlation を渡す。
	stored, err := service.RecordMutationIntent(context.Background(), IntentInput{
		OperatorID:  "01ARZ3NDEKTSV4RRFFQ69G5FAW",
		Action:      "accounts:create",
		TargetType:  "account",
		TargetID:    "",
		RequestID:   "req-admin-audit-1",
		DetailsJSON: `{"email":"customer@example.com"}`,
	})
	if err != nil {
		t.Fatalf("record mutation intent: %v", err)
	}

	// Step 3: domain.NewAdminAuditEvent 由来の pending outcome が repository port へ渡り、後続 mutation 用の audit ID が返ることを確認する。
	if stored.AuditID == "" {
		t.Fatal("expected stored audit ID")
	}
	if repository.recordedIntent.Outcome != "pending" {
		t.Fatalf("recorded outcome = %q, want pending", repository.recordedIntent.Outcome)
	}
	if repository.recordedIntent.StableErrorCode != "" || repository.recordedIntent.CompletedAt != nil {
		t.Fatalf("pending intent must not have error/completedAt: %+v", repository.recordedIntent)
	}
	if !repository.recordedIntent.OccurredAt.Equal(adminAuditUseCaseNow.UTC()) {
		t.Fatalf("occurredAt = %v, want %v", repository.recordedIntent.OccurredAt, adminAuditUseCaseNow.UTC())
	}
}

// [ADMIN-CONSOLE-BE-S066] account 作成失敗は failed outcome として監査される。
func TestAdminAuditUseCaseCompletesFailedOutcomeThroughDomainEvent(t *testing.T) {
	t.Parallel()

	// Step 1: pending intent の snapshot を fake repository に置き、failure completion が既存 intent を復元する流れを作る。
	repository := newFakeRepository()
	repository.current = Record{AuditID: "audit-1", Outcome: "pending"}
	service := mustNewAuditService(t, repository)

	// Step 2: mixed-case の stable error code を渡し、canonical 化と failed transition を domain.AdminAuditEvent.MarkFailed に委譲する。
	stored, err := service.CompleteMutationFailed(context.Background(), FailureInput{
		AuditID:         "audit-1",
		StableErrorCode: "DUPLICATE_EMAIL",
	})
	if err != nil {
		t.Fatalf("complete failed audit: %v", err)
	}

	// Step 3: repository には failed/canonical code/completedAt だけが渡り、application 側に transition duplicate logic がないことを結果で固定する。
	assertCompletedAuditRecord(t, stored, "failed", "duplicate_email")
	assertCompletedAuditCommand(t, repository.completed, "audit-1", "failed", "duplicate_email")
}

func TestAdminAuditUseCaseCompletesSucceededOutcomeThroughDomainEvent(t *testing.T) {
	t.Parallel()

	// Step 1: pending intent の snapshot を fake repository に置き、success completion が既存 intent を復元する流れを作る。
	repository := newFakeRepository()
	repository.current = Record{AuditID: "audit-2", Outcome: "pending"}
	service := mustNewAuditService(t, repository)

	// Step 2: success 用 API を呼び、domain.AdminAuditEvent.MarkSucceeded が completed timestamp を必須にする境界を通す。
	stored, err := service.CompleteMutationSucceeded(context.Background(), CompletionInput{AuditID: "audit-2"})
	if err != nil {
		t.Fatalf("complete succeeded audit: %v", err)
	}

	// Step 3: succeeded outcome には stable error code が混ざらないことを repository command と戻り値で検証する。
	assertCompletedAuditRecord(t, stored, "succeeded", "")
	assertCompletedAuditCommand(t, repository.completed, "audit-2", "succeeded", "")
}

// [ADMIN-CONSOLE-BE-S065] audit intent 作成失敗時は mutation を開始しない。
func TestAdminAuditUseCaseFailsClosedWhenIntentCannotBeRecorded(t *testing.T) {
	t.Parallel()

	// Step 1: intent 保存だけを失敗させ、account mutation use case がこの error で停止できる fail-close 境界を検証する。
	repository := newFakeRepository()
	repository.recordError = errors.New("db unavailable")
	service := mustNewAuditService(t, repository)

	// Step 2: repository error は詳細を露出せず ErrAuditInternal に写像される。
	_, err := service.RecordMutationIntent(context.Background(), IntentInput{Action: "accounts:create"})
	if !errors.Is(err, ErrAuditInternal) {
		t.Fatalf("expected ErrAuditInternal, got %v", err)
	}
}

func TestAdminAuditUseCaseRejectsAlreadyCompletedAudit(t *testing.T) {
	t.Parallel()

	// Step 1: 既に succeeded の snapshot を配置し、二重完了拒否を application ではなく domain.Reconstitute/MarkSucceeded 経由で発生させる。
	completedAt := adminAuditUseCaseNow.UTC()
	repository := newFakeRepository()
	repository.current = Record{AuditID: "audit-3", Outcome: "succeeded", CompletedAt: &completedAt}
	service := mustNewAuditService(t, repository)

	// Step 2: 完了済み audit の再完了は ErrAuditBadRequest になり、repository の CompleteAudit へ進まないことを確認する。
	_, err := service.CompleteMutationSucceeded(context.Background(), CompletionInput{AuditID: "audit-3"})
	if !errors.Is(err, ErrAuditBadRequest) {
		t.Fatalf("expected ErrAuditBadRequest, got %v", err)
	}
	if repository.completeCalls != 0 {
		t.Fatalf("CompleteAudit calls = %d, want 0", repository.completeCalls)
	}
}

func TestAdminAuditUseCaseRejectsFailedOutcomeWithoutStableErrorCode(t *testing.T) {
	t.Parallel()

	// Step 1: pending intent の snapshot を配置し、failure API が空 stable code を success と誤判定しないことを検証する。
	repository := newFakeRepository()
	repository.current = Record{AuditID: "audit-4", Outcome: "pending"}
	service := mustNewAuditService(t, repository)

	// Step 2: failed outcome に必須の stable error code を空にし、domain.AdminAuditEvent.MarkFailed による拒否へ委譲されることを確認する。
	_, err := service.CompleteMutationFailed(context.Background(), FailureInput{AuditID: "audit-4"})
	if !errors.Is(err, ErrAuditBadRequest) {
		t.Fatalf("expected ErrAuditBadRequest, got %v", err)
	}
	if repository.completeCalls != 0 {
		t.Fatalf("CompleteAudit calls = %d, want 0", repository.completeCalls)
	}
}

func mustNewAuditService(t *testing.T, repository *fakeRepository) *AuditService {
	t.Helper()

	// Step 1: test 用 clock は固定し、application code が time.Now に依存しないことを検証可能にする。
	service, err := NewAuditService(repository, func() time.Time { return adminAuditUseCaseNow })
	if err != nil {
		t.Fatalf("new admin audit service: %v", err)
	}
	return service
}

func assertCompletedAuditRecord(t *testing.T, record Record, outcome string, stableErrorCode string) {
	t.Helper()

	// Step 1: repository から返る application DTO が期待 outcome と stable code を持つことを確認する。
	if record.Outcome != outcome {
		t.Fatalf("outcome = %q, want %q", record.Outcome, outcome)
	}
	if record.StableErrorCode != stableErrorCode {
		t.Fatalf("stable error code = %q, want %q", record.StableErrorCode, stableErrorCode)
	}
	if record.CompletedAt == nil || !record.CompletedAt.Equal(adminAuditUseCaseNow.UTC()) {
		t.Fatalf("completedAt = %v, want %v", record.CompletedAt, adminAuditUseCaseNow.UTC())
	}
}

func assertCompletedAuditCommand(
	t *testing.T,
	command CompletionRecord,
	auditID string,
	outcome string,
	stableErrorCode string,
) {
	t.Helper()

	// Step 1: repository port に渡した completion command が domain method 後の canonical outcome だけを含むことを確認する。
	if command.AuditID != auditID {
		t.Fatalf("audit ID = %q, want %q", command.AuditID, auditID)
	}
	if command.Outcome != outcome {
		t.Fatalf("command outcome = %q, want %q", command.Outcome, outcome)
	}
	if command.StableErrorCode != stableErrorCode {
		t.Fatalf("command stable error code = %q, want %q", command.StableErrorCode, stableErrorCode)
	}
	if !command.CompletedAt.Equal(adminAuditUseCaseNow.UTC()) {
		t.Fatalf("command completedAt = %v, want %v", command.CompletedAt, adminAuditUseCaseNow.UTC())
	}
}

type fakeRepository struct {
	current        Record
	recordedIntent IntentRecord
	completed      CompletionRecord
	recordError    error
	findError      error
	completeError  error
	completeCalls  int
}

func newFakeRepository() *fakeRepository {
	// Step 1: default fake は audit ID を返せる pending repository として初期化し、各 test が必要な error だけ上書きする。
	return &fakeRepository{current: Record{AuditID: "audit-1", Outcome: "pending"}}
}

func (r *fakeRepository) RecordAuditIntent(ctx context.Context, record IntentRecord) (Record, error) {
	// Step 1: context cancellation を fake でも尊重し、実 repository と同じ入口条件を保つ。
	if err := ctx.Err(); err != nil {
		return Record{}, err
	}

	// Step 2: test が指定した保存 error を返し、use case の fail-close mapping を検証できるようにする。
	if r.recordError != nil {
		return Record{}, r.recordError
	}

	// Step 3: use case が渡した intent command を保持し、保存済み audit snapshot として返す。
	r.recordedIntent = record
	r.current = Record{
		AuditID:         "audit-1",
		OperatorID:      record.OperatorID,
		Action:          record.Action,
		TargetType:      record.TargetType,
		TargetID:        record.TargetID,
		RequestID:       record.RequestID,
		DetailsJSON:     record.DetailsJSON,
		Outcome:         record.Outcome,
		StableErrorCode: record.StableErrorCode,
		OccurredAt:      record.OccurredAt,
		CompletedAt:     record.CompletedAt,
	}
	return r.current, nil
}

func (r *fakeRepository) FindAudit(ctx context.Context, auditID string) (Record, error) {
	// Step 1: context cancellation を確認し、use case が cancellation を repository へ伝播する前提を fake でも維持する。
	if err := ctx.Err(); err != nil {
		return Record{}, err
	}

	// Step 2: test が指定した検索 error を返し、repository failure mapping を検証可能にする。
	if r.findError != nil {
		return Record{}, r.findError
	}

	// Step 3: 呼び出し元が指定した audit ID を snapshot に反映し、completion command の correlation を検証しやすくする。
	current := r.current
	current.AuditID = auditID
	return current, nil
}

func (r *fakeRepository) CompleteAudit(ctx context.Context, record CompletionRecord) (Record, error) {
	// Step 1: context cancellation を確認し、completion 更新も intent と同じ context 境界で扱う。
	if err := ctx.Err(); err != nil {
		return Record{}, err
	}

	// Step 2: 呼び出し回数と command を保存し、domain transition 後だけ repository に進むことを検証する。
	r.completeCalls++
	r.completed = record
	if r.completeError != nil {
		return Record{}, r.completeError
	}

	// Step 3: 更新後 snapshot を返し、application DTO の outcome/completedAt を検証可能にする。
	completedAt := record.CompletedAt
	r.current.AuditID = record.AuditID
	r.current.Outcome = record.Outcome
	r.current.StableErrorCode = record.StableErrorCode
	r.current.CompletedAt = &completedAt
	return r.current, nil
}
