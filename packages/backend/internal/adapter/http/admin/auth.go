package admin

import (
	"context"
	"errors"
	stdhttp "net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"

	"www-template/packages/backend/internal/generated/adminopenapi"
	"www-template/packages/backend/internal/platform/config"
)

const adminAuthHeader = "Authorization"
const adminCSRFHeader = "X-CSRF-Token"
const adminOriginHeader = "Origin"
const adminContextKeyOperatorID = "admin.operator.id"
const adminContextKeyOperatorEmail = "admin.operator.email"
const adminContextKeyOperatorRole = "admin.operator.role"
const adminContextKeyOperatorActive = "admin.operator.active"
const adminContextKeyOperatorPasskeyRegistrationState = "admin.operator.passkey_registration_state"
const adminContextKeySessionID = "admin.operator.session_id"
const adminContextKeyCSRFToken = "admin.operator.csrf_token"
const adminSecurityCSP = "default-src 'none'; frame-ancestors 'none'; base-uri 'none'"
const adminSecurityHSTS = "max-age=63072000; includeSubDomains"
const adminSecurityReferrerPolicy = "no-referrer"

var errAdminOperatorForbidden = errors.New("admin operator forbidden")
var errAdminOperatorInternal = errors.New("admin operator internal")

var adminExactRouteKeys = map[string]struct{}{
	"GET /api/v1/accounts":                    {},
	"POST /api/v1/accounts":                   {},
	"POST /api/v1/auth/operator-setup/finish": {},
	"POST /api/v1/auth/operator-setup/start":  {},
	"POST /api/v1/auth/operators":             {},
	"POST /api/v1/auth/setup/finish":          {},
	"POST /api/v1/auth/setup/start":           {},
	"GET /api/v1/auth/operator/current":       {},
	"GET /api/v1/auth/passkeys":               {},
	"POST /api/v1/auth/operator/logout":       {},
	"POST /api/v1/auth/operator/refresh":      {},
	"POST /api/v1/auth/passkey/finish":        {},
	"POST /api/v1/auth/passkey/start":         {},
}

type adminContextValueKey string

type adminRouterDependencies struct {
	operatorSessions          operatorSessionValidator
	operatorAuth              adminOperatorAuthenticator
	operatorSetup             adminOperatorSetupper
	operatorPasskeys          adminOperatorPasskeyVerifier
	operatorPasskeyManagement adminOperatorPasskeyManager
	accountCreation           adminAccountCreator
	accountSearch             adminAccountSearcher
}

// OperatorSessionValidationInput は Admin protected route middleware が session validator へ渡す検証入力である。
//
// 役割:
//   - AccessToken は Authorization header から取り出した Admin operator bearer token だけを保持する。
//   - CSRFToken は mutation route の X-CSRF-Token header であり、RequireCSRF が true の場合だけ必須になる。
//   - Permission は accounts:create など、application auth service が domain RBAC と CSRF binding を同時検証するための permission 名である。
//   - RequireCSRF は read route と mutation route を区別し、read route では CurrentOperator 検証だけへ委譲できるようにする。
type OperatorSessionValidationInput struct {
	AccessToken string
	CSRFToken   string
	Permission  string
	RequireCSRF bool
}

type operatorSessionValidationInput = OperatorSessionValidationInput

// OperatorSessionContext は Admin protected route middleware が handler へ束縛する検証済み Operator context である。
//
// 役割:
//   - Product account の認証状態を含めず、Admin OperatorAuth domain 由来の値だけを保持する。
//   - SessionID は accessToken payload で検証された Admin operator session selector であり、handler で token を再解析しないために使う。
//   - CSRFToken は mutation request で検証された header 値だけを保持し、未検証値を application use case へ渡さない。
type OperatorSessionContext struct {
	OperatorID                       string
	OperatorEmail                    string
	OperatorRole                     string
	OperatorActive                   bool
	OperatorPasskeyRegistrationState string
	SessionID                        string
	CSRFToken                        string
}

type operatorSessionContext = OperatorSessionContext

// OperatorSessionValidator は Admin HTTP middleware が operator session 検証へ使う公開 dependency 境界である。
//
// 役割:
//   - runtime composition から検証済み implementation を注入できるよう、unexported 型に依存しない method signature を提供する。
//   - adapter/http/admin package 内では同じ interface を package-local alias として使い、既存テストの stub も維持する。
//   - nil の場合は protected route を 503 で fail-close し、認証なし Admin mutation を防ぐ。
type OperatorSessionValidator interface {
	ValidateOperatorSession(ctx context.Context, input OperatorSessionValidationInput) (OperatorSessionContext, error)
}

type operatorSessionValidator = OperatorSessionValidator

func adminSecurityHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Step 1: Admin API 以外の `/health` などへ副作用を出さないため、`/api/v1/*` だけに browser hardening header を設定する。
		if isAdminAPIPath(c.Request.URL.Path) {
			applyAdminSecurityHeaders(c)
		}

		// Step 2: 以後の generated handler / auth middleware が status と body を決められるよう、header だけを先に固定して処理を進める。
		c.Next()
	}
}

func adminAuthMiddleware(cfg config.Config, validator operatorSessionValidator) gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path

		// Step 1: Admin API route 以外は認証境界の対象外にし、health check や Gin の NoRoute 処理を壊さない。
		if !isAdminAPIPath(path) {
			c.Next()
			return
		}
		if !isAdminRegisteredRoute(c.Request.Method, path) {
			c.Next()
			return
		}

		// Step 2: credentialed request の入口で Admin domain と Origin を比較し、session や handler へ進む前に cross-site request を拒否する。
		if !adminOriginAccepted(cfg, c.GetHeader(adminOriginHeader), c.Request.Method) {
			writeAdminAuthFailure(c, stdhttp.StatusForbidden, adminopenapi.Unauthenticated)
			return
		}

		// Step 3: passkey / setup / refresh の pre-auth flow は session-bound CSRF を持たないため、Origin 検証だけを通過条件にして handler へ委譲する。
		if isAdminPreAuthPath(c.Request.Method, path) {
			c.Next()
			return
		}

		// Step 4: protected route は bearer accessToken を必須にし、Product bearer や空 header を operator session として扱わない。
		accessToken := adminBearerToken(c.GetHeader(adminAuthHeader))
		if accessToken == "" {
			writeAdminAuthFailure(c, stdhttp.StatusUnauthorized, adminopenapi.Unauthenticated)
			return
		}

		// Step 5: session validator が未接続なら Admin API を fail-close し、暫定 handler が認証なしに公開されることを防ぐ。
		if validator == nil {
			writeAdminAuthFailure(c, stdhttp.StatusServiceUnavailable, adminopenapi.InternalError)
			return
		}

		// Step 6: mutation route だけ CSRF binding を必須化し、read route は operator session validation のみを要求する。
		input := operatorSessionValidationInput{AccessToken: accessToken, RequireCSRF: adminRouteRequiresCSRF(c.Request.Method, path), Permission: adminRoutePermission(c.Request.Method, path)}
		if input.RequireCSRF {
			input.CSRFToken = strings.TrimSpace(c.GetHeader(adminCSRFHeader))
			if input.CSRFToken == "" {
				writeAdminAuthFailure(c, stdhttp.StatusForbidden, adminopenapi.Unauthenticated)
				return
			}
		}

		// Step 7: adapter では token / CSRF の中身を判定せず、Admin operator session validator に検証を集約する。
		operatorContext, err := validator.ValidateOperatorSession(c.Request.Context(), input)
		if err != nil {
			writeAdminAuthValidationError(c, err)
			return
		}
		if input.RequireCSRF {
			operatorContext.CSRFToken = input.CSRFToken
		}

		// Step 8: 検証済み operator/session/CSRF 情報を Gin context と request context の両方へ設定し、handler が Product auth state を参照せずに済む境界を作る。
		bindAdminOperatorContext(c, operatorContext)
		c.Next()
	}
}

func applyAdminSecurityHeaders(c *gin.Context) {
	// Step 1: Admin API response は operator session や顧客 PII を含み得るため、全 API route を no-store に固定する。
	c.Header("Cache-Control", noStoreValue)

	// Step 2: Admin API JSON response に対する XSS / clickjacking / MIME sniffing / referer leakage の browser hardening header を設定する。
	c.Header("Content-Security-Policy", adminSecurityCSP)
	c.Header("Strict-Transport-Security", adminSecurityHSTS)
	c.Header("Referrer-Policy", adminSecurityReferrerPolicy)
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("X-Frame-Options", "DENY")
}

func adminOriginAccepted(cfg config.Config, originHeader string, method string) bool {
	// Step 1: unsafe method は browser が付与する Origin を必須にし、CSRF 防御を SameSite Cookie だけに依存しない。
	trimmedOrigin := strings.TrimSpace(originHeader)
	if trimmedOrigin == "" {
		return !adminMethodRequiresOrigin(method)
	}

	// Step 2: Origin header と Admin runtime domain を origin 形式へ正規化してから比較し、大文字小文字や末尾 slash の揺れを吸収する。
	requestOrigin, ok := normalizeAdminOrigin(trimmedOrigin)
	if !ok {
		return false
	}
	configuredOrigin, ok := normalizeAdminOrigin(cfg.Admin.Domain)
	if !ok {
		return false
	}
	return requestOrigin == configuredOrigin
}

func normalizeAdminOrigin(rawOrigin string) (string, bool) {
	// Step 1: URL parser で scheme / host / path を分解し、文字列 contains 判定による origin 誤認を避ける。
	parsed, err := url.Parse(strings.TrimSpace(rawOrigin))
	if err != nil {
		return "", false
	}

	// Step 2: origin は scheme と host だけを持つ値に限定し、path/query/fragment 付きの値を拒否する。
	if parsed.Scheme == "" || parsed.Host == "" || (parsed.Path != "" && parsed.Path != "/") || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", false
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", false
	}

	// Step 3: scheme と host を小文字化した canonical origin を返し、比較処理を単純な完全一致へ閉じる。
	return strings.ToLower(parsed.Scheme + "://" + parsed.Host), true
}

func adminMethodRequiresOrigin(method string) bool {
	// Step 1: state-changing method と credentialed refresh/login POST を cross-site から守るため、safe method 以外は Origin 必須にする。
	switch method {
	case stdhttp.MethodGet, stdhttp.MethodHead, stdhttp.MethodOptions:
		return false
	default:
		return true
	}
}

func isAdminAPIPath(path string) bool {
	// Step 1: Product/Admin 共通の path policy と合わせ、Admin API 境界を `/api/v1/*` に限定する。
	return strings.HasPrefix(path, "/api/v1/")
}

func isAdminRegisteredRoute(method string, path string) bool {
	// Step 1: 静的 Admin route は method/path key で確認し、Product-only path を認証 middleware で 401 に変えないようにする。
	if _, ok := adminExactRouteKeys[method+" "+path]; ok {
		return true
	}

	// Step 2: path parameter 付き route は generated binding が concrete path ではなく pattern として登録するため、prefix と method で保護対象に含める。
	return (method == stdhttp.MethodGet && strings.HasPrefix(path, "/api/v1/accounts/")) ||
		(method == stdhttp.MethodDelete && strings.HasPrefix(path, "/api/v1/auth/passkeys/"))
}

func isAdminPreAuthPath(method string, path string) bool {
	// Step 1: session 発行前または Cookie refresh 用の Admin auth route だけを session-bound CSRF 例外にする。
	if method != stdhttp.MethodPost {
		return false
	}
	preAuthPaths := map[string]struct{}{
		"/api/v1/auth/passkey/start":         {},
		"/api/v1/auth/passkey/finish":        {},
		"/api/v1/auth/operator-setup/start":  {},
		"/api/v1/auth/operator-setup/finish": {},
		"/api/v1/auth/setup/start":           {},
		"/api/v1/auth/setup/finish":          {},
		"/api/v1/auth/operator/refresh":      {},
	}
	_, ok := preAuthPaths[path]
	return ok
}

func adminRouteRequiresCSRF(method string, path string) bool {
	// Step 1: Admin operator passkey 管理は一覧を含めて session-bound CSRF を要求し、認証手段の列挙と削除を同じ session binding に閉じる。
	if (method == stdhttp.MethodGet && path == "/api/v1/auth/passkeys") ||
		(method == stdhttp.MethodDelete && strings.HasPrefix(path, "/api/v1/auth/passkeys/")) {
		return true
	}

	// Step 2: account/operator mutation と logout の transport CSRF 境界を固定し、RBAC 判定は application/domain use case に残す。
	return (method == stdhttp.MethodPost && path == "/api/v1/accounts") ||
		(method == stdhttp.MethodPost && path == "/api/v1/auth/operators") ||
		(method == stdhttp.MethodPost && path == "/api/v1/auth/operator/logout")
}

func adminRoutePermission(method string, path string) string {
	// Step 1: Admin operator passkey 管理 route は operator 自身の認証手段管理 permission と結び、account mutation 権限と分離する。
	if (method == stdhttp.MethodGet && path == "/api/v1/auth/passkeys") ||
		(method == stdhttp.MethodDelete && strings.HasPrefix(path, "/api/v1/auth/passkeys/")) {
		return "operator-passkeys:manage"
	}

	// Step 2: account 作成 route だけを accounts:create permission と結び、operator 管理権限と分離する。
	if method == stdhttp.MethodPost && path == "/api/v1/accounts" {
		return "accounts:create"
	}
	if method == stdhttp.MethodPost && path == "/api/v1/auth/operators" {
		return "operators:create"
	}
	if method == stdhttp.MethodPost && path == "/api/v1/auth/operator/logout" {
		return "operators:logout"
	}

	// Step 3: permission が未定義の CSRF route は production validator 側で fail-closed にし、CSRF を検証しない mutation を許可しない。
	return ""
}

func adminBearerToken(header string) string {
	// Step 1: Authorization header を RFC 6750 の Bearer prefix だけに限定し、Basic や Product 固有値を operator token として扱わない。
	trimmed := strings.TrimSpace(header)
	if !strings.HasPrefix(trimmed, "Bearer ") {
		return ""
	}

	// Step 2: prefix 後の token 本体を空白除去して返し、空 token は呼び出し側で unauthenticated として拒否する。
	return strings.TrimSpace(strings.TrimPrefix(trimmed, "Bearer "))
}

func bindAdminOperatorContext(c *gin.Context, operatorContext operatorSessionContext) {
	// Step 1: Gin handler から参照しやすい key に検証済み operator/session 情報を設定する。
	c.Set(adminContextKeyOperatorID, operatorContext.OperatorID)
	c.Set(adminContextKeyOperatorEmail, operatorContext.OperatorEmail)
	c.Set(adminContextKeyOperatorRole, operatorContext.OperatorRole)
	c.Set(adminContextKeyOperatorActive, operatorContext.OperatorActive)
	c.Set(adminContextKeyOperatorPasskeyRegistrationState, operatorContext.OperatorPasskeyRegistrationState)
	c.Set(adminContextKeySessionID, operatorContext.SessionID)
	c.Set(adminContextKeyCSRFToken, operatorContext.CSRFToken)

	// Step 2: generated strict handler が受け取る request.Context にも同じ値を束縛し、将来の handler 実装が Gin 依存を広げずに読めるようにする。
	ctx := c.Request.Context()
	ctx = context.WithValue(ctx, adminContextValueKey(adminContextKeyOperatorID), operatorContext.OperatorID)
	ctx = context.WithValue(ctx, adminContextValueKey(adminContextKeyOperatorEmail), operatorContext.OperatorEmail)
	ctx = context.WithValue(ctx, adminContextValueKey(adminContextKeyOperatorRole), operatorContext.OperatorRole)
	ctx = context.WithValue(ctx, adminContextValueKey(adminContextKeyOperatorActive), operatorContext.OperatorActive)
	ctx = context.WithValue(ctx, adminContextValueKey(adminContextKeyOperatorPasskeyRegistrationState), operatorContext.OperatorPasskeyRegistrationState)
	ctx = context.WithValue(ctx, adminContextValueKey(adminContextKeySessionID), operatorContext.SessionID)
	ctx = context.WithValue(ctx, adminContextValueKey(adminContextKeyCSRFToken), operatorContext.CSRFToken)
	c.Request = c.Request.WithContext(ctx)
}

func writeAdminAuthValidationError(c *gin.Context, err error) {
	// Step 1: validator の抽象 error を HTTP status と stable error classification へ変換し、内部理由や secret を response へ出さない。
	switch {
	case errors.Is(err, errAdminOperatorForbidden):
		writeAdminAuthFailure(c, stdhttp.StatusForbidden, adminopenapi.Unauthenticated)
	case errors.Is(err, errAdminOperatorInternal):
		writeAdminAuthFailure(c, stdhttp.StatusServiceUnavailable, adminopenapi.InternalError)
	default:
		writeAdminAuthFailure(c, stdhttp.StatusUnauthorized, adminopenapi.Unauthenticated)
	}
}

func writeAdminAuthFailure(c *gin.Context, status int, classification adminopenapi.WWWTemplateAuthFailureClassification) {
	// Step 1: middleware で返す失敗応答にも no-store と security headers を適用し、generated handler 到達時と同じ header 境界を保つ。
	applyAdminSecurityHeaders(c)

	// Step 2: non-secret な分類と request ID だけを返し、token/session/CSRF の詳細を外部へ露出しない。
	c.AbortWithStatusJSON(status, adminopenapi.WWWTemplateAuthFailureResponse{Error: classification, RequestId: fallbackRequestID})
}
