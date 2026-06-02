package operators

import (
	"context"
	"errors"
	"strings"
	"time"

	"www-template/packages/backend/internal/application/accounts"
	"www-template/packages/backend/internal/application/audit"
	adminauth "www-template/packages/backend/internal/application/auth"
	domain "www-template/packages/backend/internal/domain"
)

const (
	adminOperatorCreateAction                 = "operators:create"
	adminOperatorSetupAction                  = "operators:setup"
	adminOperatorTargetType                   = "operator"
	adminOperatorStableCodeInvalidInput       = "operator_invalid_input"
	adminOperatorStableCodeForbidden          = "operator_forbidden"
	adminOperatorStableCodeDeliveryFailure    = "operator_setup_delivery_failed"
	adminOperatorStableCodeRepositoryFailure  = "operator_repository_unavailable"
	adminOperatorStableCodeRegistrationFailed = "operator_registration_failed"
)

var (
	// ErrOperatorInvalidInput は operator setup / creation の入力が domain rule で拒否された場合の application error である。
	ErrOperatorInvalidInput = errors.New("admin operator invalid input")

	// ErrOperatorForbidden は bootstrap gate、setup token、または operator 権限が拒否された場合の application error である。
	ErrOperatorForbidden = errors.New("admin operator forbidden")

	// ErrOperatorConflict は初回 setup 済み環境など、現在状態と要求が衝突した場合の application error である。
	ErrOperatorConflict = errors.New("admin operator conflict")

	// ErrOperatorInternal は repository、delivery、WebAuthn provider、ID/secret 生成の失敗を隠蔽する application error である。
	ErrOperatorInternal = errors.New("admin operator internal")
)

// OperatorRepository は Admin operator setup / creation が必要とする永続化 port である。
//
// 役割:
//   - application 層から GORM や SQL を隠し、operator root と passkey credential の transaction を adapter に閉じる。
//   - setup token は opaque hash と expiry だけを保存し、平文 token は repository 境界へ渡さない。
//   - passkey 登録完了時は token 消費、passkey 保存、registration state 更新を同一 transaction で実行する。
type OperatorRepository interface {
	CountOperators(ctx context.Context) (int64, error)
	CreateInitialAdminOperatorWithPasskey(ctx context.Context, record InitialOperatorRecord) (OperatorRecord, error)
	CreateOperatorWithSetupToken(ctx context.Context, record OperatorCreationRecord) (OperatorRecord, error)
	DeletePendingOperatorSetup(ctx context.Context, operatorID string) error
	FindOperatorBySetupToken(ctx context.Context, now time.Time, match func(hash string) bool) (SetupRecord, error)
	CompleteOperatorSetupWithPasskey(ctx context.Context, record SetupCompletionRecord) (OperatorRecord, error)
}

// SetupTokenDeliveryPort は setup token 平文を backend-owned secure channel で配送する port である。
//
// 役割:
//   - operator creation response へ setup token 平文を返さないため、配送副作用を application 境界で抽象化する。
//   - 実装は SMTP などの secure delivery を使い、ログや error へ token 平文を含めてはならない。
type SetupTokenDeliveryPort interface {
	SendOperatorSetupToken(ctx context.Context, delivery SetupTokenDelivery) error
}

// SecretHasher は setup/bootstrap secret の保存用 hash を生成する port である。
//
// 役割:
//   - application service が bcrypt など platform 実装へ直接依存せず、secret 保存形式を runtime composition から受け取れるようにする。
//   - 平文 secret はこの port 呼び出し中だけ扱い、戻り値は保存用 hash のみに限定する。
//
// 引数:
//   - secretValue: setup token などの平文 secret。実装は前後空白の扱いと空値拒否を安全に処理する。
//
// 戻り値:
//   - string: 保存用 hash。
//   - error: secret が不正、または hash 生成に失敗した場合。
type SecretHasher interface {
	HashSecret(secretValue string) (string, error)
}

// SecretVerifier は保存済み secret hash と提示 secret を照合する port である。
//
// 役割:
//   - application service が bcrypt 比較実装へ直接依存せず、bootstrap/setup token の照合能力だけを受け取れるようにする。
//   - 照合失敗理由は bool に畳み、平文 secret や hash 構造を application error へ含めない。
//
// 引数:
//   - hash: 設定または DB から読み込んだ保存済み hash。
//   - secretValue: request や secure delivery flow から提示された平文 secret。
//
// 戻り値:
//   - bool: 保存済み hash と提示 secret が一致した場合だけ true。
type SecretVerifier interface {
	MatchesSecret(hash string, secretValue string) bool
}

// OperatorSessionIssuer は setup 完了直後に Admin operator session を発行する auth service 境界である。
type OperatorSessionIssuer interface {
	IssueOperatorSession(ctx context.Context, input adminauth.IssueOperatorSessionInput) (adminauth.OperatorSessionResult, error)
}

// BootstrapConfig は初回 Admin operator setup gate の application DTO である。
//
// 役割:
//   - platform/config 型を application 層へ import せず、必要な primitive だけを runtime composition から受け取る。
//   - SecretHash は opaque hash だけを保持し、bootstrap secret 平文は use case 入力として一時的に比較する。
type BootstrapConfig struct {
	Enabled    bool
	SecretHash string
	ExpiresAt  time.Time
}

// OperatorService は Admin operator の初回 setup、追加 setup、operator 作成を担う use case である。
type OperatorService struct {
	operators      OperatorRepository
	audits         *audit.AuditService
	ids            accounts.AccountIDGenerator
	secrets        adminauth.OpaqueTokenGenerator
	registrations  adminauth.OperatorPasskeyRegistrationProvider
	sessions       OperatorSessionIssuer
	delivery       SetupTokenDeliveryPort
	secretHasher   SecretHasher
	secretVerifier SecretVerifier
	clock          func() time.Time
	bootstrap      BootstrapConfig
	setupTokenTTL  time.Duration
}

// InitialSetupStartInput は初回 admin 作成 challenge 開始入力である。
type InitialSetupStartInput struct {
	Email           string
	DisplayName     string
	BootstrapSecret string
	RequestID       string
}

// InitialSetupFinishInput は初回 admin 作成完了入力である。
type InitialSetupFinishInput struct {
	Email           string
	DisplayName     string
	BootstrapSecret string
	RequestID       string
	Credential      adminauth.OperatorWebAuthnAttestationCredential
}

// SetupStartInput は追加 operator の setup challenge 開始入力である。
type SetupStartInput struct {
	SetupToken string
	RequestID  string
}

// SetupFinishInput は追加 operator の setup 完了入力である。
type SetupFinishInput struct {
	SetupToken string
	RequestID  string
	Credential adminauth.OperatorWebAuthnAttestationCredential
}

// CreateOperatorInput は admin が追加 operator を作成する入力である。
type CreateOperatorInput struct {
	Email                    string
	Role                     string
	RequestID                string
	OperatorID               string
	OperatorEmail            string
	OperatorRole             string
	OperatorActive           bool
	PasskeyRegistrationState string
}

// OperatorRecord は operator 永続化後の primitive snapshot である。
type OperatorRecord struct {
	OperatorID               string
	Email                    string
	Role                     string
	Active                   bool
	PasskeyRegistrationState string
	CreatedAt                time.Time
}

// SetupRecord は setup token に一致した operator の primitive snapshot である。
type SetupRecord struct {
	OperatorID string
	Email      string
	Role       string
	Active     bool
}

// PasskeyRecord は検証済み WebAuthn credential の保存 DTO である。
type PasskeyRecord struct {
	CredentialID     string
	CredentialHandle string
	PublicKey        []byte
	SignCount        uint32
	AAGUID           []byte
	BackupEligible   bool
	BackupState      bool
	Transports       []string
}

// InitialOperatorRecord は初回 admin と passkey を同一 transaction で保存する DTO である。
type InitialOperatorRecord struct {
	OperatorID  string
	Email       string
	Passkey     PasskeyRecord
	CompletedAt time.Time
}

// OperatorCreationRecord は追加 operator と setup token hash を保存する DTO である。
type OperatorCreationRecord struct {
	OperatorID          string
	Email               string
	Role                string
	SetupTokenHash      string
	SetupTokenExpiresAt time.Time
	CreatedAt           time.Time
}

// SetupCompletionRecord は setup token 消費と passkey 保存を同一 transaction で実行する DTO である。
type SetupCompletionRecord struct {
	OperatorID        string
	SetupTokenMatches func(hash string) bool
	Passkey           PasskeyRecord
	CompletedAt       time.Time
}

// SetupChallengeResult は passkey 登録 challenge response 用 DTO である。
type SetupChallengeResult struct {
	RequestID   string
	Challenge   string
	OptionsJSON []byte
}

// CreatedOperator は operator 作成 response 用 DTO である。
type CreatedOperator struct {
	RequestID      string
	AuditID        string
	DeliveryStatus string
	Operator       adminauth.OperatorDTO
}

// SetupTokenDelivery は secure delivery port に渡す token 配送 DTO である。
type SetupTokenDelivery struct {
	OperatorID string
	Email      string
	SetupToken string
	ExpiresAt  time.Time
	RequestID  string
}

// NewOperatorService は Admin operator setup / creation use case を構築する。
func NewOperatorService(operators OperatorRepository, audits *audit.AuditService, ids accounts.AccountIDGenerator, secrets adminauth.OpaqueTokenGenerator, registrations adminauth.OperatorPasskeyRegistrationProvider, sessions OperatorSessionIssuer, delivery SetupTokenDeliveryPort, secretHasher SecretHasher, secretVerifier SecretVerifier, clock func() time.Time, bootstrap BootstrapConfig, setupTokenTTL time.Duration) (*OperatorService, error) {
	// Step 1: 必須 port が欠けると setup や token delivery が fail-open になり得るため、構築時に拒否する。
	if operators == nil || audits == nil || ids == nil || secrets == nil || registrations == nil || sessions == nil || delivery == nil || secretHasher == nil || secretVerifier == nil || clock == nil {
		return nil, ErrOperatorInternal
	}
	if setupTokenTTL <= 0 {
		return nil, ErrOperatorInternal
	}

	// Step 2: 検証済み依存だけを service に保持し、HTTP adapter や repository へ業務 rule を散らさない。
	return &OperatorService{operators: operators, audits: audits, ids: ids, secrets: secrets, registrations: registrations, sessions: sessions, delivery: delivery, secretHasher: secretHasher, secretVerifier: secretVerifier, clock: clock, bootstrap: bootstrap, setupTokenTTL: setupTokenTTL}, nil
}

// StartInitialSetup は operator 0 件環境で初回 admin の passkey 登録 challenge を開始する。
func (s *OperatorService) StartInitialSetup(ctx context.Context, input InitialSetupStartInput) (SetupChallengeResult, error) {
	// Step 1: bootstrap gate と operator 件数を challenge 発行前に検証し、初回以外の環境で setup を開始しない。
	if err := s.validateBootstrap(ctx, input.BootstrapSecret); err != nil {
		return SetupChallengeResult{}, err
	}
	operatorID, email, displayName, err := s.initialSetupIdentity(input.RequestID, input.Email, input.DisplayName)
	if err != nil {
		return SetupChallengeResult{}, err
	}

	// Step 2: WebAuthn registration provider に discoverable credential + userVerification=required の challenge 発行を委譲する。
	challenge, err := s.registrations.BeginOperatorRegistration(ctx, adminauth.OperatorRegistrationChallengeInput{RequestID: input.RequestID, OperatorID: operatorID, Email: email, DisplayName: displayName})
	if err != nil {
		return SetupChallengeResult{}, ErrOperatorInternal
	}
	return SetupChallengeResult{RequestID: challenge.RequestID, Challenge: challenge.Challenge, OptionsJSON: challenge.OptionsJSON}, nil
}

// FinishInitialSetup は初回 admin operator と passkey credential を作成し、operator session を発行する。
func (s *OperatorService) FinishInitialSetup(ctx context.Context, input InitialSetupFinishInput) (adminauth.OperatorSessionResult, error) {
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
		return adminauth.OperatorSessionResult{}, ErrOperatorForbidden
	}
	passkey, err := s.passkeyRecord(registration)
	if err != nil {
		return adminauth.OperatorSessionResult{}, err
	}

	// Step 3: operator 作成と passkey 保存を repository transaction に委譲し、同時初回作成は conflict として扱う。
	created, err := s.operators.CreateInitialAdminOperatorWithPasskey(ctx, InitialOperatorRecord{OperatorID: operatorID, Email: email, Passkey: passkey, CompletedAt: s.clock().UTC()})
	if err != nil {
		return adminauth.OperatorSessionResult{}, mapAdminOperatorRepositoryError(err)
	}

	// Step 4: 登録済みに更新された Operator にだけ通常 Admin session を発行する。
	return s.sessions.IssueOperatorSession(ctx, adminauth.IssueOperatorSessionInput{OperatorID: created.OperatorID})
}

// StartOperatorSetup は setup token を検証し、追加 operator の passkey 登録 challenge を開始する。
func (s *OperatorService) StartOperatorSetup(ctx context.Context, input SetupStartInput) (SetupChallengeResult, error) {
	// Step 1: setup token は hash 比較 callback だけで照合し、repository へ平文 token を渡さない。
	operator, err := s.findOperatorForSetup(ctx, input.SetupToken)
	if err != nil {
		return SetupChallengeResult{}, err
	}

	// Step 2: pending operator だけに WebAuthn registration challenge を発行する。
	challenge, err := s.registrations.BeginOperatorRegistration(ctx, adminauth.OperatorRegistrationChallengeInput{RequestID: input.RequestID, OperatorID: operator.OperatorID, Email: operator.Email, DisplayName: operator.Email})
	if err != nil {
		return SetupChallengeResult{}, ErrOperatorInternal
	}
	return SetupChallengeResult{RequestID: challenge.RequestID, Challenge: challenge.Challenge, OptionsJSON: challenge.OptionsJSON}, nil
}

// FinishOperatorSetup は setup token を消費し、追加 operator の初回 passkey と operator session を発行する。
func (s *OperatorService) FinishOperatorSetup(ctx context.Context, input SetupFinishInput) (adminauth.OperatorSessionResult, error) {
	// Step 1: finish 時にも setup token を再検証し、start 後の期限切れや既消費 token を拒否する。
	operator, err := s.findOperatorForSetup(ctx, input.SetupToken)
	if err != nil {
		return adminauth.OperatorSessionResult{}, err
	}

	// Step 2: WebAuthn attestation を検証し、未検証 credential を repository へ保存しない。
	registration, err := s.registrations.FinishOperatorRegistration(ctx, input.RequestID, operator.OperatorID, input.Credential)
	if err != nil {
		return adminauth.OperatorSessionResult{}, ErrOperatorForbidden
	}
	passkey, err := s.passkeyRecord(registration)
	if err != nil {
		return adminauth.OperatorSessionResult{}, err
	}

	// Step 3: setup token の one-time consumption と passkey 保存を同一 transaction で実行する。
	trimmedToken := strings.TrimSpace(input.SetupToken)
	completed, err := s.operators.CompleteOperatorSetupWithPasskey(ctx, SetupCompletionRecord{OperatorID: operator.OperatorID, SetupTokenMatches: func(hash string) bool {
		return s.secretVerifier.MatchesSecret(hash, trimmedToken)
	}, Passkey: passkey, CompletedAt: s.clock().UTC()})
	if err != nil {
		return adminauth.OperatorSessionResult{}, mapAdminOperatorRepositoryError(err)
	}

	// Step 4: setup token 消費後の Operator にだけ通常 Admin session を発行する。
	return s.sessions.IssueOperatorSession(ctx, adminauth.IssueOperatorSessionInput{OperatorID: completed.OperatorID})
}

// CreateOperator は追加 operator を作成し、setup token を secure delivery port で配送する。
func (s *OperatorService) CreateOperator(ctx context.Context, input CreateOperatorInput) (CreatedOperator, error) {
	// Step 1: acting operator を domain object に復元し、operator 作成は admin role だけに限定する。
	acting, err := restoreAdminOperatorActor(input)
	if err != nil {
		return CreatedOperator{}, err
	}
	if !acting.HasPermission(domain.OperatorAuthPermissionOperatorsCreate.String()) {
		return CreatedOperator{}, ErrOperatorForbidden
	}

	// Step 2: 作成対象 operator の email/role と ID を domain constructor で検証する。
	operatorID, email, role, err := s.newOperatorIdentity(input.Email, input.Role)
	if err != nil {
		return CreatedOperator{}, err
	}

	// Step 3: mutation 前 audit intent を記録し、監査なし operator 作成を防ぐ。
	intent, err := s.audits.RecordMutationIntent(ctx, audit.IntentInput{OperatorID: acting.ID().String(), Action: adminOperatorCreateAction, TargetType: adminOperatorTargetType, TargetID: operatorID, RequestID: input.RequestID})
	if err != nil {
		return CreatedOperator{}, ErrOperatorInternal
	}

	// Step 4: 平文 setup token はこの use case 内だけに保持し、opaque hash と expiry だけを repository に渡す。
	plainToken, tokenHash, expiresAt, err := s.newSetupToken()
	if err != nil {
		return s.failOperatorCreation(ctx, intent.AuditID, err, adminOperatorStableCodeRepositoryFailure)
	}
	created, err := s.operators.CreateOperatorWithSetupToken(ctx, OperatorCreationRecord{OperatorID: operatorID, Email: email, Role: role, SetupTokenHash: tokenHash, SetupTokenExpiresAt: expiresAt, CreatedAt: s.clock().UTC()})
	if err != nil {
		return s.failOperatorCreation(ctx, intent.AuditID, err, adminOperatorStableCodeRepositoryFailure)
	}

	// Step 5: token 平文は secure delivery port にだけ渡し、配送失敗時は pending operator ごと削除して failed audit outcome にし、response body へ secret を出さない。
	if err := s.delivery.SendOperatorSetupToken(ctx, SetupTokenDelivery{OperatorID: created.OperatorID, Email: created.Email, SetupToken: plainToken, ExpiresAt: expiresAt, RequestID: input.RequestID}); err != nil {
		if deleteErr := s.operators.DeletePendingOperatorSetup(ctx, created.OperatorID); deleteErr != nil {
			return s.failOperatorCreation(ctx, intent.AuditID, ErrOperatorInternal, adminOperatorStableCodeRepositoryFailure)
		}
		return s.failOperatorCreation(ctx, intent.AuditID, ErrOperatorInternal, adminOperatorStableCodeDeliveryFailure)
	}
	if _, err := s.audits.CompleteMutationSucceeded(ctx, audit.CompletionInput{AuditID: intent.AuditID}); err != nil {
		if deleteErr := s.operators.DeletePendingOperatorSetup(ctx, created.OperatorID); deleteErr != nil {
			return CreatedOperator{}, ErrOperatorInternal
		}
		return CreatedOperator{}, ErrOperatorInternal
	}

	// Step 6: response は operator summary、delivery status、audit ID だけに限定し、setup token 平文を含めない。
	return CreatedOperator{RequestID: input.RequestID, AuditID: intent.AuditID, DeliveryStatus: "sent", Operator: adminauth.OperatorDTO{ID: created.OperatorID, Email: created.Email, Role: created.Role, Active: created.Active, PasskeyRegistrationState: created.PasskeyRegistrationState}}, nil
}

func (s *OperatorService) validateBootstrap(ctx context.Context, bootstrapSecret string) error {
	// Step 1: config gate が無効、期限切れ、または hash 未設定なら secret 比較へ進まず拒否する。
	if !s.bootstrap.Enabled || strings.TrimSpace(s.bootstrap.SecretHash) == "" || s.bootstrap.ExpiresAt.IsZero() || !s.clock().UTC().Before(s.bootstrap.ExpiresAt.UTC()) {
		return ErrOperatorForbidden
	}

	// Step 2: 既存 operator がある場合は初回 setup route を conflict とし、追加 operator flow へ誘導できる状態にする。
	count, err := s.operators.CountOperators(ctx)
	if err != nil {
		return ErrOperatorInternal
	}
	if count != 0 {
		return ErrOperatorConflict
	}

	// Step 3: opaque hash 比較の詳細を外へ出さず、secret 不一致は forbidden に畳む。
	if !s.secretVerifier.MatchesSecret(s.bootstrap.SecretHash, bootstrapSecret) {
		return ErrOperatorForbidden
	}
	return nil
}

func (s *OperatorService) initialSetupIdentity(requestID string, rawEmail string, rawDisplayName string) (string, string, string, error) {
	// Step 1: email は OperatorEmail domain object だけで正規化し、displayName は空なら email に倒す。
	email, err := domain.NewOperatorEmail(rawEmail)
	if err != nil {
		return "", "", "", ErrOperatorInvalidInput
	}
	displayName := strings.TrimSpace(rawDisplayName)
	if displayName == "" {
		displayName = email.String()
	}

	// Step 2: 初回 operator ID は start/finish の requestId と同じ ULID に固定し、WebAuthn session の user handle と DB 作成 ID を一致させる。
	operatorID, err := domain.NewOperatorID(requestID)
	if err != nil {
		return "", "", "", ErrOperatorInternal
	}
	return operatorID.String(), email.String(), displayName, nil
}

func (s *OperatorService) findOperatorForSetup(ctx context.Context, setupToken string) (SetupRecord, error) {
	// Step 1: 空 token は repository 探索を行わず、token 状態を区別しない forbidden に畳む。
	trimmedToken := strings.TrimSpace(setupToken)
	if trimmedToken == "" {
		return SetupRecord{}, ErrOperatorForbidden
	}

	// Step 2: opaque hash 比較 callback を repository に渡し、平文 token を DB query や audit に混ぜない。
	operator, err := s.operators.FindOperatorBySetupToken(ctx, s.clock().UTC(), func(hash string) bool {
		return s.secretVerifier.MatchesSecret(hash, trimmedToken)
	})
	if err != nil {
		return SetupRecord{}, ErrOperatorForbidden
	}
	return operator, nil
}

func (s *OperatorService) newOperatorIdentity(rawEmail string, rawRole string) (string, string, string, error) {
	// Step 1: 作成対象 email と role は domain value object で検証し、未知 role を fail-closed にする。
	email, err := domain.NewOperatorEmail(rawEmail)
	if err != nil {
		return "", "", "", ErrOperatorInvalidInput
	}
	role := domain.OperatorRole(rawRole)
	if err := role.Validate(); err != nil {
		return "", "", "", ErrOperatorInvalidInput
	}

	// Step 2: OperatorID は platform ID generator の出力を domain constructor に通してから保存する。
	rawID, err := s.ids.Next()
	if err != nil {
		return "", "", "", ErrOperatorInternal
	}
	operatorID, err := domain.NewOperatorID(rawID)
	if err != nil {
		return "", "", "", ErrOperatorInternal
	}
	return operatorID.String(), email.String(), string(role), nil
}

func (s *OperatorService) newSetupToken() (string, string, time.Time, error) {
	// Step 1: 平文 setup token は暗号学的乱数 generator から発行し、ログや response には出さない。
	plainToken, err := s.secrets.NewToken()
	if err != nil {
		return "", "", time.Time{}, ErrOperatorInternal
	}

	// Step 2: DB 保存用には bcrypt hash だけを生成し、平文 token は secure delivery port へ渡すまでの一時値に限定する。
	hash, err := s.hashAdminSecret(plainToken)
	if err != nil {
		return "", "", time.Time{}, ErrOperatorInternal
	}
	return plainToken, hash, s.clock().UTC().Add(s.setupTokenTTL), nil
}

func (s *OperatorService) hashAdminSecret(secretValue string) (string, error) {
	// Step 1: copy/paste 由来の前後空白だけを除去し、空 secret は hash 化せず拒否する。
	trimmedSecret := strings.TrimSpace(secretValue)
	if trimmedSecret == "" {
		return "", ErrOperatorInvalidInput
	}

	// Step 2: 保存形式の具体実装は SecretHasher port に委譲し、application 層から platform 依存を排除する。
	hash, err := s.secretHasher.HashSecret(trimmedSecret)
	if err != nil {
		return "", ErrOperatorInternal
	}
	return string(hash), nil
}

func (s *OperatorService) passkeyRecord(registration adminauth.OperatorPasskeyRegistration) (PasskeyRecord, error) {
	// Step 1: passkey credential ID は operator credential 専用 ULID として発行し、credential handle と分離する。
	credentialID, err := s.ids.Next()
	if err != nil {
		return PasskeyRecord{}, ErrOperatorInternal
	}
	if err := domain.ValidateAuthID(credentialID); err != nil {
		return PasskeyRecord{}, ErrOperatorInternal
	}
	if strings.TrimSpace(registration.CredentialHandle) == "" || len(registration.PublicKey) == 0 {
		return PasskeyRecord{}, ErrOperatorForbidden
	}
	return PasskeyRecord{CredentialID: credentialID, CredentialHandle: registration.CredentialHandle, PublicKey: registration.PublicKey, SignCount: registration.SignCount, AAGUID: registration.AAGUID, BackupEligible: registration.BackupEligible, BackupState: registration.BackupState, Transports: registration.Transports}, nil
}

func (s *OperatorService) failOperatorCreation(ctx context.Context, auditID string, original error, stableCode string) (CreatedOperator, error) {
	// Step 1: operator 作成または delivery 失敗を failed audit outcome として記録し、token 平文を監査へ含めない。
	if _, err := s.audits.CompleteMutationFailed(ctx, audit.FailureInput{AuditID: auditID, StableErrorCode: stableCode}); err != nil {
		return CreatedOperator{}, ErrOperatorInternal
	}

	// Step 2: 監査保存後は元 error の抽象分類だけを handler へ返す。
	return CreatedOperator{}, original
}

func restoreAdminOperatorActor(input CreateOperatorInput) (domain.Operator, error) {
	// Step 1: acting operator の primitive snapshot を domain.Operator に復元し、Product account role を混ぜない。
	operatorID, err := domain.NewOperatorID(input.OperatorID)
	if err != nil {
		return emptyAdminOperator(), ErrOperatorForbidden
	}
	operatorEmail, err := domain.NewOperatorEmail(input.OperatorEmail)
	if err != nil {
		return emptyAdminOperator(), ErrOperatorForbidden
	}
	operator, err := domain.NewOperator(operatorID, operatorEmail, domain.OperatorRole(input.OperatorRole), input.OperatorActive, domain.OperatorPasskeyRegistrationState(input.PasskeyRegistrationState))
	if err != nil {
		return emptyAdminOperator(), ErrOperatorForbidden
	}
	if !operator.Active() || operator.PasskeyRegistrationState() != domain.OperatorPasskeyRegistrationRegistered {
		return emptyAdminOperator(), ErrOperatorForbidden
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
	case errors.Is(err, ErrOperatorConflict):
		return ErrOperatorConflict
	case errors.Is(err, ErrOperatorForbidden):
		return ErrOperatorForbidden
	case errors.Is(err, ErrOperatorInvalidInput):
		return ErrOperatorInvalidInput
	default:
		return ErrOperatorInternal
	}
}
