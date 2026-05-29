package application

import (
	"context"
	"errors"
	"strings"
	"time"

	adminauth "www-template/packages/backend/internal/application/admin/auth"
	domain "www-template/packages/backend/internal/domain"
	secrethash "www-template/packages/backend/internal/platform/secret"
)

const (
	adminOperatorCreateAction                 = "operators:create"
	adminOperatorSetupAction                  = "operators:setup"
	adminOperatorTargetType                   = "operator"
	adminOperatorSetupTokenTTL                = 24 * time.Hour
	adminOperatorStableCodeInvalidInput       = "operator_invalid_input"
	adminOperatorStableCodeForbidden          = "operator_forbidden"
	adminOperatorStableCodeDeliveryFailure    = "operator_setup_delivery_failed"
	adminOperatorStableCodeRepositoryFailure  = "operator_repository_unavailable"
	adminOperatorStableCodeRegistrationFailed = "operator_registration_failed"
)

var (
	// ErrAdminOperatorInvalidInput は operator setup / creation の入力が domain rule で拒否された場合の application error である。
	ErrAdminOperatorInvalidInput = errors.New("admin operator invalid input")

	// ErrAdminOperatorForbidden は bootstrap gate、setup token、または operator 権限が拒否された場合の application error である。
	ErrAdminOperatorForbidden = errors.New("admin operator forbidden")

	// ErrAdminOperatorConflict は初回 setup 済み環境など、現在状態と要求が衝突した場合の application error である。
	ErrAdminOperatorConflict = errors.New("admin operator conflict")

	// ErrAdminOperatorInternal は repository、delivery、WebAuthn provider、ID/secret 生成の失敗を隠蔽する application error である。
	ErrAdminOperatorInternal = errors.New("admin operator internal")
)

// AdminOperatorRepository は Admin operator setup / creation が必要とする永続化 port である。
//
// 役割:
//   - application 層から GORM や SQL を隠し、operator root と passkey credential の transaction を adapter に閉じる。
//   - setup token は opaque hash と expiry だけを保存し、平文 token は repository 境界へ渡さない。
//   - passkey 登録完了時は token 消費、passkey 保存、registration state 更新を同一 transaction で実行する。
type AdminOperatorRepository interface {
	CountOperators(ctx context.Context) (int64, error)
	CreateInitialAdminOperatorWithPasskey(ctx context.Context, record AdminInitialOperatorRecord) (AdminOperatorRecord, error)
	CreateOperatorWithSetupToken(ctx context.Context, record AdminOperatorCreationRecord) (AdminOperatorRecord, error)
	DeletePendingOperatorSetup(ctx context.Context, operatorID string) error
	FindOperatorBySetupToken(ctx context.Context, now time.Time, match func(hash string) bool) (AdminOperatorSetupRecord, error)
	CompleteOperatorSetupWithPasskey(ctx context.Context, record AdminOperatorSetupCompletionRecord) (AdminOperatorRecord, error)
}

// AdminSetupTokenDelivery は setup token 平文を backend-owned secure channel で配送する port である。
//
// 役割:
//   - operator creation response へ setup token 平文を返さないため、配送副作用を application 境界で抽象化する。
//   - 実装は SMTP などの secure delivery を使い、ログや error へ token 平文を含めてはならない。
type AdminSetupTokenDelivery interface {
	SendOperatorSetupToken(ctx context.Context, delivery AdminOperatorSetupTokenDelivery) error
}

// OperatorSessionIssuer は setup 完了直後に Admin operator session を発行する auth service 境界である。
type OperatorSessionIssuer interface {
	IssueOperatorSessionForSetup(ctx context.Context, input adminauth.IssueOperatorSessionInput) (adminauth.OperatorSessionResult, error)
}

// AdminOperatorBootstrapConfig は初回 Admin operator setup gate の application DTO である。
//
// 役割:
//   - platform/config 型を application 層へ import せず、必要な primitive だけを runtime composition から受け取る。
//   - SecretHash は opaque hash だけを保持し、bootstrap secret 平文は use case 入力として一時的に比較する。
type AdminOperatorBootstrapConfig struct {
	Enabled    bool
	SecretHash string
	ExpiresAt  time.Time
}

// AdminOperatorService は Admin operator の初回 setup、追加 setup、operator 作成を担う use case である。
type AdminOperatorService struct {
	operators     AdminOperatorRepository
	audits        *AdminAuditService
	ids           AdminAccountIDGenerator
	secrets       adminauth.OpaqueTokenGenerator
	registrations adminauth.OperatorPasskeyRegistrationProvider
	sessions      OperatorSessionIssuer
	delivery      AdminSetupTokenDelivery
	clock         func() time.Time
	bootstrap     AdminOperatorBootstrapConfig
}

// AdminInitialSetupStartInput は初回 admin 作成 challenge 開始入力である。
type AdminInitialSetupStartInput struct {
	Email           string
	DisplayName     string
	BootstrapSecret string
	RequestID       string
}

// AdminInitialSetupFinishInput は初回 admin 作成完了入力である。
type AdminInitialSetupFinishInput struct {
	Email           string
	DisplayName     string
	BootstrapSecret string
	RequestID       string
	Credential      adminauth.OperatorWebAuthnAttestationCredential
}

// AdminOperatorSetupStartInput は追加 operator の setup challenge 開始入力である。
type AdminOperatorSetupStartInput struct {
	SetupToken string
	RequestID  string
}

// AdminOperatorSetupFinishInput は追加 operator の setup 完了入力である。
type AdminOperatorSetupFinishInput struct {
	SetupToken string
	RequestID  string
	Credential adminauth.OperatorWebAuthnAttestationCredential
}

// AdminCreateOperatorInput は admin が追加 operator を作成する入力である。
type AdminCreateOperatorInput struct {
	Email                    string
	Role                     string
	RequestID                string
	OperatorID               string
	OperatorEmail            string
	OperatorRole             string
	OperatorActive           bool
	PasskeyRegistrationState string
}

// AdminOperatorRecord は operator 永続化後の primitive snapshot である。
type AdminOperatorRecord struct {
	OperatorID               string
	Email                    string
	Role                     string
	Active                   bool
	PasskeyRegistrationState string
	CreatedAt                time.Time
}

// AdminOperatorSetupRecord は setup token に一致した operator の primitive snapshot である。
type AdminOperatorSetupRecord struct {
	OperatorID string
	Email      string
	Role       string
	Active     bool
}

// AdminOperatorPasskeyRecord は検証済み WebAuthn credential の保存 DTO である。
type AdminOperatorPasskeyRecord struct {
	CredentialID     string
	CredentialHandle string
	PublicKey        []byte
	SignCount        uint32
	AAGUID           []byte
	BackupEligible   bool
	BackupState      bool
	Transports       []string
}

// AdminInitialOperatorRecord は初回 admin と passkey を同一 transaction で保存する DTO である。
type AdminInitialOperatorRecord struct {
	OperatorID  string
	Email       string
	Passkey     AdminOperatorPasskeyRecord
	CompletedAt time.Time
}

// AdminOperatorCreationRecord は追加 operator と setup token hash を保存する DTO である。
type AdminOperatorCreationRecord struct {
	OperatorID          string
	Email               string
	Role                string
	SetupTokenHash      string
	SetupTokenExpiresAt time.Time
	CreatedAt           time.Time
}

// AdminOperatorSetupCompletionRecord は setup token 消費と passkey 保存を同一 transaction で実行する DTO である。
type AdminOperatorSetupCompletionRecord struct {
	OperatorID        string
	SetupTokenMatches func(hash string) bool
	Passkey           AdminOperatorPasskeyRecord
	CompletedAt       time.Time
}

// AdminOperatorSetupChallengeResult は passkey 登録 challenge response 用 DTO である。
type AdminOperatorSetupChallengeResult struct {
	RequestID   string
	Challenge   string
	OptionsJSON []byte
}

// AdminCreatedOperator は operator 作成 response 用 DTO である。
type AdminCreatedOperator struct {
	RequestID      string
	AuditID        string
	DeliveryStatus string
	Operator       adminauth.OperatorDTO
}

// AdminOperatorSetupTokenDelivery は secure delivery port に渡す token 配送 DTO である。
type AdminOperatorSetupTokenDelivery struct {
	OperatorID string
	Email      string
	SetupToken string
	ExpiresAt  time.Time
	RequestID  string
}

// NewAdminOperatorService は Admin operator setup / creation use case を構築する。
func NewAdminOperatorService(operators AdminOperatorRepository, audits *AdminAuditService, ids AdminAccountIDGenerator, secrets adminauth.OpaqueTokenGenerator, registrations adminauth.OperatorPasskeyRegistrationProvider, sessions OperatorSessionIssuer, delivery AdminSetupTokenDelivery, clock func() time.Time, bootstrap AdminOperatorBootstrapConfig) (*AdminOperatorService, error) {
	// Step 1: 必須 port が欠けると setup や token delivery が fail-open になり得るため、構築時に拒否する。
	if operators == nil || audits == nil || ids == nil || secrets == nil || registrations == nil || sessions == nil || delivery == nil || clock == nil {
		return nil, ErrAdminOperatorInternal
	}

	// Step 2: 検証済み依存だけを service に保持し、HTTP adapter や repository へ業務 rule を散らさない。
	return &AdminOperatorService{operators: operators, audits: audits, ids: ids, secrets: secrets, registrations: registrations, sessions: sessions, delivery: delivery, clock: clock, bootstrap: bootstrap}, nil
}

// StartInitialSetup は operator 0 件環境で初回 admin の passkey 登録 challenge を開始する。
func (s *AdminOperatorService) StartInitialSetup(ctx context.Context, input AdminInitialSetupStartInput) (AdminOperatorSetupChallengeResult, error) {
	// Step 1: bootstrap gate と operator 件数を challenge 発行前に検証し、初回以外の環境で setup を開始しない。
	if err := s.validateBootstrap(ctx, input.BootstrapSecret); err != nil {
		return AdminOperatorSetupChallengeResult{}, err
	}
	operatorID, email, displayName, err := s.initialSetupIdentity(input.RequestID, input.Email, input.DisplayName)
	if err != nil {
		return AdminOperatorSetupChallengeResult{}, err
	}

	// Step 2: WebAuthn registration provider に discoverable credential + userVerification=required の challenge 発行を委譲する。
	challenge, err := s.registrations.BeginOperatorRegistration(ctx, adminauth.OperatorRegistrationChallengeInput{RequestID: input.RequestID, OperatorID: operatorID, Email: email, DisplayName: displayName})
	if err != nil {
		return AdminOperatorSetupChallengeResult{}, ErrAdminOperatorInternal
	}
	return AdminOperatorSetupChallengeResult{RequestID: challenge.RequestID, Challenge: challenge.Challenge, OptionsJSON: challenge.OptionsJSON}, nil
}

// FinishInitialSetup は初回 admin operator と passkey credential を作成し、operator session を発行する。
func (s *AdminOperatorService) FinishInitialSetup(ctx context.Context, input AdminInitialSetupFinishInput) (adminauth.OperatorSessionResult, error) {
	// Step 1: start 後の期限切れや競合を防ぐため、finish 時にも bootstrap gate と operator 件数を再検証する。
	if err := s.validateBootstrap(ctx, input.BootstrapSecret); err != nil {
		return adminauth.OperatorSessionResult{}, err
	}
	operatorID, email, _, err := s.initialSetupIdentity(input.RequestID, input.Email, input.DisplayName)
	if err != nil {
		return adminauth.OperatorSessionResult{}, err
	}

	// Step 2: attestation を WebAuthn provider で検証し、検証済み credential data だけを保存 DTO へ変換する。
	registration, err := s.registrations.FinishOperatorRegistration(ctx, input.RequestID, operatorID, input.Credential)
	if err != nil {
		return adminauth.OperatorSessionResult{}, ErrAdminOperatorForbidden
	}
	passkey, err := s.passkeyRecord(registration)
	if err != nil {
		return adminauth.OperatorSessionResult{}, err
	}

	// Step 3: operator 作成と passkey 保存を repository transaction に委譲し、同時初回作成は conflict として扱う。
	created, err := s.operators.CreateInitialAdminOperatorWithPasskey(ctx, AdminInitialOperatorRecord{OperatorID: operatorID, Email: email, Passkey: passkey, CompletedAt: s.clock().UTC()})
	if err != nil {
		return adminauth.OperatorSessionResult{}, mapAdminOperatorRepositoryError(err)
	}

	// Step 4: 登録済みに更新された Operator にだけ通常 Admin session を発行する。
	return s.sessions.IssueOperatorSessionForSetup(ctx, adminauth.IssueOperatorSessionInput{OperatorID: created.OperatorID})
}

// StartOperatorSetup は setup token を検証し、追加 operator の passkey 登録 challenge を開始する。
func (s *AdminOperatorService) StartOperatorSetup(ctx context.Context, input AdminOperatorSetupStartInput) (AdminOperatorSetupChallengeResult, error) {
	// Step 1: setup token は hash 比較 callback だけで照合し、repository へ平文 token を渡さない。
	operator, err := s.findOperatorForSetup(ctx, input.SetupToken)
	if err != nil {
		return AdminOperatorSetupChallengeResult{}, err
	}

	// Step 2: pending operator だけに WebAuthn registration challenge を発行する。
	challenge, err := s.registrations.BeginOperatorRegistration(ctx, adminauth.OperatorRegistrationChallengeInput{RequestID: input.RequestID, OperatorID: operator.OperatorID, Email: operator.Email, DisplayName: operator.Email})
	if err != nil {
		return AdminOperatorSetupChallengeResult{}, ErrAdminOperatorInternal
	}
	return AdminOperatorSetupChallengeResult{RequestID: challenge.RequestID, Challenge: challenge.Challenge, OptionsJSON: challenge.OptionsJSON}, nil
}

// FinishOperatorSetup は setup token を消費し、追加 operator の初回 passkey と operator session を発行する。
func (s *AdminOperatorService) FinishOperatorSetup(ctx context.Context, input AdminOperatorSetupFinishInput) (adminauth.OperatorSessionResult, error) {
	// Step 1: finish 時にも setup token を再検証し、start 後の期限切れや既消費 token を拒否する。
	operator, err := s.findOperatorForSetup(ctx, input.SetupToken)
	if err != nil {
		return adminauth.OperatorSessionResult{}, err
	}

	// Step 2: WebAuthn attestation を検証し、未検証 credential を repository へ保存しない。
	registration, err := s.registrations.FinishOperatorRegistration(ctx, input.RequestID, operator.OperatorID, input.Credential)
	if err != nil {
		return adminauth.OperatorSessionResult{}, ErrAdminOperatorForbidden
	}
	passkey, err := s.passkeyRecord(registration)
	if err != nil {
		return adminauth.OperatorSessionResult{}, err
	}

	// Step 3: setup token の one-time consumption と passkey 保存を同一 transaction で実行する。
	trimmedToken := strings.TrimSpace(input.SetupToken)
	completed, err := s.operators.CompleteOperatorSetupWithPasskey(ctx, AdminOperatorSetupCompletionRecord{OperatorID: operator.OperatorID, SetupTokenMatches: func(hash string) bool {
		return adminSecretMatchesHash(hash, trimmedToken)
	}, Passkey: passkey, CompletedAt: s.clock().UTC()})
	if err != nil {
		return adminauth.OperatorSessionResult{}, mapAdminOperatorRepositoryError(err)
	}

	// Step 4: setup token 消費後の Operator にだけ通常 Admin session を発行する。
	return s.sessions.IssueOperatorSessionForSetup(ctx, adminauth.IssueOperatorSessionInput{OperatorID: completed.OperatorID})
}

// CreateOperator は追加 operator を作成し、setup token を secure delivery port で配送する。
func (s *AdminOperatorService) CreateOperator(ctx context.Context, input AdminCreateOperatorInput) (AdminCreatedOperator, error) {
	// Step 1: acting operator を domain object に復元し、operator 作成は admin role だけに限定する。
	acting, err := restoreAdminOperatorActor(input)
	if err != nil {
		return AdminCreatedOperator{}, err
	}
	if acting.Role() != domain.OperatorRoleAdmin {
		return AdminCreatedOperator{}, ErrAdminOperatorForbidden
	}

	// Step 2: 作成対象 operator の email/role と ID を domain constructor で検証する。
	operatorID, email, role, err := s.newOperatorIdentity(input.Email, input.Role)
	if err != nil {
		return AdminCreatedOperator{}, err
	}

	// Step 3: mutation 前 audit intent を記録し、監査なし operator 作成を防ぐ。
	intent, err := s.audits.RecordMutationIntent(ctx, AdminAuditIntentInput{OperatorID: acting.ID().String(), Action: adminOperatorCreateAction, TargetType: adminOperatorTargetType, TargetID: operatorID, RequestID: input.RequestID})
	if err != nil {
		return AdminCreatedOperator{}, ErrAdminOperatorInternal
	}

	// Step 4: 平文 setup token はこの use case 内だけに保持し、opaque hash と expiry だけを repository に渡す。
	plainToken, tokenHash, expiresAt, err := s.newSetupToken()
	if err != nil {
		return s.failOperatorCreation(ctx, intent.AuditID, err, adminOperatorStableCodeRepositoryFailure)
	}
	created, err := s.operators.CreateOperatorWithSetupToken(ctx, AdminOperatorCreationRecord{OperatorID: operatorID, Email: email, Role: role, SetupTokenHash: tokenHash, SetupTokenExpiresAt: expiresAt, CreatedAt: s.clock().UTC()})
	if err != nil {
		return s.failOperatorCreation(ctx, intent.AuditID, err, adminOperatorStableCodeRepositoryFailure)
	}

	// Step 5: token 平文は secure delivery port にだけ渡し、配送失敗時は pending operator ごと削除して failed audit outcome にし、response body へ secret を出さない。
	if err := s.delivery.SendOperatorSetupToken(ctx, AdminOperatorSetupTokenDelivery{OperatorID: created.OperatorID, Email: created.Email, SetupToken: plainToken, ExpiresAt: expiresAt, RequestID: input.RequestID}); err != nil {
		if deleteErr := s.operators.DeletePendingOperatorSetup(ctx, created.OperatorID); deleteErr != nil {
			return s.failOperatorCreation(ctx, intent.AuditID, ErrAdminOperatorInternal, adminOperatorStableCodeRepositoryFailure)
		}
		return s.failOperatorCreation(ctx, intent.AuditID, ErrAdminOperatorInternal, adminOperatorStableCodeDeliveryFailure)
	}
	if _, err := s.audits.CompleteMutationSucceeded(ctx, AdminAuditCompletionInput{AuditID: intent.AuditID}); err != nil {
		if deleteErr := s.operators.DeletePendingOperatorSetup(ctx, created.OperatorID); deleteErr != nil {
			return AdminCreatedOperator{}, ErrAdminOperatorInternal
		}
		return AdminCreatedOperator{}, ErrAdminOperatorInternal
	}

	// Step 6: response は operator summary、delivery status、audit ID だけに限定し、setup token 平文を含めない。
	return AdminCreatedOperator{RequestID: input.RequestID, AuditID: intent.AuditID, DeliveryStatus: "sent", Operator: adminauth.OperatorDTO{ID: created.OperatorID, Email: created.Email, Role: created.Role, Active: created.Active, PasskeyRegistrationState: created.PasskeyRegistrationState}}, nil
}

func (s *AdminOperatorService) validateBootstrap(ctx context.Context, bootstrapSecret string) error {
	// Step 1: config gate が無効、期限切れ、または hash 未設定なら secret 比較へ進まず拒否する。
	if !s.bootstrap.Enabled || strings.TrimSpace(s.bootstrap.SecretHash) == "" || s.bootstrap.ExpiresAt.IsZero() || !s.clock().UTC().Before(s.bootstrap.ExpiresAt.UTC()) {
		return ErrAdminOperatorForbidden
	}

	// Step 2: 既存 operator がある場合は初回 setup route を conflict とし、追加 operator flow へ誘導できる状態にする。
	count, err := s.operators.CountOperators(ctx)
	if err != nil {
		return ErrAdminOperatorInternal
	}
	if count != 0 {
		return ErrAdminOperatorConflict
	}

	// Step 3: opaque hash 比較の詳細を外へ出さず、secret 不一致は forbidden に畳む。
	if !adminSecretMatchesHash(s.bootstrap.SecretHash, bootstrapSecret) {
		return ErrAdminOperatorForbidden
	}
	return nil
}

func (s *AdminOperatorService) initialSetupIdentity(requestID string, rawEmail string, rawDisplayName string) (string, string, string, error) {
	// Step 1: email は OperatorEmail domain object だけで正規化し、displayName は空なら email に倒す。
	email, err := domain.NewOperatorEmail(rawEmail)
	if err != nil {
		return "", "", "", ErrAdminOperatorInvalidInput
	}
	displayName := strings.TrimSpace(rawDisplayName)
	if displayName == "" {
		displayName = email.String()
	}

	// Step 2: 初回 operator ID は start/finish の requestId と同じ ULID に固定し、WebAuthn session の user handle と DB 作成 ID を一致させる。
	operatorID, err := domain.NewOperatorID(requestID)
	if err != nil {
		return "", "", "", ErrAdminOperatorInternal
	}
	return operatorID.String(), email.String(), displayName, nil
}

func (s *AdminOperatorService) findOperatorForSetup(ctx context.Context, setupToken string) (AdminOperatorSetupRecord, error) {
	// Step 1: 空 token は repository 探索を行わず、token 状態を区別しない forbidden に畳む。
	trimmedToken := strings.TrimSpace(setupToken)
	if trimmedToken == "" {
		return AdminOperatorSetupRecord{}, ErrAdminOperatorForbidden
	}

	// Step 2: opaque hash 比較 callback を repository に渡し、平文 token を DB query や audit に混ぜない。
	operator, err := s.operators.FindOperatorBySetupToken(ctx, s.clock().UTC(), func(hash string) bool {
		return adminSecretMatchesHash(hash, trimmedToken)
	})
	if err != nil {
		return AdminOperatorSetupRecord{}, ErrAdminOperatorForbidden
	}
	return operator, nil
}

func (s *AdminOperatorService) newOperatorIdentity(rawEmail string, rawRole string) (string, string, string, error) {
	// Step 1: 作成対象 email と role は domain value object で検証し、未知 role を fail-closed にする。
	email, err := domain.NewOperatorEmail(rawEmail)
	if err != nil {
		return "", "", "", ErrAdminOperatorInvalidInput
	}
	role := domain.OperatorRole(rawRole)
	if err := role.Validate(); err != nil {
		return "", "", "", ErrAdminOperatorInvalidInput
	}

	// Step 2: OperatorID は platform ID generator の出力を domain constructor に通してから保存する。
	rawID, err := s.ids.Next()
	if err != nil {
		return "", "", "", ErrAdminOperatorInternal
	}
	operatorID, err := domain.NewOperatorID(rawID)
	if err != nil {
		return "", "", "", ErrAdminOperatorInternal
	}
	return operatorID.String(), email.String(), string(role), nil
}

func (s *AdminOperatorService) newSetupToken() (string, string, time.Time, error) {
	// Step 1: 平文 setup token は暗号学的乱数 generator から発行し、ログや response には出さない。
	plainToken, err := s.secrets.NewOpaqueToken()
	if err != nil {
		return "", "", time.Time{}, ErrAdminOperatorInternal
	}

	// Step 2: DB 保存用には bcrypt hash だけを生成し、平文 token は secure delivery port へ渡すまでの一時値に限定する。
	hash, err := adminHashSecret(plainToken)
	if err != nil {
		return "", "", time.Time{}, ErrAdminOperatorInternal
	}
	return plainToken, hash, s.clock().UTC().Add(adminOperatorSetupTokenTTL), nil
}

func adminHashSecret(secretValue string) (string, error) {
	// Step 1: copy/paste 由来の前後空白だけを除去し、空 secret は hash 化せず拒否する。
	trimmedSecret := strings.TrimSpace(secretValue)
	if trimmedSecret == "" {
		return "", ErrAdminOperatorInvalidInput
	}

	// Step 2: bcrypt 実装は platform helper に閉じ、application 層は secret 保存形式の policy だけを表現する。
	hash, err := secrethash.HashBcryptSecret(trimmedSecret)
	if err != nil {
		return "", ErrAdminOperatorInternal
	}
	return string(hash), nil
}

func adminSecretMatchesHash(hash string, secretValue string) bool {
	// Step 1: bootstrap secret と setup token の照合は bcrypt helper だけに委譲し、高速 digest 形式は一致させない。
	return secrethash.MatchesBcryptSecret(hash, secretValue)
}

func (s *AdminOperatorService) passkeyRecord(registration adminauth.OperatorPasskeyRegistration) (AdminOperatorPasskeyRecord, error) {
	// Step 1: passkey credential ID は operator credential 専用 ULID として発行し、credential handle と分離する。
	credentialID, err := s.ids.Next()
	if err != nil {
		return AdminOperatorPasskeyRecord{}, ErrAdminOperatorInternal
	}
	if err := domain.ValidateAuthID(credentialID); err != nil {
		return AdminOperatorPasskeyRecord{}, ErrAdminOperatorInternal
	}
	if strings.TrimSpace(registration.CredentialHandle) == "" || len(registration.PublicKey) == 0 {
		return AdminOperatorPasskeyRecord{}, ErrAdminOperatorForbidden
	}
	return AdminOperatorPasskeyRecord{CredentialID: credentialID, CredentialHandle: registration.CredentialHandle, PublicKey: registration.PublicKey, SignCount: registration.SignCount, AAGUID: registration.AAGUID, BackupEligible: registration.BackupEligible, BackupState: registration.BackupState, Transports: registration.Transports}, nil
}

func (s *AdminOperatorService) failOperatorCreation(ctx context.Context, auditID string, original error, stableCode string) (AdminCreatedOperator, error) {
	// Step 1: operator 作成または delivery 失敗を failed audit outcome として記録し、token 平文を監査へ含めない。
	if _, err := s.audits.CompleteMutationFailed(ctx, AdminAuditFailureInput{AuditID: auditID, StableErrorCode: stableCode}); err != nil {
		return AdminCreatedOperator{}, ErrAdminOperatorInternal
	}

	// Step 2: 監査保存後は元 error の抽象分類だけを handler へ返す。
	return AdminCreatedOperator{}, original
}

func restoreAdminOperatorActor(input AdminCreateOperatorInput) (domain.Operator, error) {
	// Step 1: acting operator の primitive snapshot を domain.Operator に復元し、Product account role を混ぜない。
	operatorID, err := domain.NewOperatorID(input.OperatorID)
	if err != nil {
		return emptyAdminOperator(), ErrAdminOperatorForbidden
	}
	operatorEmail, err := domain.NewOperatorEmail(input.OperatorEmail)
	if err != nil {
		return emptyAdminOperator(), ErrAdminOperatorForbidden
	}
	operator, err := domain.NewOperator(operatorID, operatorEmail, domain.OperatorRole(input.OperatorRole), input.OperatorActive, domain.OperatorPasskeyRegistrationState(input.PasskeyRegistrationState))
	if err != nil {
		return emptyAdminOperator(), ErrAdminOperatorForbidden
	}
	if !operator.Active() || operator.PasskeyRegistrationState() != domain.OperatorPasskeyRegistrationRegistered {
		return emptyAdminOperator(), ErrAdminOperatorForbidden
	}
	return operator, nil
}

func emptyAdminOperator() domain.Operator {
	// Step 1: guardrail が禁止する domain composite literal を使わず、error return 用のゼロ値を作る。
	var operator domain.Operator
	return operator
}

func mapAdminOperatorRepositoryError(err error) error {
	// Step 1: repository の抽象 error を HTTP adapter が扱う application error へ畳む。
	switch {
	case errors.Is(err, ErrAdminOperatorConflict):
		return ErrAdminOperatorConflict
	case errors.Is(err, ErrAdminOperatorForbidden):
		return ErrAdminOperatorForbidden
	case errors.Is(err, ErrAdminOperatorInvalidInput):
		return ErrAdminOperatorInvalidInput
	default:
		return ErrAdminOperatorInternal
	}
}
