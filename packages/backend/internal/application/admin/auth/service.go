package application

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	tokenprimitive "www-template/packages/backend/internal/application/shared/tokenprimitive"
	domain "www-template/packages/backend/internal/domain"
)

const adminRefreshCookieName = "admin_refresh_token"

// Service は Admin operator auth の application boundary である。
//
// 役割:
//   - Admin operator login、refresh rotation、current operator、mutation CSRF validation、logout を提供する。
//   - Product auth application や Product-specific domain object を import せず、OperatorAuth domain object だけを使う。
//   - refreshToken 平文を response body DTO に置かず、HttpOnly Cookie command 専用値として adapter へ渡す。
type Service struct {
	operators  OperatorRepository
	passkeys   OperatorPasskeyRepository
	sessions   OperatorSessionStore
	challenges OperatorPasskeyChallengeProvider
	signer     tokenprimitive.JSONSignVerifier
	secrets    OpaqueTokenGenerator
	ids        IDGenerator
	clock      func() time.Time
	config     AdminAuthConfig
}

// NewService は Admin operator auth Service を生成する。
//
// 引数:
//   - operators: Admin Operator snapshot を取得する repository port。
//   - sessions: Admin refresh session を保存・rotation・revoke する store port。
//   - challenges: passkey login challenge を発行する provider port。
//   - signer: Admin accessToken payload を署名・検証する中立 signer/verifier。
//   - secrets: refreshToken と CSRF token の opaque secret generator。
//   - ids: session ID、JTI、request ID の ID generator。
//   - clock: 現在時刻を外部から注入する関数。nil は許可しない。
//   - config: Admin auth の TTL と Cookie lifetime 設定。
//
// 戻り値:
//   - *Service: 検証済み依存を保持する use case service。
//   - error: TTL/Cookie lifetime が不正、または必須依存が nil の場合。
func NewService(
	operators OperatorRepository,
	sessions OperatorSessionStore,
	challenges OperatorPasskeyChallengeProvider,
	signer tokenprimitive.JSONSignVerifier,
	secrets OpaqueTokenGenerator,
	ids IDGenerator,
	clock func() time.Time,
	config AdminAuthConfig,
) (*Service, error) {
	// Step 1: 必須 port が欠けた状態で認証境界を作ると fail-open になり得るため拒否する。
	if operators == nil || sessions == nil || signer == nil || secrets == nil || ids == nil || clock == nil {
		return nil, ErrAdminAuthInternal
	}

	// Step 2: token TTL と Cookie lifetime の関係を中立 primitive で検証し、Cookie の長すぎる保持を防ぐ。
	if _, err := tokenprimitive.ValidateDurations(config.RefreshSessionTTL, config.RefreshCookieLifetime); err != nil {
		return nil, ErrAdminAuthInternal
	}
	if _, err := tokenprimitive.ValidateTTL(config.AccessTokenTTL); err != nil {
		return nil, ErrAdminAuthInternal
	}

	// Step 3: 検証済み依存だけを Service に保持し、以後の use case が同じ境界を共有する。
	return &Service{
		operators:  operators,
		sessions:   sessions,
		challenges: challenges,
		signer:     signer,
		secrets:    secrets,
		ids:        ids,
		clock:      clock,
		config:     config,
	}, nil
}

// AttachOperatorPasskeyRepository は Admin operator passkey 管理用 repository を既存 Service に接続する。
//
// 引数:
//   - passkeys: Admin schema の operator_passkeys だけを扱う repository port。nil は拒否する。
//
// 戻り値:
//   - nil: repository が接続された場合。
//   - ErrAdminAuthInternal: nil repository が渡された場合。
//
// 使用例:
//
//	service, err := auth.NewService(...)
//	if err == nil {
//		err = service.AttachOperatorPasskeyRepository(passkeyRepo)
//	}
func (s *Service) AttachOperatorPasskeyRepository(passkeys OperatorPasskeyRepository) error {
	// Step 1: nil repository を接続すると list/delete が fail-open になり得るため拒否する。
	if passkeys == nil {
		return ErrAdminAuthInternal
	}

	// Step 2: runtime composition 済み Admin passkey repository を保持し、以後の list/delete use case で利用する。
	s.passkeys = passkeys
	return nil
}

// ListOperatorPasskeys は Admin operator 自身に登録された passkey credential 一覧を返す。
//
// OperatorID は HTTP middleware が Admin operator session と CSRF binding を検証した後に渡す値である。
// response には credential handle や public key を含めず、Product passkey repository も参照しない。
func (s *Service) ListOperatorPasskeys(ctx context.Context, input ListOperatorPasskeysInput) (OperatorPasskeyListResult, error) {
	// Step 1: repository 未接続時は passkey 一覧を公開せず、Admin auth boundary を fail-closed にする。
	if s.passkeys == nil {
		return OperatorPasskeyListResult{}, ErrAdminAuthInternal
	}

	// Step 2: OperatorID を domain value object として検証し、Product AccountID などの未検証文字列を拒否する。
	operatorID, err := domain.NewOperatorID(input.OperatorID)
	if err != nil {
		return OperatorPasskeyListResult{}, ErrAdminAuthUnauthenticated
	}

	// Step 3: Admin operator passkey repository へ所有者 ID を渡し、Admin schema の credential だけを取得する。
	passkeys, err := s.passkeys.ListOperatorPasskeys(ctx, operatorID.String())
	if err != nil {
		return OperatorPasskeyListResult{}, mapAdminPasskeyStoreError(err)
	}

	// Step 4: 非秘匿 DTO の一覧だけを返し、handler が credential handle や公開鍵を扱わない境界を保つ。
	return OperatorPasskeyListResult{Passkeys: passkeys}, nil
}

// DeleteOperatorPasskey は Admin operator 自身の passkey credential を削除する。
//
// 最後の 1 件削除は domain.EnsureOperatorPasskeyDeletionAllowed で拒否し、repository でも所有者条件つき削除に限定する。
func (s *Service) DeleteOperatorPasskey(ctx context.Context, input DeleteOperatorPasskeyInput) error {
	// Step 1: repository 未接続時は passkey 削除を行わず、Admin auth boundary を fail-closed にする。
	if s.passkeys == nil {
		return ErrAdminAuthInternal
	}

	// Step 2: 所有者 OperatorID と削除対象 passkey ID を domain ULID rule で検証し、保存層へ不正 selector を渡さない。
	operatorID, err := domain.NewOperatorID(input.OperatorID)
	if err != nil {
		return ErrAdminAuthUnauthenticated
	}
	if err := domain.ValidateAuthID(input.PasskeyID); err != nil {
		return ErrAdminAuthBadRequest
	}

	// Step 3: 削除前の credential 数を Admin repository から取得し、最後の 1 件削除 rule を domain に委譲する。
	passkeys, err := s.passkeys.ListOperatorPasskeys(ctx, operatorID.String())
	if err != nil {
		return mapAdminPasskeyStoreError(err)
	}
	if err := domain.EnsureOperatorPasskeyDeletionAllowed(len(passkeys)); err != nil {
		return mapAdminPasskeyDomainError(err)
	}

	// Step 4: repository に所有者 ID と passkey ID の両方を渡し、他 Operator の credential 削除を防ぐ。
	if err := s.passkeys.DeleteOperatorPasskey(ctx, operatorID.String(), input.PasskeyID); err != nil {
		return mapAdminPasskeyStoreError(err)
	}
	return nil
}

// StartOperatorPasskey は Admin operator passkey login challenge を開始する。
//
// 戻り値には WebAuthn ceremony に必要な challenge 情報だけを含め、session secret は発行しない。
// challenges port が未設定の場合は、adapter が別 provider を使う構成として ErrAdminAuthInternal を返す。
func (s *Service) StartOperatorPasskey(ctx context.Context, input StartOperatorPasskeyInput) (OperatorPasskeyChallenge, error) {
	// Step 1: challenge provider がない構成では passkey 開始を fail-closed にする。
	if s.challenges == nil {
		return OperatorPasskeyChallenge{}, ErrAdminAuthInternal
	}

	// Step 2: WebAuthn challenge 発行は Admin 専用 provider port へ委譲する。
	challengeKey, optionsJSON, err := s.challenges.BeginOperatorLogin(ctx, input.Identifier)
	if err != nil {
		return OperatorPasskeyChallenge{}, ErrAdminAuthInternal
	}

	// Step 3: adapter/frontend が ceremony を継続できる DTO に変換して返す。
	return OperatorPasskeyChallenge{
		ChallengeID:     challengeKey,
		Challenge:       challengeKey,
		WebAuthnRPID:    s.config.WebAuthnRPID,
		WebAuthnOptions: optionsJSON,
	}, nil
}

// FinishOperatorPasskey は WebAuthn 検証済み credential から Admin operator session を発行する。
//
// refreshToken 平文は OperatorSessionResult.RefreshCookie.Value にだけ入り、response body 用 DTO field には含めない。
func (s *Service) FinishOperatorPasskey(ctx context.Context, input FinishOperatorPasskeyInput) (OperatorSessionResult, error) {
	// Step 1: credential handle から現在の Operator snapshot を取得し、Product account repository を使わない。
	operator, err := s.findOperatorByCredential(ctx, input.CredentialHandle)
	if err != nil {
		return OperatorSessionResult{}, err
	}

	// Step 2: Admin OperatorAuth domain object で refresh session、accessToken、CSRF を組み立てる。
	issued, err := s.issueSession(ctx, operator)
	if err != nil {
		return OperatorSessionResult{}, err
	}

	// Step 3: session store には hash と snapshot だけを保存し、平文 refreshToken は保存しない。
	if err := s.sessions.SaveOperatorSession(ctx, issued.record, s.config.RefreshSessionTTL); err != nil {
		return OperatorSessionResult{}, ErrAdminAuthInternal
	}

	// Step 4: response body と Cookie command を分離した DTO として返す。
	return s.sessionResult(operator, issued), nil
}

// RefreshOperatorSession は Admin refresh Cookie を rotation し、新しい accessToken と refresh Cookie command を返す。
//
// 古い refreshToken は session store の RotateOperatorSession で hash 一致を確認しながら置換される。
func (s *Service) RefreshOperatorSession(ctx context.Context, input RefreshOperatorSessionInput) (OperatorSessionResult, error) {
	// Step 1: Cookie value から session selector と opaque token を分離する。
	sessionID, err := parseRefreshCookieSessionID(input.RefreshCookieValue)
	if err != nil {
		return OperatorSessionResult{}, err
	}

	// Step 2: 保存済み Admin session state を復元し、Product refresh state を参照しない。
	record, err := s.sessions.GetOperatorSession(ctx, sessionID.String())
	if err != nil {
		return OperatorSessionResult{}, mapAdminStoreError(err)
	}
	operator, session, err := s.operatorAndSession(ctx, record)
	if err != nil {
		return OperatorSessionResult{}, err
	}

	// Step 3: Admin OperatorAuth domain object で refreshToken が session と一致することを検証する。
	if err := session.ValidateRefreshToken(input.RefreshCookieValue, s.clock()); err != nil {
		return OperatorSessionResult{}, mapAdminDomainAuthError(err)
	}

	// Step 4: 新しい session state を発行し、store の atomic rotation port へ渡す。
	issued, err := s.issueSession(ctx, operator)
	if err != nil {
		return OperatorSessionResult{}, err
	}
	if err := s.sessions.RotateOperatorSession(ctx, session.ID().String(), session.RefreshTokenHash().String(), issued.record, s.config.RefreshSessionTTL); err != nil {
		return OperatorSessionResult{}, mapAdminStoreError(err)
	}

	// Step 5: 新しい accessToken/CSRF と HttpOnly Cookie command を返し、refreshToken body exposure を避ける。
	return s.sessionResult(operator, issued), nil
}

// CurrentOperator は Admin accessToken から現在の Operator DTO を返す。
//
// current operator は mutation ではないため CSRF は要求せず、session と operator snapshot の一致だけを検証する。
func (s *Service) CurrentOperator(ctx context.Context, input CurrentOperatorInput) (OperatorDTO, error) {
	// Step 1: 署名済み accessToken を検証し、Admin operator claim payload として復元する。
	payload, err := s.verifyAccessPayload(input.AccessToken)
	if err != nil {
		return OperatorDTO{}, err
	}

	// Step 2: payload の session ID から Admin session state と現在 Operator を取得する。
	operator, session, err := s.operatorAndSessionByID(ctx, payload.SessionID)
	if err != nil {
		return OperatorDTO{}, err
	}

	// Step 3: current endpoint 用に session/snapshot/expiry を検証し、viewer は許可する。
	if err := validateCurrentOperatorPayload(operator, session, payload, s.clock()); err != nil {
		return OperatorDTO{}, err
	}

	// Step 4: adapter/frontend 向けの Admin Operator DTO と、middleware context binding に必要な session ID を返す。
	return mapOperatorDTOWithSession(operator, payload.SessionID), nil
}

// ValidateOperatorMutation は Admin mutation 前に accessToken、session、permission、CSRF binding を検証する。
//
// Product bearer token や Product session ID は Admin OperatorAuth domain 型へ復元できないため拒否される。
func (s *Service) ValidateOperatorMutation(ctx context.Context, input ValidateOperatorMutationInput) (OperatorDTO, error) {
	// Step 1: accessToken payload を検証し、Admin operator claim として扱える値だけにする。
	payload, err := s.verifyAccessPayload(input.AccessToken)
	if err != nil {
		return OperatorDTO{}, err
	}

	// Step 2: session state と Operator snapshot を取得し、domain object へ復元する。
	operator, session, err := s.operatorAndSessionByID(ctx, payload.SessionID)
	if err != nil {
		return OperatorDTO{}, err
	}
	claims, err := s.claimsFromPayload(operator, session, payload)
	if err != nil {
		return OperatorDTO{}, err
	}
	permission := domain.OperatorAuthPermission(input.Permission)

	// Step 3: Admin OperatorAuth domain の ValidateAccess に CSRF と permission 判定を委譲する。
	if err := session.ValidateAccess(operator, claims, permission, input.CSRFToken, s.clock()); err != nil {
		return OperatorDTO{}, mapAdminDomainAuthError(err)
	}

	// Step 4: 検証済み Operator context として adapter が扱える DTO を、session selector とともに返す。
	return mapOperatorDTOWithSession(operator, payload.SessionID), nil
}

// LogoutOperator は Admin accessToken が示す Operator session を revoke する。
//
// refreshToken 平文は不要であり、返却する Cookie command は adapter が refresh Cookie を削除するためだけに使う。
func (s *Service) LogoutOperator(ctx context.Context, input LogoutOperatorInput) (RefreshCookieCommand, error) {
	// Step 1: accessToken payload を検証し、対象 session selector を取得する。
	payload, err := s.verifyAccessPayload(input.AccessToken)
	if err != nil {
		return RefreshCookieCommand{}, err
	}

	// Step 2: session state から現在 Operator を復元し、logout 対象の所有者を確認する。
	operator, _, err := s.operatorAndSessionByID(ctx, payload.SessionID)
	if err != nil {
		return RefreshCookieCommand{}, err
	}

	// Step 3: store port へ revoke を委譲し、access/refresh eligibility を同じ session ID で失効させる。
	if err := s.sessions.RevokeOperatorSession(ctx, operator.ID().String(), payload.SessionID); err != nil {
		return RefreshCookieCommand{}, mapAdminStoreError(err)
	}

	// Step 4: adapter が HttpOnly Cookie を削除するための command を返す。
	return clearRefreshCookieCommand(), nil
}

// IssueOperatorSessionForSetup は setup 完了済み Operator に Admin session を発行する。
//
// 引数:
//   - ctx: session store への保存に使う cancellation context。
//   - input.OperatorID: setup transaction で passkey 登録済みに更新された Operator の canonical ID。
//
// 戻り値:
//   - OperatorSessionResult: accessToken、CSRF token、HttpOnly refresh Cookie command を分離した session DTO。
//   - error: Operator が存在しない、inactive、未登録状態、または session 保存に失敗した場合の stable application error。
func (s *Service) IssueOperatorSessionForSetup(ctx context.Context, input IssueOperatorSessionInput) (OperatorSessionResult, error) {
	// Step 1: setup 完了後の Operator snapshot を repository から復元し、Product account auth state を参照しない。
	operator, err := s.findOperatorByID(ctx, input.OperatorID)
	if err != nil {
		return OperatorSessionResult{}, err
	}

	// Step 2: passkey 未登録や inactive の Operator に session を出さないよう、domain permission と同じ前提を検証する。
	if !operator.Active() || operator.PasskeyRegistrationState() != domain.OperatorPasskeyRegistrationRegistered {
		return OperatorSessionResult{}, ErrAdminAuthForbidden
	}

	// Step 3: 通常 login と同じ issueSession path を使い、setup 専用 token 形式を作らない。
	issued, err := s.issueSession(ctx, operator)
	if err != nil {
		return OperatorSessionResult{}, err
	}
	if err := s.sessions.SaveOperatorSession(ctx, issued.record, s.config.RefreshSessionTTL); err != nil {
		return OperatorSessionResult{}, ErrAdminAuthInternal
	}
	return s.sessionResult(operator, issued), nil
}

type issuedOperatorSession struct {
	session      domain.OperatorAuthSession
	record       OperatorSessionRecord
	accessToken  string
	refreshToken string
	csrfToken    string
}

func (s *Service) issueSession(ctx context.Context, operator domain.Operator) (issuedOperatorSession, error) {
	// Step 1: session ID、refreshToken、CSRF token、accessToken JTI を外部 port から発行する。
	sessionID, tokenID, refreshToken, csrfToken, err := s.nextSessionSecrets()
	if err != nil {
		return issuedOperatorSession{}, err
	}
	issuedAt := s.clock()
	ttl, err := domain.NewTokenTTL(s.config.RefreshSessionTTL)
	if err != nil {
		return issuedOperatorSession{}, ErrAdminAuthInternal
	}

	// Step 2: Admin OperatorAuth domain object で refresh session state を構築する。
	session, err := domain.NewOperatorAuthSession(operator, sessionID, refreshToken, csrfToken, ttl, issuedAt)
	if err != nil {
		return issuedOperatorSession{}, mapAdminDomainAuthError(err)
	}

	// Step 3: accessToken claims も Admin OperatorAuth domain object で構築する。
	accessTTL, err := domain.NewTokenTTL(s.config.AccessTokenTTL)
	if err != nil {
		return issuedOperatorSession{}, ErrAdminAuthInternal
	}
	claims, err := domain.NewOperatorAccessTokenClaims(operator, session, tokenID, accessTTL, issuedAt)
	if err != nil {
		return issuedOperatorSession{}, mapAdminDomainAuthError(err)
	}

	// Step 4: claim DTO を中立 signer で署名し、payload 意味はこの Admin use case に閉じる。
	accessToken, err := s.signClaims(claims)
	if err != nil {
		return issuedOperatorSession{}, err
	}

	// Step 5: 保存用 record は hash/snapshot だけを含め、平文 secret を store へ渡さない。
	return issuedOperatorSession{
		session:      session,
		record:       recordFromSession(session),
		accessToken:  accessToken,
		refreshToken: refreshToken,
		csrfToken:    csrfToken,
	}, nil
}

func (s *Service) nextSessionSecrets() (domain.OperatorSessionID, domain.TokenJTI, string, string, error) {
	// Step 1: session ID と JTI は生成後に domain constructor で形式検証する。
	rawSessionID, err := s.ids.Next()
	if err != nil {
		return "", "", "", "", ErrAdminAuthInternal
	}
	sessionID, err := domain.NewOperatorSessionID(rawSessionID)
	if err != nil {
		return "", "", "", "", ErrAdminAuthInternal
	}
	rawTokenID, err := s.ids.Next()
	if err != nil {
		return "", "", "", "", ErrAdminAuthInternal
	}
	tokenID, err := domain.NewTokenJTI(rawTokenID)
	if err != nil {
		return "", "", "", "", ErrAdminAuthInternal
	}

	// Step 2: refreshToken と CSRF token は opaque secret として別々に発行する。
	refreshToken, err := s.refreshCookieValue(sessionID)
	if err != nil {
		return "", "", "", "", err
	}
	csrfToken, err := s.secrets.NewOpaqueToken()
	if err != nil {
		return "", "", "", "", ErrAdminAuthInternal
	}

	// Step 3: 生成済みの Admin session secrets を返す。
	return sessionID, tokenID, refreshToken, csrfToken, nil
}

func (s *Service) refreshCookieValue(sessionID domain.OperatorSessionID) (string, error) {
	// Step 1: opaque secret を発行し、session selector と組み合わせて Cookie value にする。
	secret, err := s.secrets.NewOpaqueToken()
	if err != nil {
		return "", ErrAdminAuthInternal
	}

	// Step 2: Cookie value は sessionID.secret の形にして、refresh 時に O(1) で session を選択できるようにする。
	return sessionID.String() + "." + secret, nil
}

func (s *Service) signClaims(claims domain.OperatorAccessTokenClaims) (string, error) {
	// Step 1: Admin claim payload へ変換し、Product account claim を混入させない。
	payload := operatorAccessTokenPayload{
		OperatorID: claims.OperatorID().String(),
		SessionID:  claims.SessionID().String(),
		TokenID:    claims.TokenID().String(),
		Role:       string(claims.RoleSnapshot()),
		Active:     claims.ActiveSnapshot(),
		IssuedAt:   claims.IssuedAt().Unix(),
		ExpiresAt:  claims.ExpiresAt().Unix(),
	}

	// Step 2: JSON object payload として encode し、中立 signer へ渡す。
	encoded, err := json.Marshal(payload)
	if err != nil {
		return "", ErrAdminAuthInternal
	}
	tokenString, err := s.signer.SignJSON(encoded)
	if err != nil {
		return "", ErrAdminAuthInternal
	}

	// Step 3: compact token を accessToken body field として返す。
	return tokenString, nil
}

func (s *Service) verifyAccessPayload(accessToken string) (operatorAccessTokenPayload, error) {
	// Step 1: 空 token は署名検証へ進めず、認証なしとして扱う。
	if strings.TrimSpace(accessToken) == "" {
		return operatorAccessTokenPayload{}, ErrAdminAuthUnauthenticated
	}

	// Step 2: 中立 verifier で署名と JSON object 性を検証する。
	verifiedPayload, err := s.signer.VerifyJSON(accessToken)
	if err != nil {
		return operatorAccessTokenPayload{}, ErrAdminAuthUnauthenticated
	}

	// Step 3: Admin operator claim DTO として decode し、必須値は domain constructor 側で検証する。
	var payload operatorAccessTokenPayload
	if err := json.Unmarshal(verifiedPayload, &payload); err != nil {
		return operatorAccessTokenPayload{}, ErrAdminAuthUnauthenticated
	}
	if err := payload.validate(); err != nil {
		return operatorAccessTokenPayload{}, ErrAdminAuthUnauthenticated
	}

	// Step 4: 有効な Admin access payload として返す。
	return payload, nil
}

func (s *Service) findOperatorByCredential(ctx context.Context, credentialHandle string) (domain.Operator, error) {
	// Step 1: repository port から Admin Operator snapshot を取得する。
	snapshot, err := s.operators.FindOperatorByCredential(ctx, credentialHandle)
	if err != nil {
		return zeroAdminOperator(), mapAdminStoreError(err)
	}

	// Step 2: snapshot を Admin Operator domain object へ復元し、不正 role/state を拒否する。
	return operatorFromSnapshot(snapshot)
}

func (s *Service) findOperatorByID(ctx context.Context, operatorID string) (domain.Operator, error) {
	// Step 1: OperatorID を domain value object として検証し、未検証 selector を repository へ渡さない。
	validatedID, err := domain.NewOperatorID(operatorID)
	if err != nil {
		return zeroAdminOperator(), ErrAdminAuthUnauthenticated
	}

	// Step 2: repository port から Admin Operator snapshot を取得し、Product account repository へは委譲しない。
	snapshot, err := s.operators.FindOperatorByID(ctx, validatedID.String())
	if err != nil {
		return zeroAdminOperator(), mapAdminStoreError(err)
	}

	// Step 3: snapshot を Admin Operator domain object へ復元し、不正 role/state を拒否する。
	return operatorFromSnapshot(snapshot)
}

func (s *Service) operatorAndSessionByID(ctx context.Context, sessionID string) (domain.Operator, domain.OperatorAuthSession, error) {
	// Step 1: session ID で保存済み Admin refresh session state を取得する。
	record, err := s.sessions.GetOperatorSession(ctx, sessionID)
	if err != nil {
		return zeroAdminOperator(), zeroAdminOperatorSession(), mapAdminStoreError(err)
	}

	// Step 2: record から Operator と session domain object を復元する。
	return s.operatorAndSession(ctx, record)
}

func (s *Service) operatorAndSession(ctx context.Context, record OperatorSessionRecord) (domain.Operator, domain.OperatorAuthSession, error) {
	// Step 1: session record の owner OperatorID から現在 snapshot を取得する。
	operatorSnapshot, err := s.operators.FindOperatorByID(ctx, record.OperatorID)
	if err != nil {
		return zeroAdminOperator(), zeroAdminOperatorSession(), mapAdminStoreError(err)
	}
	operator, err := operatorFromSnapshot(operatorSnapshot)
	if err != nil {
		return zeroAdminOperator(), zeroAdminOperatorSession(), err
	}

	// Step 2: session record を Admin OperatorAuth domain object へ復元する。
	session, err := sessionFromRecord(record)
	if err != nil {
		return zeroAdminOperator(), zeroAdminOperatorSession(), mapAdminDomainAuthError(err)
	}

	// Step 3: 復元した Operator と session を返す。
	return operator, session, nil
}

func (s *Service) claimsFromPayload(operator domain.Operator, session domain.OperatorAuthSession, payload operatorAccessTokenPayload) (domain.OperatorAccessTokenClaims, error) {
	// Step 1: payload の JTI と時刻を domain constructor へ渡せる値にする。
	tokenID, err := domain.NewTokenJTI(payload.TokenID)
	if err != nil {
		return zeroAdminOperatorClaims(), ErrAdminAuthUnauthenticated
	}
	issuedAt := time.Unix(payload.IssuedAt, 0).UTC()
	accessTTL, err := domain.NewTokenTTL(time.Duration(payload.ExpiresAt-payload.IssuedAt) * time.Second)
	if err != nil {
		return zeroAdminOperatorClaims(), ErrAdminAuthUnauthenticated
	}

	// Step 2: Admin OperatorAuth domain constructor で claims を復元し、snapshot を検証する。
	claims, err := domain.NewOperatorAccessTokenClaims(operator, session, tokenID, accessTTL, issuedAt)
	if err != nil {
		return zeroAdminOperatorClaims(), mapAdminDomainAuthError(err)
	}

	// Step 3: payload の snapshot が domain claims と一致することを確認する。
	if !payload.matchesClaims(claims) {
		return zeroAdminOperatorClaims(), ErrAdminAuthUnauthenticated
	}

	// Step 4: 復元済み claims を返す。
	return claims, nil
}

func (s *Service) sessionResult(operator domain.Operator, issued issuedOperatorSession) OperatorSessionResult {
	// Step 1: refreshToken は Cookie command の Value だけに置き、body DTO へは field を作らない。
	return OperatorSessionResult{
		AccessToken: issued.accessToken,
		CSRFToken:   issued.csrfToken,
		Operator:    mapOperatorDTO(operator),
		SessionID:   issued.session.ID().String(),
		ExpiresAt:   issued.session.ExpiresAt(),
		RefreshCookie: RefreshCookieCommand{
			Name:     adminRefreshCookieName,
			Value:    issued.refreshToken,
			MaxAge:   s.config.RefreshCookieLifetime,
			HTTPOnly: true,
			Secure:   true,
			SameSite: "Lax",
			Path:     "/",
		},
	}
}

func operatorFromSnapshot(snapshot OperatorSnapshot) (domain.Operator, error) {
	// Step 1: application DTO の primitive 値を Admin Operator domain value object へ変換する。
	operatorID, err := domain.NewOperatorID(snapshot.ID)
	if err != nil {
		return zeroAdminOperator(), ErrAdminAuthUnauthenticated
	}
	operatorEmail, err := domain.NewOperatorEmail(snapshot.Email)
	if err != nil {
		return zeroAdminOperator(), ErrAdminAuthUnauthenticated
	}

	// Step 2: domain.NewOperator に role/active/passkey state の不変条件を委譲する。
	operator, err := domain.NewOperator(
		operatorID,
		operatorEmail,
		domain.OperatorRole(snapshot.Role),
		snapshot.Active,
		domain.OperatorPasskeyRegistrationState(snapshot.PasskeyRegistrationState),
	)
	if err != nil {
		return zeroAdminOperator(), mapAdminDomainAuthError(err)
	}

	// Step 3: 復元済み Operator domain object を返す。
	return operator, nil
}

func sessionFromRecord(record OperatorSessionRecord) (domain.OperatorAuthSession, error) {
	// Step 1: session record の primitive 値を domain value object へ変換する。
	sessionID, err := domain.NewOperatorSessionID(record.SessionID)
	if err != nil {
		return zeroAdminOperatorSession(), err
	}
	operatorID, err := domain.NewOperatorID(record.OperatorID)
	if err != nil {
		return zeroAdminOperatorSession(), err
	}

	// Step 2: ReconstituteOperatorAuthSession で state 全体の不変条件を検証する。
	return domain.ReconstituteOperatorAuthSession(
		sessionID,
		operatorID,
		domain.OpaqueTokenHash(record.RefreshTokenHash),
		domain.OpaqueTokenHash(record.CSRFTokenHash),
		domain.OperatorRole(record.RoleSnapshot),
		record.ActiveSnapshot,
		record.IssuedAt,
		record.ExpiresAt,
		record.Revoked,
	)
}

func zeroAdminOperator() domain.Operator {
	// Step 1: error return 専用の zero value を var 経由で作り、Admin Operator の生成は constructor/reconstitution に限定する。
	var operator domain.Operator
	return operator
}

func zeroAdminOperatorSession() domain.OperatorAuthSession {
	// Step 1: error return 専用の zero value を var 経由で作り、OperatorAuthSession の復元は domain helper だけに集約する。
	var session domain.OperatorAuthSession
	return session
}

func zeroAdminOperatorClaims() domain.OperatorAccessTokenClaims {
	// Step 1: error return 専用の zero value を var 経由で作り、claims の成功 path は domain constructor の結果だけにする。
	var claims domain.OperatorAccessTokenClaims
	return claims
}

func recordFromSession(session domain.OperatorAuthSession) OperatorSessionRecord {
	// Step 1: domain session から保存に必要な hash/snapshot だけを抜き出す。
	return OperatorSessionRecord{
		SessionID:        session.ID().String(),
		OperatorID:       session.OperatorID().String(),
		RefreshTokenHash: session.RefreshTokenHash().String(),
		CSRFTokenHash:    session.CSRFTokenHash().String(),
		RoleSnapshot:     string(session.RoleSnapshot()),
		ActiveSnapshot:   session.ActiveSnapshot(),
		IssuedAt:         session.IssuedAt(),
		ExpiresAt:        session.ExpiresAt(),
		Revoked:          session.Revoked(),
	}
}

func mapOperatorDTO(operator domain.Operator) OperatorDTO {
	// Step 1: Admin Operator domain object を external-facing DTO へ変換する。
	return OperatorDTO{
		ID:                       operator.ID().String(),
		Email:                    operator.Email().String(),
		Role:                     string(operator.Role()),
		Active:                   operator.Active(),
		PasskeyRegistrationState: string(operator.PasskeyRegistrationState()),
	}
}

func mapOperatorDTOWithSession(operator domain.Operator, sessionID string) OperatorDTO {
	// Step 1: 共通 profile DTO を先に作り、role/active/passkey state の写像を一箇所に集約する。
	dto := mapOperatorDTO(operator)

	// Step 2: middleware が request context へ session selector を束縛できるよう、検証済み accessToken payload 由来の session ID だけを追加する。
	dto.SessionID = sessionID
	return dto
}

func mapAdminStoreError(err error) error {
	// Step 1: 保存層の not found/expired/revoked 系は認証失敗へ畳み、詳細を隠す。
	if errors.Is(err, domain.ErrSessionNotFound) || errors.Is(err, domain.ErrSessionExpired) || errors.Is(err, domain.ErrSessionRevoked) {
		return ErrAdminAuthUnauthenticated
	}

	// Step 2: 保存層の利用不能は内部エラーとして fail-closed にする。
	if errors.Is(err, domain.ErrAuthStoreUnavailable) {
		return ErrAdminAuthInternal
	}

	// Step 3: 未分類の store error も外部へ詳細を出さず内部エラーにする。
	return ErrAdminAuthInternal
}

func mapAdminDomainAuthError(err error) error {
	// Step 1: 期限切れ、失効、形式不備は認証失敗として扱う。
	if errors.Is(err, domain.ErrTokenExpired) || errors.Is(err, domain.ErrSessionExpired) || errors.Is(err, domain.ErrSessionRevoked) || errors.Is(err, domain.ErrInvalidToken) {
		return ErrAdminAuthUnauthenticated
	}

	// Step 2: inactive、permission、CSRF mismatch は mutation 禁止として扱う。
	if errors.Is(err, domain.ErrOperatorAuthInactive) || errors.Is(err, domain.ErrOperatorAuthPermissionDenied) || errors.Is(err, domain.ErrOperatorAuthCSRFMismatch) {
		return ErrAdminAuthForbidden
	}

	// Step 3: snapshot/session mismatch は Product token 混入も含むため認証失敗へ畳む。
	if errors.Is(err, domain.ErrOperatorAuthSnapshotMismatch) || errors.Is(err, domain.ErrOperatorAuthSessionMismatch) {
		return ErrAdminAuthUnauthenticated
	}

	// Step 4: その他の domain 不変条件違反は request を進めず認証失敗として扱う。
	return ErrAdminAuthUnauthenticated
}

func mapAdminPasskeyStoreError(err error) error {
	// Step 1: passkey 不在は selector 不正として扱い、他 Operator 所有 credential の存在有無を詳細に出さない。
	if errors.Is(err, domain.ErrSessionNotFound) || errors.Is(err, domain.ErrAccountAuthNotFound) {
		return ErrAdminAuthPasskeyNotFound
	}

	// Step 2: repository 側の二重防御で最後の passkey 削除が検出された場合も domain error と同じ conflict に畳む。
	if errors.Is(err, domain.ErrOperatorLastPasskeyDeletion) {
		return ErrAdminAuthLastPasskey
	}

	// Step 3: 保存層の利用不能は内部エラーとして fail-closed にする。
	if errors.Is(err, domain.ErrAuthStoreUnavailable) {
		return ErrAdminAuthInternal
	}

	// Step 4: 未分類の store error は外部へ詳細を出さず内部エラーにする。
	return ErrAdminAuthInternal
}

func mapAdminPasskeyDomainError(err error) error {
	// Step 1: 最後の credential 削除拒否は UI が再取得できる conflict として扱う。
	if errors.Is(err, domain.ErrOperatorLastPasskeyDeletion) {
		return ErrAdminAuthLastPasskey
	}

	// Step 2: その他の domain 不変条件違反は request を進めず認証失敗として扱う。
	return ErrAdminAuthUnauthenticated
}

func clearRefreshCookieCommand() RefreshCookieCommand {
	// Step 1: adapter が同一 Cookie を削除できるよう、属性と Clear flag をまとめる。
	return RefreshCookieCommand{
		Name:     adminRefreshCookieName,
		MaxAge:   0,
		HTTPOnly: true,
		Secure:   true,
		SameSite: "Lax",
		Path:     "/",
		Clear:    true,
	}
}
