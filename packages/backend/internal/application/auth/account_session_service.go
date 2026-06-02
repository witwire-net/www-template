package auth

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"time"

	domain "www-template/packages/backend/internal/domain"
)

// AccountSessionService は Product account auth の login / refresh / revoke / bearer validation use case を提供する。
//
// 役割:
//   - Product AccountAuth domain object と application signer capability だけを使って Product 認証を組み立てる。
//   - Admin operator auth application/domain を import せず、Product-only application boundary を維持する。
//   - refreshToken は Cookie command へだけ渡し、response body DTO へ含めない。
//
// 使用例:
//
//	service, err := NewAccountSessionService(deps, cfg)
//	if err != nil {
//		return err
//	}
//	result, err := service.IssueAccountSession(ctx, input)
type AccountSessionService struct {
	accounts              AccountAuthRepository
	refreshSessions       AccountRefreshSessionStore
	sessions              AccountSessionMetadataStore
	signer                JSONSignVerifier
	idGenerator           IDGenerator
	tokenGenerator        OpaqueTokenGenerator
	clock                 func() time.Time
	accessTTL             domain.TokenTTL
	refreshTTL            domain.TokenTTL
	refreshCookieLifetime time.Duration
}

// AccountSessionDependencies は Product auth AccountSessionService の必須 port をまとめた DTO である。
//
// 役割:
//   - NewService の引数増加を抑えつつ、nil 依存を一括検証する。
//   - platform / adapter の具象型を application public API に持ち込まない。
type AccountSessionDependencies struct {
	Accounts        AccountAuthRepository
	RefreshSessions AccountRefreshSessionStore
	Sessions        AccountSessionMetadataStore
	Signer          JSONSignVerifier
	IDGenerator     IDGenerator
	TokenGenerator  OpaqueTokenGenerator
	Clock           func() time.Time
}

// NewAccountSessionService は Product account auth の AccountSessionService を生成する。
//
// 役割:
//   - 必須依存、accessToken TTL、refreshToken TTL、refresh Cookie lifetime を fail-close に検証する。
//   - Cookie lifetime が refreshToken server-side TTL を超えないことを domain primitive で確認する。
//
// 引数:
//   - deps: repository、store、signer、ID/token generator、clock の必須依存。
//   - cfg: accessToken / refreshToken / Cookie lifetime の設定 DTO。
//
// 戻り値:
//   - *AccountSessionService: 検証済み依存だけを持つ Product auth service。
//   - error: 必須依存または duration が不正な場合の application/domain error。
func NewAccountSessionService(deps AccountSessionDependencies, cfg AccountSessionConfig) (*AccountSessionService, error) {
	// Step 1: nil 依存を最初に拒否し、認証境界が部分的に fail-open しないようにする。
	if err := validateAccountSessionDependencies(deps); err != nil {
		return nil, err
	}

	// Step 2: accessToken TTL は Product accessToken claim の exp 計算に使うため正の duration に限定する。
	accessTTL, err := domain.ValidateTokenTTL(cfg.AccessTokenTTL)
	if err != nil {
		return nil, err
	}

	// Step 3: refreshToken TTL と Cookie lifetime の大小関係を domain primitive で検証する。
	refreshTTL, err := domain.ValidateTokenTTL(cfg.RefreshTokenTTL)
	if err != nil {
		return nil, err
	}
	if err := domain.ValidateTokenCookieLifetime(cfg.RefreshCookieLifetime, refreshTTL); err != nil {
		return nil, err
	}

	// Step 4: 検証済み依存だけを Service に保持し、以降の use case が同じ境界を使うようにする。
	return &AccountSessionService{
		accounts:              deps.Accounts,
		refreshSessions:       deps.RefreshSessions,
		sessions:              deps.Sessions,
		signer:                deps.Signer,
		idGenerator:           deps.IDGenerator,
		tokenGenerator:        deps.TokenGenerator,
		clock:                 deps.Clock,
		accessTTL:             accessTTL,
		refreshTTL:            refreshTTL,
		refreshCookieLifetime: cfg.RefreshCookieLifetime,
	}, nil
}

// IssueAccountSession は確定済み Product AccountID から canonical account auth session を発行する。
//
// 役割:
//   - WebAuthn 検証、recovery session 消費、passkey 追加など外側フローで AccountID が確定した後の session issuance を一箇所へ集約する。
//   - accessToken、refresh session、session metadata、refresh Cookie command を Product AccountAuth lifecycle の同じ手順で作る。
//   - 旧 root TokenService の Issue caller を production path から外し、Product session 発行の true owner をこの package に固定する。
//
// 引数:
//   - ctx: 呼び出し単位のキャンセル・期限情報。
//   - input: 確定済み AccountID と device metadata seed。
//
// 戻り値:
//   - AccountSessionResult: body 用 accessToken/session DTO と Set-Cookie 用 refresh command。
//   - error: Account 不在、停止、保存失敗、署名失敗などの application error。
func (s *AccountSessionService) IssueAccountSession(ctx context.Context, input IssueAccountSessionInput) (AccountSessionResult, error) {
	// Step 1: 確定済み AccountID の現在 projection を取得し、削除済み・停止済み account への発行を防ぐ。
	accountAuth, err := s.accounts.FindByID(ctx, input.AccountID)
	if err != nil {
		return AccountSessionResult{}, mapAccountAuthError(err)
	}
	if accountAuth.AccountID() != input.AccountID {
		return AccountSessionResult{}, ErrAccountAuthUnauthorized
	}

	// Step 2: AccountAuth projection を Account root へ戻し、status と session_revoked_after を domain constructor に検証させる。
	account, err := accountRootFromAuth(accountAuth)
	if err != nil {
		return AccountSessionResult{}, mapAccountAuthError(err)
	}

	// Step 3: Account root が確定した後の issuance 手順は outer flow と分離し、root legacy TokenService への迂回を作らない。
	return s.issueAccountSessionForAccount(ctx, account, deviceInput{clientIP: input.ClientIP, userAgent: input.UserAgent})
}

func (s *AccountSessionService) issueAccountSessionForAccount(ctx context.Context, account domain.Account, device deviceInput) (AccountSessionResult, error) {
	// Step 1: 新しい Product account session ID を canonical lifecycle が発行し、root legacy service に ID 生成責務を戻さない。
	sessionID, err := s.nextAccountSessionID()
	if err != nil {
		return AccountSessionResult{}, err
	}

	// Step 2: accessToken・refreshToken・metadata を同じ helper で組み立て、発行経路ごとの処理差分を作らない。
	issued, err := s.issueSession(ctx, account, sessionID, device)
	if err != nil {
		return AccountSessionResult{}, err
	}

	// Step 3: refresh session state を保存し、平文 refreshToken は Cookie command 以外へ流さない。
	if err := s.refreshSessions.Save(ctx, issued.refreshSession, s.refreshTTL.Duration()); err != nil {
		return AccountSessionResult{}, mapAccountAuthError(err)
	}

	// Step 4: bearer validation 用 metadata を保存し、protected request が accessToken だけで session 所属を検証できる状態にする。
	if err := s.sessions.Save(ctx, issued.metadata, s.refreshTTL.Duration()); err != nil {
		return AccountSessionResult{}, mapAccountAuthError(err)
	}

	// Step 5: response body 用 DTO と Cookie command を分離した canonical 結果を返す。
	return AccountSessionResult{Session: issued.session, RefreshCookie: issued.cookie}, nil
}

// RevokeAccountSession は Product account の対象 session と紐づく refresh state を失効する。
//
// 役割:
//   - AccountID と session selector を domain value として検証する。
//   - Product AccountAuth projection を取得して Account lifecycle 境界に存在する account だけを対象にする。
//   - session metadata と refresh session state の両方を失効し、以後の access/refresh を拒否できる状態にする。
//
// 引数:
//   - ctx: 呼び出し単位のキャンセル・期限情報。
//   - input: bearer validation 済み AccountID と失効対象 session ID。
//
// 戻り値:
//   - error: 入力不正、所有権不一致、保存先失敗などの application error。
func (s *AccountSessionService) RevokeAccountSession(ctx context.Context, input RevokeAccountSessionInput) error {
	// Step 1: session selector を Product AccountAuth 専用 value object に変換し、不正形式を拒否する。
	sessionID, err := domain.NewAccountAuthSessionID(input.SessionID)
	if err != nil {
		return ErrAccountAuthInvalidInput
	}

	// Step 2: AccountAuth projection を取得し、Product account として存在することを確認する。
	accountAuth, err := s.accounts.FindByID(ctx, input.AccountID)
	if err != nil {
		return mapAccountAuthError(err)
	}
	if accountAuth.AccountID() != input.AccountID {
		return ErrAccountAuthUnauthorized
	}

	// Step 3: metadata の所有権を確認し、別 account の session selector を拒否する。
	metadata, err := s.sessions.Get(ctx, sessionID)
	if err != nil {
		return mapAccountAuthError(err)
	}
	if metadata.AccountID != input.AccountID || metadata.SessionID != sessionID.String() {
		return ErrAccountAuthUnauthorized
	}

	// Step 4: accessToken 検証用 metadata を先に消し、bearer token の継続利用を拒否できる状態にする。
	if err := s.sessions.Revoke(ctx, input.AccountID, sessionID); err != nil {
		return mapAccountAuthError(err)
	}

	// Step 5: refresh session state を同じ selector で失効し、Cookie refresh の再利用を防ぐ。
	if err := s.refreshSessions.RevokeSession(ctx, input.AccountID, sessionID, s.clock().UTC()); err != nil {
		return mapAccountAuthError(err)
	}

	// Step 6: 両方の失効が完了したため成功として返す。
	return nil
}

// RevokeAllAccountSessions は Product account に紐づく全 session と refresh state を失効する。
//
// 役割:
//   - account suspension など、Account 単位で全 session を閉じる後続 adapter/use case から再利用できる境界を提供する。
//   - Product AccountAuth projection を確認し、Admin operator session へ影響しない Product-only revoke に閉じる。
//
// 引数:
//   - ctx: 呼び出し単位のキャンセル・期限情報。
//   - accountID: 失効対象 Product AccountID。
//
// 戻り値:
//   - error: Account 不在、保存先失敗などの application error。
func (s *AccountSessionService) RevokeAllAccountSessions(ctx context.Context, accountID domain.AccountID) error {
	// Step 1: Product AccountAuth projection が存在する account だけを対象にする。
	accountAuth, err := s.accounts.FindByID(ctx, accountID)
	if err != nil {
		return mapAccountAuthError(err)
	}
	if accountAuth.AccountID() != accountID {
		return ErrAccountAuthUnauthorized
	}

	// Step 2: bearer validation 用 metadata を全失効する。
	if err := s.sessions.RevokeAllForAccount(ctx, accountID); err != nil {
		return mapAccountAuthError(err)
	}

	// Step 3: refresh session state を全失効し、Cookie refresh をまとめて拒否する。
	if err := s.refreshSessions.RevokeAllForAccount(ctx, accountID, s.clock().UTC()); err != nil {
		return mapAccountAuthError(err)
	}

	// Step 4: Product account auth state の全失効が完了したため成功として返す。
	return nil
}

// ListAccountSessions は Product account に紐づく canonical session metadata を返す。
//
// 役割:
//   - session 管理画面が必要とする一覧取得を legacy root SessionService ではなく AccountSessionService に集約する。
//   - Product AccountAuth projection の存在確認を先に行い、他 surface や削除済み account の session metadata を返さない。
//   - 保存層の障害や壊れた session ID は Product auth error に畳み、HTTP adapter へ store 実装詳細を漏らさない。
//
// 引数:
//   - ctx: 呼び出し単位のキャンセル・期限情報。
//   - accountID: 一覧取得対象の Product AccountID。
//
// 戻り値:
//   - []SessionMetadata: Product session metadata の snapshot。
//   - error: Account 不在、保存先障害、または認可失敗を表す application error。
func (s *AccountSessionService) ListAccountSessions(ctx context.Context, accountID domain.AccountID) ([]SessionMetadata, error) {
	// Step 1: Product AccountAuth projection が存在する account だけを対象にし、孤立 metadata の列挙を拒否する。
	accountAuth, err := s.accounts.FindByID(ctx, accountID)
	if err != nil {
		return nil, mapAccountAuthError(err)
	}
	if accountAuth.AccountID() != accountID {
		return nil, ErrAccountAuthUnauthorized
	}

	// Step 2: canonical session metadata store から一覧を読み、legacy SessionStore へ戻る経路を作らない。
	sessions, err := s.sessions.List(ctx, accountID)
	if err != nil {
		return nil, mapAccountAuthError(err)
	}
	return sessions, nil
}

// RevokeOtherAccountSessions は現在 session を除く Product account session をすべて失効する。
//
// 役割:
//   - session metadata と refresh state の両方を canonical Product account auth lifecycle で失効する。
//   - 現在 session selector を domain value として検証し、壊れた入力で他 session 削除へ進まない。
//   - legacy root RefreshTokenStore / SessionStore の二重管理を排除し、Product Valkey account session store だけを使う。
//
// 引数:
//   - ctx: 呼び出し単位のキャンセル・期限情報。
//   - accountID: bearer validation 済み Product AccountID。
//   - currentSessionID: 保持する現在 session の selector。
//
// 戻り値:
//   - error: 入力不正、Account 不在、保存先障害などの application error。
func (s *AccountSessionService) RevokeOtherAccountSessions(ctx context.Context, accountID domain.AccountID, currentSessionID string) error {
	// Step 1: 現在 session selector を Product AccountAuth session ID として検証し、不正値では削除処理へ進まない。
	current, err := domain.NewAccountAuthSessionID(currentSessionID)
	if err != nil {
		return ErrAccountAuthInvalidInput
	}

	// Step 2: 一覧取得と Account 存在確認は ListAccountSessions に集約し、AccountAuth 検証を重複実装しない。
	sessions, err := s.ListAccountSessions(ctx, accountID)
	if err != nil {
		return err
	}

	// Step 3: 現在 session 以外を個別 revoke し、metadata と refresh state を同じ canonical helper で同期して削除する。
	for _, metadata := range sessions {
		if metadata.SessionID == current.String() {
			continue
		}
		if err := s.RevokeAccountSession(ctx, RevokeAccountSessionInput{AccountID: accountID, SessionID: metadata.SessionID}); err != nil {
			return err
		}
	}

	// Step 4: 対象 session の失効が完了したため成功として返す。
	return nil
}

func validateAccountSessionDependencies(deps AccountSessionDependencies) error {
	// Step 1: AccountAuth repository がない場合、Product account の lifecycle 判定ができないため拒否する。
	if deps.Accounts == nil {
		return ErrAccountAuthUnavailable
	}

	// Step 2: refresh state store がない場合、Cookie refresh を安全に rotation / revoke できないため拒否する。
	if deps.RefreshSessions == nil {
		return ErrAccountAuthUnavailable
	}

	// Step 3: session metadata store がない場合、accessToken sid の失効判定ができないため拒否する。
	if deps.Sessions == nil {
		return ErrAccountAuthUnavailable
	}

	// Step 4: signer がない場合、accessToken を発行・検証できないため拒否する。
	if deps.Signer == nil {
		return ErrAccountAuthUnavailable
	}

	// Step 5: ID generator がない場合、session ID / jti を安全に発行できないため拒否する。
	if deps.IDGenerator == nil {
		return ErrAccountAuthUnavailable
	}

	// Step 6: opaque token generator がない場合、refreshToken secret を発行できないため拒否する。
	if deps.TokenGenerator == nil {
		return ErrAccountAuthUnavailable
	}

	// Step 7: clock がない場合、token/session lifetime を deterministic に扱えないため拒否する。
	if deps.Clock == nil {
		return ErrAccountAuthUnavailable
	}

	// Step 8: すべての必須依存が揃っているため成功とする。
	return nil
}

type deviceInput struct {
	clientIP  string
	userAgent string
}

type issuedSession struct {
	session        AuthenticatedSession
	cookie         AccountRefreshCookieCommand
	refreshSession domain.AccountRefreshSession
	metadata       SessionMetadata
}

type accessTokenPayload struct {
	Subject         string `json:"sub"`
	SessionID       string `json:"sid"`
	TokenID         string `json:"jti"`
	Status          string `json:"status"`
	IssuedAt        int64  `json:"iat"`
	ExpiresAt       int64  `json:"exp"`
	IssuedAtMillis  int64  `json:"iat_ms,omitempty"`
	ExpiresAtMillis int64  `json:"exp_ms,omitempty"`
}

func (s *AccountSessionService) issueSession(ctx context.Context, account domain.Account, sessionID domain.AccountAuthSessionID, device deviceInput) (issuedSession, error) {
	// Step 1: 現在時刻を注入 clock から取得し、domain/application が time.Now を直接読まない規約を守る。
	issuedAt := s.clock().UTC()

	// Step 2: accessToken jti を生成し、中立 TokenJTI value object として検証する。
	jti, err := s.nextTokenJTI()
	if err != nil {
		return issuedSession{}, err
	}

	// Step 3: Account lifecycle と session ID を domain constructor へ渡し、Product accessToken claim を生成する。
	claims, err := domain.NewAccountAccessTokenClaims(account, sessionID, jti, issuedAt, s.accessTTLValue())
	if err != nil {
		return issuedSession{}, mapAccountAuthError(err)
	}
	// Step 3-a: canonical lifecycle へ渡す Product subject payload を明示的に作り、Account/Operator discriminator や互換 shim を使わない。
	subject, err := NewAccountSubjectPayload(claims.AccountID(), claims.SessionID().String())
	if err != nil {
		return issuedSession{}, mapAccountAuthError(err)
	}

	// Step 4: Product AccountAuth claim を JSON payload に変換し、中立 signer で署名する。
	accessToken, err := s.signAccessToken(claims)
	if err != nil {
		return issuedSession{}, mapAccountAuthError(err)
	}

	// Step 5: refreshToken 平文を生成し、保存用 hash だけを Product refresh session domain object に渡す。
	refreshToken, tokenHash, err := s.newRefreshTokenHash()
	if err != nil {
		return issuedSession{}, err
	}

	// Step 6: refresh session state を Product domain constructor で生成し、停止・revoke 境界を発行時に検証する。
	refreshSession, err := domain.NewAccountRefreshSession(account, sessionID, tokenHash, issuedAt, s.refreshTTL.ExpiresAt(issuedAt))
	if err != nil {
		return issuedSession{}, mapAccountAuthError(err)
	}

	// Step 7: clientIP / userAgent から保存用 device metadata を生成する。生 IP は HMAC 化し、metadata に直接残さない。
	metadata := s.sessionMetadata(subject.AccountID(), subject.SessionID(), device, issuedAt)

	// Step 8: ctx は store 実装の呼び出し側期限を保持するため、この helper では未使用でも明示的に保持する。
	_ = ctx

	// Step 9: response body 用 accessToken と Set-Cookie 用 refreshToken を別 DTO に分けて返す。
	return issuedSession{
		session: AuthenticatedSession{
			AccountID:   subject.AccountID(),
			SessionID:   subject.SessionID().String(),
			AccessToken: accessToken,
			ExpiresAt:   claims.ExpiresAt(),
			DeviceName:  metadata.DeviceName,
		},
		cookie: AccountRefreshCookieCommand{
			Value:     refreshToken,
			MaxAge:    s.refreshCookieLifetime,
			ExpiresAt: issuedAt.Add(s.refreshCookieLifetime).UTC(),
			Clear:     false,
		},
		refreshSession: refreshSession,
		metadata:       metadata,
	}, nil
}

func (s *AccountSessionService) accessTTLValue() domain.TokenTTL {
	// Step 1: Service 構築時に検証済みの TTL を domain constructor へ渡すため、同じ duration から再構成する。
	ttl, err := domain.ValidateTokenTTL(s.accessTTL.Duration())
	if err != nil {
		return zeroProductTokenTTL()
	}

	// Step 2: 検証済み TTL を返す。NewService で検証済みのため通常 error path には入らない。
	return ttl
}

func (s *AccountSessionService) nextAccountSessionID() (domain.AccountAuthSessionID, error) {
	// Step 1: 外側から注入された ID generator で ULID 文字列を生成する。
	value, err := s.idGenerator.Next()
	if err != nil {
		return "", ErrAccountAuthUnavailable
	}

	// Step 2: Product AccountAuth session ID として domain constructor で検証する。
	sessionID, err := domain.NewAccountAuthSessionID(value)
	if err != nil {
		return "", ErrAccountAuthUnavailable
	}

	// Step 3: 検証済み session ID を返す。
	return sessionID, nil
}

func (s *AccountSessionService) nextTokenJTI() (domain.TokenJTI, error) {
	// Step 1: accessToken jti 用の ULID 文字列を生成する。
	value, err := s.idGenerator.Next()
	if err != nil {
		return "", ErrAccountAuthUnavailable
	}

	// Step 2: 中立 TokenJTI として検証し、Product/Admin の意味をここでは足さない。
	jti, err := domain.NewTokenJTI(value)
	if err != nil {
		return "", ErrAccountAuthUnavailable
	}

	// Step 3: 検証済み jti を返す。
	return jti, nil
}

func (s *AccountSessionService) signAccessToken(claims domain.AccountAccessTokenClaims) (string, error) {
	// Step 1: Product claim object から JWT payload DTO へ写像する。
	payload := accessTokenPayload{
		Subject:         claims.AccountID().String(),
		SessionID:       claims.SessionID().String(),
		TokenID:         claims.JTI().String(),
		Status:          claims.Status().String(),
		IssuedAt:        claims.IssuedAt().Unix(),
		ExpiresAt:       claims.ExpiresAt().Unix(),
		IssuedAtMillis:  claims.IssuedAt().UnixMilli(),
		ExpiresAtMillis: claims.ExpiresAt().UnixMilli(),
	}

	// Step 2: JSON marshal により signer へ渡す payload を作る。
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", ErrAccountAuthUnavailable
	}

	// Step 3: 署名処理は application signer capability へ委譲し、Product claim の意味を signer に持たせない。
	return s.signer.SignJSON(payloadBytes)
}

func (s *AccountSessionService) newRefreshTokenHash() (string, domain.OpaqueTokenHash, error) {
	// Step 1: 平文 refreshToken secret を生成する。保存層にはこの値を渡さない。
	token, err := s.tokenGenerator.NewToken()
	if err != nil {
		return "", "", ErrAccountAuthUnavailable
	}

	// Step 2: canonical auth lifecycle helper で保存用 hash を生成し、Product 固有の判断を hash helper に持たせない。
	tokenHash, err := HashRefreshCredential(token)
	if err != nil {
		return "", "", ErrAccountAuthUnavailable
	}

	// Step 3: 平文は Cookie command 用、hash は refresh session state 用として分離して返す。
	return token, tokenHash, nil
}

func (s *AccountSessionService) sessionMetadata(accountID domain.AccountID, sessionID domain.AccountAuthSessionID, device deviceInput, issuedAt time.Time) SessionMetadata {
	// Step 1: User-Agent 由来の表示名は空でも保存可能な metadata とし、長すぎる場合は保存境界で切り詰める。
	deviceName := device.userAgent
	if len(deviceName) > 255 {
		deviceName = deviceName[:255]
	}

	// Step 2: IP は可逆な形で保存せず、HMAC fingerprint として metadata に残す。
	ipHash := keyedHash(device.clientIP, sessionID.String())

	// Step 3: session metadata DTO を組み立て、store が adapter 型なしで保存できるようにする。
	return SessionMetadata{
		AccountID:    accountID,
		SessionID:    sessionID.String(),
		DeviceName:   deviceName,
		LoginAt:      issuedAt.UTC(),
		LastActiveAt: issuedAt.UTC(),
		IPHash:       ipHash,
	}
}

func accountRootFromAuth(accountAuth domain.AccountAuth) (domain.Account, error) {
	// Step 1: AccountAuth projection の email を Product AccountEmail として再検証し、壊れた永続化値を拒否する。
	email, err := domain.NewAccountEmail(accountAuth.Email())
	if err != nil {
		return zeroProductAccount(), err
	}

	// Step 2: AccountAuth projection の status を Product lifecycle status として再検証する。
	status, err := domain.NewAccountStatus(accountAuth.Status())
	if err != nil {
		return zeroProductAccount(), err
	}

	// Step 3: Account root の必須 child である AccountSetting は Product 既定値で再構成し、認証可否判断には使わない。
	setting, err := domain.NewDefaultAccountSetting(accountAuth.AccountID())
	if err != nil {
		return zeroProductAccount(), err
	}

	// Step 4: status と sessionRevokedAfter を反映した Product Account root を domain constructor で生成する。
	return domain.NewAccount(accountAuth.AccountID(), email, status, setting, accountAuth.SessionRevokedAfter())
}

func zeroProductAccount() domain.Account {
	// Step 1: error return 専用の zero value を var 経由で作り、domain entity の composite literal を application 層に置かない。
	var account domain.Account
	return account
}

func zeroProductRefreshSession() domain.AccountRefreshSession {
	// Step 1: error return 専用の zero value を var 経由で作り、refresh session 生成は通常 path の constructor に限定する。
	var session domain.AccountRefreshSession
	return session
}

func zeroProductAccessTokenClaims() domain.AccountAccessTokenClaims {
	// Step 1: error return 専用の zero value を var 経由で作り、claim 生成は domain constructor/reconstitution helper に限定する。
	var claims domain.AccountAccessTokenClaims
	return claims
}

func zeroProductTokenTTL() domain.TokenTTL {
	// Step 1: error return 専用の zero value を var 経由で作り、成功 path では domain TTL constructor の結果だけを使う。
	var ttl domain.TokenTTL
	return ttl
}

var productAuthUnavailableErrors = []error{
	ErrAccountAuthUnavailable,
	domain.ErrAuthStoreUnavailable,
	domain.ErrInvalidSecret,
}

var productAuthUnauthorizedErrors = []error{
	ErrAccountAuthUnauthorized,
	domain.ErrAccountAuthNotFound,
	domain.ErrSessionNotFound,
	domain.ErrSessionExpired,
	domain.ErrSessionRevoked,
	domain.ErrTokenExpired,
	domain.ErrInvalidSignature,
	domain.ErrMalformedToken,
}

var productAuthInvalidInputErrors = []error{
	ErrAccountAuthInvalidInput,
	domain.ErrInvalidAccountID,
	domain.ErrInvalidToken,
	domain.ErrInvalidAuthID,
	domain.ErrInvalidSessionID,
	domain.ErrInvalidTokenTTL,
	domain.ErrInvalidSessionExpiry,
	domain.ErrInvalidAccountStatus,
	domain.ErrInvalidAccountEmail,
}

func mapAccountAuthError(err error) error {
	// Step 1: nil error はそのまま返し、呼び出し側の分岐を単純に保つ。
	if err == nil {
		return nil
	}

	// Step 2: 保存層障害や署名境界の不備は Product auth unavailable に畳む。
	if matchesProductAuthError(err, productAuthUnavailableErrors) {
		return ErrAccountAuthUnavailable
	}

	// Step 3: token reuse は session family 失効など別処理が必要になるため専用 error に畳む。
	if errors.Is(err, ErrAccountAuthTokenReuseDetected) {
		return ErrAccountAuthTokenReuseDetected
	}

	// Step 4: Account lifecycle による現在状態拒否は署名不正と分け、外側が stable な 403 分類へ畳めるようにする。
	if errors.Is(err, domain.ErrAccountAuthTokenIneligible) || errors.Is(err, ErrAccountAuthIneligible) {
		return ErrAccountAuthIneligible
	}

	// Step 5: 認証不可を表す domain error は詳細を漏らさない unauthorized に畳む。
	if matchesProductAuthError(err, productAuthUnauthorizedErrors) {
		return ErrAccountAuthUnauthorized
	}

	// Step 6: domain constructor の入力不備は invalid input として扱う。
	if matchesProductAuthError(err, productAuthInvalidInputErrors) {
		return ErrAccountAuthInvalidInput
	}

	// Step 7: 未分類 error は fail-closed に unavailable として扱う。
	return ErrAccountAuthUnavailable
}

func matchesProductAuthError(err error, candidates []error) bool {
	// Step 1: application error の分類表を順に確認し、errors.Is による wrapping 互換の照合だけを許可する。
	for _, candidate := range candidates {
		if errors.Is(err, candidate) {
			return true
		}
	}

	// Step 2: どの分類表にも一致しない error は呼び出し側の fail-closed default へ委ねる。
	return false
}

func keyedHash(value string, key string) string {
	// Step 1: session ID を key として HMAC を作り、生 IP を保存せず session-local fingerprint に変換する。
	mac := hmac.New(sha256.New, []byte(key))

	// Step 2: hash.Hash.Write は仕様上 nil error だけだが、errcheck のため戻り値を受ける。
	_, _ = mac.Write([]byte(value))

	// Step 3: 保存しやすい Base64URL 文字列として返す。
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
