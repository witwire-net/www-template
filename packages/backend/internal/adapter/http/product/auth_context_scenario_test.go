package product

import (
	"fmt"
	stdhttp "net/http"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestProductContextScopedAuthIssuanceScenarioTitles(t *testing.T) {
	t.Parallel()

	t.Run("[AUTH-BE-S094] Product adapter returns explicit account subject payload only", func(t *testing.T) {
		// Step 1: Product Cookie mode login を実行し、HTTP adapter が generated response の account field を Product subject として出す経路を観測する。
		env := newJWTAuthTestEnv(t)
		loginBody, refreshToken := productScenarioLogin(t, env.router, "member@example.com", "existing-credential", "cookie")

		// Step 2: login response は account subject payload だけを持ち、Admin operator payload を混入しないことを境界 evidence として固定する。
		assertProductAccountSubject(t, loginBody, true)
		if _, ok := loginBody["operator"]; ok {
			t.Fatalf("expected Product login response not to contain operator payload, got %#v", loginBody)
		}

		// Step 3: context refresh response でも同じ Product account subject payload を返し、refreshToken 平文や operator payload を含めない。
		response := performRefreshWithCookie(t, env.router, refreshToken)
		assertStatus(t, response, stdhttp.StatusOK)
		refreshBody := decodeJSONBody(t, response)
		assertProductAccountSubject(t, refreshBody, false)
		if _, ok := refreshBody["operator"]; ok {
			t.Fatalf("expected Product refresh response not to contain operator payload, got %#v", refreshBody)
		}
	})

	t.Run("[AUTH-BE-S060] Cookie login returns accessToken body and path-scoped refresh Cookie", func(t *testing.T) {
		// Step 1: Cookie mode の passkey login を完了し、browser 用 response body と HttpOnly Cookie を同時に取得する。
		env := newJWTAuthTestEnv(t)
		body, refreshToken := productScenarioLogin(t, env.router, "member@example.com", "existing-credential", "cookie")

		// Step 2: body は短命 accessToken と context/session metadata を返し、refreshToken 平文を含めないことを検証する。
		if body["accessToken"] == "" || body["authContextId"] == "" || body["sessionId"] == "" || body["account"] == nil {
			t.Fatalf("expected Product Cookie login body with accessToken and metadata, got %#v", body)
		}
		productScenarioAssertNoBodyRefreshToken(t, body, refreshToken)

		// Step 3: Set-Cookie の Path が authContextId に bind された context refresh path であることを固定する。
		if got, want := productScenarioRefreshCookiePath(t, env.router, body), productScenarioRefreshPath(t, body); got != want {
			t.Fatalf("expected refresh Cookie path %q, got %q", want, got)
		}
	})

	t.Run("[AUTH-BE-S063] Cookie login and refresh never expose browser refreshToken in response body", func(t *testing.T) {
		// Step 1: Cookie mode login で refreshToken を Cookie に隔離し、body には公開しない初期状態を作る。
		env := newJWTAuthTestEnv(t)
		loginBody, refreshToken := productScenarioLogin(t, env.router, "member@example.com", "existing-credential", "cookie")
		productScenarioAssertNoBodyRefreshToken(t, loginBody, refreshToken)

		// Step 2: context refresh でも新しい refreshToken は Set-Cookie にだけ現れ、body へ漏れないことを確認する。
		response := performRefreshWithCookie(t, env.router, refreshToken)
		assertStatus(t, response, stdhttp.StatusOK)
		refreshBody := decodeJSONBody(t, response)
		rotatedRefreshToken := refreshCookieValueFromResponse(t, response)
		productScenarioAssertNoBodyRefreshToken(t, refreshBody, rotatedRefreshToken)
	})

	t.Run("[AUTH-BE-S062] Cookie refresh rotates path-scoped Cookie and omits body refreshToken", func(t *testing.T) {
		// Step 1: 有効な Cookie mode session を作成し、context-scoped refresh endpoint の入力を準備する。
		env := newJWTAuthTestEnv(t)
		loginBody, oldRefreshToken := productScenarioLogin(t, env.router, "member@example.com", "existing-credential", "cookie")

		// Step 2: URL path の authContextId と Cookie credential を使って refresh し、旧 token が rotation されることを確認する。
		response := performRefreshWithCookie(t, env.router, oldRefreshToken)
		assertStatus(t, response, stdhttp.StatusOK)
		refreshBody := decodeJSONBody(t, response)
		newRefreshToken := refreshCookieValueFromResponse(t, response)
		if newRefreshToken == oldRefreshToken {
			t.Fatalf("expected rotated refresh token, got unchanged value %q", newRefreshToken)
		}
		if refreshBody["authContextId"] != loginBody["authContextId"] || refreshBody["accessToken"] == "" {
			t.Fatalf("expected refreshed session for same authContextId, got %#v", refreshBody)
		}
		productScenarioAssertNoBodyRefreshToken(t, refreshBody, newRefreshToken)
	})

}

func TestProductContextScopedAuthRefreshOwnershipScenarioTitles(t *testing.T) {
	t.Parallel()

	t.Run("[AUTH-BE-S066] Refresh rotation preserves non-target session state", func(t *testing.T) {
		// Step 1: 2 つの Product account session を同一 router 上に作り、対象外 session の refresh credential を記録する。
		clock := &mutableClock{current: time.Date(2026, time.March, 21, 0, 0, 0, 0, time.UTC)}
		stateRepo := newStubAuthStateRepository(clock.Now)
		accountRepo := newMultiAccountStubAccountAuthRepository(
			stubAccountAuthRepositoryWithMember(),
			stubAccountAuthRepositoryWithAccount("01ARZ3NDEKTSV4RRFFQ69G5FB1", "other@example.com", "other-credential"),
		)
		env := newJWTAuthTestEnvWithRepository(t, clock, stateRepo, accountRepo)
		targetBody, targetRefreshToken := productScenarioLogin(t, env.router, "member@example.com", "existing-credential", "cookie")
		otherBody, otherRefreshToken := productScenarioLogin(t, env.router, "other@example.com", "other-credential", "cookie")

		// Step 2: 対象 session だけを refresh し、対象外 session の旧 refresh credential がまだ利用できることを実行で確認する。
		targetResponse := performRefreshWithCookie(t, env.router, targetRefreshToken)
		assertStatus(t, targetResponse, stdhttp.StatusOK)
		otherResponse := performJSONWithHeaders(t, env.router, stdhttp.MethodPost, productScenarioRefreshPath(t, otherBody), nil, "", productScenarioRefreshCookieHeader(otherRefreshToken))
		assertStatus(t, otherResponse, stdhttp.StatusOK)
		if targetBody["authContextId"] == otherBody["authContextId"] {
			t.Fatalf("expected distinct authContextId values, got %#v and %#v", targetBody, otherBody)
		}
	})

	t.Run("[AUTH-BE-S083] Refresh rejects credential whose owner does not match path authContextId", func(t *testing.T) {
		// Step 1: 2 つの session を作り、path と Cookie の所属を意図的に食い違わせる。
		clock := &mutableClock{current: time.Date(2026, time.March, 21, 0, 0, 0, 0, time.UTC)}
		stateRepo := newStubAuthStateRepository(clock.Now)
		accountRepo := newMultiAccountStubAccountAuthRepository(
			stubAccountAuthRepositoryWithMember(),
			stubAccountAuthRepositoryWithAccount("01ARZ3NDEKTSV4RRFFQ69G5FB1", "other@example.com", "other-credential"),
		)
		env := newJWTAuthTestEnvWithRepository(t, clock, stateRepo, accountRepo)
		targetBody, _ := productScenarioLogin(t, env.router, "member@example.com", "existing-credential", "cookie")
		_, otherRefreshToken := productScenarioLogin(t, env.router, "other@example.com", "other-credential", "cookie")
		storedBefore := len(env.productRefreshStore.sessions)

		// Step 2: account A の path に account B の Cookie を送っても、新 credential を発行しないことを確認する。
		response := performJSONWithHeaders(t, env.router, stdhttp.MethodPost, productScenarioRefreshPath(t, targetBody), nil, "", productScenarioRefreshCookieHeader(otherRefreshToken))
		assertStatus(t, response, stdhttp.StatusUnauthorized)
		assertNoStore(t, response)
		if len(env.productRefreshStore.sessions) != storedBefore-1 {
			t.Fatalf("expected mismatched credential to consume only presented token without issuing a replacement, before=%d after=%d", storedBefore, len(env.productRefreshStore.sessions))
		}
		productScenarioAssertNoSetCookie(t, response.Header().Values("Set-Cookie"))
	})

	t.Run("[AUTH-BE-S090] Cookie Path alone is not authorization boundary for refresh", func(t *testing.T) {
		// Step 1: Cookie Path が browser 側で送信可に見える状況でも、server-side record の authContextId を別 path と照合させる。
		env := newJWTAuthTestEnv(t)
		loginBody, refreshToken := productScenarioLogin(t, env.router, "member@example.com", "existing-credential", "cookie")
		storedBefore := len(env.productRefreshStore.sessions)
		wrongPath := strings.Replace(productScenarioRefreshPath(t, loginBody), productScenarioAuthContextID(t, loginBody), "01ARZ3NDEKTSV4RRFFQ69G5FB9", 1)

		// Step 2: Cookie 値が存在していても path の authContextId と record が一致しなければ拒否され、Set-Cookie を返さない。
		response := performJSONWithHeaders(t, env.router, stdhttp.MethodPost, wrongPath, nil, "", productScenarioRefreshCookieHeader(refreshToken))
		assertStatus(t, response, stdhttp.StatusUnauthorized)
		assertNoStore(t, response)
		if len(env.productRefreshStore.sessions) != storedBefore-1 {
			t.Fatalf("expected path mismatch to consume only presented token without replacement, before=%d after=%d", storedBefore, len(env.productRefreshStore.sessions))
		}
		productScenarioAssertNoSetCookie(t, response.Header().Values("Set-Cookie"))
	})

	t.Run("[AUTH-BE-S086] Bearer refresh rotates body refreshToken and does not set Cookie", func(t *testing.T) {
		// Step 1: Bearer mode login で external client 用の body refreshToken を発行し、Cookie がないことを確認する。
		env := newJWTAuthTestEnv(t)
		loginBody, _ := productScenarioLogin(t, env.router, "member@example.com", "existing-credential", "bearer")
		oldRefreshToken, ok := loginBody["refreshToken"].(string)
		if !ok || oldRefreshToken == "" {
			t.Fatalf("expected bearer login body refreshToken, got %#v", loginBody)
		}

		// Step 2: body credentialMode=bearer と refreshToken だけで rotation し、新 refreshToken も body に返し、Cookie を設定しない。
		response := performJSON(t, env.router, stdhttp.MethodPost, productScenarioRefreshPath(t, loginBody), map[string]any{"credentialMode": "bearer", "refreshToken": oldRefreshToken}, "")
		assertStatus(t, response, stdhttp.StatusOK)
		body := decodeJSONBody(t, response)
		newRefreshToken, ok := body["refreshToken"].(string)
		if !ok || newRefreshToken == "" || newRefreshToken == oldRefreshToken {
			t.Fatalf("expected rotated bearer body refreshToken, got %#v", body["refreshToken"])
		}
		productScenarioAssertNoSetCookie(t, response.Header().Values("Set-Cookie"))
	})

}

func TestProductContextScopedAuthRefreshRejectionScenarioTitles(t *testing.T) {
	t.Parallel()

	t.Run("[AUTH-BE-S080] Refresh endpoint rejects Authorization header before rotation", func(t *testing.T) {
		// Step 1: 有効な Cookie refresh request に Authorization header を追加し、accessToken が refresh credential として使われない状態を作る。
		env := newJWTAuthTestEnv(t)
		loginBody, refreshToken := productScenarioLogin(t, env.router, "member@example.com", "existing-credential", "cookie")
		storedBefore := len(env.productRefreshStore.sessions)

		// Step 2: Authorization header がある時点で rotation 前に拒否され、refresh store が変化しないことを確認する。
		response := performJSONWithHeaders(t, env.router, stdhttp.MethodPost, productScenarioRefreshPath(t, loginBody), nil, "access-token-not-a-refresh-token", productScenarioRefreshCookieHeader(refreshToken))
		assertStatus(t, response, stdhttp.StatusUnauthorized)
		assertNoStore(t, response)
		if len(env.productRefreshStore.sessions) != storedBefore {
			t.Fatalf("expected Authorization rejection before rotation, before=%d after=%d", storedBefore, len(env.productRefreshStore.sessions))
		}
	})

	t.Run("[AUTH-BE-S087] Refresh rejects simultaneous Cookie and body refreshToken", func(t *testing.T) {
		// Step 1: 有効な Cookie refreshToken を持つ request に body refreshToken も混ぜ、credential ambiguity を作る。
		env := newJWTAuthTestEnv(t)
		loginBody, refreshToken := productScenarioLogin(t, env.router, "member@example.com", "existing-credential", "cookie")
		storedBefore := len(env.productRefreshStore.sessions)

		// Step 2: exactly-one 条件違反として 400 で拒否され、rotation されないことを確認する。
		response := performJSONWithHeaders(t, env.router, stdhttp.MethodPost, productScenarioRefreshPath(t, loginBody), map[string]any{"refreshToken": "body-refresh-token"}, "", productScenarioRefreshCookieHeader(refreshToken))
		assertStatus(t, response, stdhttp.StatusBadRequest)
		assertNoStore(t, response)
		if len(env.productRefreshStore.sessions) != storedBefore {
			t.Fatalf("expected ambiguous credential rejection before rotation, before=%d after=%d", storedBefore, len(env.productRefreshStore.sessions))
		}
	})

	t.Run("[AUTH-BE-S091] Reusing old context refresh token allows only one successful rotation", func(t *testing.T) {
		// Step 1: 同じ old refreshToken を 2 回提示し、atomic consume 後の再利用を再現する。
		env := newJWTAuthTestEnv(t)
		_, refreshToken := productScenarioLogin(t, env.router, "member@example.com", "existing-credential", "cookie")

		// Step 2: 1 回目だけ成功し、2 回目は old token reuse として拒否されることを確認する。
		first := performRefreshWithCookie(t, env.router, refreshToken)
		assertStatus(t, first, stdhttp.StatusOK)
		second := performRefreshWithCookie(t, env.router, refreshToken)
		assertStatus(t, second, stdhttp.StatusUnauthorized)
		assertNoStore(t, second)
	})

	t.Run("[AUTH-BE-S044] Consumed refresh token reuse is rejected", func(t *testing.T) {
		// Step 1: refreshToken を一度 rotation して旧 token を consume する。
		env := newJWTAuthTestEnv(t)
		_, refreshToken := productScenarioLogin(t, env.router, "member@example.com", "existing-credential", "cookie")
		first := performRefreshWithCookie(t, env.router, refreshToken)
		assertStatus(t, first, stdhttp.StatusOK)

		// Step 2: 同じ旧 token を再利用すると unauthorized になり、新 credential が発行されないことを確認する。
		second := performRefreshWithCookie(t, env.router, refreshToken)
		assertStatus(t, second, stdhttp.StatusUnauthorized)
		assertNoStore(t, second)
		productScenarioAssertNoSetCookie(t, second.Header().Values("Set-Cookie"))
	})

	t.Run("[AUTH-BE-S045] Invalid refresh token is rejected without credential issuance", func(t *testing.T) {
		// Step 1: 存在しない refreshToken を Cookie として提示し、不正 token の fail-close 経路を作る。
		env := newJWTAuthTestEnv(t)

		// Step 2: invalid token は unauthorized で拒否され、Set-Cookie を返さないことを確認する。
		response := performRefreshWithCookie(t, env.router, "invalid-token")
		assertStatus(t, response, stdhttp.StatusUnauthorized)
		assertNoStore(t, response)
		productScenarioAssertNoSetCookie(t, response.Header().Values("Set-Cookie"))
	})

}

func TestProductProtectedRouteScenarioTitles(t *testing.T) {
	t.Parallel()

	t.Run("[AUTH-BE-S084] Protected routes ignore browser Cookie credentials without bearer", func(t *testing.T) {
		// Step 1: 有効な refresh Cookie を持つ browser request だが、Authorization header は持たない protected route 呼び出しを作る。
		env := newJWTAuthTestEnv(t)
		_, refreshToken := productScenarioLogin(t, env.router, "member@example.com", "existing-credential", "cookie")

		// Step 2: refresh Cookie は認可材料にならず、protected route は unauthenticated で拒否される。
		response := performJSONWithHeaders(t, env.router, stdhttp.MethodGet, "/api/v1/passkeys", nil, "", productScenarioRefreshCookieHeader(refreshToken))
		assertStatus(t, response, stdhttp.StatusUnauthorized)
		assertFailureCode(t, response, "unauthenticated")
	})

	t.Run("[AUTH-BE-S085] Protected mutation accepts bearer without X-Auth-Context-Id or CSRF", func(t *testing.T) {
		// Step 1: 有効な bearer accessToken だけを持つ state-changing logout request を準備する。
		env := newJWTAuthTestEnv(t)
		loginBody, _ := productScenarioLogin(t, env.router, "member@example.com", "existing-credential", "cookie")
		accessToken := loginBody["accessToken"].(string)

		// Step 2: X-Auth-Context-Id と CSRF header がなくても accessToken claims と session record で成功することを検証する。
		response := performJSON(t, env.router, stdhttp.MethodPost, "/api/v1/auth/logout", nil, accessToken)
		assertStatus(t, response, stdhttp.StatusOK)
	})

	t.Run("[AUTH-BE-S009] Request without session is unauthenticated not expired", func(t *testing.T) {
		// Step 1: protected route へ session credential を一切提示しない request を送る。
		env := newJWTAuthTestEnv(t)

		// Step 2: missing session は session-expired ではなく unauthenticated として分類されることを確認する。
		response := performJSON(t, env.router, stdhttp.MethodGet, "/api/v1/passkeys", nil, "")
		assertStatus(t, response, stdhttp.StatusUnauthorized)
		assertFailureCode(t, response, "unauthenticated")
	})

	t.Run("[AUTH-BE-S046] Expired bearer accessToken is rejected as session-expired", func(t *testing.T) {
		// Step 1: 有効な accessToken を発行後、test clock を accessToken TTL より後へ進める。
		env := newJWTAuthTestEnv(t)
		loginBody, _ := productScenarioLogin(t, env.router, "member@example.com", "existing-credential", "cookie")
		env.advance(20 * time.Minute)

		// Step 2: expired bearer は session-expired として拒否されることを確認する。
		response := performJSON(t, env.router, stdhttp.MethodGet, "/api/v1/passkeys", nil, loginBody["accessToken"].(string))
		assertStatus(t, response, stdhttp.StatusUnauthorized)
		assertFailureCode(t, response, "session-expired")
	})

}

func TestProductLogoutAndSuspensionScenarioTitles(t *testing.T) {
	t.Parallel()

	t.Run("[AUTH-BE-S042] Logout revokes only target session and preserves another session", func(t *testing.T) {
		// Step 1: 同じ account に 2 つの session を発行し、片方だけ logout 対象にする。
		env := newJWTAuthTestEnv(t)
		firstBody, _ := productScenarioLogin(t, env.router, "member@example.com", "existing-credential", "cookie")
		secondBody, _ := productScenarioLogin(t, env.router, "member@example.com", "existing-credential", "cookie")

		// Step 2: first session を logout しても second session の bearer は引き続き protected route に使えることを確認する。
		logoutResponse := performJSON(t, env.router, stdhttp.MethodPost, "/api/v1/auth/logout", nil, firstBody["accessToken"].(string))
		assertStatus(t, logoutResponse, stdhttp.StatusOK)
		preservedResponse := performJSON(t, env.router, stdhttp.MethodGet, "/api/v1/passkeys", nil, secondBody["accessToken"].(string))
		assertStatus(t, preservedResponse, stdhttp.StatusOK)
	})

	t.Run("[AUTH-BE-S092] Logout returns clear command for target refresh Cookie path", func(t *testing.T) {
		// Step 1: Cookie mode session を作り、logout 対象の authContextId を response body から取得する。
		env := newJWTAuthTestEnv(t)
		loginBody, _ := productScenarioLogin(t, env.router, "member@example.com", "existing-credential", "cookie")

		// Step 2: logout response が同じ authContextId の Cookie Path を削除 command と Set-Cookie で返すことを検証する。
		response := performJSON(t, env.router, stdhttp.MethodPost, "/api/v1/auth/logout", nil, loginBody["accessToken"].(string))
		assertStatus(t, response, stdhttp.StatusOK)
		assertRefreshCookieCleared(t, response)
		body := decodeJSONBody(t, response)
		commands, ok := body["clearCookieCommands"].([]any)
		if !ok || len(commands) != 1 {
			t.Fatalf("expected one clearCookieCommands item, got %#v", body["clearCookieCommands"])
		}
		command := commands[0].(map[string]any)
		if command["authContextId"] != loginBody["authContextId"] || command["path"] != productScenarioRefreshPath(t, loginBody) {
			t.Fatalf("expected clear command for target context, got %#v", command)
		}
	})

}

func TestProductSuspensionScenarioTitles(t *testing.T) {
	t.Parallel()

	t.Run("[AUTH-BE-S054] Suspended account cannot receive new accessToken", func(t *testing.T) {
		// Step 1: account を suspended に変更してから valid passkey assertion を完了する。
		env, accountRepo := newJWTAuthTestEnvWithAccountRepo(t)
		revokedAt := env.now()
		accountRepo.account = accountRepo.account.WithStatus("suspended", &revokedAt)
		challenge := startPasskey(t, env.router, "member@example.com")

		// Step 2: token pair は発行されず、account-suspended の stable failure になることを確認する。
		response := performJSON(t, env.router, stdhttp.MethodPost, "/api/v1/auth/passkey/finish", map[string]any{"credential": assertionCredentialJSON("existing-credential", challengeValue(challenge))}, "")
		assertStatus(t, response, stdhttp.StatusForbidden)
		assertFailureCode(t, response, "account-suspended")
		productScenarioAssertNoSetCookie(t, response.Header().Values("Set-Cookie"))
	})

	t.Run("[AUTH-BE-S055] Suspended account existing bearer is rejected", func(t *testing.T) {
		// Step 1: active account の bearer を発行後、account status を suspended へ変更する。
		env, accountRepo := newJWTAuthTestEnvWithAccountRepo(t)
		loginBody, _ := productScenarioLogin(t, env.router, "member@example.com", "existing-credential", "cookie")
		revokedAt := env.now().Add(time.Second)
		accountRepo.account = accountRepo.account.WithStatus("suspended", &revokedAt)

		// Step 2: 既存 bearer は protected route で 403 account-suspended として拒否される。
		response := performJSON(t, env.router, stdhttp.MethodGet, "/api/v1/passkeys", nil, loginBody["accessToken"].(string))
		assertStatus(t, response, stdhttp.StatusForbidden)
		assertFailureCode(t, response, "account-suspended")
	})

	t.Run("[AUTH-BE-S058] Suspended account refresh does not rotate", func(t *testing.T) {
		// Step 1: valid refreshToken 発行後に account を suspended にし、refresh rotation の入口を作る。
		env, accountRepo := newJWTAuthTestEnvWithAccountRepo(t)
		_, refreshToken := productScenarioLogin(t, env.router, "member@example.com", "existing-credential", "cookie")
		revokedAt := env.now().Add(time.Second)
		accountRepo.account = accountRepo.account.WithStatus("suspended", &revokedAt)

		// Step 2: refresh は 403 で拒否され、新しい refresh credential を Set-Cookie しない。
		response := performRefreshWithCookie(t, env.router, refreshToken)
		assertStatus(t, response, stdhttp.StatusForbidden)
		assertFailureCode(t, response, "account-suspended")
		productScenarioAssertNoSetCookie(t, response.Header().Values("Set-Cookie"))
	})

	t.Run("[AUTH-BE-S056] Session revoked timestamp rejects older bearer", func(t *testing.T) {
		// Step 1: session 発行時刻より後の session_revoked_after を account projection に設定する。
		env, accountRepo := newJWTAuthTestEnvWithAccountRepo(t)
		loginBody, _ := productScenarioLogin(t, env.router, "member@example.com", "existing-credential", "cookie")
		revokedAt := env.now()
		accountRepo.account = accountRepo.account.WithStatus("active", &revokedAt)

		// Step 2: 古い bearer は account-suspended 相当の stable failure で拒否される。
		response := performJSON(t, env.router, stdhttp.MethodGet, "/api/v1/passkeys", nil, loginBody["accessToken"].(string))
		assertStatus(t, response, stdhttp.StatusForbidden)
		assertFailureCode(t, response, "account-suspended")
	})

	t.Run("[AUTH-BE-S057] Restored account rejects pre-suspend session and accepts relogin", func(t *testing.T) {
		// Step 1: suspend/restore 後に古い session が残っている状況を session_revoked_after で表現する。
		env, accountRepo := newJWTAuthTestEnvWithAccountRepo(t)
		oldBody, _ := productScenarioLogin(t, env.router, "member@example.com", "existing-credential", "cookie")
		revokedAt := env.now()
		accountRepo.account = accountRepo.account.WithStatus("active", &revokedAt)

		// Step 2: suspend 前 bearer は拒否され、時刻を進めた再ログインの bearer は成功することを検証する。
		oldResponse := performJSON(t, env.router, stdhttp.MethodGet, "/api/v1/passkeys", nil, oldBody["accessToken"].(string))
		assertStatus(t, oldResponse, stdhttp.StatusForbidden)
		env.advance(time.Second)
		newBody, _ := productScenarioLogin(t, env.router, "member@example.com", "existing-credential", "cookie")
		newResponse := performJSON(t, env.router, stdhttp.MethodGet, "/api/v1/passkeys", nil, newBody["accessToken"].(string))
		assertStatus(t, newResponse, stdhttp.StatusOK)
	})

	t.Run("[AUTH-BE-S059] Account suspended failure shape is stable and no-store", func(t *testing.T) {
		// Step 1: protected route で suspended 判定を発生させ、failure response の安定 shape を確認する。
		env, accountRepo := newJWTAuthTestEnvWithAccountRepo(t)
		loginBody, _ := productScenarioLogin(t, env.router, "member@example.com", "existing-credential", "cookie")
		revokedAt := env.now().Add(time.Second)
		accountRepo.account = accountRepo.account.WithStatus("suspended", &revokedAt)

		// Step 2: status、error code、no-store が揃っていることを固定する。
		response := performJSON(t, env.router, stdhttp.MethodGet, "/api/v1/passkeys", nil, loginBody["accessToken"].(string))
		assertStatus(t, response, stdhttp.StatusForbidden)
		assertFailureCode(t, response, "account-suspended")
		assertNoStore(t, response)
	})
}

func TestProductCookieSettingFlowRegistrationOriginAndFetchMetadata(t *testing.T) {
	t.Parallel()

	t.Run("[AUTH-BE-S088] Cookie mode registration finish rejects disallowed Origin without Set-Cookie", func(t *testing.T) {
		// Step 1: recovery registration finish を Cookie mode で呼ぶが、Origin は allowlist 外にする。
		env := newJWTAuthTestEnv(t)
		recoverySession := consumeRecoverySession(t, env)
		response := performJSONWithHeaders(t, env.router, stdhttp.MethodPost, "/api/v1/auth/passkey/register", map[string]any{"recovery_session": recoverySession, "credential": attestationCredentialJSON("new-credential", "")}, "", map[string]string{productOriginHeader: "https://evil.example.com"})

		// Step 2: application の registration state 消費や Cookie 設定へ進まず、fail-close response だけを返すことを確認する。
		assertStatus(t, response, stdhttp.StatusForbidden)
		assertNoStore(t, response)
		productScenarioAssertNoSetCookie(t, response.Header().Values("Set-Cookie"))
	})

	t.Run("[AUTH-BE-S089] Cookie mode registration finish rejects cross-site Fetch Metadata without Set-Cookie", func(t *testing.T) {
		// Step 1: allowed Origin を持つ registration finish でも、Sec-Fetch-Site が cross-site の request を作る。
		env := newJWTAuthTestEnv(t)
		recoverySession := consumeRecoverySession(t, env)
		response := performJSONWithHeaders(t, env.router, stdhttp.MethodPost, "/api/v1/auth/passkey/register", map[string]any{"recovery_session": recoverySession, "credential": attestationCredentialJSON("new-credential", "")}, "", map[string]string{productOriginHeader: productTestAllowedOrigin, productFetchSiteHeader: "cross-site"})

		// Step 2: Fetch Metadata によって Cookie 設定が拒否され、refreshToken は発行されないことを検証する。
		assertStatus(t, response, stdhttp.StatusBadRequest)
		assertNoStore(t, response)
		productScenarioAssertNoSetCookie(t, response.Header().Values("Set-Cookie"))
	})
}

func productScenarioLogin(t *testing.T, router *gin.Engine, identifier string, credentialHandle string, credentialMode string) (map[string]any, string) {
	t.Helper()

	// Step 1: Product passkey challenge を開始し、指定 credential handle で mock WebAuthn assertion を作る。
	challenge := startPasskey(t, router, identifier)
	response := performJSON(t, router, stdhttp.MethodPost, "/api/v1/auth/passkey/finish", map[string]any{"credentialMode": credentialMode, "credential": assertionCredentialJSON(credentialHandle, challengeValue(challenge))}, "")
	assertStatus(t, response, stdhttp.StatusOK)
	assertNoStore(t, response)

	// Step 2: response body を返し、Cookie mode の場合だけ refresh Cookie 値を抽出する。
	body := decodeJSONBody(t, response)
	if credentialMode == "cookie" {
		body["__refreshCookiePath"] = productScenarioCookiePathFromResponse(t, response)
		return body, refreshCookieValueFromResponse(t, response)
	}
	productScenarioAssertNoSetCookie(t, response.Header().Values("Set-Cookie"))
	return body, ""
}

func productScenarioAuthContextID(t *testing.T, body map[string]any) string {
	t.Helper()

	// Step 1: response body の authContextId は refresh path / clear command の selector なので、空値を許さず文字列として取り出す。
	authContextID, ok := body["authContextId"].(string)
	if !ok || strings.TrimSpace(authContextID) == "" {
		t.Fatalf("expected authContextId in response body, got %#v", body["authContextId"])
	}
	return authContextID
}

func productScenarioRefreshPath(t *testing.T, body map[string]any) string {
	t.Helper()

	// Step 1: contract 上の relative refresh path を authContextId から構築し、request と Cookie Path の期待値を統一する。
	return "/api/v1/auth/contexts/" + productScenarioAuthContextID(t, body) + "/refresh"
}

func productScenarioRefreshCookiePath(t *testing.T, _ *gin.Engine, body map[string]any) string {
	t.Helper()

	// Step 1: login response 取得時に helper が保存した Set-Cookie Path を読み、body の authContextId と同じ context を検証する。
	path, ok := body["__refreshCookiePath"].(string)
	if !ok || path == "" {
		t.Fatalf("expected cached refresh Cookie path for body %#v", body)
	}
	return path
}

func productScenarioCookiePathFromResponse(t *testing.T, response interface{ Result() *stdhttp.Response }) string {
	t.Helper()

	// Step 1: Product refresh Cookie の Path 属性だけを取り出し、Cookie value を assertion message に含めない。
	for _, cookie := range response.Result().Cookies() {
		if cookie.Name == productRefreshCookieName {
			return cookie.Path
		}
	}
	t.Fatalf("expected %s Set-Cookie path", productRefreshCookieName)
	return ""
}

func productScenarioRefreshCookieHeader(refreshToken string) map[string]string {
	// Step 1: Cookie header を 1 箇所で作り、refreshToken 値を body や log へ広げず HTTP boundary に閉じ込める。
	return map[string]string{"Cookie": productRefreshCookieName + "=" + refreshToken}
}

func productScenarioAssertNoBodyRefreshToken(t *testing.T, body map[string]any, refreshToken string) {
	t.Helper()

	// Step 1: JSON body に refreshToken field が存在しないことを map と serialized 文字列の両方で確認する。
	if _, exposed := body["refreshToken"]; exposed {
		t.Fatalf("refreshToken must not be exposed in Cookie mode body: %#v", body)
	}
	if refreshToken != "" && strings.Contains(fmt.Sprint(body), refreshToken) {
		t.Fatalf("Cookie refresh token value must not appear in response body: %#v", body)
	}
}

func productScenarioAssertNoSetCookie(t *testing.T, cookies []string) {
	t.Helper()

	// Step 1: fail-close / Bearer mode response が browser Cookie を設定していないことを確認する。
	if len(cookies) != 0 {
		t.Fatalf("expected no Set-Cookie header, got %q", cookies)
	}
}
