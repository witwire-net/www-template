package auth

import (
	"context"
	"errors"
	"time"

	domain "www-template/packages/backend/internal/domain"
)

var (
	// ErrAccountAuthUnauthorized は AccountAuth の提示資格情報を受け入れられない場合に返す application error である。
	//
	// 役割:
	//   - accessToken、refreshToken、session selector、Account 状態のどこで失敗したかを transport 層へ漏らさない。
	//   - Admin operator auth の error と共有せず、Product account auth 境界だけで使う。
	//
	// 使用例:
	//
	//	if errors.Is(err, ErrAccountAuthUnauthorized) {
	//		return unauthorizedResponse()
	//	}
	ErrAccountAuthUnauthorized = errors.New("account auth unauthorized")

	// ErrAccountAuthUnavailable は AccountAuth の保存先や署名器など必須依存が利用できない場合に返す application error である。
	ErrAccountAuthUnavailable = errors.New("account auth unavailable")

	// ErrAccountAuthInvalidInput は AccountAuth use case の入力 DTO が domain constructor に渡せない場合に返す application error である。
	ErrAccountAuthInvalidInput = errors.New("account auth invalid input")

	// ErrAccountAuthTokenReuseDetected は消費済み refreshToken の再利用など、token family を拒否すべき場合に返す application error である。
	ErrAccountAuthTokenReuseDetected = errors.New("account auth token reuse detected")

	// ErrAccountAuthIneligible は Account lifecycle により既存 token/session が拒否された場合に返す application error である。
	//
	// 役割:
	//   - suspended、sessionRevokedAfter、発行時 status snapshot mismatch など、Account 側の現在状態で token/session が失効したケースを表す。
	//   - 署名不正や存在しない session とは分け、HTTP adapter が stable な account-suspended 形状へ畳めるようにする。
	//   - 平文 token や詳細な停止理由は含めず、外部 response で情報漏洩しない分類だけを提供する。
	ErrAccountAuthIneligible = errors.New("account auth account ineligible")
)

// AccountAuthRepository は Product AccountAuth use case が AccountAuth projection を取得するための port である。
//
// 役割:
//   - Product account login / refresh / session validation で必要な AccountAuth domain object だけを返す。
//   - Admin operator auth domain object や Admin application DTO を一切扱わない。
//
// 引数:
//   - context.Context: 呼び出し単位のキャンセル・期限情報。
//   - credentialHandle: WebAuthn 検証済み credential handle。login 完了時の Account 解決に使う。
//   - accountID: Product Account の canonical ULID。refresh / bearer validation の Account 解決に使う。
//
// 戻り値:
//   - domain.AccountAuth: Product AccountAuth projection。
//   - error: domain.ErrAccountAuthNotFound、domain.ErrAuthStoreUnavailable、または実装固有 error。
//
// 使用例:
//
//	accountAuth, err := repo.FindByCredential(ctx, credentialHandle)
//	if err != nil {
//		return err
//	}
type AccountAuthRepository interface {
	FindByCredential(ctx context.Context, credentialHandle string) (domain.AccountAuth, error)
	FindByID(ctx context.Context, accountID domain.AccountID) (domain.AccountAuth, error)
}

// RefreshRotationBuilder は保存層が旧 refresh session を消費した直後に、新 session を domain rule で組み立てる callback である。
//
// 役割:
//   - 旧 token の原子消費と新 token 保存の間に Product AccountAuth domain validation を挟む。
//   - adapter が Product 認証の可否判断を再実装せず、application が組み立てた next session だけを保存できるようにする。
//
// 引数:
//   - consumed: 保存層が old refreshToken hash から復元した Product AccountRefreshSession。
//
// 戻り値:
//   - domain.AccountRefreshSession: 保存すべき次の Product refresh session。
//   - error: Product AccountAuth domain validation に失敗した場合の application/domain error。
type RefreshRotationBuilder func(consumed domain.AccountRefreshSession) (domain.AccountRefreshSession, error)

// AccountRefreshSessionStore は Product Account refreshToken の server-side state を扱う port である。
//
// 役割:
//   - login では Product AccountRefreshSession を保存する。
//   - refresh では旧 token hash の消費、domain validation callback、新 token 保存を単一操作として実装できる境界を提供する。
//   - revoke では対象 Product account session の refresh state を明示失効する。
//
// 注意:
//   - 平文 refreshToken は受け取らず、domain.OpaqueTokenHash だけを保存・検索キーとして扱う。
//   - response body へ refreshToken を返す責務はこの port に存在しない。
type AccountRefreshSessionStore interface {
	Save(ctx context.Context, session domain.AccountRefreshSession, ttl time.Duration) error
	Rotate(ctx context.Context, tokenHash domain.OpaqueTokenHash, ttl time.Duration, build RefreshRotationBuilder) (domain.AccountRefreshSession, domain.AccountRefreshSession, error)
	RevokeSession(ctx context.Context, accountID domain.AccountID, sessionID domain.AccountAuthSessionID, revokedAt time.Time) error
	RevokeAllForAccount(ctx context.Context, accountID domain.AccountID, revokedAt time.Time) error
}

// AccountSessionMetadataStore は Product account session metadata を扱う port である。
//
// 役割:
//   - accessToken の sid が実在する Product session か検証する。
//   - login / refresh 時に最終利用時刻と device metadata を保存する。
//   - revoke 時に bearer validation で使う session selector を失効させる。
type AccountSessionMetadataStore interface {
	Save(ctx context.Context, metadata SessionMetadata, ttl time.Duration) error
	Get(ctx context.Context, sessionID domain.AccountAuthSessionID) (SessionMetadata, error)
	List(ctx context.Context, accountID domain.AccountID) ([]SessionMetadata, error)
	Revoke(ctx context.Context, accountID domain.AccountID, sessionID domain.AccountAuthSessionID) error
	RevokeAllForAccount(ctx context.Context, accountID domain.AccountID) error
}

// IDGenerator は auth application が ULID 系 ID を発行するための port である。
//
// 役割:
//   - application 層が platform/id の具象型へ固定されないよう、Next だけに依存する。
//   - Account/Operator session ID と accessToken jti の生成順を use case 内で明示できるようにする。
type IDGenerator interface {
	Next() (string, error)
}

// OpaqueTokenGenerator は refreshToken の平文 secret を生成する port である。
//
// 役割:
//   - production では crypto/rand による十分な entropy の token を生成する。
//   - Account と Operator の両方が同じ NewToken shape を使い、adapter 側で別名 method を再実装しない。
//   - test では deterministic な token を注入し、保存 hash と Cookie command を検証できるようにする。
type OpaqueTokenGenerator interface {
	NewToken() (string, error)
}

// Config は Product AccountAuth use case の token lifetime と Cookie lifetime を表す設定 DTO である。
//
// 役割:
//   - accessToken TTL、refreshToken server-side TTL、refresh Cookie lifetime を Product auth 境界へ注入する。
//   - domain token primitive により Cookie lifetime が refresh TTL を超えないことを NewService で検証する。
type AccountSessionConfig struct {
	AccessTokenTTL        time.Duration
	RefreshTokenTTL       time.Duration
	RefreshCookieLifetime time.Duration
}

// RefreshAccountSessionInput は Product refresh Cookie から account session を rotation する入力 DTO である。
//
// 役割:
//   - refreshToken は Cookie から取得した平文 secret として受け取り、response body DTO へは戻さない。
//   - SessionID は request が対象にする Product session selector として domain.CanRotate に渡す。
type RefreshAccountSessionInput struct {
	RefreshToken string
	SessionID    string
	ClientIP     string
	UserAgent    string
}

// IssueAccountSessionInput は Product account session を canonical lifecycle で発行する入力 DTO である。
//
// 役割:
//   - WebAuthn や recovery の外側フローで確定済みの AccountID を、旧 root token service を経由せず canonical account auth lifecycle へ渡す。
//   - clientIP / userAgent から device metadata を生成し、refresh session と bearer validation metadata を同時に保存する。
type IssueAccountSessionInput struct {
	AccountID domain.AccountID
	ClientIP  string
	UserAgent string
}

// RevokeAccountSessionInput は Product account session を明示失効する入力 DTO である。
//
// 役割:
//   - logout や session 管理画面から対象 Product session だけを失効する。
//   - AccountID は bearer validation 済みの caller account を想定し、SessionID の所有権検証に使う。
type RevokeAccountSessionInput struct {
	AccountID domain.AccountID
	SessionID string
}

// ValidateAccountBearerInput は Product bearer accessToken を検証する入力 DTO である。
//
// 役割:
//   - Authorization header から取り出した accessToken と、transport が選択した session selector を検証する。
//   - token payload と session metadata と現在 AccountAuth projection を照合し、停止や revoke 境界を反映する。
type ValidateAccountBearerInput struct {
	AccessToken string
	SessionID   string
}

// RefreshCookieCommand は Product refreshToken を HttpOnly Cookie として設定・削除するための application DTO である。
//
// 役割:
//   - response body ではなく Set-Cookie adapter へ渡す命令として refreshToken secret を閉じ込める。
//   - Cookie 属性そのものは HTTP adapter の責務に残し、この DTO は値・寿命・削除要否だけを保持する。
type AccountRefreshCookieCommand struct {
	Value     string
	MaxAge    time.Duration
	ExpiresAt time.Time
	Clear     bool
}

// AuthenticatedSession は Product account auth 成功時に transport body へ返せる session DTO である。
//
// 役割:
//   - accessToken と session metadata だけを公開し、refreshToken 平文を body へ含めない。
//   - Cookie command は別 field として adapter の Set-Cookie 処理へ渡す。
type AuthenticatedSession struct {
	AccountID   domain.AccountID
	SessionID   string
	AccessToken string
	ExpiresAt   time.Time
	DeviceName  string
}

// AccountSessionResult は Product account session 発行 use case の結果 DTO である。
//
// 役割:
//   - Body に載せる AuthenticatedSession と、Set-Cookie 用 RefreshCookieCommand を分離する。
//   - refreshToken を response body DTO に混入させない構造を固定する。
type AccountSessionResult struct {
	Session       AuthenticatedSession
	RefreshCookie AccountRefreshCookieCommand `json:"-"`
}

// AccountRefreshResult は Product account refresh use case の結果 DTO である。
//
// 役割:
//   - rotation 後の accessToken と refresh Cookie command だけを返す。
//   - 旧 refreshToken / 新 refreshToken を response body field として公開しない。
type AccountRefreshResult struct {
	RequestID     string
	Session       AuthenticatedSession
	RefreshCookie AccountRefreshCookieCommand `json:"-"`
}

// ValidatedSession は Product bearer validation 成功時の caller context DTO である。
//
// 役割:
//   - downstream use case が必要とする AccountID、SessionID、token jti、期限だけを application DTO として渡す。
//   - domain Account root や AccountAuth projection を adapter へ公開しない。
type ValidatedSession struct {
	AccountID domain.AccountID
	SessionID string
	TokenID   string
	ExpiresAt time.Time
}

// SessionMetadata は Product session list / bearer validation に使う application DTO である。
//
// 役割:
//   - session store が保持する metadata を adapter 型に依存せず表す。
//   - AccountID は ownership 検証、SessionID は accessToken sid / refresh session selector との照合に使う。
type SessionMetadata struct {
	AccountID        domain.AccountID
	SessionID        string
	DeviceName       string
	LoginAt          time.Time
	LastActiveAt     time.Time
	IPHash           string
	IsCurrentSession bool
}
