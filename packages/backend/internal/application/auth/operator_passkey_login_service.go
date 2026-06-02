package auth

import "context"

// OperatorPasskeyLoginService は Admin operator passkey login の外側 WebAuthn flow を扱う application service である。
//
// 役割:
//   - passkey challenge の開始と、WebAuthn adapter が検証済みにした credential handle からの session 発行を担当する。
//   - OperatorSessionService には session lifecycle だけを委譲し、challenge provider や credential lookup を session service へ漏らさない。
//   - Product Account の passkey flow や repository を使わず、Admin Operator 専用 DTO と port だけを扱う。
//
// 使用例:
//
//	service, err := NewOperatorPasskeyLoginService(deps, config)
//	if err != nil {
//		return err
//	}
//	challenge, err := service.StartOperatorPasskey(ctx, input)
type OperatorPasskeyLoginService struct {
	operators  OperatorRepository
	challenges OperatorPasskeyChallengeProvider
	sessions   OperatorSessionIssuer
	rpID       string
}

// OperatorSessionIssuer は OperatorPasskeyLoginService が session lifecycle owner へ委譲する発行 port である。
//
// 役割:
//   - passkey login outer flow から確定済み OperatorID だけを渡し、access/refresh 発行の詳細を再実装しない。
//   - OperatorSessionService の公開 method だけへ依存し、保存先や signer の具象型を login service へ持ち込まない。
type OperatorSessionIssuer interface {
	IssueOperatorSession(ctx context.Context, input IssueOperatorSessionInput) (OperatorSessionResult, error)
}

// OperatorPasskeyLoginDependencies は Admin passkey login outer flow に必要な port をまとめる DTO である。
//
// 役割:
//   - repository、challenge provider、session issuer を constructor 時点で必須検証する。
//   - OperatorSessionDependencies から challenge provider を分離し、session lifecycle と WebAuthn ceremony の所有者を明確にする。
type OperatorPasskeyLoginDependencies struct {
	Operators  OperatorRepository
	Challenges OperatorPasskeyChallengeProvider
	Sessions   OperatorSessionIssuer
}

// NewOperatorPasskeyLoginService は Admin operator passkey login 用 service を生成する。
//
// 引数:
//   - deps: Operator repository、challenge provider、session issuer の必須依存。
//   - config: WebAuthn RP ID を含む Admin operator auth 設定。
//
// 戻り値:
//   - *OperatorPasskeyLoginService: 検証済み依存を保持する passkey login service。
//   - error: 必須依存が nil、または WebAuthn RP ID が空の場合の internal error。
func NewOperatorPasskeyLoginService(deps OperatorPasskeyLoginDependencies, config OperatorSessionConfig) (*OperatorPasskeyLoginService, error) {
	// Step 1: passkey login に必要な outer flow 依存を constructor で検証し、route 実行時の nil fallback を作らない。
	if deps.Operators == nil || deps.Challenges == nil || deps.Sessions == nil || config.WebAuthnRPID == "" {
		return nil, ErrOperatorAuthUnavailable
	}

	// Step 2: 検証済み依存だけを保持し、session 発行の詳細は Sessions port へ閉じ込める。
	return &OperatorPasskeyLoginService{operators: deps.Operators, challenges: deps.Challenges, sessions: deps.Sessions, rpID: config.WebAuthnRPID}, nil
}

// StartOperatorPasskey は Admin operator passkey login challenge を開始する。
//
// 引数:
//   - ctx: challenge provider 呼び出しに使う cancellation context。
//   - input: operator email など、provider が challenge 発行に使う識別子。
//
// 戻り値:
//   - OperatorPasskeyChallenge: WebAuthn ceremony に必要な公開 challenge 情報。
//   - error: challenge provider 失敗時の stable internal error。
func (s *OperatorPasskeyLoginService) StartOperatorPasskey(ctx context.Context, input StartOperatorPasskeyInput) (OperatorPasskeyChallenge, error) {
	// Step 1: WebAuthn challenge 発行は Admin 専用 provider port へ委譲し、session secret はまだ発行しない。
	challengeKey, optionsJSON, err := s.challenges.BeginOperatorLogin(ctx, input.Identifier)
	if err != nil {
		return OperatorPasskeyChallenge{}, ErrOperatorAuthUnavailable
	}

	// Step 2: adapter/frontend が ceremony を継続できる DTO に変換して返す。
	return OperatorPasskeyChallenge{ChallengeID: challengeKey, Challenge: challengeKey, WebAuthnRPID: s.rpID, WebAuthnOptions: optionsJSON}, nil
}

// FinishOperatorPasskey は WebAuthn 検証済み credential から Admin operator session を発行する。
//
// 引数:
//   - ctx: repository/session issuer 呼び出しに使う cancellation context。
//   - input: WebAuthn provider が署名検証後に確定した credential handle と challenge selector。
//
// 戻り値:
//   - OperatorSessionResult: accessToken と refresh Cookie command を分離した session DTO。
//   - error: credential 不一致、Operator 不適格、session 保存失敗などの stable application error。
func (s *OperatorPasskeyLoginService) FinishOperatorPasskey(ctx context.Context, input FinishOperatorPasskeyInput) (OperatorSessionResult, error) {
	// Step 1: credential handle から現在の Admin Operator snapshot を取得し、Product account repository を使わない。
	snapshot, err := s.operators.FindOperatorByCredential(ctx, input.CredentialHandle)
	if err != nil {
		return OperatorSessionResult{}, mapAdminStoreError(err)
	}

	// Step 2: snapshot を domain object へ復元し、壊れた role/state を session issuer へ渡さない。
	operator, err := operatorFromSnapshot(snapshot)
	if err != nil {
		return OperatorSessionResult{}, err
	}

	// Step 3: WebAuthn provider が challenge を消費済みであることを入力上の境界として記録し、未使用警告を避ける。
	_ = input.ChallengeID

	// Step 4: access/refresh 発行と保存は canonical session lifecycle owner へ委譲する。
	return s.sessions.IssueOperatorSession(ctx, IssueOperatorSessionInput{OperatorID: operator.ID().String()})
}
