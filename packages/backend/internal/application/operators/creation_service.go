package operators

import (
	"context"
	"errors"
	"strings"
	"time"

	"www-template/packages/backend/internal/application/accounts"
	"www-template/packages/backend/internal/application/audit"
	authapplication "www-template/packages/backend/internal/application/auth"
	domain "www-template/packages/backend/internal/domain"
)

// ─── Use case 定数 ──────────────────────────────────────────────────────────

const (
	// operatorCreateAction は operator 作成時の audit action である。
	operatorCreateAction = "operators:create"
	// operatorTargetType は audit target type として使う operator の種別である。
	operatorTargetType = "operator"
	// operatorStableCodeDeliveryFailure は token 配送失敗時の stable error code である。
	operatorStableCodeDeliveryFailure = "operator_setup_delivery_failed"
	// operatorStableCodeRepositoryFailure は repository 障害時の stable error code である。
	operatorStableCodeRepositoryFailure = "operator_repository_unavailable"
)

// ─── Service ───────────────────────────────────────────────────────────────

// OperatorService は Operator の初回 setup、追加 setup、operator 作成を担う use case である。
//
// 役割:
//   - bootstrap gate、setup token、passkey 登録、operator 作成の orchestration を application 境界で担う。
//   - 永続化、配送、WebAuthn provider、ID/secret 生成は port interface に委譲し、具体実装へ直接依存しない。
//   - 平文 setup token は use case 内だけに保持し、response や audit へ出さない。
type OperatorService struct {
	operators      OperatorRepository
	audits         *audit.AuditService
	ids            accounts.AccountIDGenerator
	secrets        authapplication.OpaqueTokenGenerator
	registrations  authapplication.OperatorPasskeyRegistrationProvider
	sessions       OperatorSessionIssuer
	delivery       SetupTokenDeliveryPort
	secretHasher   SecretHasher
	secretVerifier SecretVerifier
	clock          func() time.Time
	bootstrap      BootstrapConfig
	setupTokenTTL  time.Duration
}

// ─── Constructor ───────────────────────────────────────────────────────────

// NewOperatorService は Operator setup / creation use case を構築する。
//
// 役割:
//   - 必須 port を constructor 時点で検証し、nil port による fail-open を防止する。
//   - setupTokenTTL が正数でない場合も拒否し、token 有効期限の非決定性を防ぐ。
//
// 引数:
//   - operators: operator 永続化 port。
//   - audits: 監査 event 記録 port。
//   - ids: ID 生成 port。
//   - secrets: opaque token 生成 port。
//   - registrations: WebAuthn registration provider port。
//   - sessions: operator session 発行 port。
//   - delivery: setup token 配送 port。
//   - secretHasher: secret hash 生成 port。
//   - secretVerifier: secret 照合 port。
//   - clock: 時刻副作用を注入する関数。
//   - bootstrap: 初回 setup gate 設定。
//   - setupTokenTTL: setup token の有効期間。
//
// 戻り値:
//   - *OperatorService: 検証済み依存だけを保持する use case。
//   - error: 必須依存が欠けている場合、または setupTokenTTL が不正な場合。
func NewOperatorService(operators OperatorRepository, audits *audit.AuditService, ids accounts.AccountIDGenerator, secrets authapplication.OpaqueTokenGenerator, registrations authapplication.OperatorPasskeyRegistrationProvider, sessions OperatorSessionIssuer, delivery SetupTokenDeliveryPort, secretHasher SecretHasher, secretVerifier SecretVerifier, clock func() time.Time, bootstrap BootstrapConfig, setupTokenTTL time.Duration) (*OperatorService, error) {
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

// ─── Use case method ───────────────────────────────────────────────────────

// StartInitialSetup は operator 0 件環境で初回 operator の passkey 登録 challenge を開始する。
//
// 役割:
//   - bootstrap gate と operator 件数を challenge 発行前に検証し、初回以外の環境で setup を開始しない。
//   - WebAuthn registration provider に discoverable credential + userVerification=required の challenge 発行を委譲する。
//
// 引数:
//   - ctx: 呼び出し単位のキャンセル・期限情報。
//   - input: bootstrap secret、email、表示名、request ID を含む入力 DTO。
//
// 戻り値:
//   - SetupChallengeResult: WebAuthn ceremony に必要な challenge 情報。
//   - error: bootstrap gate 不正、operator 既存、ID 生成失敗などの stable application error。
//
// エラーケース:
//   - bootstrap gate が無効、期限切れ、secret 不一致の場合は ErrOperatorForbidden を返す。
//   - 既存 operator が存在する場合は ErrOperatorConflict を返す。
//   - ID 生成や email 形式不正の場合は ErrOperatorInvalidInput または ErrOperatorInternal を返す。
//
// 使用例:
//
//	result, err := service.StartInitialSetup(ctx, operators.InitialSetupStartInput{
//		Email: "operator@example.com", DisplayName: "Operator", BootstrapSecret: "secret", RequestID: requestID,
//	})
//	if err != nil { return err }
//	// result.Challenge, result.OptionsJSON を browser WebAuthn API に渡す
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
	challenge, err := s.registrations.BeginOperatorRegistration(ctx, authapplication.OperatorRegistrationChallengeInput{RequestID: input.RequestID, OperatorID: operatorID, Email: email, DisplayName: displayName})
	if err != nil {
		return SetupChallengeResult{}, ErrOperatorInternal
	}
	return SetupChallengeResult{RequestID: challenge.RequestID, Challenge: challenge.Challenge, OptionsJSON: challenge.OptionsJSON}, nil
}

// FinishInitialSetup は初回 operator と passkey credential を作成し、operator session を発行する。
//
// 役割:
//   - start 後の期限切れや競合を防ぐため、finish 時にも bootstrap gate と operator 件数を再検証する。
//   - attestation を WebAuthn provider で検証し、検証済み credential data だけを保存 DTO へ変換する。
//   - operator 作成と passkey 保存を repository transaction に委譲し、同時初回作成は conflict として扱う。
//
// 引数:
//   - ctx: 呼び出し単位のキャンセル・期限情報。
//   - input: bootstrap secret、email、表示名、request ID、WebAuthn credential を含む入力 DTO。
//
// 戻り値:
//   - authapplication.OperatorSessionResult: accessToken と refresh Cookie command を分離した session DTO。
//   - error: bootstrap gate 不正、credential 検証失敗、repository 障害などの stable application error。
//
// エラーケース:
//   - bootstrap gate 不正、credential 検証失敗の場合は ErrOperatorForbidden を返す。
//   - repository 障害の場合は ErrOperatorInternal または ErrOperatorConflict を返す。
//
// 使用例:
//
//	result, err := service.FinishInitialSetup(ctx, operators.InitialSetupFinishInput{
//		Email: "operator@example.com", DisplayName: "Operator", BootstrapSecret: "secret",
//		RequestID: requestID, Credential: webauthnCredential,
//	})
//	if err != nil { return err }
//	// result.AccessToken を response body に、result.RefreshCookie を Set-Cookie に返す
func (s *OperatorService) FinishInitialSetup(ctx context.Context, input InitialSetupFinishInput) (authapplication.OperatorSessionResult, error) {
	// Step 1: start 後の期限切れや競合を防ぐため、finish 時にも bootstrap gate と operator 件数を再検証する。
	if err := s.validateBootstrap(ctx, input.BootstrapSecret); err != nil {
		return authapplication.OperatorSessionResult{}, err
	}
	operatorID, email, _, err := s.initialSetupIdentity(input.RequestID, input.Email, input.DisplayName)
	if err != nil {
		return authapplication.OperatorSessionResult{}, err
	}

	// Step 2: attestation を WebAuthn provider で検証し、検証済み credential data だけを保存 DTO へ変換する。
	registration, err := s.registrations.FinishOperatorRegistration(ctx, input.RequestID, operatorID, input.Credential)
	if err != nil {
		return authapplication.OperatorSessionResult{}, ErrOperatorForbidden
	}
	passkey, err := s.passkeyRecord(registration)
	if err != nil {
		return authapplication.OperatorSessionResult{}, err
	}

	// Step 3: operator 作成と passkey 保存を repository transaction に委譲し、同時初回作成は conflict として扱う。
	created, err := s.operators.CreateInitialOperatorWithPasskey(ctx, InitialOperatorRecord{OperatorID: operatorID, Email: email, Passkey: passkey, CompletedAt: s.clock().UTC()})
	if err != nil {
		return authapplication.OperatorSessionResult{}, mapOperatorRepositoryError(err)
	}

	// Step 4: 登録済みに更新された Operator にだけ通常 Operator session を発行する。
	return s.sessions.IssueOperatorSession(ctx, authapplication.IssueOperatorSessionInput{OperatorID: created.OperatorID})
}

// StartOperatorSetup は setup token を検証し、追加 operator の passkey 登録 challenge を開始する。
//
// 役割:
//   - setup token は hash 比較 callback だけで照合し、repository へ平文 token を渡さない。
//   - pending operator だけに WebAuthn registration challenge を発行する。
//
// 引数:
//   - ctx: 呼び出し単位のキャンセル・期限情報。
//   - input: setup token と request ID を含む入力 DTO。
//
// 戻り値:
//   - SetupChallengeResult: WebAuthn ceremony に必要な challenge 情報。
//   - error: token 不正、WebAuthn provider 障害などの stable application error。
//
// エラーケース:
//   - token が空、期限切れ、一致しない場合は ErrOperatorForbidden を返す。
//   - WebAuthn provider 障害の場合は ErrOperatorInternal を返す。
//
// 使用例:
//
//	result, err := service.StartOperatorSetup(ctx, operators.SetupStartInput{
//		SetupToken: token, RequestID: requestID,
//	})
//	if err != nil { return err }
//	// result.Challenge, result.OptionsJSON を browser WebAuthn API に渡す
func (s *OperatorService) StartOperatorSetup(ctx context.Context, input SetupStartInput) (SetupChallengeResult, error) {
	// Step 1: setup token は hash 比較 callback だけで照合し、repository へ平文 token を渡さない。
	operator, err := s.findOperatorForSetup(ctx, input.SetupToken)
	if err != nil {
		return SetupChallengeResult{}, err
	}

	// Step 2: pending operator だけに WebAuthn registration challenge を発行する。
	challenge, err := s.registrations.BeginOperatorRegistration(ctx, authapplication.OperatorRegistrationChallengeInput{RequestID: input.RequestID, OperatorID: operator.OperatorID, Email: operator.Email, DisplayName: operator.Email})
	if err != nil {
		return SetupChallengeResult{}, ErrOperatorInternal
	}
	return SetupChallengeResult{RequestID: challenge.RequestID, Challenge: challenge.Challenge, OptionsJSON: challenge.OptionsJSON}, nil
}

// FinishOperatorSetup は setup token を消費し、追加 operator の初回 passkey と operator session を発行する。
//
// 役割:
//   - finish 時にも setup token を再検証し、start 後の期限切れや既消費 token を拒否する。
//   - WebAuthn attestation を検証し、未検証 credential を repository へ保存しない。
//   - setup token の one-time consumption と passkey 保存を同一 transaction で実行する。
//
// 引数:
//   - ctx: 呼び出し単位のキャンセル・期限情報。
//   - input: setup token、request ID、WebAuthn credential を含む入力 DTO。
//
// 戻り値:
//   - authapplication.OperatorSessionResult: accessToken と refresh Cookie command を分離した session DTO。
//   - error: token 不正、credential 検証失敗、repository 障害などの stable application error。
//
// エラーケース:
//   - token 不正、credential 検証失敗の場合は ErrOperatorForbidden を返す。
//   - repository 障害の場合は ErrOperatorInternal を返す。
//
// 使用例:
//
//	result, err := service.FinishOperatorSetup(ctx, operators.SetupFinishInput{
//		SetupToken: token, RequestID: requestID, Credential: webauthnCredential,
//	})
//	if err != nil { return err }
//	// result.AccessToken を response body に、result.RefreshCookie を Set-Cookie に返す
func (s *OperatorService) FinishOperatorSetup(ctx context.Context, input SetupFinishInput) (authapplication.OperatorSessionResult, error) {
	// Step 1: finish 時にも setup token を再検証し、start 後の期限切れや既消費 token を拒否する。
	operator, err := s.findOperatorForSetup(ctx, input.SetupToken)
	if err != nil {
		return authapplication.OperatorSessionResult{}, err
	}

	// Step 2: WebAuthn attestation を検証し、未検証 credential を repository へ保存しない。
	registration, err := s.registrations.FinishOperatorRegistration(ctx, input.RequestID, operator.OperatorID, input.Credential)
	if err != nil {
		return authapplication.OperatorSessionResult{}, ErrOperatorForbidden
	}
	passkey, err := s.passkeyRecord(registration)
	if err != nil {
		return authapplication.OperatorSessionResult{}, err
	}

	// Step 3: setup token の one-time consumption と passkey 保存を同一 transaction で実行する。
	trimmedToken := strings.TrimSpace(input.SetupToken)
	completed, err := s.operators.CompleteOperatorSetupWithPasskey(ctx, SetupCompletionRecord{OperatorID: operator.OperatorID, SetupTokenMatches: func(hash string) bool {
		return s.secretVerifier.MatchesSecret(hash, trimmedToken)
	}, Passkey: passkey, CompletedAt: s.clock().UTC()})
	if err != nil {
		return authapplication.OperatorSessionResult{}, mapOperatorRepositoryError(err)
	}

	// Step 4: setup token 消費後の Operator にだけ通常 Operator session を発行する。
	return s.sessions.IssueOperatorSession(ctx, authapplication.IssueOperatorSessionInput{OperatorID: completed.OperatorID})
}

// CreateOperator は追加 operator を作成し、setup token を secure delivery port で配送する。
//
// 役割:
//   - acting operator を domain object に復元し、operators:create 権限を持つ acting operator に限定する。
//   - 作成対象 operator の email/role と ID を domain constructor で検証する。
//   - mutation 前 audit intent を記録し、監査なし operator 作成を防ぐ。
//   - 平文 setup token はこの use case 内だけに保持し、opaque hash と expiry だけを repository に渡す。
//   - token 平文は secure delivery port にだけ渡し、配送失敗時は pending operator ごと削除して failed audit outcome にする。
//
// 引数:
//   - ctx: 呼び出し単位のキャンセル・期限情報。
//   - input: acting operator の snapshot、作成対象の email/role、request ID を含む入力 DTO。
//
// 戻り値:
//   - CreatedOperator: 作成された operator の非秘匿 DTO、audit ID、delivery status。
//   - error: 権限不足、入力不正、repository/配送/audit 障害などの stable application error。
//
// エラーケース:
//   - acting operator の権限不足（operators:create 権限なし）の場合は ErrOperatorForbidden を返す。
//   - email/role 形式不正の場合は ErrOperatorInvalidInput を返す。
//   - repository/配送/audit 障害の場合は ErrOperatorInternal を返す。
//
// 使用例:
//
//	result, err := service.CreateOperator(ctx, operators.CreateOperatorInput{
//		Email: "new@example.com", Role: "viewer", RequestID: requestID,
//		OperatorID: actingID, OperatorEmail: "acting@example.com", OperatorRole: "admin",
//		OperatorActive: true, PasskeyRegistrationState: "registered",
//	})
//	if err != nil { return err }
//	// result.Operator.ID, result.DeliveryStatus を response に返す
func (s *OperatorService) CreateOperator(ctx context.Context, input CreateOperatorInput) (CreatedOperator, error) {
	// Step 1: acting operator を domain object に復元し、operators:create 権限を持つ acting operator に限定する。
	acting, err := restoreOperatorActor(input)
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
	intent, err := s.audits.RecordMutationIntent(ctx, audit.IntentInput{OperatorID: acting.ID().String(), Action: operatorCreateAction, TargetType: operatorTargetType, TargetID: operatorID, RequestID: input.RequestID})
	if err != nil {
		return CreatedOperator{}, ErrOperatorInternal
	}

	// Step 4: 平文 setup token はこの use case 内だけに保持し、opaque hash と expiry だけを repository に渡す。
	plainToken, tokenHash, expiresAt, err := s.newSetupToken()
	if err != nil {
		return s.failOperatorCreation(ctx, intent.AuditID, err, operatorStableCodeRepositoryFailure)
	}
	created, err := s.operators.CreateOperatorWithSetupToken(ctx, OperatorCreationRecord{OperatorID: operatorID, Email: email, Role: role, SetupTokenHash: tokenHash, SetupTokenExpiresAt: expiresAt, CreatedAt: s.clock().UTC()})
	if err != nil {
		return s.failOperatorCreation(ctx, intent.AuditID, err, operatorStableCodeRepositoryFailure)
	}

	// Step 5: token 平文は secure delivery port にだけ渡し、配送失敗時は pending operator ごと削除して failed audit outcome にし、response body へ secret を出さない。
	if err := s.delivery.SendOperatorSetupToken(ctx, SetupTokenDelivery{OperatorID: created.OperatorID, Email: created.Email, SetupToken: plainToken, ExpiresAt: expiresAt, RequestID: input.RequestID}); err != nil {
		if deleteErr := s.operators.DeletePendingOperatorSetup(ctx, created.OperatorID); deleteErr != nil {
			return s.failOperatorCreation(ctx, intent.AuditID, ErrOperatorInternal, operatorStableCodeRepositoryFailure)
		}
		return s.failOperatorCreation(ctx, intent.AuditID, ErrOperatorInternal, operatorStableCodeDeliveryFailure)
	}
	if _, err := s.audits.CompleteMutationSucceeded(ctx, audit.CompletionInput{AuditID: intent.AuditID}); err != nil {
		if deleteErr := s.operators.DeletePendingOperatorSetup(ctx, created.OperatorID); deleteErr != nil {
			return CreatedOperator{}, ErrOperatorInternal
		}
		return CreatedOperator{}, ErrOperatorInternal
	}

	// Step 6: response は operator summary、delivery status、audit ID だけに限定し、setup token 平文を含めない。
	return CreatedOperator{RequestID: input.RequestID, AuditID: intent.AuditID, DeliveryStatus: "sent", Operator: authapplication.OperatorDTO{ID: created.OperatorID, Email: created.Email, Role: created.Role, Active: created.Active, PasskeyRegistrationState: created.PasskeyRegistrationState}}, nil
}

// ─── Unexported helper ─────────────────────────────────────────────────────

// validateBootstrap は bootstrap gate と operator 件数を検証する。
//
// 役割:
//   - config gate が無効、期限切れ、または hash 未設定なら secret 比較へ進まず拒否する。
//   - 既存 operator がある場合は初回 setup route を conflict とし、追加 operator flow へ誘導できる状態にする。
//   - opaque hash 比較の詳細を外へ出さず、secret 不一致は forbidden に畳む。
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

// initialSetupIdentity は初回 setup の email/displayName/operatorID を検証して返す。
//
// 役割:
//   - email は OperatorEmail domain object だけで正規化し、displayName は空なら email に倒す。
//   - 初回 operator ID は start/finish の requestId と同じ ULID に固定し、WebAuthn session の user handle と DB 作成 ID を一致させる。
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

// findOperatorForSetup は setup token を検証し、対象 operator を返す。
//
// 役割:
//   - 空 token は repository 探索を行わず、token 状態を区別しない forbidden に畳む。
//   - opaque hash 比較 callback を repository に渡し、平文 token を DB query や audit に混ぜない。
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

// newOperatorIdentity は作成対象 operator の email/role/ID を検証して返す。
//
// 役割:
//   - 作成対象 email と role は domain value object で検証し、未知 role を fail-closed にする。
//   - OperatorID は platform ID generator の出力を domain constructor に通してから保存する。
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

// newSetupToken は平文 setup token とその hash を生成する。
//
// 役割:
//   - 平文 setup token は暗号学的乱数 generator から発行し、ログや response には出さない。
//   - DB 保存用には bcrypt hash だけを生成し、平文 token は secure delivery port へ渡すまでの一時値に限定する。
func (s *OperatorService) newSetupToken() (string, string, time.Time, error) {
	// Step 1: 平文 setup token は暗号学的乱数 generator から発行し、ログや response には出さない。
	plainToken, err := s.secrets.NewToken()
	if err != nil {
		return "", "", time.Time{}, ErrOperatorInternal
	}

	// Step 2: DB 保存用には bcrypt hash だけを生成し、平文 token は secure delivery port へ渡すまでの一時値に限定する。
	hash, err := s.hashOperatorSecret(plainToken)
	if err != nil {
		return "", "", time.Time{}, ErrOperatorInternal
	}
	return plainToken, hash, s.clock().UTC().Add(s.setupTokenTTL), nil
}

// hashOperatorSecret は operator secret の保存用 hash を生成する。
//
// 役割:
//   - copy/paste 由来の前後空白だけを除去し、空 secret は hash 化せず拒否する。
//   - 保存形式の具体実装は SecretHasher port に委譲し、application 層から platform 依存を排除する。
func (s *OperatorService) hashOperatorSecret(secretValue string) (string, error) {
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

// passkeyRecord は WebAuthn registration 結果を保存用 DTO へ変換する。
//
// 役割:
//   - passkey credential ID は operator credential 専用 ULID として発行し、credential handle と分離する。
//   - credential handle と public key が空の場合は ErrOperatorForbidden を返す。
func (s *OperatorService) passkeyRecord(registration authapplication.OperatorPasskeyRegistration) (PasskeyRecord, error) {
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

// failOperatorCreation は operator 作成または delivery 失敗を failed audit outcome として記録する。
//
// 役割:
//   - operator 作成または delivery 失敗を failed audit outcome として記録し、token 平文を監査へ含めない。
//   - 監査保存後は元 error の抽象分類だけを handler へ返す。
func (s *OperatorService) failOperatorCreation(ctx context.Context, auditID string, original error, stableCode string) (CreatedOperator, error) {
	// Step 1: operator 作成または delivery 失敗を failed audit outcome として記録し、token 平文を監査へ含めない。
	if _, err := s.audits.CompleteMutationFailed(ctx, audit.FailureInput{AuditID: auditID, StableErrorCode: stableCode}); err != nil {
		return CreatedOperator{}, ErrOperatorInternal
	}

	// Step 2: 監査保存後は元 error の抽象分類だけを handler へ返す。
	return CreatedOperator{}, original
}

// restoreOperatorActor は acting operator の primitive snapshot を domain.Operator に復元する。
//
// 役割:
//   - acting operator の primitive snapshot を domain.Operator に復元し、Account role を混ぜない。
//   - operator が inactive または passkey 未登録の場合は ErrOperatorForbidden を返す。
func restoreOperatorActor(input CreateOperatorInput) (domain.Operator, error) {
	// Step 1: acting operator の primitive snapshot を domain.Operator に復元し、Account role を混ぜない。
	operatorID, err := domain.NewOperatorID(input.OperatorID)
	if err != nil {
		return emptyOperator(), ErrOperatorForbidden
	}
	operatorEmail, err := domain.NewOperatorEmail(input.OperatorEmail)
	if err != nil {
		return emptyOperator(), ErrOperatorForbidden
	}
	operator, err := domain.NewOperator(operatorID, operatorEmail, domain.OperatorRole(input.OperatorRole), input.OperatorActive, domain.OperatorPasskeyRegistrationState(input.PasskeyRegistrationState))
	if err != nil {
		return emptyOperator(), ErrOperatorForbidden
	}
	if !operator.Active() || operator.PasskeyRegistrationState() != domain.OperatorPasskeyRegistrationRegistered {
		return emptyOperator(), ErrOperatorForbidden
	}
	return operator, nil
}

// emptyOperator は error return 用の domain.Operator ゼロ値を返す。
//
// 役割:
//   - guardrail が禁止する domain composite literal を使わず、error return 用のゼロ値を作る。
func emptyOperator() domain.Operator {
	// Step 1: guardrail が禁止する domain composite literal を使わず、error return 用のゼロ値を作る。
	var operator domain.Operator
	return operator
}

// mapOperatorRepositoryError は repository の抽象 error を HTTP adapter が扱う application error へ畳む。
//
// 役割:
//   - repository の抽象 error を HTTP adapter が扱う application error へ畳む。
//   - 未分類の error は ErrOperatorInternal として fail-closed にする。
func mapOperatorRepositoryError(err error) error {
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
