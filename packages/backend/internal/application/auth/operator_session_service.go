package auth

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	domain "www-template/packages/backend/internal/domain"
)

// OperatorSessionService は Operator auth の application boundary である。
//
// 役割:
//   - Operator session 発行、refresh rotation、current operator、mutation authorization、logout を提供する。
//   - Product auth application や Product-specific domain object を import せず、OperatorAuth domain object だけを使う。
//   - refreshToken 平文を response body DTO に置かず、HttpOnly Cookie command 専用値として adapter へ渡す。
type OperatorSessionService struct {
	operators OperatorRepository
	sessions  OperatorRefreshSessionStore
	signer    JSONSignVerifier
	secrets   OpaqueTokenGenerator
	ids       IDGenerator
	clock     func() time.Time
	config    OperatorSessionConfig
}

// OperatorSessionDependencies は OperatorSessionService の必須 port をまとめた DTO である。
//
// 役割:
//   - AccountSessionDependencies と同じ constructor-time validation 形式にそろえ、欠落依存を fail-close に拒否する。
//   - repository、store、signer、secret generator、ID generator、clock の具象型を application public API へ持ち込まない。
type OperatorSessionDependencies struct {
	Operators       OperatorRepository
	RefreshSessions OperatorRefreshSessionStore
	Signer          JSONSignVerifier
	TokenGenerator  OpaqueTokenGenerator
	IDGenerator     IDGenerator
	Clock           func() time.Time
}

// NewOperatorSessionService は Operator auth OperatorSessionService を生成する。
//
// 引数:
//   - deps: Operator snapshot repository、refresh session store、signer、secret generator、ID generator、clock。
//   - config: Operator auth の TTL と Cookie lifetime 設定。
//
// 戻り値:
//   - *OperatorSessionService: 検証済み依存を保持する use case service。
//   - error: TTL/Cookie lifetime が不正、または必須依存が nil の場合。
func NewOperatorSessionService(deps OperatorSessionDependencies, config OperatorSessionConfig) (*OperatorSessionService, error) {
	// Step 1: 必須 port が欠けた状態で認証境界を作ると fail-open になり得るため拒否する。
	if err := validateOperatorSessionDependencies(deps); err != nil {
		return nil, ErrOperatorAuthUnavailable
	}

	// Step 2: token TTL と Cookie lifetime の関係を domain primitive で検証し、Cookie の長すぎる保持を防ぐ。
	refreshTTL, err := domain.ValidateTokenTTL(config.OperatorRefreshSessionTTL)
	if err != nil {
		return nil, ErrOperatorAuthUnavailable
	}
	if err := domain.ValidateTokenCookieLifetime(config.OperatorRefreshCookieLifetime, refreshTTL); err != nil {
		return nil, ErrOperatorAuthUnavailable
	}
	if _, err := domain.ValidateTokenTTL(config.OperatorAccessTokenTTL); err != nil {
		return nil, ErrOperatorAuthUnavailable
	}

	// Step 3: 検証済み依存だけを Service に保持し、以後の use case が同じ境界を共有する。
	return &OperatorSessionService{
		operators: deps.Operators,
		sessions:  deps.RefreshSessions,
		signer:    deps.Signer,
		secrets:   deps.TokenGenerator,
		ids:       deps.IDGenerator,
		clock:     deps.Clock,
		config:    config,
	}, nil
}

func validateOperatorSessionDependencies(deps OperatorSessionDependencies) error {
	// Step 1: session lifecycle が使う必須依存をまとめて検証し、部分的な service 構築を防ぐ。
	if deps.Operators == nil || deps.RefreshSessions == nil || deps.Signer == nil || deps.TokenGenerator == nil || deps.IDGenerator == nil || deps.Clock == nil {
		return ErrOperatorAuthUnavailable
	}

	// Step 2: passkey login の challenge provider は OperatorPasskeyLoginService 側で必須検証するため、session lifecycle service には保持しない。
	return nil
}

// RefreshOperatorSession は refresh credential を rotation し、新しい accessToken と refresh credential command を返す。
//
// 古い refreshToken は OperatorRefreshSessionStore.Rotate で hash 一致を確認しながら置換される。
func (s *OperatorSessionService) RefreshOperatorSession(ctx context.Context, input RefreshOperatorSessionInput) (OperatorSessionResult, error) {
	// Step 1: Cookie または Bearer body で提示された refresh credential から session selector と opaque token を分離する。
	sessionID, err := parseRefreshCookieSessionID(input.RefreshTokenValue)
	if err != nil {
		return OperatorSessionResult{}, err
	}

	// Step 2: 保存済み session state を復元し、refresh state を参照しない。
	record, err := s.sessions.Get(ctx, sessionID.String())
	if err != nil {
		return OperatorSessionResult{}, mapOperatorStoreError(err)
	}
	operator, session, err := s.operatorAndSession(ctx, record)
	if err != nil {
		return OperatorSessionResult{}, err
	}

	// Step 3: canonical auth lifecycle の context ownership check で path と server-side session selector を一致検証し、Cookie Path だけを認証境界として信頼しない。
	if err := domain.EnsureRefreshContext(input.AuthContextID, session.ID().String()); err != nil {
		return OperatorSessionResult{}, ErrOperatorAuthUnauthenticated
	}

	// Step 4: OperatorAuth domain object で refreshToken が session と一致することを検証する。
	if err := session.ValidateRefreshToken(input.RefreshTokenValue, s.clock()); err != nil {
		return OperatorSessionResult{}, mapOperatorDomainAuthError(err)
	}

	// Step 5: 新しい session state を発行し、store の atomic rotation port へ渡す。
	issued, err := s.issueSession(ctx, operator)
	if err != nil {
		return OperatorSessionResult{}, err
	}
	if err := s.sessions.Rotate(ctx, session.ID().String(), session.RefreshTokenHash().String(), issued.record, s.config.OperatorRefreshSessionTTL); err != nil {
		return OperatorSessionResult{}, mapOperatorStoreError(err)
	}

	// Step 6: 新しい accessToken と refresh credential command を返し、transport mode ごとの露出判断は HTTP adapter に閉じる。
	return s.sessionResult(operator, issued), nil
}

// CurrentOperator は accessToken から現在の Operator DTO を返す。
//
// current operator は mutation ではないため CSRF は要求せず、session と operator snapshot の一致だけを検証する。
func (s *OperatorSessionService) CurrentOperator(ctx context.Context, input CurrentOperatorInput) (OperatorDTO, error) {
	// Step 1: 署名済み accessToken を検証し、operator claim payload として復元する。
	payload, err := s.verifyAccessPayload(input.AccessToken)
	if err != nil {
		return OperatorDTO{}, err
	}

	// Step 2: payload の session ID から Operator session state と現在 Operator を取得する。
	operator, session, err := s.operatorAndSessionByID(ctx, payload.SessionID)
	if err != nil {
		return OperatorDTO{}, err
	}

	// Step 3: current endpoint 用に session/snapshot/expiry を検証し、viewer は許可する。
	if err := validateCurrentOperatorPayload(operator, session, payload, s.clock()); err != nil {
		return OperatorDTO{}, err
	}

	// Step 4: adapter/frontend 向けの Operator DTO と、middleware context binding に必要な session ID を返す。
	return mapOperatorDTOWithSession(operator, payload.SessionID), nil
}

// AuthorizeOperatorSession は mutation route の bearer/session/snapshot 検証後に permission を判定する。
//
// 役割:
//   - HTTP adapter が RBAC role matrix を再実装せず、OperatorAuth domain の ValidateAccess へ判定を委譲できるようにする。
//   - Product bearer token、refresh Cookie、CSRF token を permission 判定材料にせず、署名済み operator accessToken と server-side session record だけを使う。
//   - read/current route とは別に mutation permission を必須化し、permission 未指定 route を fail-closed にする。
func (s *OperatorSessionService) AuthorizeOperatorSession(ctx context.Context, input AuthorizeOperatorSessionInput) (OperatorAuthorizationDecision, error) {
	// Step 1: route permission が空の場合、mutation を RBAC なしで進めないため forbidden として拒否する。
	if input.Permission == "" {
		return OperatorAuthorizationDecision{}, ErrOperatorAuthForbidden
	}

	// Step 2: 署名済み accessToken を検証し、operator claim payload として復元する。
	payload, err := s.verifyAccessPayload(input.AccessToken)
	if err != nil {
		return OperatorAuthorizationDecision{}, err
	}

	// Step 3: payload の session ID から server-side session state と現在 Operator snapshot を取得する。
	operator, session, err := s.operatorAndSessionByID(ctx, payload.SessionID)
	if err != nil {
		return OperatorAuthorizationDecision{}, err
	}

	// Step 4: permission 文字列を domain permission として扱い、session.ValidateAccess に session/snapshot/permission 照合を委譲する。
	permission := domain.OperatorAuthPermission(input.Permission)
	if err := validateOperatorAccessPayload(operator, session, payload, permission, s.clock()); err != nil {
		return OperatorAuthorizationDecision{}, err
	}

	// Step 5: 許可済み operator DTO と permission を返し、HTTP adapter が role matrix を再判定しないようにする。
	return OperatorAuthorizationDecision{
		Operator:   mapOperatorDTOWithSession(operator, payload.SessionID),
		SessionID:  payload.SessionID,
		Permission: input.Permission,
		Allowed:    true,
	}, nil
}

// LogoutOperator は accessToken が示す Operator session を revoke する。
//
// refreshToken 平文は不要であり、返却する Cookie command は adapter が refresh Cookie を削除するためだけに使う。
func (s *OperatorSessionService) LogoutOperator(ctx context.Context, input LogoutOperatorInput) (OperatorRefreshCookieCommand, error) {
	// Step 1: accessToken payload を検証し、対象 session selector を取得する。
	payload, err := s.verifyAccessPayload(input.AccessToken)
	if err != nil {
		return OperatorRefreshCookieCommand{}, err
	}

	// Step 2: session state から現在 Operator と refresh session を復元し、logout 対象の所有者を確認する。
	operator, session, err := s.operatorAndSessionByID(ctx, payload.SessionID)
	if err != nil {
		return OperatorRefreshCookieCommand{}, err
	}

	// Step 3: logout も session 維持 mutation として OperatorAuth domain の ValidateAccess に通し、失効済み・期限切れ・snapshot mismatch を拒否する。
	if err := validateOperatorAccessPayload(operator, session, payload, domain.OperatorAuthPermissionOperatorsLogout, s.clock()); err != nil {
		return OperatorRefreshCookieCommand{}, err
	}

	// Step 4: store port へ revoke を委譲し、access/refresh eligibility を同じ session ID で失効させる。
	if err := s.sessions.Revoke(ctx, operator.ID().String(), payload.SessionID); err != nil {
		return OperatorRefreshCookieCommand{}, mapOperatorStoreError(err)
	}

	// Step 5: adapter が対象 auth context の HttpOnly Cookie だけを削除するための command を返す。
	return clearRefreshCookieCommand(payload.SessionID), nil
}

// IssueOperatorSession は確定済み Operator に session を発行する。
//
// 引数:
//   - ctx: session store への保存に使う cancellation context。
//   - input.OperatorID: passkey login や setup transaction で認証済みになった Operator の canonical ID。
//
// 戻り値:
//   - OperatorSessionResult: accessToken と HttpOnly refresh Cookie command を分離した session DTO。
//   - error: Operator が存在しない、inactive、未登録状態、または session 保存に失敗した場合の stable application error。
func (s *OperatorSessionService) IssueOperatorSession(ctx context.Context, input IssueOperatorSessionInput) (OperatorSessionResult, error) {
	// Step 1: 認証済み Operator snapshot を repository から復元し、account auth state を参照しない。
	operator, err := s.findOperatorByID(ctx, input.OperatorID)
	if err != nil {
		return OperatorSessionResult{}, err
	}

	// Step 2: passkey 未登録や inactive の Operator に session を出さないよう、domain permission と同じ前提を検証する。
	if !operator.Active() || operator.PasskeyRegistrationState() != domain.OperatorPasskeyRegistrationRegistered {
		return OperatorSessionResult{}, ErrOperatorAuthForbidden
	}

	// Step 3: login/setup で同じ issueSession path を使い、flow 別 token 形式を作らない。
	issued, err := s.issueSession(ctx, operator)
	if err != nil {
		return OperatorSessionResult{}, err
	}
	if err := s.sessions.Save(ctx, issued.record, s.config.OperatorRefreshSessionTTL); err != nil {
		return OperatorSessionResult{}, ErrOperatorAuthUnavailable
	}
	return s.sessionResult(operator, issued), nil
}

type issuedOperatorSession struct {
	session      domain.OperatorAuthSession
	record       OperatorSessionRecord
	accessToken  string
	refreshToken string
}

func (s *OperatorSessionService) issueSession(ctx context.Context, operator domain.Operator) (issuedOperatorSession, error) {
	// Step 1: session ID、refreshToken、accessToken JTI を外部 port から発行する。
	sessionID, tokenID, refreshToken, err := s.nextSessionSecrets()
	if err != nil {
		return issuedOperatorSession{}, err
	}
	issuedAt := s.clock()
	ttl, err := domain.NewTokenTTL(s.config.OperatorRefreshSessionTTL)
	if err != nil {
		return issuedOperatorSession{}, ErrOperatorAuthUnavailable
	}

	// Step 2: OperatorAuth domain object で refresh session state を構築する。
	session, err := domain.NewOperatorAuthSession(operator, sessionID, refreshToken, ttl, issuedAt)
	if err != nil {
		return issuedOperatorSession{}, mapOperatorDomainAuthError(err)
	}

	// Step 3: accessToken claims も OperatorAuth domain object で構築する。
	accessTTL, err := domain.NewTokenTTL(s.config.OperatorAccessTokenTTL)
	if err != nil {
		return issuedOperatorSession{}, ErrOperatorAuthUnavailable
	}
	claims, err := domain.NewOperatorAccessTokenClaims(operator, session, tokenID, accessTTL, issuedAt)
	if err != nil {
		return issuedOperatorSession{}, mapOperatorDomainAuthError(err)
	}
	// Step 3-a: canonical lifecycle へ渡す subject payload を明示的に作り、account subject と discriminator で切り替えない。
	subject, err := NewOperatorSubjectPayload(operator.ID().String(), session.ID().String())
	if err != nil {
		return issuedOperatorSession{}, mapOperatorDomainAuthError(err)
	}

	// Step 4: claim DTO を中立 signer で署名し、payload 意味はこの use case に閉じる。
	accessToken, err := s.signClaims(claims)
	if err != nil {
		return issuedOperatorSession{}, err
	}

	// Step 5: 保存用 record は hash/snapshot だけを含め、平文 secret を store へ渡さない。
	return issuedOperatorSession{
		session:      session,
		record:       recordFromSessionWithSubject(session, subject),
		accessToken:  accessToken,
		refreshToken: refreshToken,
	}, nil
}

func (s *OperatorSessionService) nextSessionSecrets() (domain.OperatorSessionID, domain.TokenJTI, string, error) {
	// Step 1: session ID と JTI は生成後に domain constructor で形式検証する。
	rawSessionID, err := s.ids.Next()
	if err != nil {
		return "", "", "", ErrOperatorAuthUnavailable
	}
	sessionID, err := domain.NewOperatorSessionID(rawSessionID)
	if err != nil {
		return "", "", "", ErrOperatorAuthUnavailable
	}
	rawTokenID, err := s.ids.Next()
	if err != nil {
		return "", "", "", ErrOperatorAuthUnavailable
	}
	tokenID, err := domain.NewTokenJTI(rawTokenID)
	if err != nil {
		return "", "", "", ErrOperatorAuthUnavailable
	}

	// Step 2: refreshToken は opaque secret として発行する。
	refreshToken, err := s.refreshCookieValue(sessionID)
	if err != nil {
		return "", "", "", err
	}

	// Step 3: 生成済みの session secrets を返す。
	return sessionID, tokenID, refreshToken, nil
}

func (s *OperatorSessionService) refreshCookieValue(sessionID domain.OperatorSessionID) (string, error) {
	// Step 1: opaque secret を発行し、session selector と組み合わせて Cookie value にする。
	secret, err := s.secrets.NewToken()
	if err != nil {
		return "", ErrOperatorAuthUnavailable
	}

	// Step 2: Cookie value は sessionID.secret の形にして、refresh 時に O(1) で session を選択できるようにする。
	return sessionID.String() + "." + secret, nil
}

func (s *OperatorSessionService) signClaims(claims domain.OperatorAccessTokenClaims) (string, error) {
	// Step 1: claim payload へ変換し、account claim を混入させない。
	payload := operatorAccessPayload{
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
		return "", ErrOperatorAuthUnavailable
	}
	tokenString, err := s.signer.SignJSON(encoded)
	if err != nil {
		return "", ErrOperatorAuthUnavailable
	}

	// Step 3: compact token を accessToken body field として返す。
	return tokenString, nil
}

func (s *OperatorSessionService) verifyAccessPayload(accessToken string) (operatorAccessPayload, error) {
	// Step 1: 空 token は署名検証へ進めず、認証なしとして扱う。
	if strings.TrimSpace(accessToken) == "" {
		return operatorAccessPayload{}, ErrOperatorAuthUnauthenticated
	}

	// Step 2: 中立 verifier で署名と JSON object 性を検証する。
	verifiedPayload, err := s.signer.VerifyJSON(accessToken)
	if err != nil {
		return operatorAccessPayload{}, ErrOperatorAuthUnauthenticated
	}

	// Step 3: operator claim DTO として decode し、必須値は domain constructor 側で検証する。
	var payload operatorAccessPayload
	if err := json.Unmarshal(verifiedPayload, &payload); err != nil {
		return operatorAccessPayload{}, ErrOperatorAuthUnauthenticated
	}
	if err := payload.validate(); err != nil {
		return operatorAccessPayload{}, ErrOperatorAuthUnauthenticated
	}

	// Step 4: 有効な access payload として返す。
	return payload, nil
}

func (s *OperatorSessionService) findOperatorByID(ctx context.Context, operatorID string) (domain.Operator, error) {
	// Step 1: OperatorID を domain value object として検証し、未検証 selector を repository へ渡さない。
	validatedID, err := domain.NewOperatorID(operatorID)
	if err != nil {
		return zeroOperator(), ErrOperatorAuthUnauthenticated
	}

	// Step 2: repository port から Operator snapshot を取得し、account repository へは委譲しない。
	snapshot, err := s.operators.FindOperatorByID(ctx, validatedID.String())
	if err != nil {
		return zeroOperator(), mapOperatorStoreError(err)
	}

	// Step 3: snapshot を Operator domain object へ復元し、不正 role/state を拒否する。
	return operatorFromSnapshot(snapshot)
}

func (s *OperatorSessionService) operatorAndSessionByID(ctx context.Context, sessionID string) (domain.Operator, domain.OperatorAuthSession, error) {
	// Step 1: session ID で保存済み refresh session state を取得する。
	record, err := s.sessions.Get(ctx, sessionID)
	if err != nil {
		return zeroOperator(), zeroOperatorSession(), mapOperatorStoreError(err)
	}

	// Step 2: record から Operator と session domain object を復元する。
	return s.operatorAndSession(ctx, record)
}

func (s *OperatorSessionService) operatorAndSession(ctx context.Context, record OperatorSessionRecord) (domain.Operator, domain.OperatorAuthSession, error) {
	// Step 1: session record の owner OperatorID から現在 snapshot を取得する。
	operatorSnapshot, err := s.operators.FindOperatorByID(ctx, record.OperatorID)
	if err != nil {
		return zeroOperator(), zeroOperatorSession(), mapOperatorStoreError(err)
	}
	operator, err := operatorFromSnapshot(operatorSnapshot)
	if err != nil {
		return zeroOperator(), zeroOperatorSession(), err
	}

	// Step 2: session record を OperatorAuth domain object へ復元する。
	session, err := sessionFromRecord(record)
	if err != nil {
		return zeroOperator(), zeroOperatorSession(), mapOperatorDomainAuthError(err)
	}

	// Step 3: 復元した Operator と session を返す。
	return operator, session, nil
}

func (s *OperatorSessionService) sessionResult(operator domain.Operator, issued issuedOperatorSession) OperatorSessionResult {
	// Step 1: refreshToken は Cookie command の Value だけに置き、body DTO へは field を作らない。
	return OperatorSessionResult{
		AccessToken: issued.accessToken,
		Operator:    mapOperatorDTO(operator),
		SessionID:   issued.session.ID().String(),
		ExpiresAt:   issued.session.ExpiresAt(),
		RefreshCookie: OperatorRefreshCookieCommand{
			AuthContextID: issued.session.ID().String(),
			Value:         issued.refreshToken,
			MaxAge:        s.config.OperatorRefreshCookieLifetime,
		},
	}
}

func operatorFromSnapshot(snapshot OperatorSnapshot) (domain.Operator, error) {
	// Step 1: application DTO の primitive 値を Operator domain value object へ変換する。
	operatorID, err := domain.NewOperatorID(snapshot.ID)
	if err != nil {
		return zeroOperator(), ErrOperatorAuthUnauthenticated
	}
	operatorEmail, err := domain.NewOperatorEmail(snapshot.Email)
	if err != nil {
		return zeroOperator(), ErrOperatorAuthUnauthenticated
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
		return zeroOperator(), mapOperatorDomainAuthError(err)
	}

	// Step 3: 復元済み Operator domain object を返す。
	return operator, nil
}

func sessionFromRecord(record OperatorSessionRecord) (domain.OperatorAuthSession, error) {
	// Step 1: session record の primitive 値を domain value object へ変換する。
	sessionID, err := domain.NewOperatorSessionID(record.SessionID)
	if err != nil {
		return zeroOperatorSession(), err
	}
	operatorID, err := domain.NewOperatorID(record.OperatorID)
	if err != nil {
		return zeroOperatorSession(), err
	}

	// Step 2: ReconstituteOperatorAuthSession で state 全体の不変条件を検証する。
	return domain.ReconstituteOperatorAuthSession(
		sessionID,
		operatorID,
		domain.OpaqueTokenHash(record.RefreshTokenHash),
		domain.OperatorRole(record.RoleSnapshot),
		record.ActiveSnapshot,
		record.IssuedAt,
		record.ExpiresAt,
		record.Revoked,
	)
}

func zeroOperator() domain.Operator {
	// Step 1: error return 専用の zero value を var 経由で作り、Operator の生成は constructor/reconstitution に限定する。
	var operator domain.Operator
	return operator
}

func zeroOperatorSession() domain.OperatorAuthSession {
	// Step 1: error return 専用の zero value を var 経由で作り、OperatorAuthSession の復元は domain helper だけに集約する。
	var session domain.OperatorAuthSession
	return session
}

func recordFromSessionWithSubject(session domain.OperatorAuthSession, subject OperatorSubjectPayload) OperatorSessionRecord {
	// Step 1: domain session と explicit subject payload の owner/session が一致することを保存 DTO の入力境界で固定する。
	if subject.OperatorID() != session.OperatorID() || subject.SessionID() != session.ID() {
		return OperatorSessionRecord{}
	}

	// Step 2: domain session から保存に必要な hash/snapshot だけを抜き出す。
	return OperatorSessionRecord{
		SessionID:        subject.SessionID().String(),
		OperatorID:       subject.OperatorID().String(),
		RefreshTokenHash: session.RefreshTokenHash().String(),
		RoleSnapshot:     string(session.RoleSnapshot()),
		ActiveSnapshot:   session.ActiveSnapshot(),
		IssuedAt:         session.IssuedAt(),
		ExpiresAt:        session.ExpiresAt(),
		Revoked:          session.Revoked(),
	}
}

func mapOperatorDTO(operator domain.Operator) OperatorDTO {
	// Step 1: Operator domain object を external-facing DTO へ変換する。
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

func mapOperatorStoreError(err error) error {
	// Step 1: 保存層の not found/expired/revoked 系は認証失敗へ畳み、詳細を隠す。
	if errors.Is(err, domain.ErrSessionNotFound) || errors.Is(err, domain.ErrSessionExpired) || errors.Is(err, domain.ErrSessionRevoked) {
		return ErrOperatorAuthUnauthenticated
	}

	// Step 2: 保存層の利用不能は内部エラーとして fail-closed にする。
	if errors.Is(err, domain.ErrAuthStoreUnavailable) {
		return ErrOperatorAuthUnavailable
	}

	// Step 3: 未分類の store error も外部へ詳細を出さず内部エラーにする。
	return ErrOperatorAuthUnavailable
}

func mapOperatorDomainAuthError(err error) error {
	// Step 1: 期限切れ、失効、形式不備は認証失敗として扱う。
	if errors.Is(err, domain.ErrTokenExpired) || errors.Is(err, domain.ErrSessionExpired) || errors.Is(err, domain.ErrSessionRevoked) || errors.Is(err, domain.ErrInvalidToken) {
		return ErrOperatorAuthUnauthenticated
	}

	// Step 2: inactive、permission mismatch は mutation 禁止として扱う。
	if errors.Is(err, domain.ErrOperatorAuthInactive) || errors.Is(err, domain.ErrOperatorAuthPermissionDenied) {
		return ErrOperatorAuthForbidden
	}

	// Step 3: snapshot/session mismatch は Product token 混入も含むため認証失敗へ畳む。
	if errors.Is(err, domain.ErrOperatorAuthSnapshotMismatch) || errors.Is(err, domain.ErrOperatorAuthSessionMismatch) {
		return ErrOperatorAuthUnauthenticated
	}

	// Step 4: その他の domain 不変条件違反は request を進めず認証失敗として扱う。
	return ErrOperatorAuthUnauthenticated
}

func mapOperatorPasskeyStoreError(err error) error {
	// Step 1: passkey 不在は selector 不正として扱い、他 Operator 所有 credential の存在有無を詳細に出さない。
	if errors.Is(err, domain.ErrSessionNotFound) || errors.Is(err, domain.ErrAccountAuthNotFound) {
		return ErrOperatorAuthPasskeyNotFound
	}

	// Step 2: repository 側の二重防御で最後の passkey 削除が検出された場合も domain error と同じ conflict に畳む。
	if errors.Is(err, domain.ErrOperatorLastPasskeyDeletion) {
		return ErrOperatorAuthLastPasskey
	}

	// Step 3: 保存層の利用不能は内部エラーとして fail-closed にする。
	if errors.Is(err, domain.ErrAuthStoreUnavailable) {
		return ErrOperatorAuthUnavailable
	}

	// Step 4: 未分類の store error は外部へ詳細を出さず内部エラーにする。
	return ErrOperatorAuthUnavailable
}

func mapOperatorPasskeyDomainError(err error) error {
	// Step 1: 最後の credential 削除拒否は UI が再取得できる conflict として扱う。
	if errors.Is(err, domain.ErrOperatorLastPasskeyDeletion) {
		return ErrOperatorAuthLastPasskey
	}

	// Step 2: その他の domain 不変条件違反は request を進めず認証失敗として扱う。
	return ErrOperatorAuthUnauthenticated
}

func clearRefreshCookieCommand(authContextID string) OperatorRefreshCookieCommand {
	// Step 1: application 層は auth context selector と削除意図だけを返し、Cookie 名・Path・browser policy などの transport 属性は HTTP adapter に委譲する。
	return OperatorRefreshCookieCommand{
		AuthContextID: authContextID,
		Clear:         true,
	}
}
