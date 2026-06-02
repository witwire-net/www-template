package admin

import (
	"context"
	"errors"
	stdhttp "net/http"
	"strings"

	"github.com/gin-gonic/gin"

	sharedhttp "www-template/packages/backend/internal/adapter/http/shared"
	"www-template/packages/backend/internal/generated/adminopenapi"
	"www-template/packages/backend/internal/platform/config"
)

const adminAuthHeader = "Authorization"
const adminOriginHeader = "Origin"
const adminFetchSiteHeader = "Sec-Fetch-Site"
const adminSecurityCSP = sharedhttp.SecurityCSP
const adminSecurityHSTS = sharedhttp.SecurityHSTS
const adminSecurityReferrerPolicy = sharedhttp.SecurityReferrerPolicy
const adminContextKeyOperatorID = "admin.operator.id"
const adminContextKeyOperatorEmail = "admin.operator.email"
const adminContextKeyOperatorRole = "admin.operator.role"
const adminContextKeyOperatorActive = "admin.operator.active"
const adminContextKeyOperatorPasskeyRegistrationState = "admin.operator.passkey_registration_state"
const adminContextKeySessionID = "admin.operator.session_id"

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
	"POST /api/v1/auth/passkey/finish":        {},
	"POST /api/v1/auth/passkey/start":         {},
}

type adminContextValueKey string

type adminRouterDependencies struct {
	operatorSessions          operatorSessionValidator
	operatorAuth              adminOperatorAuthenticator
	operatorPasskeyAuth       adminOperatorPasskeyAuthenticator
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
//   - Permission は accounts:create など、application auth service が domain RBAC を検証するための permission 名である。
//   - Permission が空の場合は read route として CurrentOperator 検証だけへ委譲する。
type OperatorSessionValidationInput struct {
	AccessToken string
	Permission  string
}

type operatorSessionValidationInput = OperatorSessionValidationInput

// OperatorSessionContext は Admin protected route middleware が handler へ束縛する検証済み Operator context である。
//
// 役割:
//   - Product account の認証状態を含めず、Admin OperatorAuth domain 由来の値だけを保持する。
//   - SessionID は accessToken payload で検証された Admin operator session selector であり、handler で token を再解析しないために使う。
type OperatorSessionContext struct {
	OperatorID                       string
	OperatorEmail                    string
	OperatorRole                     string
	OperatorActive                   bool
	OperatorPasskeyRegistrationState string
	SessionID                        string
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
		if !sharedhttp.FetchMetadataAccepted(c.GetHeader(adminFetchSiteHeader)) {
			writeAdminAuthFailure(c, stdhttp.StatusForbidden, adminopenapi.Unauthenticated)
			return
		}

		// Step 3: passkey / setup / refresh の pre-auth flow は bearer accessToken を持たないため、Origin 検証だけを通過条件にして handler へ委譲する。
		if isAdminPreAuthPath(c.Request.Method, path) {
			c.Next()
			return
		}

		// Step 4: protected route は bearer accessToken を必須にし、Product bearer や空 header を operator session として扱わない。
		accessToken := sharedhttp.BearerToken(c.GetHeader(adminAuthHeader))
		if accessToken == "" {
			writeAdminAuthFailure(c, stdhttp.StatusUnauthorized, adminopenapi.Unauthenticated)
			return
		}

		// Step 5: session validator が未接続なら Admin API を fail-close し、暫定 handler が認証なしに公開されることを防ぐ。
		if validator == nil {
			writeAdminAuthFailure(c, stdhttp.StatusServiceUnavailable, adminopenapi.InternalError)
			return
		}

		// Step 6: protected route は CSRF header を要求せず、Bearer accessToken と route permission だけを validator へ渡す。
		input := operatorSessionValidationInput{AccessToken: accessToken, Permission: adminRoutePermission(c.Request.Method, path)}

		// Step 7: adapter では token の中身や RBAC を判定せず、Admin operator session validator に検証を集約する。
		operatorContext, err := validator.ValidateOperatorSession(c.Request.Context(), input)
		if err != nil {
			writeAdminAuthValidationError(c, err)
			return
		}

		// Step 8: 検証済み operator/session 情報を Gin context と request context の両方へ設定し、handler が Product auth state を参照せずに済む境界を作る。
		bindAdminOperatorContext(c, operatorContext)
		c.Next()
	}
}

func applyAdminSecurityHeaders(c *gin.Context) {
	// Step 1: Admin API response は operator session や顧客 PII を含み得るため、no-store と browser hardening header を shared helper concept からまとめて適用する。
	sharedhttp.ApplyBrowserSecurityHeaders(c, noStoreValue)
}

func adminOriginAccepted(cfg config.Config, originHeader string, method string) bool {
	// Step 1: unsafe method は browser が付与する Origin を必須にし、cross-site request を Bearer 検証前に拒否する。
	trimmedOrigin := strings.TrimSpace(originHeader)
	if trimmedOrigin == "" {
		return !adminMethodRequiresOrigin(method)
	}

	// Step 2: Origin header と Admin runtime domain を origin 形式へ正規化してから比較し、大文字小文字や末尾 slash の揺れを吸収する。
	return sharedhttp.OriginMatches(trimmedOrigin, cfg.Admin.Domain)
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
		(method == stdhttp.MethodDelete && strings.HasPrefix(path, "/api/v1/auth/passkeys/")) ||
		isAdminContextRefreshPath(method, path)
}

func isAdminPreAuthPath(method string, path string) bool {
	// Step 1: session 発行前または Cookie refresh 用の Admin auth route だけを bearer session 検証の例外にする。
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
	}
	_, ok := preAuthPaths[path]
	return ok || isAdminContextRefreshPath(method, path)
}

func isAdminContextRefreshPath(method string, path string) bool {
	// Step 1: Admin context refresh は path scoped Cookie を認証材料にするため、旧固定 refresh path ではなく context ID 付き path だけを pre-auth 対象にする。
	if method != stdhttp.MethodPost {
		return false
	}

	// Step 2: prefix / suffix / 空 context の三点を検査し、generated route と同じ `/auth/contexts/{authContextId}/refresh` だけを Origin 検証対象に含める。
	const prefix = "/api/v1/auth/contexts/"
	const suffix = "/refresh"
	if !strings.HasPrefix(path, prefix) || !strings.HasSuffix(path, suffix) {
		return false
	}
	contextID := strings.TrimSuffix(strings.TrimPrefix(path, prefix), suffix)
	return contextID != "" && !strings.Contains(contextID, "/")
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

	// Step 3: permission が未定義の protected route は read-only current validation として扱う。
	return ""
}

func bindAdminOperatorContext(c *gin.Context, operatorContext operatorSessionContext) {
	// Step 1: Gin handler から参照しやすい key に検証済み operator/session 情報を設定する。
	c.Set(adminContextKeyOperatorID, operatorContext.OperatorID)
	c.Set(adminContextKeyOperatorEmail, operatorContext.OperatorEmail)
	c.Set(adminContextKeyOperatorRole, operatorContext.OperatorRole)
	c.Set(adminContextKeyOperatorActive, operatorContext.OperatorActive)
	c.Set(adminContextKeyOperatorPasskeyRegistrationState, operatorContext.OperatorPasskeyRegistrationState)
	c.Set(adminContextKeySessionID, operatorContext.SessionID)

	// Step 2: generated strict handler が受け取る request.Context にも同じ値を束縛し、将来の handler 実装が Gin 依存を広げずに読めるようにする。
	ctx := c.Request.Context()
	ctx = context.WithValue(ctx, adminContextValueKey(adminContextKeyOperatorID), operatorContext.OperatorID)
	ctx = context.WithValue(ctx, adminContextValueKey(adminContextKeyOperatorEmail), operatorContext.OperatorEmail)
	ctx = context.WithValue(ctx, adminContextValueKey(adminContextKeyOperatorRole), operatorContext.OperatorRole)
	ctx = context.WithValue(ctx, adminContextValueKey(adminContextKeyOperatorActive), operatorContext.OperatorActive)
	ctx = context.WithValue(ctx, adminContextValueKey(adminContextKeyOperatorPasskeyRegistrationState), operatorContext.OperatorPasskeyRegistrationState)
	ctx = context.WithValue(ctx, adminContextValueKey(adminContextKeySessionID), operatorContext.SessionID)
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

	// Step 2: non-secret な分類と request ID だけを返し、token/session の詳細を外部へ露出しない。
	c.AbortWithStatusJSON(status, adminopenapi.WWWTemplateAuthFailureResponse{Error: classification, RequestId: fallbackRequestID})
}
