package app

import (
	adminauth "www-template/packages/backend/internal/application/auth"
	"www-template/packages/backend/internal/platform/config"
)

// operatorAuthConfigFromRuntime は共通 auth runtime 設定を Admin operator session 設定へ変換する。
//
// 役割:
//   - Product account auth と Admin operator auth が同じ config.AuthConfig を起点にしつつ、
//     各自の session semantics に合う DTO へ変換する。
//   - admin_container.go の newAdminOperatorAuthService / newAdminOperatorPasskeyLoginService と、
//     product_runtime_test.go の TestAdminOperatorAuthRuntimeConfigUsesOperatorSemantics の両方から使われるため、
//     admin_runtime.go ではなくこの中立ファイルに配置する。
//
// 振る舞い:
//   - RefreshTokenTTL が未設定（0）の場合は SessionAbsoluteTTL を operator refresh session TTL として使う。
//     これにより、無期限 operator session を避ける。
//   - OperatorRefreshCookieLifetime は server-side operator session TTL と同じ長さにし、
//     server 期限を超えて refresh Cookie が残らないようにする。
//
// 引数:
//   - authRuntime: Product / Admin 共通の auth 設定。TTL と WebAuthn RPID を含む。
//
// 戻り値:
//   - adminauth.OperatorSessionConfig: Admin operator session lifecycle に使う設定値。
//
// 使用例:
//
//	cfg := operatorAuthConfigFromRuntime(authRuntime)
//	service, err := adminauth.NewOperatorSessionService(deps, cfg)
func operatorAuthConfigFromRuntime(authRuntime config.AuthConfig) adminauth.OperatorSessionConfig {
	// Step 1: Admin operator refresh session は共通 auth runtime の refresh TTL を operator session TTL として解釈し、未設定時は絶対 session TTL に丸めて無期限 operator session を避ける。
	refreshSessionTTL := authRuntime.RefreshTokenTTL
	if refreshSessionTTL == 0 {
		refreshSessionTTL = authRuntime.SessionAbsoluteTTL
	}

	// Step 2: Admin operator refresh Cookie lifetime を server-side operator session TTL と同じ長さにし、server 期限を超えて refresh Cookie が残らないようにする。
	return adminauth.OperatorSessionConfig{
		OperatorAccessTokenTTL:        authRuntime.SessionIdleTTL,
		OperatorRefreshSessionTTL:     refreshSessionTTL,
		OperatorRefreshCookieLifetime: refreshSessionTTL,
		WebAuthnRPID:                  authRuntime.WebAuthnRPID,
	}
}
