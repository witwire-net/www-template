package application

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	domain "www-template/packages/backend/internal/domain"
)

const (
	adminAccountCreateAction                 = "accounts:create"
	adminAccountCreateTargetType             = "account"
	adminAccountStableCodeDuplicateEmail     = "duplicate_email"
	adminAccountStableCodeInvalidInput       = "invalid_account_input"
	adminAccountStableCodeInternal           = "account_creation_internal"
	adminAccountStableCodeRepositoryFailure  = "account_repository_unavailable"
	adminAccountStableCodeAuditTargetMissing = "audit_target_not_found"
)

var (
	// ErrAdminAccountCreationForbidden は Operator が Admin account 作成を実行できない場合に返す application error である。
	// domain.Operator.HasPermission が拒否した結果だけを表し、handler が 403 に写像できる stable boundary として使う。
	ErrAdminAccountCreationForbidden = errors.New("admin account creation forbidden")

	// ErrAdminAccountCreationInvalidInput は account 作成入力が Account domain object により拒否された場合に返す application error である。
	// email 形式や AccountID 生成値の不正を handler へ domain 型なしで伝え、400 系の stable error に写像できるようにする。
	ErrAdminAccountCreationInvalidInput = errors.New("admin account creation invalid input")

	// ErrAdminAccountCreationInternal は account 作成 use case の必須依存や内部生成処理が利用できない場合に返す application error である。
	// repository/audit/id generator の詳細を外へ出さず、fail-closed な 5xx 系分類へ畳む。
	ErrAdminAccountCreationInternal = errors.New("admin account creation internal")
)

// AdminAccountIDGenerator は AccountID を発行する最小 port である。
//
// 役割:
//   - application 層が platform/id の具象実装に直接依存しないよう、Next だけを要求する。
//   - 生成値の形式検証は domain.NewAccountID に委譲し、不正な ID が root 作成へ進まないようにする。
//   - test では deterministic generator を注入し、account creation use case の振る舞いを安定して検証できるようにする。
type AdminAccountIDGenerator interface {
	Next() (string, error)
}

// AdminAccountCreationService は Admin 経由の顧客 account 作成 orchestration を担う application use case である。
//
// 役割:
//   - Operator 復元と `accounts:create` 権限判定を domain.Operator に委譲する。
//   - AccountEmail と NewAdminCreatedAccount を使って Account root の不変条件を domain 層で検証する。
//   - audit intent、Product Account root 作成、audit outcome 完了の順序だけを制御し、handler や repository に業務 rule を置かない。
//
// 使用例:
//
//	service, err := NewAdminAccountCreationService(accounts, audits, ids, projector, projectionFailures)
//	if err != nil {
//		return err
//	}
//	created, err := service.CreateAccount(ctx, input)
//	_ = created
type AdminAccountCreationService struct {
	accounts           AdminAccountRepository
	audits             *AdminAuditService
	ids                AdminAccountIDGenerator
	projector          AdminAuditProjector
	projectionFailures AdminAuditProjectionFailureObserver
}

// AdminCreateAccountInput は Admin account creation use case の入力 DTO である。
//
// 役割:
//   - HTTP/generated 型ではなく primitive だけで handler 境界から受け取る。
//   - Operator の snapshot は domain.NewOperator で復元され、Product account auth 情報を含めない。
//   - Email は raw 入力のまま受け取り、正規化と検証を domain.NewAccountEmail に委譲する。
type AdminCreateAccountInput struct {
	Email                    string
	RequestID                string
	OperatorID               string
	OperatorEmail            string
	OperatorRole             string
	OperatorActive           bool
	PasskeyRegistrationState string
}

// AdminCreatedAccount は Admin account creation use case の成功結果 DTO である。
//
// 役割:
//   - handler が response DTO へ変換するための primitive snapshot だけを保持する。
//   - Account root の email/status/locale は domain と repository を通った canonical 値である。
//   - AuditID は mutation intent/outcome と response correlation を結び付けるために返す。
type AdminCreatedAccount struct {
	AccountID    string
	Email        string
	Status       string
	Locale       string
	PasskeyCount int32
	AuditID      string
	RequestID    string
	CreatedAt    time.Time
}

// NewAdminAccountCreationService は Admin account creation use case を生成する。
//
// 引数:
//   - accounts: Product Account root と Admin audit target を同一 transaction で保存する repository port。
//   - audits: mutation intent/outcome を管理する AdminAuditService。
//   - ids: 新規 AccountID を発行する ID generator port。
//   - projector: 成功済み audit event を Admin audit search namespace へ投影する port。
//   - projectionFailures: projection failure を warning log / metric / retry marker へ委譲する observer port。
//
// 戻り値:
//   - *AdminAccountCreationService: 検証済み依存を保持する use case。
//   - error: 必須依存が nil の場合は ErrAdminAccountCreationInternal。
func NewAdminAccountCreationService(accounts AdminAccountRepository, audits *AdminAuditService, ids AdminAccountIDGenerator, projector AdminAuditProjector, projectionFailures AdminAuditProjectionFailureObserver) (*AdminAccountCreationService, error) {
	// Step 1: 未監査 mutation や AccountID なしの root 作成を防ぐため、必須 port が欠けている構成を拒否する。
	if accounts == nil || audits == nil || ids == nil || projector == nil || projectionFailures == nil {
		return nil, ErrAdminAccountCreationInternal
	}

	// Step 2: 検証済み依存だけを保持し、handler/runtime composition から再利用できる service として返す。
	return &AdminAccountCreationService{accounts: accounts, audits: audits, ids: ids, projector: projector, projectionFailures: projectionFailures}, nil
}

// CreateAccount は Operator 権限、audit intent、Account domain 構築、repository transaction、audit outcome を順に実行する。
//
// ctx は audit repository と account repository へ deadline/cancellation を伝播する。
// input は raw email と検証済み operator snapshot を含む primitive DTO である。
// 成功時は作成済み account snapshot と audit correlation を返し、失敗時は duplicate/forbidden/invalid/internal の application error を返す。
func (s *AdminAccountCreationService) CreateAccount(ctx context.Context, input AdminCreateAccountInput) (AdminCreatedAccount, error) {
	// Step 1: cancellation 済み request では audit intent も mutation も開始しない。
	if err := ctx.Err(); err != nil {
		return AdminCreatedAccount{}, ErrAdminAccountCreationInternal
	}

	// Step 2: operator snapshot を concrete domain object に復元し、accounts:create 権限判定を domain に委譲する。
	permission := domain.OperatorAuthPermissionAccountsCreate.String()
	operator, err := restoreAccountCreationOperator(input, permission)
	if err != nil {
		return AdminCreatedAccount{}, err
	}

	// Step 3: mutation 前 intent を必ず記録し、保存できない場合は Account root 作成へ進まない。
	intent, err := s.recordAccountCreationIntent(ctx, input, operator)
	if err != nil {
		return AdminCreatedAccount{}, err
	}

	// Step 4: raw email と発行 ID を domain Account root へ変換し、失敗は failed audit outcome にして返す。
	account, err := s.newAccountRootForCreation(input.Email)
	if err != nil {
		return s.failAccountCreation(ctx, intent.AuditID, err, stableCodeForAccountRootError(err))
	}

	// Step 5: success audit completion を domain.AdminAuditEvent 経由で作り、Account 作成 transaction に同梱する。
	completion, err := s.audits.BuildMutationSucceededCompletion(AdminAuditCompletionInput{AuditID: intent.AuditID})
	if err != nil {
		return s.failAccountCreation(ctx, intent.AuditID, err, adminAccountStableCodeInternal)
	}

	// Step 6: repository transaction に domain.Account と success audit completion を渡し、重複や DB failure を application error として受け取る。
	created, err := s.accounts.CreateAccountWithAuditTarget(ctx, AdminAccountCreationRecord{Account: account, AuditID: intent.AuditID, AuditCompletion: completion})
	if err != nil {
		return s.failAccountCreation(ctx, intent.AuditID, err, stableCodeForAccountCreationError(err))
	}

	// Step 7: Account 作成 transaction の成功後にだけ audit projection を実行し、projection failure は mutation 成功を取り消さず observer へ渡す。
	s.projectAccountCreationAudit(ctx, intent, completion, created)

	// Step 8: handler 用の primitive DTO に変換し、domain/repository 型を transport 境界へ漏らさない。
	return createdAccountResult(created, intent), nil
}

func (s *AdminAccountCreationService) projectAccountCreationAudit(ctx context.Context, intent AdminAuditRecord, completion AdminAuditCompletionRecord, created AdminAccountRecord) {
	// Step 1: projection document は commit 済み Account snapshot と intent correlation から組み立て、未確定の target ID を使わない。
	projection := AdminAuditProjectionRecord{
		AuditID:     intent.AuditID,
		OperatorID:  intent.OperatorID,
		Action:      intent.Action,
		TargetType:  intent.TargetType,
		TargetID:    created.AccountID,
		RequestID:   intent.RequestID,
		DetailsJSON: intent.DetailsJSON,
		Outcome:     string(domain.AdminAuditOutcomeSucceeded),
		OccurredAt:  intent.OccurredAt,
		CompletedAt: &completion.CompletedAt,
	}

	// Step 2: OpenSearch 側の一時障害は成功済み DB mutation を rollback できないため、error は observer へ渡して成功 response を維持する。
	if err := s.projector.ProjectAdminAuditEvent(ctx, projection); err != nil {
		s.projectionFailures.ObserveAdminAuditProjectionFailure(ctx, intent.AuditID, err)
	}
}

func (s *AdminAccountCreationService) recordAccountCreationIntent(ctx context.Context, input AdminCreateAccountInput, operator domain.Operator) (AdminAuditRecord, error) {
	// Step 1: audit details は machine-readable JSON に限定し、handler message や secret を混ぜない。
	detailsJSON, err := accountCreationDetailsJSON(input.Email)
	if err != nil {
		return AdminAuditRecord{}, ErrAdminAccountCreationInternal
	}

	// Step 2: operator/action/request correlation を audit service へ渡し、pending intent の組み立ては AdminAuditEvent に委譲する。
	return s.audits.RecordMutationIntent(ctx, AdminAuditIntentInput{
		OperatorID:  operator.ID().String(),
		Action:      adminAccountCreateAction,
		TargetType:  adminAccountCreateTargetType,
		RequestID:   input.RequestID,
		DetailsJSON: detailsJSON,
	})
}

func (s *AdminAccountCreationService) newAccountRootForCreation(rawEmail string) (domain.Account, error) {
	// Step 1: email の trim/lowercase/形式検証は AccountEmail domain object だけに委譲する。
	email, err := domain.NewAccountEmail(rawEmail)
	if err != nil {
		return emptyDomainAccount(), ErrAdminAccountCreationInvalidInput
	}

	// Step 2: ID generator の raw 値は AccountID constructor に通し、未検証 ID を Account root に入れない。
	rawID, err := s.ids.Next()
	if err != nil {
		return emptyDomainAccount(), ErrAdminAccountCreationInternal
	}
	accountID, err := domain.NewAccountID(rawID)
	if err != nil {
		return emptyDomainAccount(), ErrAdminAccountCreationInternal
	}

	// Step 3: active 初期状態、DefaultAccountSetting、session revoke 境界なしの決定は NewAdminCreatedAccount に委譲する。
	account, err := domain.NewAdminCreatedAccount(accountID, email)
	if err != nil {
		return emptyDomainAccount(), ErrAdminAccountCreationInternal
	}
	return account, nil
}

func (s *AdminAccountCreationService) failAccountCreation(ctx context.Context, auditID string, original error, stableCode string) (AdminCreatedAccount, error) {
	// Step 1: mutation 失敗を failed audit outcome として完了し、失敗監査が保存できない場合は audit error を優先する。
	if _, err := s.audits.CompleteMutationFailed(ctx, AdminAuditFailureInput{AuditID: auditID, StableErrorCode: stableCode}); err != nil {
		return AdminCreatedAccount{}, err
	}

	// Step 2: 監査が完了した場合は元の application error を返し、handler が duplicate/invalid などを安定写像できるようにする。
	return AdminCreatedAccount{}, original
}

func restoreAccountCreationOperator(input AdminCreateAccountInput, permission string) (domain.Operator, error) {
	// Step 1: operator ID を Admin 専用 value object として復元し、不正値は権限不足に畳む。
	operatorID, err := domain.NewOperatorID(input.OperatorID)
	if err != nil {
		return emptyDomainOperator(), ErrAdminAccountCreationForbidden
	}

	// Step 2: operator email も domain value object へ戻し、未検証 snapshot の mutation 使用を防ぐ。
	operatorEmail, err := domain.NewOperatorEmail(input.OperatorEmail)
	if err != nil {
		return emptyDomainOperator(), ErrAdminAccountCreationForbidden
	}

	// Step 3: role/active/passkey state を Operator domain object に集約し、permission 判定の source of truth を domain に固定する。
	operator, err := domain.NewOperator(operatorID, operatorEmail, domain.OperatorRole(input.OperatorRole), input.OperatorActive, domain.OperatorPasskeyRegistrationState(input.PasskeyRegistrationState))
	if err != nil {
		return emptyDomainOperator(), ErrAdminAccountCreationForbidden
	}

	// Step 4: accounts:create の許可可否は HasPermission に委譲し、application 側で role matrix を複製しない。
	if !operator.HasPermission(permission) {
		return emptyDomainOperator(), ErrAdminAccountCreationForbidden
	}

	// Step 5: 許可済み Operator だけを後続 audit correlation に渡す。
	return operator, nil
}

func emptyDomainAccount() domain.Account {
	// Step 1: guardrail が禁止する domain composite literal を使わず、error return 用のゼロ値を作る。
	var account domain.Account
	return account
}

func emptyDomainOperator() domain.Operator {
	// Step 1: guardrail が禁止する domain composite literal を使わず、error return 用のゼロ値を作る。
	var operator domain.Operator
	return operator
}

func accountCreationDetailsJSON(rawEmail string) (string, error) {
	// Step 1: audit intent には入力値の追跡に必要な email だけを JSON として保持し、動的 error message や secret は含めない。
	details, err := json.Marshal(struct {
		RequestedEmail string `json:"requested_email"`
	}{RequestedEmail: rawEmail})
	if err != nil {
		return "", err
	}

	// Step 2: repository port が扱う primitive DTO として JSON 文字列へ変換する。
	return string(details), nil
}

func stableCodeForAccountCreationError(err error) string {
	// Step 1: duplicate email は Admin API が 409 に写像する stable error として監査する。
	if errors.Is(err, ErrAdminAccountDuplicateEmail) {
		return adminAccountStableCodeDuplicateEmail
	}

	// Step 2: audit target 不整合は repository 境界の内部不整合として別 stable code に分ける。
	if errors.Is(err, ErrAdminAccountAuditNotFound) {
		return adminAccountStableCodeAuditTargetMissing
	}

	// Step 3: その他の永続化 failure は詳細を露出しない stable code へ畳む。
	return adminAccountStableCodeRepositoryFailure
}

func stableCodeForAccountRootError(err error) string {
	// Step 1: AccountEmail などユーザー入力の domain validation failure は input 用 stable code として監査する。
	if errors.Is(err, ErrAdminAccountCreationInvalidInput) {
		return adminAccountStableCodeInvalidInput
	}

	// Step 2: ID generator や root 構築の内部不整合は詳細を隠した internal stable code として監査する。
	return adminAccountStableCodeInternal
}

func createdAccountResult(record AdminAccountRecord, audit AdminAuditRecord) AdminCreatedAccount {
	// Step 1: 新規作成 account は passkey credential を持たないため、response 用 snapshot は 0 として返す。
	return AdminCreatedAccount{
		AccountID:    record.AccountID,
		Email:        record.Email,
		Status:       record.Status,
		Locale:       record.Locale,
		PasskeyCount: 0,
		AuditID:      audit.AuditID,
		RequestID:    audit.RequestID,
		CreatedAt:    record.CreatedAt,
	}
}
