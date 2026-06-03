package accounts

import (
	"context"
	"errors"
	"testing"
	"time"

	"www-template/packages/backend/internal/application/audit"
	domain "www-template/packages/backend/internal/domain"
)

var adminAccountCreationCreatedAt = time.Date(2026, 5, 26, 3, 0, 0, 0, time.UTC)

// [ADMIN-CONSOLE-BE-S062] Admin API が顧客 account を作成し、[ADMIN-CONSOLE-BE-S067] Account domain rule 共有も同じ happy path で追跡する。
func TestAdminAccountCreationUseCaseCreatesAccountAndCompletesAudit(t *testing.T) {
	t.Parallel()

	// Step 1: account repository、audit service、ID generator を fake として注入し、application orchestration だけを検証する。
	accounts := newFakeAdminAccountCreationRepository()
	audits := newFakeRepository()
	service := mustNewAccountCreationService(t, accounts, audits, newTestAccountIDGenerator(testAdminAccountID))

	// Step 2: 表記揺れのある email と許可済み Operator snapshot を渡し、正規化と権限判定を domain object に通す。
	created, err := service.CreateAccount(context.Background(), validCreateAccountInput())
	if err != nil {
		t.Fatalf("create admin account: %v", err)
	}

	// Step 3: repository へ渡った Account root が domain.NewCreatedAccount 由来の active/canonical/default locale を持つことを確認する。
	if accounts.createCalls != 1 {
		t.Fatalf("CreateAccountWithAuditTarget calls = %d, want 1", accounts.createCalls)
	}
	assertAdminCreatedDomainAccount(t, accounts.creationRecord)

	// Step 4: audit intent と transaction 用 success completion が audit.AuditService 経由で作られ、response correlation が返ることを確認する。
	assertAdminCreateAuditIntent(t, audits.recordedIntent)
	assertAdminAccountCreationSuccessCompletion(t, accounts.creationRecord.AuditCompletion)
	assertCreatedAccountResult(t, created)
}

// [ADMIN-CONSOLE-BE-S063] Duplicate email は 409 と failed audit を返す。
func TestAdminAccountCreationUseCaseRecordsFailedAuditForDuplicateEmail(t *testing.T) {
	t.Parallel()

	// Step 1: repository に duplicate email error を返させ、use case が stable failed audit を完了する経路を作る。
	accounts := newFakeAdminAccountCreationRepository()
	accounts.createError = ErrAccountDuplicateEmail
	audits := newFakeRepository()
	service := mustNewAccountCreationService(t, accounts, audits, newTestAccountIDGenerator(testAdminAccountID))

	// Step 2: duplicate error はそのまま application boundary として返し、handler が 409 に写像できる状態を保つ。
	_, err := service.CreateAccount(context.Background(), validCreateAccountInput())
	if !errors.Is(err, ErrAccountDuplicateEmail) {
		t.Fatalf("expected ErrAccountDuplicateEmail, got %v", err)
	}

	// Step 3: failed outcome は duplicate_email の stable code で記録される。
	assertCompletedAuditCommand(t, audits.completed, "audit-1", "failed", "duplicate_email")
}

// [ADMIN-CONSOLE-BE-S064] account create permission を持たない Operator は拒否される。
func TestAdminAccountCreationUseCaseRejectsOperatorWithoutPermissionBeforeAudit(t *testing.T) {
	t.Parallel()

	// Step 1: viewer Operator は domain.Operator.HasPermission が拒否するため、audit/mutation fake を呼ばない構成を検証する。
	accounts := newFakeAdminAccountCreationRepository()
	audits := newFakeRepository()
	service := mustNewAccountCreationService(t, accounts, audits, newTestAccountIDGenerator(testAdminAccountID))
	input := validCreateAccountInput()
	input.OperatorRole = string(domain.OperatorRoleViewer)

	// Step 2: 権限不足は account/audit target state を変更せず forbidden error として返る。
	_, err := service.CreateAccount(context.Background(), input)
	if !errors.Is(err, ErrAccountCreationForbidden) {
		t.Fatalf("expected ErrAccountCreationForbidden, got %v", err)
	}
	if accounts.createCalls != 0 || audits.recordedIntent.Action != "" {
		t.Fatalf("forbidden operator must not create account or audit intent: calls=%d intent=%+v", accounts.createCalls, audits.recordedIntent)
	}
}

// [ADMIN-CONSOLE-BE-S065] Audit intent failure は account mutation を防ぐ。
func TestAdminAccountCreationUseCaseStopsWhenAuditIntentFails(t *testing.T) {
	t.Parallel()

	// Step 1: intent 保存に失敗する audit repository を注入し、mutation 前 fail-close を検証する。
	accounts := newFakeAdminAccountCreationRepository()
	audits := newFakeRepository()
	audits.recordError = errors.New("audit db unavailable")
	service := mustNewAccountCreationService(t, accounts, audits, newTestAccountIDGenerator(testAdminAccountID))

	// Step 2: audit failure は ErrAuditInternal のまま返り、account repository は呼ばれない。
	_, err := service.CreateAccount(context.Background(), validCreateAccountInput())
	if !errors.Is(err, audit.ErrAuditInternal) {
		t.Fatalf("expected ErrAuditInternal, got %v", err)
	}
	if accounts.createCalls != 0 {
		t.Fatalf("account repository calls = %d, want 0", accounts.createCalls)
	}
}

// [ADMIN-CONSOLE-BE-S066] Account creation failure は failed audit outcome を記録する。
func TestAdminAccountCreationUseCaseRecordsFailedAuditForDomainValidation(t *testing.T) {
	t.Parallel()

	// Step 1: 不正 email を渡し、AccountEmail domain object が拒否した後に failed audit を完了する経路を検証する。
	accounts := newFakeAdminAccountCreationRepository()
	audits := newFakeRepository()
	service := mustNewAccountCreationService(t, accounts, audits, newTestAccountIDGenerator(testAdminAccountID))
	input := validCreateAccountInput()
	input.Email = "not-an-email"

	// Step 2: domain validation error は application の invalid input boundary に畳まれ、repository mutation は実行されない。
	_, err := service.CreateAccount(context.Background(), input)
	if !errors.Is(err, ErrAccountCreationInvalidInput) {
		t.Fatalf("expected ErrAccountCreationInvalidInput, got %v", err)
	}
	if accounts.createCalls != 0 {
		t.Fatalf("account repository calls = %d, want 0", accounts.createCalls)
	}
	assertCompletedAuditCommand(t, audits.completed, "audit-1", "failed", "invalid_account_input")
}

// [ADMIN-CONSOLE-BE-S085] Admin audit event は Go backend の account creation use case から projection port へ送られる。
func TestAdminAccountCreationUseCaseProjectsAuditAfterSuccessfulMutation(t *testing.T) {
	t.Parallel()

	// Step 1: 成功する account repository と projection fake を組み合わせ、DB mutation 成功後の projection 境界だけを検証する。
	accounts := newFakeAdminAccountCreationRepository()
	audits := newFakeRepository()
	projector := newFakeProjector()
	service := mustNewAccountCreationServiceWithProjector(t, accounts, audits, newTestAccountIDGenerator(testAdminAccountID), projector)

	// Step 2: account 作成を実行し、projection が Account commit 後の target account ID を持つことを確認する。
	_, err := service.CreateAccount(context.Background(), validCreateAccountInput())
	if err != nil {
		t.Fatalf("create admin account: %v", err)
	}
	if projector.projectCalls != 1 || projector.record.AuditID != "audit-1" || projector.record.TargetID != testAdminAccountID {
		t.Fatalf("unexpected projection record: calls=%d record=%+v", projector.projectCalls, projector.record)
	}
	if projector.record.Action != "accounts:create" || projector.record.Outcome != "succeeded" {
		t.Fatalf("unexpected projection audit attributes: %+v", projector.record)
	}
}

// [ADMIN-CONSOLE-BE-S087] OpenSearch indexing failure は mutation 成功を取り消さず observer へ渡される。
func TestAdminAccountCreationUseCaseDoesNotRollbackWhenAuditProjectionFails(t *testing.T) {
	t.Parallel()

	// Step 1: projection fake に error を返させ、DB mutation 成功済みの response が維持されることを再現する。
	accounts := newFakeAdminAccountCreationRepository()
	audits := newFakeRepository()
	projector := newFakeProjector()
	projector.projectError = errors.New("opensearch down")
	service := mustNewAccountCreationServiceWithProjector(t, accounts, audits, newTestAccountIDGenerator(testAdminAccountID), projector)

	// Step 2: projection failure があっても CreateAccount は成功し、observer が audit ID と error を受け取ることを確認する。
	created, err := service.CreateAccount(context.Background(), validCreateAccountInput())
	if err != nil {
		t.Fatalf("projection failure must not fail account creation: %v", err)
	}
	if created.AccountID != testAdminAccountID || accounts.createCalls != 1 {
		t.Fatalf("mutation success was not preserved: created=%+v calls=%d", created, accounts.createCalls)
	}
	if projector.failureCalls != 1 || projector.failureAuditID != "audit-1" || !errors.Is(projector.failureError, projector.projectError) {
		t.Fatalf("projection failure was not observed: %+v", projector)
	}
}

const testAdminAccountID = "01ARZ3NDEKTSV4RRFFQ69G5FAV"

func validCreateAccountInput() CreateAccountInput {
	// Step 1: 許可済み Operator snapshot と raw email を組み合わせ、各 test が必要な差分だけ上書きできる基準入力を返す。
	return CreateAccountInput{
		Email:                    "  Customer@Example.COM  ",
		RequestID:                "req-admin-account-create-1",
		OperatorID:               "01ARZ3NDEKTSV4RRFFQ69G5FAW",
		OperatorEmail:            "operator@example.com",
		OperatorRole:             string(domain.OperatorRoleAdmin),
		OperatorActive:           true,
		PasskeyRegistrationState: string(domain.OperatorPasskeyRegistrationRegistered),
	}
}

func mustNewAccountCreationService(
	t *testing.T,
	accounts *fakeAdminAccountCreationRepository,
	audits *fakeRepository,
	ids *testAccountIDGenerator,
) *AccountCreationService {
	return mustNewAccountCreationServiceWithProjector(t, accounts, audits, ids, newFakeProjector())
}

func mustNewAccountCreationServiceWithProjector(
	t *testing.T,
	accounts *fakeAdminAccountCreationRepository,
	audits *fakeRepository,
	ids *testAccountIDGenerator,
	projector *fakeProjector,
) *AccountCreationService {
	t.Helper()

	// Step 1: audit service は既存 use case constructor を通し、account creation test でも OperatorAuditEvent transition を共有する。
	auditService := mustNewAuditService(t, audits)
	service, err := NewAccountCreationService(accounts, auditService, ids, projector, projector)
	if err != nil {
		t.Fatalf("new admin account creation service: %v", err)
	}

	// Step 2: 構築済み service だけを返し、各 test が orchestration method に集中できるようにする。
	return service
}

type fakeProjector struct {
	record         audit.ProjectionRecord
	projectError   error
	failureError   error
	failureAuditID string
	projectCalls   int
	failureCalls   int
}

func newFakeProjector() *fakeProjector {
	// Step 1: default fake は projection success として動作し、必要な test だけ error を上書きする。
	return &fakeProjector{}
}

func (p *fakeProjector) ProjectOperatorAuditEvent(ctx context.Context, record audit.ProjectionRecord) error {
	// Step 1: context cancellation は production projector と同じ入口条件として扱い、projection の呼び出し可否を検証可能にする。
	if err := ctx.Err(); err != nil {
		return err
	}

	// Step 2: projection document を保存し、use case が Admin audit event を検索投影へ渡したことを test で確認できるようにする。
	p.projectCalls++
	p.record = record
	return p.projectError
}

func (p *fakeProjector) ObserveOperatorAuditProjectionFailure(_ context.Context, auditID string, err error) {
	// Step 1: warning log / metric adapter の代わりに失敗情報を保持し、projection failure が沈黙しないことを検証する。
	p.failureCalls++
	p.failureAuditID = auditID
	p.failureError = err
}

func assertAdminCreatedDomainAccount(t *testing.T, record AccountCreationRecord) {
	t.Helper()

	// Step 1: Account root は domain constructor により canonical ID/email/active/default locale になっていることを確認する。
	if record.AuditID != "audit-1" {
		t.Fatalf("audit ID = %q, want audit-1", record.AuditID)
	}
	if record.Account.ID().String() != testAdminAccountID {
		t.Fatalf("account ID = %q, want %q", record.Account.ID().String(), testAdminAccountID)
	}
	if record.Account.Email().String() != "customer@example.com" {
		t.Fatalf("account email = %q, want customer@example.com", record.Account.Email().String())
	}
	if record.Account.Status() != domain.AccountStatusActive || record.Account.Setting().Locale().String() != "ja" {
		t.Fatalf("account lifecycle = status %q locale %q", record.Account.Status(), record.Account.Setting().Locale().String())
	}
}

func assertAdminCreateAuditIntent(t *testing.T, record audit.IntentRecord) {
	t.Helper()

	// Step 1: audit intent は accounts:create と request/operator correlation を持ち、pending outcome として記録される。
	if record.OperatorID != "01ARZ3NDEKTSV4RRFFQ69G5FAW" || record.Action != "accounts:create" {
		t.Fatalf("unexpected audit intent correlation: %+v", record)
	}
	if record.TargetType != "account" || record.RequestID != "req-admin-account-create-1" {
		t.Fatalf("unexpected audit intent target/request: %+v", record)
	}
	if record.DetailsJSON != `{"requested_email":"  Customer@Example.COM  "}` {
		t.Fatalf("details JSON = %s", record.DetailsJSON)
	}
}

func assertAdminAccountCreationSuccessCompletion(t *testing.T, record audit.CompletionRecord) {
	t.Helper()

	// Step 1: Account repository transaction に渡す success completion は domain.OperatorAuditEvent.MarkSucceeded 済みの値だけを含む。
	if record.AuditID != "audit-1" || record.Outcome != "succeeded" || record.StableErrorCode != "" {
		t.Fatalf("unexpected success completion: %+v", record)
	}
	if record.CompletedAt.IsZero() || !record.CompletedAt.Equal(operatorAuditUseCaseNow.UTC()) {
		t.Fatalf("completion time = %v, want %v", record.CompletedAt, operatorAuditUseCaseNow.UTC())
	}
}

func assertCreatedAccountResult(t *testing.T, created CreatedAccount) {
	t.Helper()

	// Step 1: success result は handler が response DTO に変換できる primitive snapshot と audit correlation を保持する。
	if created.AccountID != testAdminAccountID || created.Email != "customer@example.com" {
		t.Fatalf("created account identity = %+v", created)
	}
	if created.Status != "active" || created.Locale != "ja" || created.PasskeyCount != 0 {
		t.Fatalf("created account lifecycle = %+v", created)
	}
	if created.AuditID != "audit-1" || created.RequestID != "req-admin-account-create-1" {
		t.Fatalf("created account audit correlation = %+v", created)
	}
}

type fakeAdminAccountCreationRepository struct {
	creationRecord AccountCreationRecord
	createError    error
	createCalls    int
}

func newFakeAdminAccountCreationRepository() *fakeAdminAccountCreationRepository {
	// Step 1: default fake は successful repository として動作し、各 test が必要な error だけ上書きする。
	return &fakeAdminAccountCreationRepository{}
}

func (r *fakeAdminAccountCreationRepository) CreateAccountWithAuditTarget(ctx context.Context, record AccountCreationRecord) (AccountRecord, error) {
	// Step 1: context cancellation を fake でも尊重し、実 repository と同じ入口条件を保つ。
	if err := ctx.Err(); err != nil {
		return AccountRecord{}, err
	}

	// Step 2: 呼び出し回数と入力 record を保存し、use case が domain Account と audit ID を渡したことを test で検証可能にする。
	r.createCalls++
	r.creationRecord = record
	if r.createError != nil {
		return AccountRecord{}, r.createError
	}

	// Step 3: repository 成功時の primitive snapshot を返し、handler-facing result 変換を検証できるようにする。
	return AccountRecord{
		AccountID: record.Account.ID().String(),
		Email:     record.Account.Email().String(),
		Status:    record.Account.Status().String(),
		Locale:    record.Account.Setting().Locale().String(),
		CreatedAt: adminAccountCreationCreatedAt,
	}, nil
}

type testAccountIDGenerator struct {
	value string
	err   error
}

var operatorAuditUseCaseNow = time.Date(2026, 5, 26, 12, 0, 0, 0, time.FixedZone("JST", 9*60*60))

type fakeRepository struct {
	current        audit.Record
	recordedIntent audit.IntentRecord
	completed      audit.CompletionRecord
	recordError    error
	findError      error
	completeError  error
	completeCalls  int
}

func newFakeRepository() *fakeRepository {
	// Step 1: account creation test 用に pending audit を返せる repository fake を用意し、各 test が error だけを差し替えられるようにする。
	return &fakeRepository{current: audit.Record{AuditID: "audit-1", Outcome: "pending"}}
}

func mustNewAuditService(t *testing.T, repository *fakeRepository) *audit.AuditService {
	t.Helper()

	// Step 1: account creation use case から見る audit capability を固定 clock 付きで生成し、時刻依存を deterministic にする。
	service, err := audit.NewAuditService(repository, func() time.Time { return operatorAuditUseCaseNow })
	if err != nil {
		t.Fatalf("new audit service: %v", err)
	}
	return service
}

func (r *fakeRepository) RecordAuditIntent(ctx context.Context, record audit.IntentRecord) (audit.Record, error) {
	// Step 1: context cancellation と保存 error を production repository と同じ順で扱い、mutation 前 fail-close を検証する。
	if err := ctx.Err(); err != nil {
		return audit.Record{}, err
	}
	if r.recordError != nil {
		return audit.Record{}, r.recordError
	}

	// Step 2: use case から渡された intent を保持し、保存済み audit snapshot として返す。
	r.recordedIntent = record
	r.current = audit.Record{AuditID: "audit-1", OperatorID: record.OperatorID, Action: record.Action, TargetType: record.TargetType, TargetID: record.TargetID, RequestID: record.RequestID, DetailsJSON: record.DetailsJSON, Outcome: record.Outcome, StableErrorCode: record.StableErrorCode, OccurredAt: record.OccurredAt, CompletedAt: record.CompletedAt}
	return r.current, nil
}

func (r *fakeRepository) FindAudit(ctx context.Context, auditID string) (audit.Record, error) {
	// Step 1: completion path が pending audit を復元できるよう、指定された audit ID を現在 snapshot に反映して返す。
	if err := ctx.Err(); err != nil {
		return audit.Record{}, err
	}
	if r.findError != nil {
		return audit.Record{}, r.findError
	}
	current := r.current
	current.AuditID = auditID
	return current, nil
}

func (r *fakeRepository) CompleteAudit(ctx context.Context, record audit.CompletionRecord) (audit.Record, error) {
	// Step 1: completion command を保持し、account creation use case が stable failed/succeeded outcome を保存したことを確認できるようにする。
	if err := ctx.Err(); err != nil {
		return audit.Record{}, err
	}
	r.completeCalls++
	r.completed = record
	if r.completeError != nil {
		return audit.Record{}, r.completeError
	}
	completedAt := record.CompletedAt
	r.current.AuditID = record.AuditID
	r.current.Outcome = record.Outcome
	r.current.StableErrorCode = record.StableErrorCode
	r.current.CompletedAt = &completedAt
	return r.current, nil
}

func assertCompletedAuditCommand(t *testing.T, command audit.CompletionRecord, auditID string, outcome string, stableErrorCode string) {
	t.Helper()

	// Step 1: account creation failure/success が audit capability の completion DTO として canonical outcome を保存していることを検証する。
	if command.AuditID != auditID || command.Outcome != outcome || command.StableErrorCode != stableErrorCode {
		t.Fatalf("unexpected audit completion command: %+v", command)
	}
	if !command.CompletedAt.Equal(operatorAuditUseCaseNow.UTC()) {
		t.Fatalf("command completedAt = %v, want %v", command.CompletedAt, operatorAuditUseCaseNow.UTC())
	}
}

func newTestAccountIDGenerator(value string) *testAccountIDGenerator {
	// Step 1: deterministic な AccountID generator として、domain.NewAccountID に通す raw 値を保持する。
	return &testAccountIDGenerator{value: value}
}

func (g *testAccountIDGenerator) Next() (string, error) {
	// Step 1: test が指定した generator error を返し、ID 生成失敗 path を必要に応じて検証できるようにする。
	if g.err != nil {
		return "", g.err
	}

	// Step 2: deterministic な raw ID を返し、AccountID validation は production と同じ domain constructor に委譲する。
	return g.value, nil
}
