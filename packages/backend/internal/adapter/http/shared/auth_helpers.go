package shared

import (
	"context"
	"errors"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	refreshRoutePrefix = "/api/v1/auth/contexts/"
	refreshRouteSuffix = "/refresh"
)

var (
	errInvalidRefreshCookieName = errors.New("refresh cookie name must not be empty")
	errInvalidRefreshContextID  = errors.New("refresh context id must be a valid ULID")
	refreshContextIDPattern     = regexp.MustCompile(`^[0-9A-HJKMNP-TV-Z]{26}$`)
)

// RefreshCookieClearCommand は HTTP adapter が refresh Cookie 削除 header を組み立てるための shared DTO である。
//
// 役割:
//   - Product/Admin の transport adapter が同じ context-scoped refresh path と Max-Age=0 規則を使えるようにする。
//   - application 層へ Cookie 名、Path、Expires などの transport 属性を持ち込まないため、HTTP shared helper が所有する。
//
// 使用例:
//
//	command, err := shared.NewRefreshCookieClearCommand("refresh_token", contextID)
//	if err != nil { return err }
type RefreshCookieClearCommand struct {
	Name      string
	Path      string
	MaxAge    time.Duration
	ExpiresAt time.Time
	Clear     bool
}

// BuildRefreshPath は contextID に対応する refresh endpoint path を構築する。
//
// 役割:
//   - contextID を ULID として検証したうえで `/api/v1/auth/contexts/{contextID}/refresh` を返す。
//   - Cookie Path と generated refresh route path の対応関係を HTTP adapter 境界で一元管理する。
//
// 引数:
//   - contextID: path segment として使う ULID 文字列。前後空白は除去される。
//
// 戻り値:
//   - string: refresh endpoint の絶対 path。
//   - error: contextID が ULID 形式でない場合の transport validation error。
func BuildRefreshPath(contextID string) (string, error) {
	// Step 1: path segment にする値を正規化し、余分な前後空白だけを取り除く。
	normalizedID := strings.TrimSpace(contextID)

	// Step 2: slash や任意文字列を受け入れないよう、既存の ULID 検証を使って path segment を固定する。
	if !refreshContextIDPattern.MatchString(normalizedID) {
		return "", errInvalidRefreshContextID
	}

	// Step 3: 固定 prefix/suffix と検証済み ID だけで refresh endpoint path を構築する。
	return refreshRoutePrefix + normalizedID + refreshRouteSuffix, nil
}

// NewRefreshCookieClearCommand は contextID に対応する refresh Cookie 削除命令を生成する。
//
// 役割:
//   - 削除対象 Cookie 名と検証済み refresh path を組み合わせ、Max-Age=0 の削除命令へ変換する。
//   - HTTP response への書き込みは行わず、副作用のない DTO だけを返す。
//
// 引数:
//   - cookieName: 削除対象 Cookie 名。前後空白は除去され、空は拒否される。
//   - contextID: refresh endpoint path に埋め込む ULID 文字列。
//
// 戻り値:
//   - RefreshCookieClearCommand: 削除対象 Cookie の名前、Path、期限、削除 flag を持つ DTO。
//   - error: Cookie 名または contextID が不正な場合の transport validation error。
func NewRefreshCookieClearCommand(cookieName string, contextID string) (RefreshCookieClearCommand, error) {
	// Step 1: Cookie 名の前後空白だけを取り除き、空名による曖昧な削除命令を拒否する。
	normalizedName := strings.TrimSpace(cookieName)
	if normalizedName == "" {
		return RefreshCookieClearCommand{}, errInvalidRefreshCookieName
	}

	// Step 2: contextID から refresh path を構築し、Cookie Path と endpoint path の規則を揃える。
	path, err := BuildRefreshPath(contextID)
	if err != nil {
		return RefreshCookieClearCommand{}, err
	}

	// Step 3: 削除用 Expires は固定の過去時刻にし、clock 依存のない deterministic な命令にする。
	expiresAt := time.Unix(0, 0).UTC()

	// Step 4: adapter が Set-Cookie 削除処理へ渡せる shared DTO として返す。
	return RefreshCookieClearCommand{Name: normalizedName, Path: path, MaxAge: 0, ExpiresAt: expiresAt, Clear: true}, nil
}

// SecurityCSP は Product/Admin の API response に共通適用する Content-Security-Policy の baseline 値である。
//
// 役割:
//   - API response が HTML や外部 resource を実行しない前提を browser に伝え、frame 埋め込みと base URI 変更を拒否する。
//   - Product/Admin surface 間で CSP 文字列が分岐しないよう、shared HTTP helper の単一値として所有する。
//
// 使用例:
//
//	c.Header("Content-Security-Policy", shared.SecurityCSP)
const SecurityCSP = "default-src 'none'; frame-ancestors 'none'; base-uri 'none'"

// SecurityHSTS は Product/Admin の API response に共通適用する Strict-Transport-Security の baseline 値である。
//
// 役割:
//   - browser に HTTPS の継続利用を要求し、subdomain を含む downgrade exposure を抑止する。
//   - HSTS max-age と includeSubDomains の値を shared HTTP helper に集約し、surface ごとの drift を防ぐ。
//
// 使用例:
//
//	c.Header("Strict-Transport-Security", shared.SecurityHSTS)
const SecurityHSTS = "max-age=63072000; includeSubDomains"

// SecurityReferrerPolicy は Product/Admin の API response に共通適用する Referrer-Policy の baseline 値である。
//
// 役割:
//   - browser が遷移先へ URL や path 由来の情報を referrer として送らないようにする。
//   - no-referrer policy を shared HTTP helper で一元管理し、認証 surface の情報漏えい対策をそろえる。
//
// 使用例:
//
//	c.Header("Referrer-Policy", shared.SecurityReferrerPolicy)
const SecurityReferrerPolicy = "no-referrer"

// BearerToken は Authorization header から RFC 6750 形式の bearer token 本体だけを抽出する。
//
// 役割:
//   - Product/Admin HTTP adapter で重複していた Bearer prefix 判定を shared helper concept に集約する。
//   - `Bearer ` 以外の認証方式、空白だけの値、空 token を認証材料として扱わない。
//
// 引数:
//   - header: HTTP Authorization header の生値。
//
// 戻り値:
//   - string: `Bearer ` prefix の後ろにある token 本体。形式不一致または空の場合は空文字。
//
// エラーケース:
//   - この関数は error を返さない。不正形式は fail-close できるよう空文字で表す。
//
// 使用例:
//
//	token := shared.BearerToken(c.GetHeader("Authorization"))
//	if token == "" { c.AbortWithStatus(http.StatusUnauthorized) }
func BearerToken(header string) string {
	// Step 1: 前後空白を取り除き、proxy や test fixture が付けた余分な空白で prefix 判定が揺れないようにする。
	trimmed := strings.TrimSpace(header)
	if !strings.HasPrefix(trimmed, "Bearer ") {
		return ""
	}

	// Step 2: prefix 後の token 本体も空白除去し、空 token は呼び出し側が unauthenticated として拒否できるようにする。
	return strings.TrimSpace(strings.TrimPrefix(trimmed, "Bearer "))
}

// AuthorizationHeaderPresent は context に束縛された Gin request から Authorization header の存在を判定する。
//
// 役割:
//   - refresh endpoint で Authorization header を refresh credential として扱わない invariant を Product/Admin で共有する。
//   - Gin context 以外から呼ばれた場合は header を観測できないため「存在しない」として返す。
//
// 引数:
//   - ctx: generated strict handler に渡される request context。通常は *gin.Context。
//   - headerName: 検査対象 header 名。通常は `Authorization`。
//
// 戻り値:
//   - bool: 空白以外の header 値が存在する場合 true。
func AuthorizationHeaderPresent(ctx context.Context, headerName string) bool {
	// Step 1: Gin context だけを HTTP header source として扱い、任意 context value に置かれた token を信頼しない。
	ginContext, ok := ctx.(*gin.Context)
	if !ok {
		return false
	}

	// Step 2: 空白だけの header は credential ではないため、trim 後に値がある場合だけ存在とみなす。
	return strings.TrimSpace(ginContext.GetHeader(headerName)) != ""
}

// BearerTokenFromContext は context に束縛された Gin request から bearer token を抽出する。
//
// 役割:
//   - generated strict handler 内で HTTP header を読む処理を Product/Admin で共有する。
//   - context value に token を保存する設計を避け、transport header からだけ access token を得る。
//
// 引数:
//   - ctx: generated strict handler に渡される request context。通常は *gin.Context。
//   - headerName: 検査対象 header 名。通常は `Authorization`。
//
// 戻り値:
//   - string: bearer token 本体。
//   - bool: token が空でない場合 true。
func BearerTokenFromContext(ctx context.Context, headerName string) (string, bool) {
	// Step 1: Gin context 以外では HTTP header を安全に読めないため、未認証として扱える値を返す。
	ginContext, ok := ctx.(*gin.Context)
	if !ok {
		return "", false
	}

	// Step 2: 共通 Bearer parser を使い、Product/Admin 間で Basic や空 token の扱いが分岐しないようにする。
	token := BearerToken(ginContext.GetHeader(headerName))
	return token, token != ""
}

// CookieValueFromContext は context に束縛された Gin request から指定 Cookie の非空値を取り出す。
//
// 役割:
//   - Product/Admin の path-scoped refresh Cookie 抽出を共有し、Cookie 欠落と空値を同じ fail-close 条件にする。
//   - Cookie 平文値は application service に渡す直前だけ返し、response body 用 DTO には関与しない。
//
// 引数:
//   - ctx: generated strict handler に渡される request context。通常は *gin.Context。
//   - cookieName: 読み取る Cookie 名。
//
// 戻り値:
//   - string: Cookie の非空値。
//   - bool: Cookie が存在し、値が空白だけではない場合 true。
func CookieValueFromContext(ctx context.Context, cookieName string) (string, bool) {
	// Step 1: Gin context 以外からの呼び出しは Cookie header を検証できないため、credential 不在として扱う。
	ginContext, ok := ctx.(*gin.Context)
	if !ok {
		return "", false
	}

	// Step 2: 指定 Cookie だけを読み、欠落・空白値・parser error はすべて false に正規化する。
	value, err := ginContext.Cookie(cookieName)
	if err != nil || strings.TrimSpace(value) == "" {
		return "", false
	}

	// Step 3: Cookie 平文は最小限の範囲で返し、log や error message には含めない。
	return value, true
}

// ClientIPFromContext は context に束縛された Gin request から client IP を返す。
//
// 役割:
//   - Product/Admin route adapter が request metadata を application DTO に詰め替えるときの fallback を共有する。
//   - Gin が解決できない場合は secret ではない `unknown` に正規化し、空文字で downstream validation を揺らさない。
//
// 引数:
//   - ctx: generated strict handler に渡される request context。通常は *gin.Context。
//
// 戻り値:
//   - string: Gin が解決した client IP、または `unknown`。
func ClientIPFromContext(ctx context.Context) string {
	// Step 1: Gin context 以外では proxy/trusted header 設定を読めないため、非 secret fallback を返す。
	ginContext, ok := ctx.(*gin.Context)
	if !ok {
		return "unknown"
	}

	// Step 2: Gin の trusted proxy 設定に従った ClientIP を採用し、空なら `unknown` に正規化する。
	if ip := strings.TrimSpace(ginContext.ClientIP()); ip != "" {
		return ip
	}
	return "unknown"
}

// UserAgentFromContext は context に束縛された Gin request から User-Agent を返す。
//
// 役割:
//   - refresh/session store に保存する device hint を route adapter 境界で抽出する。
//   - Gin context 以外の場合は空文字を返し、HTTP transport 以外の context value を信頼しない。
//
// 引数:
//   - ctx: generated strict handler に渡される request context。通常は *gin.Context。
//
// 戻り値:
//   - string: request の User-Agent。取得不能なら空文字。
func UserAgentFromContext(ctx context.Context) string {
	// Step 1: Gin context 以外では request header を読まず、transport 外の値を device hint として採用しない。
	ginContext, ok := ctx.(*gin.Context)
	if !ok {
		return ""
	}

	// Step 2: net/http request の UserAgent helper から値を返す。
	return ginContext.Request.UserAgent()
}

// ApplyBrowserSecurityHeaders は API response に共通 browser hardening header を設定する。
//
// 役割:
//   - Product/Admin HTTP adapter に重複していた CSP/HSTS/referrer/MIME/frame header を shared helper concept に集約する。
//   - cache policy は surface ごとに違うため、必要な場合だけ呼び出し側から `cacheControl` を渡す。
//
// 引数:
//   - c: header を設定する Gin context。
//   - cacheControl: `Cache-Control` に設定する値。空文字の場合は cache header を変更しない。
//
// 戻り値:
//   - なし。副作用として response header を設定する。
func ApplyBrowserSecurityHeaders(c *gin.Context, cacheControl string) {
	// Step 1: no-store が必要な surface では cache policy も同じ helper 呼び出しで固定し、header 設定漏れを防ぐ。
	if strings.TrimSpace(cacheControl) != "" {
		c.Header("Cache-Control", cacheControl)
	}

	// Step 2: browser hardening header は Product/Admin で同じ baseline とし、surface ごとの drift を防ぐ。
	c.Header("Content-Security-Policy", SecurityCSP)
	c.Header("Strict-Transport-Security", SecurityHSTS)
	c.Header("Referrer-Policy", SecurityReferrerPolicy)
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("X-Frame-Options", "DENY")
}

// OriginAllowed は Origin header が許可 origin 一覧のいずれかと完全一致するかを判定する。
//
// 役割:
//   - Cookie-setting / rotation flow の allowlist 比較を Product/Admin で共有し、path/query/fragment や wildcard 的な曖昧一致を拒否する。
//   - 設定値と request 値をどちらも canonical origin に正規化してから比較する。
//
// 引数:
//   - allowedOrigins: config 由来の許可 origin 一覧。
//   - originHeader: request の Origin header 生値。
//
// 戻り値:
//   - bool: request origin が許可一覧に完全一致する場合 true。
func OriginAllowed(allowedOrigins []string, originHeader string) bool {
	// Step 1: request Origin が欠落または不正な場合は Cookie 発行・rotation を fail-close する。
	requestOrigin, ok := NormalizeOrigin(originHeader)
	if !ok {
		return false
	}

	// Step 2: 許可 origin も同じ正規化を通し、不正な設定値は無視して完全一致だけを認める。
	for _, allowed := range allowedOrigins {
		configuredOrigin, configuredOK := NormalizeOrigin(allowed)
		if configuredOK && requestOrigin == configuredOrigin {
			return true
		}
	}
	return false
}

// OriginMatches は request Origin と単一 configured origin が完全一致するかを判定する。
//
// 役割:
//   - Admin runtime domain のように許可 origin が 1 つだけの surface でも、Product allowlist と同じ正規化規則を使う。
//   - 不正な Origin や不正な設定値は fail-close として false を返す。
//
// 引数:
//   - originHeader: request の Origin header 生値。
//   - configuredOrigin: runtime config 由来の許可 origin。
//
// 戻り値:
//   - bool: 両者が canonical origin として完全一致する場合 true。
func OriginMatches(originHeader string, configuredOrigin string) bool {
	// Step 1: request Origin を canonical origin へ正規化し、path/query/fragment 付き値を拒否する。
	requestOrigin, ok := NormalizeOrigin(originHeader)
	if !ok {
		return false
	}

	// Step 2: 設定値も同じ規則で検証し、誤設定時は安全側の false を返す。
	allowedOrigin, ok := NormalizeOrigin(configuredOrigin)
	if !ok {
		return false
	}
	return requestOrigin == allowedOrigin
}

// NormalizeOrigin は raw origin を `scheme://host` の canonical 形式へ正規化する。
//
// 役割:
//   - Origin allowlist 判定で文字列 contains や prefix 判定を使わないよう、URL parser による構文検証を一箇所に集約する。
//   - scheme は http/https だけを認め、path/query/fragment を持つ値を拒否する。
//
// 引数:
//   - rawOrigin: request header または config 由来の origin 値。
//
// 戻り値:
//   - string: 小文字化した `scheme://host`。
//   - bool: origin として有効な場合 true。
func NormalizeOrigin(rawOrigin string) (string, bool) {
	// Step 1: URL parser で scheme / host / path を分解し、手書き文字列処理による origin 誤認を避ける。
	parsed, err := url.Parse(strings.TrimSpace(rawOrigin))
	if err != nil {
		return "", false
	}

	// Step 2: Origin は scheme と host だけを持つ値に限定し、path/query/fragment 付き値を拒否する。
	if parsed.Scheme == "" || parsed.Host == "" || (parsed.Path != "" && parsed.Path != "/") || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", false
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", false
	}

	// Step 3: scheme と host を小文字化した canonical origin を返し、比較を完全一致に閉じる。
	return strings.ToLower(parsed.Scheme + "://" + parsed.Host), true
}

// FetchMetadataAccepted は Sec-Fetch-Site header が same-origin/same-site/none/empty のいずれかかを判定する。
//
// 役割:
//   - Product/Admin の Cookie-setting、rotation、mutation 入口で cross-site request を handler 到達前に拒否する。
//   - Fetch Metadata が欠落する legacy client は Origin allowlist を主境界として扱うため、空値は許可する。
//
// 引数:
//   - fetchSiteHeader: request の Sec-Fetch-Site header 生値。
//
// 戻り値:
//   - bool: 安全側として許可できる値の場合 true。
func FetchMetadataAccepted(fetchSiteHeader string) bool {
	// Step 1: 欠落値は Origin allowlist による検証へ委ねるため true とし、古い browser/test harness を誤拒否しない。
	fetchSite := strings.ToLower(strings.TrimSpace(fetchSiteHeader))
	if fetchSite == "" {
		return true
	}

	// Step 2: cross-site は SameSite や Bearer の有無に依存せず fail-close する。
	if fetchSite == "cross-site" {
		return false
	}

	// Step 3: browser が定義する安全側の値だけを許可し、未知値は intermediary 誤設定に備えて拒否する。
	return fetchSite == "same-origin" || fetchSite == "same-site" || fetchSite == "none"
}
