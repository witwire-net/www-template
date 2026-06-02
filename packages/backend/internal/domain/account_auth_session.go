package domain

import (
	"errors"
	"time"
)

var (
	// ErrAccountAuthTokenIneligible は Product AccountAuth token/session が現在の Account 状態で利用できない場合に返すエラーである。
	// suspended、sessionRevokedAfter 境界、session ID mismatch は外側で詳細を漏らさず同じ拒否系 error として扱える。
	ErrAccountAuthTokenIneligible = errors.New("account auth token is ineligible")
)

// AccountAuthSessionID は Product AccountAuth の session を表す canonical ULID 値オブジェクトである。
//
// 役割:
//   - Product account の accessToken `sid` claim と refresh session state を同じ識別子で結びつける。
//   - Admin operator session の識別子とは別型にし、Product AccountAuth domain 内の語彙として保持する。
//
// 引数:
//   - value: NewAccountAuthSessionID に渡す ULID 文字列。前後空白は許容せず、既存 auth ID 規則で検証される。
//
// 戻り値:
//   - AccountAuthSessionID: 検証済み Product account session ID。
//   - error: value が ULID 形式ではない場合に ErrInvalidSessionID。
//
// 使用例:
//
//	sessionID, err := NewAccountAuthSessionID("01ARZ3NDEKTSV4RRFFQ69G5FAW")
//	if err != nil {
//		return err
//	}
type AccountAuthSessionID string

// AccountAccessTokenClaims は Product AccountAuth accessToken の domain claim snapshot である。
//
// 役割:
//   - AccountID、Product session ID、jti、発行時刻、失効時刻、発行時点の AccountStatus を保持する。
//   - token payload の意味づけを Product AccountAuth domain に閉じ、Admin operator claim と混在させない。
//
// 引数:
//   - NewAccountAccessTokenClaims の account: 発行対象 Account root。suspended または revoke 境界により発行時刻が拒否される場合は失敗する。
//   - NewAccountAccessTokenClaims の sessionID: Product account refresh session と一致する session ID。
//   - NewAccountAccessTokenClaims の jti: accessToken 自体の ULID claim。
//   - NewAccountAccessTokenClaims の issuedAt: 外側の clock から渡された発行時刻。zero time は拒否する。
//   - NewAccountAccessTokenClaims の ttl: 正の token TTL。未初期化 TTL は拒否する。
//
// 戻り値:
//   - AccountAccessTokenClaims: Product AccountAuth 専用の検証済み claim snapshot。
//   - error: Account/session/token/時刻/eligibility が不正な場合の domain error。
//
// 使用例:
//
//	claims, err := NewAccountAccessTokenClaims(account, sessionID, jti, issuedAt, ttl)
//	if err != nil {
//		return err
//	}
type AccountAccessTokenClaims struct {
	accountID AccountID
	sessionID AccountAuthSessionID
	jti       TokenJTI
	status    AccountStatus
	issuedAt  time.Time
	expiresAt time.Time
}

// AccountRefreshSession は Product AccountAuth refreshToken の server-side session state である。
//
// 役割:
//   - Product account refreshToken の hash、session ID、AccountID、有効期限、revoke 状態を保持する。
//   - refresh rotation 前に Account lifecycle と session selector を検証し、停止・revoke・mismatch を拒否する。
//
// 引数:
//   - NewAccountRefreshSession の account: refresh session を所有する Product Account root。
//   - NewAccountRefreshSession の sessionID: Product account session を表す ULID。
//   - NewAccountRefreshSession の tokenHash: refreshToken 平文ではなく保存用 opaque hash。
//   - NewAccountRefreshSession の issuedAt: session 発行時刻。zero time は拒否する。
//   - NewAccountRefreshSession の expiresAt: session 失効時刻。issuedAt より後でなければならない。
//
// 戻り値:
//   - AccountRefreshSession: 検証済み Product refresh session state。
//   - error: Account/session/hash/時刻/eligibility が不正な場合の domain error。
//
// 使用例:
//
//	session, err := NewAccountRefreshSession(account, sessionID, tokenHash, issuedAt, expiresAt)
//	if err != nil {
//		return err
//	}
type AccountRefreshSession struct {
	accountID AccountID
	sessionID AccountAuthSessionID
	tokenHash OpaqueTokenHash
	issuedAt  time.Time
	expiresAt time.Time
	revokedAt *time.Time
}

// NewAccountAuthSessionID は Product AccountAuth session ID を検証して返す。
//
// value は canonical ULID でなければならない。
// 不正な値は ErrInvalidSessionID に畳み込み、application/adapter が session 境界の入力不備として扱えるようにする。
func NewAccountAuthSessionID(value string) (AccountAuthSessionID, error) {
	// Step 1: Product session ID は token primitive の ULID 検証を再利用し、識別子形式だけを確認する。
	id, err := NewTokenULID(value)
	if err != nil {
		return "", ErrInvalidSessionID
	}

	// Step 2: 検証済み ULID を Product AccountAuth 専用型へ写像する。
	return AccountAuthSessionID(id.String()), nil
}

// String は AccountAuthSessionID の canonical 文字列表現を返す。
//
// 戻り値は NewAccountAuthSessionID によって検証済みの ULID 文字列であり、JWT claim、DB key、監査参照に使える。
func (id AccountAuthSessionID) String() string {
	// Step 1: value object の内部表現をそのまま文字列として返す。
	return string(id)
}

// NewAccountAccessTokenClaims は Product AccountAuth accessToken claim snapshot を生成する。
//
// account は active かつ issuedAt が sessionRevokedAfter 境界より後でなければならない。
// sessionID、jti、issuedAt、ttl を検証し、成功時は expiresAt を ttl から deterministic に計算する。
func NewAccountAccessTokenClaims(account Account, sessionID AccountAuthSessionID, jti TokenJTI, issuedAt time.Time, ttl TokenTTL) (AccountAccessTokenClaims, error) {
	// Step 1: Account root と token 発行時刻の組み合わせが Product 認証に使えるか検証する。
	if err := ensureAccountAuthEligibleAt(account, issuedAt); err != nil {
		return AccountAccessTokenClaims{}, err
	}

	// Step 2: session ID と jti の値オブジェクトが未初期化ではないことを確認する。
	if err := validateAccountAuthSessionAndJTI(sessionID, jti); err != nil {
		return AccountAccessTokenClaims{}, err
	}

	// Step 3: TTL が正しく設定済みであることを確認し、未初期化 TTL による即時失効を拒否する。
	if ttl.Duration() <= 0 {
		return AccountAccessTokenClaims{}, ErrInvalidTokenTTL
	}

	// Step 4: 発行時刻と TTL から失効時刻を UTC で計算し、snapshot を構築する。
	return AccountAccessTokenClaims{
		accountID: account.ID(),
		sessionID: sessionID,
		jti:       jti,
		status:    account.Status(),
		issuedAt:  issuedAt.UTC(),
		expiresAt: ttl.ExpiresAt(issuedAt),
	}, nil
}

// ReconstituteAccountAccessTokenClaims は署名済み JSON payload から Product AccountAuth claims を復元する。
//
// 役割:
//   - application 層で decode した primitive 値を domain claim snapshot に戻し、status/current account/expiry の判定を EnsureEligible へ集約する。
//   - JWT 署名や JSON field の必須性は application が担当し、この helper は Product AccountAuth として意味を持つ値だけを検証する。
//   - Admin Operator claims と混在しないよう、AccountID と AccountAuthSessionID の Product 専用 constructor を必ず通す。
//
// 引数:
//   - accountID: accessToken `sub` から復元した Product AccountID。
//   - sessionID: accessToken `sid` から復元した Product AccountAuth session ID。
//   - jti: accessToken `jti` から復元した token ID。
//   - status: 発行時点の Product AccountStatus snapshot。
//   - issuedAt: accessToken `iat` の UTC 時刻。zero time は拒否される。
//   - expiresAt: accessToken `exp` の UTC 時刻。issuedAt より後でなければならない。
//
// 戻り値:
//   - AccountAccessTokenClaims: 復元済み Product AccountAuth claims。
//   - error: ID、status、時刻、TTL が不正な場合の domain error。
//
// 使用例:
//
//	claims, err := ReconstituteAccountAccessTokenClaims(accountID, sessionID, jti, status, issuedAt, expiresAt)
//	if err != nil {
//		return err
//	}
func ReconstituteAccountAccessTokenClaims(
	accountID AccountID,
	sessionID AccountAuthSessionID,
	jti TokenJTI,
	status AccountStatus,
	issuedAt time.Time,
	expiresAt time.Time,
) (AccountAccessTokenClaims, error) {
	// Step 1: Product AccountID は canonical constructor に通し、別 domain の ID や空値を拒否する。
	validatedAccountID, err := NewAccountID(accountID.String())
	if err != nil {
		return AccountAccessTokenClaims{}, err
	}

	// Step 2: Product AccountAuth session ID と JTI の形式を検証し、手組み zero value を拒否する。
	if err := validateAccountAuthSessionAndJTI(sessionID, jti); err != nil {
		return AccountAccessTokenClaims{}, err
	}

	// Step 3: status snapshot は Product lifecycle status として検証し、未知 status を fail-closed にする。
	validatedStatus, err := NewAccountStatus(status.String())
	if err != nil {
		return AccountAccessTokenClaims{}, err
	}

	// Step 4: iat/exp から TTL を検証し、期限が逆転した署名済み payload を拒否する。
	if issuedAt.IsZero() || !expiresAt.UTC().After(issuedAt.UTC()) {
		return AccountAccessTokenClaims{}, ErrInvalidTokenTTL
	}
	if _, err := ValidateTokenTTL(expiresAt.UTC().Sub(issuedAt.UTC())); err != nil {
		return AccountAccessTokenClaims{}, err
	}

	// Step 5: 復元済み snapshot を返し、現在 Account との eligibility は EnsureEligible に委譲する。
	return AccountAccessTokenClaims{
		accountID: validatedAccountID,
		sessionID: sessionID,
		jti:       jti,
		status:    validatedStatus,
		issuedAt:  issuedAt.UTC(),
		expiresAt: expiresAt.UTC(),
	}, nil
}

// EnsureEligible は accessToken claim が現在の Product Account 状態と session selector に対して有効か検証する。
//
// account は現在の Account root、expectedSessionID は request/session store から選択された Product session ID を表す。
// now は外側の clock から渡される現在時刻であり、期限切れ、suspended、sessionRevokedAfter、session mismatch を拒否する。
func (c AccountAccessTokenClaims) EnsureEligible(account Account, expectedSessionID AccountAuthSessionID, now time.Time) error {
	// Step 1: claim 自体の AccountID、session ID、jti、時刻が検証済み状態を保っているか確認する。
	if err := c.validate(); err != nil {
		return err
	}

	// Step 2: 提示された Account root と claim の subject が一致しない場合は Product token として拒否する。
	if account.ID() != c.accountID {
		return ErrAccountAuthTokenIneligible
	}

	// Step 3: request が選択した session と claim の sid が違う場合は対象 session 外の token として拒否する。
	if expectedSessionID != c.sessionID {
		return ErrAccountAuthTokenIneligible
	}

	// Step 4: token 失効時刻に到達している場合は期限切れとして拒否する。
	if !now.UTC().Before(c.expiresAt.UTC()) {
		return ErrTokenExpired
	}

	// Step 5: 発行時点の status snapshot が suspended の token は発行経路外の値として拒否する。
	if c.status.IsSuspended() {
		return ErrAccountAuthTokenIneligible
	}

	// Step 6: 発行時点 status と現在 status がずれた token は lifecycle 変更後の古い snapshot として拒否する。
	if c.status != account.Status() {
		return ErrAccountAuthTokenIneligible
	}

	// Step 7: 現在の Account lifecycle により token 発行時刻が失効済みかを最終判定する。
	return ensureAccountAuthEligibleAt(account, c.issuedAt)
}

// AccountID は accessToken claim の subject である Product AccountID を返す。
//
// 戻り値は constructor で検証済みの AccountID であり、Admin operator ID など他 domain の識別子を含まない。
func (c AccountAccessTokenClaims) AccountID() AccountID {
	// Step 1: claim snapshot が保持する AccountID を返す。
	return c.accountID
}

// SessionID は accessToken claim の Product AccountAuth session ID を返す。
//
// 戻り値は refresh session selector と照合するための Product 専用 session ID である。
func (c AccountAccessTokenClaims) SessionID() AccountAuthSessionID {
	// Step 1: claim snapshot が保持する session ID を返す。
	return c.sessionID
}

// JTI は accessToken claim の JWT ID を返す。
//
// 戻り値は TokenJTI として ULID 形式だけを検証済みであり、権限や利用者種別の意味は持たない。
func (c AccountAccessTokenClaims) JTI() TokenJTI {
	// Step 1: claim snapshot が保持する jti を返す。
	return c.jti
}

// Status は accessToken 発行時点の Product Account status snapshot を返す。
//
// 現在状態の判定は EnsureEligible の account 引数で行うため、この値だけで認証可否を最終判断しない。
func (c AccountAccessTokenClaims) Status() AccountStatus {
	// Step 1: claim snapshot が保持する status を返す。
	return c.status
}

// IssuedAt は accessToken claim の発行時刻を UTC で返す。
//
// 戻り値は sessionRevokedAfter 境界と比較するために使われる。
func (c AccountAccessTokenClaims) IssuedAt() time.Time {
	// Step 1: 発行時刻を UTC に正規化して返す。
	return c.issuedAt.UTC()
}

// ExpiresAt は accessToken claim の失効時刻を UTC で返す。
//
// 戻り値は EnsureEligible が now と比較する期限境界である。
func (c AccountAccessTokenClaims) ExpiresAt() time.Time {
	// Step 1: 失効時刻を UTC に正規化して返す。
	return c.expiresAt.UTC()
}

// NewAccountRefreshSession は Product AccountAuth refresh session state を新規発行用に生成する。
//
// account は active かつ issuedAt が sessionRevokedAfter より後でなければならない。
// refreshToken 平文は受け取らず、HashOpaqueToken 済みの OpaqueTokenHash だけを保持する。
func NewAccountRefreshSession(account Account, sessionID AccountAuthSessionID, tokenHash OpaqueTokenHash, issuedAt time.Time, expiresAt time.Time) (AccountRefreshSession, error) {
	// Step 1: Account と発行時刻が Product refresh session を作れる状態か確認する。
	if err := ensureAccountAuthEligibleAt(account, issuedAt); err != nil {
		return AccountRefreshSession{}, err
	}

	// Step 2: 共通の復元 constructor へ委譲し、発行時と永続化復元で検証を揃える。
	return ReconstituteAccountRefreshSession(account.ID(), sessionID, tokenHash, issuedAt, expiresAt, nil)
}

// ReconstituteAccountRefreshSession は永続化済み Product AccountAuth refresh session state を復元する。
//
// accountID、sessionID、tokenHash、issuedAt、expiresAt、revokedAt を検証し、壊れた永続化値が domain 外へ出ないようにする。
// revokedAt は nil または non-zero time のみを許可し、返却値では defensive copy として保持する。
func ReconstituteAccountRefreshSession(accountID AccountID, sessionID AccountAuthSessionID, tokenHash OpaqueTokenHash, issuedAt time.Time, expiresAt time.Time, revokedAt *time.Time) (AccountRefreshSession, error) {
	// Step 1: AccountID、session ID、hash、時刻範囲をまとめて検証する。
	if err := validateAccountRefreshSessionInputs(accountID, sessionID, tokenHash, issuedAt, expiresAt, revokedAt); err != nil {
		return AccountRefreshSession{}, err
	}

	// Step 2: revokedAt は外部 pointer を共有しないよう UTC defensive copy として保持する。
	return AccountRefreshSession{
		accountID: accountID,
		sessionID: sessionID,
		tokenHash: tokenHash,
		issuedAt:  issuedAt.UTC(),
		expiresAt: expiresAt.UTC(),
		revokedAt: optionalTimePointer(revokedAt),
	}, nil
}

// CanRotate は Product refresh session が現在の Account 状態と selector で rotation 可能か検証する。
//
// account は現在の Product Account root、selector は request が対象にした session ID、now は外側の clock である。
// suspended、sessionRevokedAfter、revokedAt、expiresAt、AccountID mismatch、session ID mismatch はすべて拒否される。
func (s AccountRefreshSession) CanRotate(account Account, selector AccountAuthSessionID, now time.Time) error {
	// Step 1: session state 自体が永続化値として妥当か再検証する。
	if err := s.validate(); err != nil {
		return err
	}

	// Step 2: refresh session の所有 Account と現在 Account が違う場合は Product session として拒否する。
	if account.ID() != s.accountID {
		return ErrAccountAuthTokenIneligible
	}

	// Step 3: request の selector と保存済み session ID が違う場合は対象 session 外の rotation として拒否する。
	if selector != s.sessionID {
		return ErrAccountAuthTokenIneligible
	}

	// Step 4: 明示 revoke 済み session は rotation できない。
	if s.revokedAt != nil {
		return ErrSessionRevoked
	}

	// Step 5: refresh session の失効時刻に到達している場合は期限切れとして拒否する。
	if !now.UTC().Before(s.expiresAt.UTC()) {
		return ErrSessionExpired
	}

	// Step 6: 現在の Account lifecycle と発行時刻を照合し、停止・境界失効を拒否する。
	return ensureAccountAuthEligibleAt(account, s.issuedAt)
}

// Revoke は Product refresh session を指定時刻で失効済みにした新しい値を返す。
//
// at は外側の clock から渡す revoke 時刻であり、zero time は ErrInvalidSessionRevocationBoundary として拒否される。
// 元の値は変更せず、値オブジェクトとして revoke 済みの copy を返す。
func (s AccountRefreshSession) Revoke(at time.Time) (AccountRefreshSession, error) {
	// Step 1: revoke 時刻が zero の場合は有効な境界にならないため拒否する。
	if at.IsZero() {
		return AccountRefreshSession{}, ErrInvalidSessionRevocationBoundary
	}

	// Step 2: 既存 session を複製し、revokedAt を UTC defensive copy として設定する。
	s.revokedAt = cloneTimePointer(at.UTC())

	// Step 3: revoke 後の session state が引き続き妥当であることを検証する。
	if err := s.validate(); err != nil {
		return AccountRefreshSession{}, err
	}

	// Step 4: 検証済みの revoke 済み session state を返す。
	return s, nil
}

// AccountID は refresh session を所有する Product AccountID を返す。
//
// 戻り値は constructor で検証済みの AccountID である。
func (s AccountRefreshSession) AccountID() AccountID {
	// Step 1: session state が保持する AccountID を返す。
	return s.accountID
}

// SessionID は Product AccountAuth refresh session ID を返す。
//
// 戻り値は accessToken claim の sid や rotation selector と照合するために使う。
func (s AccountRefreshSession) SessionID() AccountAuthSessionID {
	// Step 1: session state が保持する session ID を返す。
	return s.sessionID
}

// TokenHash は保存済み refreshToken hash を返す。
//
// 戻り値は平文 token ではなく OpaqueTokenHash であり、Matches で提示 token と照合できる。
func (s AccountRefreshSession) TokenHash() OpaqueTokenHash {
	// Step 1: session state が保持する opaque token hash を返す。
	return s.tokenHash
}

// IssuedAt は refresh session の発行時刻を UTC で返す。
//
// 戻り値は Account の sessionRevokedAfter 境界と比較するために使われる。
func (s AccountRefreshSession) IssuedAt() time.Time {
	// Step 1: 発行時刻を UTC に正規化して返す。
	return s.issuedAt.UTC()
}

// ExpiresAt は refresh session の失効時刻を UTC で返す。
//
// 戻り値は rotation 可否の期限境界である。
func (s AccountRefreshSession) ExpiresAt() time.Time {
	// Step 1: 失効時刻を UTC に正規化して返す。
	return s.expiresAt.UTC()
}

// RevokedAt は refresh session の明示 revoke 時刻を返す。
//
// nil は未 revoke を表し、非 nil の場合は defensive copy を返して内部状態の外部変更を防ぐ。
func (s AccountRefreshSession) RevokedAt() *time.Time {
	// Step 1: revoke されていない場合は nil を返す。
	if s.revokedAt == nil {
		return nil
	}

	// Step 2: revoke 時刻は defensive copy として返す。
	return cloneTimePointer(*s.revokedAt)
}

func (c AccountAccessTokenClaims) validate() error {
	// Step 1: AccountID は Product Account の canonical ULID として再検証する。
	if err := validateAccountID(c.accountID.String()); err != nil {
		return ErrInvalidAccountID
	}

	// Step 2: session ID と jti は Product token/session を結ぶ必須識別子として検証する。
	if err := validateAccountAuthSessionAndJTI(c.sessionID, c.jti); err != nil {
		return err
	}

	// Step 3: status snapshot は対応済み AccountStatus のみを許可する。
	if _, err := NewAccountStatus(c.status.String()); err != nil {
		return err
	}

	// Step 4: issuedAt/expiresAt の zero と逆転を拒否する。
	return validateAccountAuthTimeRange(c.issuedAt, c.expiresAt)
}

func (s AccountRefreshSession) validate() error {
	// Step 1: 復元 constructor と同じ入力検証へ戻し、値の一貫性を確認する。
	return validateAccountRefreshSessionInputs(s.accountID, s.sessionID, s.tokenHash, s.issuedAt, s.expiresAt, s.revokedAt)
}

func validateAccountRefreshSessionInputs(accountID AccountID, sessionID AccountAuthSessionID, tokenHash OpaqueTokenHash, issuedAt time.Time, expiresAt time.Time, revokedAt *time.Time) error {
	// Step 1: AccountID は Product Account root と接続するため canonical ULID として検証する。
	if err := validateAccountID(accountID.String()); err != nil {
		return ErrInvalidAccountID
	}

	// Step 2: session ID は Product AccountAuth session として検証する。
	if err := validateAccountAuthSessionID(sessionID); err != nil {
		return err
	}

	// Step 3: refreshToken hash は空だと平文照合境界を構成できないため拒否する。
	if tokenHash.String() == "" {
		return ErrInvalidToken
	}

	// Step 4: issuedAt/expiresAt の zero と逆転を拒否する。
	if err := validateAccountAuthTimeRange(issuedAt, expiresAt); err != nil {
		return err
	}

	// Step 5: revokedAt が存在する場合は zero time を拒否し、revoke 境界を明確にする。
	if revokedAt != nil && revokedAt.IsZero() {
		return ErrInvalidSessionRevocationBoundary
	}

	// Step 6: すべての refresh session state 入力が妥当であるため成功とする。
	return nil
}

func validateAccountAuthSessionAndJTI(sessionID AccountAuthSessionID, jti TokenJTI) error {
	// Step 1: session ID を Product AccountAuth session として検証する。
	if err := validateAccountAuthSessionID(sessionID); err != nil {
		return err
	}

	// Step 2: jti は token primitive の ULID 検証へ戻し、未初期化値を拒否する。
	if _, err := NewTokenJTI(jti.String()); err != nil {
		return err
	}

	// Step 3: session ID と jti の両方が有効なため成功とする。
	return nil
}

func validateAccountAuthSessionID(sessionID AccountAuthSessionID) error {
	// Step 1: 専用型の文字列表現を constructor に戻し、zero value や不正形式を拒否する。
	if _, err := NewAccountAuthSessionID(sessionID.String()); err != nil {
		return err
	}

	// Step 2: Product AccountAuth session ID として妥当なため成功とする。
	return nil
}

func validateAccountAuthTimeRange(issuedAt time.Time, expiresAt time.Time) error {
	// Step 1: 発行時刻または失効時刻の zero は token/session lifetime を構成できないため拒否する。
	if issuedAt.IsZero() || expiresAt.IsZero() {
		return ErrInvalidSessionExpiry
	}

	// Step 2: 失効時刻は発行時刻より後でなければならない。
	if !expiresAt.UTC().After(issuedAt.UTC()) {
		return ErrInvalidSessionExpiry
	}

	// Step 3: 時刻範囲が妥当なため成功とする。
	return nil
}

func ensureAccountAuthEligibleAt(account Account, issuedAt time.Time) error {
	// Step 1: 発行時刻が zero の場合、revoke 境界との比較が意味を持たないため拒否する。
	if issuedAt.IsZero() {
		return ErrInvalidSessionExpiry
	}

	// Step 2: Account root 自体を再構成し、手組み値や壊れた lifecycle を拒否する。
	if _, err := NewAccount(account.ID(), account.Email(), account.Status(), account.Setting(), account.SessionRevokedAfter()); err != nil {
		return err
	}

	// Step 3: suspended または sessionRevokedAfter 境界以前の token/session は Product AccountAuth として拒否する。
	if account.RejectsTokenIssuedAt(issuedAt) {
		return ErrAccountAuthTokenIneligible
	}

	// Step 4: Account lifecycle が token/session を許可しているため成功とする。
	return nil
}
