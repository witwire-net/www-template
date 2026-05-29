package application

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"time"

	tokenprimitive "www-template/packages/backend/internal/application/shared/tokenprimitive"
	domain "www-template/packages/backend/internal/domain"
)

// Service は Product account auth の login / refresh / revoke / bearer validation use case を提供する。
//
// 役割:
//   - Product AccountAuth domain object と中立 tokenprimitive だけを使って Product 認証を組み立てる。
//   - Admin operator auth application/domain を import せず、Product-only application boundary を維持する。
//   - refreshToken は Cookie command へだけ渡し、response body DTO へ含めない。
//
// 使用例:
//
//	service, err := NewService(deps, cfg)
//	if err != nil {
//		return err
//	}
//	result, err := service.LoginWithPasskey(ctx, input)
type Service struct {
	accounts              AccountAuthRepository
	refreshSessions       RefreshSessionStore
	sessions              SessionMetadataStore
	signer                tokenprimitive.JSONSignVerifier
	idGenerator           IDGenerator
	tokenGenerator        OpaqueTokenGenerator
	clock                 func() time.Time
	accessTTL             tokenprimitive.TTL
	refreshTTL            tokenprimitive.TTL
	refreshCookieLifetime time.Duration
}

// Dependencies は Product auth Service の必須 port をまとめた DTO である。
//
// 役割:
//   - NewService の引数増加を抑えつつ、nil 依存を一括検証する。
//   - platform / adapter の具象型を application public API に持ち込まない。
type Dependencies struct {
	Accounts        AccountAuthRepository
	RefreshSessions RefreshSessionStore
	Sessions        SessionMetadataStore
	Signer          tokenprimitive.JSONSignVerifier
	IDGenerator     IDGenerator
	TokenGenerator  OpaqueTokenGenerator
	Clock           func() time.Time
}

// NewService は Product account auth の Service を生成する。
//
// 役割:
//   - 必須依存、accessToken TTL、refreshToken TTL、refresh Cookie lifetime を fail-close に検証する。
//   - Cookie lifetime が refreshToken server-side TTL を超えないことを shared tokenprimitive で確認する。
//
// 引数:
//   - deps: repository、store、signer、ID/token generator、clock の必須依存。
//   - cfg: accessToken / refreshToken / Cookie lifetime の設定 DTO。
//
// 戻り値:
//   - *Service: 検証済み依存だけを持つ Product auth service。
//   - error: 必須依存または duration が不正な場合の application/domain error。
func NewService(deps Dependencies, cfg Config) (*Service, error) {
	// Step 1: nil 依存を最初に拒否し、認証境界が部分的に fail-open しないようにする。
	if err := validateDependencies(deps); err != nil {
		return nil, err
	}

	// Step 2: accessToken TTL は Product accessToken claim の exp 計算に使うため正の duration に限定する。
	accessTTL, err := tokenprimitive.ValidateTTL(cfg.AccessTokenTTL)
	if err != nil {
		return nil, err
	}

	// Step 3: refreshToken TTL と Cookie lifetime の大小関係を中立 helper で検証する。
	refreshTTL, err := tokenprimitive.ValidateDurations(cfg.RefreshTokenTTL, cfg.RefreshCookieLifetime)
	if err != nil {
		return nil, err
	}

	// Step 4: 検証済み依存だけを Service に保持し、以降の use case が同じ境界を使うようにする。
	return &Service{
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

// LoginWithPasskey は検証済み passkey credential から Product account session を新規発行する。
//
// 役割:
//   - credential handle から Product AccountAuth projection を取得する。
//   - Product AccountAuth domain object で accessToken claim と refresh session state を生成する。
//   - accessToken は body DTO、refreshToken は Cookie command に分離して返す。
//
// 引数:
//   - ctx: 呼び出し単位のキャンセル・期限情報。
//   - input: WebAuthn 検証済み credential handle と device metadata seed。
//
// 戻り値:
//   - LoginResult: accessToken body DTO と refresh Cookie command。
//   - error: 認証拒否、保存失敗、署名失敗などの application error。
func (s *Service) LoginWithPasskey(ctx context.Context, input LoginWithPasskeyInput) (LoginResult, error) {
	// Step 1: credential handle から Product AccountAuth projection を取得し、Admin auth と混在しない境界を保つ。
	accountAuth, err := s.accounts.FindByCredential(ctx, input.CredentialHandle)
	if err != nil {
		return LoginResult{}, mapProductAuthError(err)
	}

	// Step 2: AccountAuth projection を Product Account root へ写像し、lifecycle / revoke 境界を domain method で判定する。
	account, err := accountRootFromAuth(accountAuth)
	if err != nil {
		return LoginResult{}, mapProductAuthError(err)
	}

	// Step 3: Product account session ID を新規生成し、domain 専用 session ID として検証する。
	sessionID, err := s.nextAccountSessionID()
	if err != nil {
		return LoginResult{}, err
	}

	// Step 4: session ID と Account root から accessToken / refreshToken / metadata を一貫して発行する。
	issued, err := s.issueSession(ctx, account, sessionID, deviceInput{clientIP: input.ClientIP, userAgent: input.UserAgent})
	if err != nil {
		return LoginResult{}, err
	}

	// Step 5: refresh session state を保存し、平文 refreshToken は Cookie command 以外へ流さない。
	if err := s.refreshSessions.Save(ctx, issued.refreshSession, s.refreshTTL.Duration()); err != nil {
		return LoginResult{}, mapProductAuthError(err)
	}

	// Step 6: bearer validation で参照する session metadata を保存する。
	if err := s.sessions.Save(ctx, issued.metadata, s.refreshTTL.Duration()); err != nil {
		return LoginResult{}, mapProductAuthError(err)
	}

	// Step 7: body 用 session DTO と Set-Cookie 用 command を分離した結果を返す。
	return LoginResult{Session: issued.session, RefreshCookie: issued.cookie}, nil
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
func (s *Service) RevokeAccountSession(ctx context.Context, input RevokeAccountSessionInput) error {
	// Step 1: session selector を Product AccountAuth 専用 value object に変換し、不正形式を拒否する。
	sessionID, err := domain.NewAccountAuthSessionID(input.SessionID)
	if err != nil {
		return ErrProductAuthInvalidInput
	}

	// Step 2: AccountAuth projection を取得し、Product account として存在することを確認する。
	accountAuth, err := s.accounts.FindByID(ctx, input.AccountID)
	if err != nil {
		return mapProductAuthError(err)
	}
	if accountAuth.AccountID() != input.AccountID {
		return ErrProductAuthUnauthorized
	}

	// Step 3: metadata の所有権を確認し、別 account の session selector を拒否する。
	metadata, err := s.sessions.Get(ctx, sessionID)
	if err != nil {
		return mapProductAuthError(err)
	}
	if metadata.AccountID != input.AccountID || metadata.SessionID != sessionID.String() {
		return ErrProductAuthUnauthorized
	}

	// Step 4: accessToken 検証用 metadata を先に消し、bearer token の継続利用を拒否できる状態にする。
	if err := s.sessions.Revoke(ctx, input.AccountID, sessionID); err != nil {
		return mapProductAuthError(err)
	}

	// Step 5: refresh session state を同じ selector で失効し、Cookie refresh の再利用を防ぐ。
	if err := s.refreshSessions.RevokeSession(ctx, input.AccountID, sessionID, s.clock().UTC()); err != nil {
		return mapProductAuthError(err)
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
func (s *Service) RevokeAllAccountSessions(ctx context.Context, accountID domain.AccountID) error {
	// Step 1: Product AccountAuth projection が存在する account だけを対象にする。
	accountAuth, err := s.accounts.FindByID(ctx, accountID)
	if err != nil {
		return mapProductAuthError(err)
	}
	if accountAuth.AccountID() != accountID {
		return ErrProductAuthUnauthorized
	}

	// Step 2: bearer validation 用 metadata を全失効する。
	if err := s.sessions.RevokeAllForAccount(ctx, accountID); err != nil {
		return mapProductAuthError(err)
	}

	// Step 3: refresh session state を全失効し、Cookie refresh をまとめて拒否する。
	if err := s.refreshSessions.RevokeAllForAccount(ctx, accountID, s.clock().UTC()); err != nil {
		return mapProductAuthError(err)
	}

	// Step 4: Product account auth state の全失効が完了したため成功として返す。
	return nil
}

func validateDependencies(deps Dependencies) error {
	// Step 1: AccountAuth repository がない場合、Product account の lifecycle 判定ができないため拒否する。
	if deps.Accounts == nil {
		return ErrProductAuthUnavailable
	}

	// Step 2: refresh state store がない場合、Cookie refresh を安全に rotation / revoke できないため拒否する。
	if deps.RefreshSessions == nil {
		return ErrProductAuthUnavailable
	}

	// Step 3: session metadata store がない場合、accessToken sid の失効判定ができないため拒否する。
	if deps.Sessions == nil {
		return ErrProductAuthUnavailable
	}

	// Step 4: signer がない場合、accessToken を発行・検証できないため拒否する。
	if deps.Signer == nil {
		return ErrProductAuthUnavailable
	}

	// Step 5: ID generator がない場合、session ID / jti を安全に発行できないため拒否する。
	if deps.IDGenerator == nil {
		return ErrProductAuthUnavailable
	}

	// Step 6: opaque token generator がない場合、refreshToken secret を発行できないため拒否する。
	if deps.TokenGenerator == nil {
		return ErrProductAuthUnavailable
	}

	// Step 7: clock がない場合、token/session lifetime を deterministic に扱えないため拒否する。
	if deps.Clock == nil {
		return ErrProductAuthUnavailable
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
	cookie         RefreshCookieCommand
	refreshSession domain.AccountRefreshSession
	metadata       SessionMetadata
}

type accessTokenPayload struct {
	Subject   string `json:"sub"`
	SessionID string `json:"sid"`
	TokenID   string `json:"jti"`
	Status    string `json:"status"`
	IssuedAt  int64  `json:"iat"`
	ExpiresAt int64  `json:"exp"`
}

func (s *Service) issueSession(ctx context.Context, account domain.Account, sessionID domain.AccountAuthSessionID, device deviceInput) (issuedSession, error) {
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
		return issuedSession{}, mapProductAuthError(err)
	}

	// Step 4: Product AccountAuth claim を JSON payload に変換し、中立 signer で署名する。
	accessToken, err := s.signAccessToken(claims)
	if err != nil {
		return issuedSession{}, mapProductAuthError(err)
	}

	// Step 5: refreshToken 平文を生成し、保存用 hash だけを Product refresh session domain object に渡す。
	refreshToken, tokenHash, err := s.newRefreshTokenHash()
	if err != nil {
		return issuedSession{}, err
	}

	// Step 6: refresh session state を Product domain constructor で生成し、停止・revoke 境界を発行時に検証する。
	refreshSession, err := domain.NewAccountRefreshSession(account, sessionID, tokenHash, issuedAt, s.refreshTTL.ExpiresAt(issuedAt))
	if err != nil {
		return issuedSession{}, mapProductAuthError(err)
	}

	// Step 7: clientIP / userAgent から保存用 device metadata を生成する。生 IP は HMAC 化し、metadata に直接残さない。
	metadata := s.sessionMetadata(account.ID(), sessionID, device, issuedAt)

	// Step 8: ctx は store 実装の呼び出し側期限を保持するため、この helper では未使用でも明示的に保持する。
	_ = ctx

	// Step 9: response body 用 accessToken と Set-Cookie 用 refreshToken を別 DTO に分けて返す。
	return issuedSession{
		session: AuthenticatedSession{
			AccountID:   account.ID(),
			SessionID:   sessionID.String(),
			AccessToken: accessToken,
			ExpiresAt:   claims.ExpiresAt(),
			DeviceName:  metadata.DeviceName,
		},
		cookie: RefreshCookieCommand{
			Value:     refreshToken,
			MaxAge:    s.refreshCookieLifetime,
			ExpiresAt: issuedAt.Add(s.refreshCookieLifetime).UTC(),
			Clear:     false,
		},
		refreshSession: refreshSession,
		metadata:       metadata,
	}, nil
}

func (s *Service) accessTTLValue() domain.TokenTTL {
	// Step 1: Service 構築時に検証済みの TTL を domain constructor へ渡すため、同じ duration から再構成する。
	ttl, err := domain.ValidateTokenTTL(s.accessTTL.Duration())
	if err != nil {
		return zeroProductTokenTTL()
	}

	// Step 2: 検証済み TTL を返す。NewService で検証済みのため通常 error path には入らない。
	return ttl
}

func (s *Service) nextAccountSessionID() (domain.AccountAuthSessionID, error) {
	// Step 1: 外側から注入された ID generator で ULID 文字列を生成する。
	value, err := s.idGenerator.Next()
	if err != nil {
		return "", ErrProductAuthUnavailable
	}

	// Step 2: Product AccountAuth session ID として domain constructor で検証する。
	sessionID, err := domain.NewAccountAuthSessionID(value)
	if err != nil {
		return "", ErrProductAuthUnavailable
	}

	// Step 3: 検証済み session ID を返す。
	return sessionID, nil
}

func (s *Service) nextTokenJTI() (domain.TokenJTI, error) {
	// Step 1: accessToken jti 用の ULID 文字列を生成する。
	value, err := s.idGenerator.Next()
	if err != nil {
		return "", ErrProductAuthUnavailable
	}

	// Step 2: 中立 TokenJTI として検証し、Product/Admin の意味をここでは足さない。
	jti, err := domain.NewTokenJTI(value)
	if err != nil {
		return "", ErrProductAuthUnavailable
	}

	// Step 3: 検証済み jti を返す。
	return jti, nil
}

func (s *Service) signAccessToken(claims domain.AccountAccessTokenClaims) (string, error) {
	// Step 1: Product claim object から JWT payload DTO へ写像する。
	payload := accessTokenPayload{
		Subject:   claims.AccountID().String(),
		SessionID: claims.SessionID().String(),
		TokenID:   claims.JTI().String(),
		Status:    claims.Status().String(),
		IssuedAt:  claims.IssuedAt().Unix(),
		ExpiresAt: claims.ExpiresAt().Unix(),
	}

	// Step 2: JSON marshal により signer へ渡す payload を作る。
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", ErrProductAuthUnavailable
	}

	// Step 3: 署名処理は中立 tokenprimitive へ委譲し、Product claim の意味を signer に持たせない。
	return s.signer.SignJSON(payloadBytes)
}

func (s *Service) newRefreshTokenHash() (string, domain.OpaqueTokenHash, error) {
	// Step 1: 平文 refreshToken secret を生成する。保存層にはこの値を渡さない。
	token, err := s.tokenGenerator.NewToken()
	if err != nil {
		return "", "", ErrProductAuthUnavailable
	}

	// Step 2: domain primitive で保存用 hash を生成する。
	tokenHash, err := domain.HashOpaqueToken(token)
	if err != nil {
		return "", "", ErrProductAuthUnavailable
	}

	// Step 3: 平文は Cookie command 用、hash は refresh session state 用として分離して返す。
	return token, tokenHash, nil
}

func (s *Service) sessionMetadata(accountID domain.AccountID, sessionID domain.AccountAuthSessionID, device deviceInput, issuedAt time.Time) SessionMetadata {
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

func zeroProductTokenTTL() domain.TokenTTL {
	// Step 1: error return 専用の zero value を var 経由で作り、成功 path では domain TTL constructor の結果だけを使う。
	var ttl domain.TokenTTL
	return ttl
}

var productAuthUnavailableErrors = []error{
	ErrProductAuthUnavailable,
	domain.ErrAuthStoreUnavailable,
	domain.ErrInvalidSecret,
}

var productAuthUnauthorizedErrors = []error{
	ErrProductAuthUnauthorized,
	domain.ErrAccountAuthNotFound,
	domain.ErrSessionNotFound,
	domain.ErrSessionExpired,
	domain.ErrSessionRevoked,
	domain.ErrAccountAuthTokenIneligible,
	domain.ErrTokenExpired,
	domain.ErrInvalidSignature,
	domain.ErrMalformedToken,
}

var productAuthInvalidInputErrors = []error{
	ErrProductAuthInvalidInput,
	domain.ErrInvalidAccountID,
	domain.ErrInvalidToken,
	domain.ErrInvalidAuthID,
	domain.ErrInvalidSessionID,
	domain.ErrInvalidTokenTTL,
	domain.ErrInvalidSessionExpiry,
	domain.ErrInvalidAccountStatus,
	domain.ErrInvalidAccountEmail,
}

func mapProductAuthError(err error) error {
	// Step 1: nil error はそのまま返し、呼び出し側の分岐を単純に保つ。
	if err == nil {
		return nil
	}

	// Step 2: 保存層障害や署名境界の不備は Product auth unavailable に畳む。
	if matchesProductAuthError(err, productAuthUnavailableErrors) {
		return ErrProductAuthUnavailable
	}

	// Step 3: token reuse は session family 失効など別処理が必要になるため専用 error に畳む。
	if errors.Is(err, ErrProductAuthTokenReuseDetected) {
		return ErrProductAuthTokenReuseDetected
	}

	// Step 4: 認証不可を表す domain error は詳細を漏らさない unauthorized に畳む。
	if matchesProductAuthError(err, productAuthUnauthorizedErrors) {
		return ErrProductAuthUnauthorized
	}

	// Step 5: domain constructor の入力不備は invalid input として扱う。
	if matchesProductAuthError(err, productAuthInvalidInputErrors) {
		return ErrProductAuthInvalidInput
	}

	// Step 6: 未分類 error は fail-closed に unavailable として扱う。
	return ErrProductAuthUnavailable
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
