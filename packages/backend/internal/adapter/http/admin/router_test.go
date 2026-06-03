package admin

import (
	"context"
	"encoding/json"
	stdhttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	accountsapplication "www-template/packages/backend/internal/application/accounts"
	adminauth "www-template/packages/backend/internal/application/auth"
	operatorsapplication "www-template/packages/backend/internal/application/operators"
	domain "www-template/packages/backend/internal/domain"
	"www-template/packages/backend/internal/platform/config"
)

// OpenSpec 追跡: ADMIN-AUTH-BE-S057 / task 4.40 は TestAdminOperatorContextIsBoundAfterSessionValidation で、operator accessToken 検証後の operator/session context binding を固定する。
// OpenSpec 追跡: ADMIN-AUTH-BE-S058 / task 4.41 は TestAdminAPIRejectsProductBearerToken で、Product bearer を Admin operator session として扱わない境界を固定する。
// OpenSpec 追跡: ADMIN-AUTH-BE-S059 / task 4.42 は TestAdminMutationRejectsDisallowedOriginBeforeSessionValidation で、許可外 Origin を mutation 実行前に 403 へ止める境界を固定する。
// OpenSpec 追跡: ADMIN-AUTH-BE-S060 / task 4.43 は TestAdminMutationValidatesSessionAndPermissionWithoutCSRF で、protected mutation が CSRF token を要求しない境界を固定する。
// OpenSpec 追跡: ADMIN-AUTH-BE-S061 / task 4.44 は TestAdminPreAuthPasskeyStartRequiresOriginButNotSessionCSRF で、pre-auth passkey start が session-bound CSRF なしでも許可済み Origin を要求する境界を固定する。
// OpenSpec 追跡: ADMIN-AUTH-BE-S066 / task 4.49 は assertAdminSecurityHeaders で、Admin API response の no-store と browser security header baseline を固定する。

func TestAdminAPISetsNoStoreAndSecurityHeaders(t *testing.T) {
	t.Parallel()

	// Step 1: 有効な operator session を返す validator を注入し、middleware 通過後の generated 503 response header を検査対象にする。
	validator := &stubOperatorSessionValidator{contextToReturn: validOperatorSessionContext()}
	router := newAdminTestRouter(validator)
	request := httptest.NewRequest(stdhttp.MethodGet, "/api/v1/auth/operator/current", nil)
	request.Header.Set(adminAuthHeader, "Bearer valid-admin-token")
	response := httptest.NewRecorder()

	// Step 2: request を実行し、handler 未実装の 503 でも middleware が header 境界を維持することを確認する。
	router.ServeHTTP(response, request)

	if response.Code != stdhttp.StatusServiceUnavailable {
		t.Fatalf("expected generated handler to fail closed with 503, got %d body=%s", response.Code, response.Body.String())
	}
	assertAdminSecurityHeaders(t, response)
	if validator.currentInput.AccessToken != "valid-admin-token" || validator.currentInput.Permission != "" {
		t.Fatalf("expected read route to validate operator session without mutation permission, got input=%+v", validator.currentInput)
	}
}

func TestAdminProtectedRouteRequiresOperatorSession(t *testing.T) {
	t.Parallel()

	// Step 1: Authorization header が無い request を protected current route へ送り、validator 到達前に拒否されることを確認する。
	validator := &stubOperatorSessionValidator{contextToReturn: validOperatorSessionContext()}
	router := newAdminTestRouter(validator)
	request := httptest.NewRequest(stdhttp.MethodGet, "/api/v1/auth/operator/current", nil)
	response := httptest.NewRecorder()

	// Step 2: Product bearer などを operator session と誤認しない最小境界として、空 bearer は 401 に固定する。
	router.ServeHTTP(response, request)

	if response.Code != stdhttp.StatusUnauthorized {
		t.Fatalf("expected missing operator bearer to return 401, got %d body=%s", response.Code, response.Body.String())
	}
	if validator.calls != 0 {
		t.Fatalf("expected missing bearer to stop before validator, got calls=%d", validator.calls)
	}
	assertAdminSecurityHeaders(t, response)
}

func TestAdminProtectedRouteBearerOnlyScenarioTitles(t *testing.T) {
	t.Parallel()

	t.Run("[ADMIN-AUTH-BE-S057] Admin middleware binds operator context after accessToken validation", func(t *testing.T) {
		// Step 1: protected current route に validator と auth service を注入し、Bearer accessToken 検証後だけ handler へ進む状態を作る。
		validator := &stubOperatorSessionValidator{contextToReturn: validOperatorSessionContext()}
		auth := &stubAdminOperatorAuth{operator: validAdminOperatorDTO()}
		router := newAdminTestRouterWithAuth(validator, auth)
		request := httptest.NewRequest(stdhttp.MethodGet, "/api/v1/auth/operator/current", nil)
		request.Header.Set(adminAuthHeader, "Bearer valid-admin-token")
		response := httptest.NewRecorder()

		// Step 2: request を実行し、validator と handler の両方が同じ bearer だけを operator credential として使うことを確認する。
		router.ServeHTTP(response, request)

		if response.Code != stdhttp.StatusOK {
			t.Fatalf("expected current operator to return 200, got %d body=%s", response.Code, response.Body.String())
		}
		if validator.currentInput.AccessToken != "valid-admin-token" || auth.currentInput.AccessToken != "valid-admin-token" {
			t.Fatalf("expected bearer accessToken to bind operator context, validator=%+v auth=%+v", validator.currentInput, auth.currentInput)
		}
		assertCurrentOperatorResponse(t, response)
	})

	t.Run("[ADMIN-AUTH-BE-S058] Product bearer token is rejected before Admin handler", func(t *testing.T) {
		// Step 1: Product account claim 形状の bearer を Admin signer で署名し、署名ではなく payload 意味の不一致を検査する。
		signer := newAdminBoundarySignVerifier(t)
		productBearer := signedProductBearerForAdminBoundaryTest(t, signer)
		validator := &serviceBackedOperatorSessionValidator{auth: newAdminAuthServiceForBoundaryTest(t, signer)}
		auth := &stubAdminOperatorAuth{operator: validAdminOperatorDTO()}
		router := newAdminTestRouterWithAuth(validator, auth)
		request := httptest.NewRequest(stdhttp.MethodGet, "/api/v1/auth/operator/current", nil)
		request.Header.Set(adminAuthHeader, "Bearer "+productBearer)
		response := httptest.NewRecorder()

		// Step 2: Admin validator が Product bearer を拒否し、handler の CurrentOperator へ到達しないことを確認する。
		router.ServeHTTP(response, request)

		if response.Code != stdhttp.StatusUnauthorized {
			t.Fatalf("expected Product bearer token to be rejected with 401, got %d body=%s", response.Code, response.Body.String())
		}
		if auth.currentInput.AccessToken != "" {
			t.Fatalf("expected rejected Product bearer not to reach Admin handler, got %+v", auth.currentInput)
		}
	})

	t.Run("[ADMIN-AUTH-BE-S080] Admin protected route rejects refresh and legacy cookies without bearer", func(t *testing.T) {
		// Step 1: protected current route に Admin refresh Cookie と legacy access Cookie だけを付け、ambient Cookie 認可にならない状態を作る。
		validator := &stubOperatorSessionValidator{contextToReturn: validOperatorSessionContext()}
		router := newAdminTestRouter(validator)
		request := httptest.NewRequest(stdhttp.MethodGet, "/api/v1/auth/operator/current", nil)
		request.Header.Set("Cookie", adminRefreshCookieName+"=refresh-cookie-value; admin_access_token=legacy-access-cookie")
		response := httptest.NewRecorder()

		// Step 2: bearer 不在として 401 になり、Cookie 値が validator へ渡らないことを確認する。
		router.ServeHTTP(response, request)

		if response.Code != stdhttp.StatusUnauthorized {
			t.Fatalf("expected Cookie-only protected request to return 401, got %d body=%s", response.Code, response.Body.String())
		}
		if validator.calls != 0 {
			t.Fatalf("expected Cookie-only request to stop before validator, got calls=%d", validator.calls)
		}
		assertAdminSecurityHeaders(t, response)
	})

	t.Run("[ADMIN-AUTH-BE-S060] Admin protected mutation validates bearer and permission without CSRF", func(t *testing.T) {
		// Step 1: protected mutation に CSRF header を付けず、許可済み Origin と bearer だけを提示する。
		validator := &stubOperatorSessionValidator{contextToReturn: validOperatorSessionContext()}
		router := newAdminTestRouter(validator)
		request := newAdminJSONRequest(stdhttp.MethodPost, "/api/v1/accounts", `{"email":"customer@example.com"}`)
		request.Header.Set(adminOriginHeader, "https://admin.example.com")
		request.Header.Set(adminAuthHeader, "Bearer valid-admin-token")
		response := httptest.NewRecorder()

		// Step 2: CSRF 欠落では止まらず、permission 付き session validation まで進むことを確認する。
		router.ServeHTTP(response, request)

		if response.Code != stdhttp.StatusServiceUnavailable {
			t.Fatalf("expected generated handler to fail closed with 503, got %d body=%s", response.Code, response.Body.String())
		}
		if validator.mutationInput.AccessToken != "valid-admin-token" || validator.mutationInput.Permission != "accounts:create" {
			t.Fatalf("expected bearer and accounts:create permission without CSRF, got %+v", validator.mutationInput)
		}
	})

	t.Run("[ADMIN-AUTH-BE-S078] Admin protected request does not issue sliding accessToken or Cookie", func(t *testing.T) {
		// Step 1: 同じ bearer で current route を 2 回呼び、通常 protected request が token rotation endpoint にならないことを観測する。
		validator := &stubOperatorSessionValidator{contextToReturn: validOperatorSessionContext()}
		auth := &stubAdminOperatorAuth{operator: validAdminOperatorDTO()}
		router := newAdminTestRouterWithAuth(validator, auth)

		// Step 2: 2 回とも response body に新 accessToken を返さず、Set-Cookie も出さないことを確認する。
		for i := 0; i < 2; i++ {
			request := httptest.NewRequest(stdhttp.MethodGet, "/api/v1/auth/operator/current", nil)
			request.Header.Set(adminAuthHeader, "Bearer valid-admin-token")
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			if response.Code != stdhttp.StatusOK {
				t.Fatalf("expected current operator attempt %d to return 200, got %d body=%s", i+1, response.Code, response.Body.String())
			}
			if strings.Contains(response.Body.String(), "accessToken") {
				t.Fatalf("protected request must not mint a replacement accessToken, body=%s", response.Body.String())
			}
			assertNoSetCookie(t, response)
		}
	})
}

// TestAdminAPIRejectsProductBearerToken は Product accessToken 形状の bearer が Admin API の operator session として受理されないことを検証する。
// Product と Admin は同じ署名 primitive を共有できても claim 意味は別ドメインなので、Admin middleware は operator session validator の拒否を 401 として返す。
func TestAdminAPIRejectsProductBearerToken(t *testing.T) {
	t.Parallel()

	// Step 1: Admin auth service と同じ signer で Product AccountAuth 形状の bearer を署名し、署名不一致ではなく claim domain 不一致を検査する。
	signer := newAdminBoundarySignVerifier(t)
	productBearer := signedProductBearerForAdminBoundaryTest(t, signer)
	validator := &serviceBackedOperatorSessionValidator{auth: newAdminAuthServiceForBoundaryTest(t, signer)}
	auth := &stubAdminOperatorAuth{operator: validAdminOperatorDTO()}
	router := newAdminTestRouterWithPasskeyAuth(validator, auth)
	request := httptest.NewRequest(stdhttp.MethodGet, "/api/v1/auth/operator/current", nil)
	request.Header.Set(adminAuthHeader, "Bearer "+productBearer)
	response := httptest.NewRecorder()

	// Step 2: current operator API を実行し、Product bearer が Admin operator として handler へ到達せず 401 で拒否されることを確認する。
	router.ServeHTTP(response, request)

	if response.Code != stdhttp.StatusUnauthorized {
		t.Fatalf("expected Product bearer token to be rejected by Admin API with 401, got %d body=%s", response.Code, response.Body.String())
	}
	if validator.currentInput.AccessToken != productBearer || validator.currentInput.Permission != "" {
		t.Fatalf("expected Admin validator to receive Product bearer as read-session input, got %+v", validator.currentInput)
	}
	if auth.currentInput.AccessToken != "" {
		t.Fatalf("expected rejected Product bearer not to reach current operator handler, got %+v", auth.currentInput)
	}
	assertAdminSecurityHeaders(t, response)
}

// [ADMIN-CONSOLE-BE-S057] TestAdminRuntimeDoesNotRegisterProductOperations は Admin runtime の router が Product 専用 operation を公開しないことを検証する。
// Product と Admin は同じ `/api/v1/*` path 空間を別 origin / 別 binary で使うため、Admin 側で Product 専用 path が 404 になることを route table 境界の証拠にする。
func TestAdminRuntimeDoesNotRegisterProductOperations(t *testing.T) {
	t.Parallel()

	// Step 1: Admin router に session validator を注入し、Product 専用 route が Admin 登録済み route として扱われた場合は validator 呼び出しや 401/403 で検出できる状態にする。
	validator := &stubOperatorSessionValidator{contextToReturn: validOperatorSessionContext()}
	router := newAdminTestRouter(validator)
	productOnlyRoutes := []struct {
		method string
		path   string
	}{
		{method: stdhttp.MethodGet, path: "/api/v1/status"},
		{method: stdhttp.MethodGet, path: "/api/v1/account/settings"},
		{method: stdhttp.MethodPatch, path: "/api/v1/account/settings"},
		{method: stdhttp.MethodPost, path: "/api/v1/auth/passkey/register/start"},
		{method: stdhttp.MethodPost, path: "/api/v1/auth/passkey/register"},
		{method: stdhttp.MethodPost, path: "/api/v1/auth/recovery"},
		{method: stdhttp.MethodPost, path: "/api/v1/auth/recovery/consume"},
		{method: stdhttp.MethodPost, path: "/api/v1/auth/refresh"},
		{method: stdhttp.MethodGet, path: "/api/v1/passkeys"},
		{method: stdhttp.MethodPost, path: "/api/v1/passkeys/start"},
		{method: stdhttp.MethodPost, path: "/api/v1/passkeys/finish"},
		{method: stdhttp.MethodDelete, path: "/api/v1/passkeys/01ARZ3NDEKTSV4RRFFQ69G5FAV"},
		{method: stdhttp.MethodGet, path: "/api/v1/sessions"},
		{method: stdhttp.MethodDelete, path: "/api/v1/sessions/others"},
		{method: stdhttp.MethodDelete, path: "/api/v1/sessions/01B7X9BN4X2Y3Z4A5B6C7D8E9F"},
	}

	// Step 2: Product 専用 path が Admin router で見つからないことを確認し、Admin binary への Product operation 混入を検出できるようにする。
	for _, route := range productOnlyRoutes {
		request := newAdminJSONRequest(route.method, route.path, `{}`)
		request.Header.Set(adminOriginHeader, "https://admin.example.com")
		request.Header.Set(adminAuthHeader, "Bearer valid-admin-token")
		response := httptest.NewRecorder()

		router.ServeHTTP(response, request)

		if response.Code != stdhttp.StatusNotFound {
			t.Fatalf("expected Admin router to reject Product route %s %s with 404, got %d body=%s", route.method, route.path, response.Code, response.Body.String())
		}
	}

	// Step 3: Product 専用 path が Admin 認証 middleware の登録済み route にも入っていないことを、validator 未呼び出しで確認する。
	if validator.calls != 0 {
		t.Fatalf("expected Product-only routes to skip Admin session validator, got calls=%d", validator.calls)
	}
}

func TestAdminMutationRejectsDisallowedOriginBeforeSessionValidation(t *testing.T) {
	t.Parallel()

	// Step 1: mutation request に許可されない Origin を付け、session validator より前に拒否されることを検証する。
	validator := &stubOperatorSessionValidator{contextToReturn: validOperatorSessionContext()}
	router := newAdminTestRouter(validator)
	request := newAdminJSONRequest(stdhttp.MethodPost, "/api/v1/accounts", `{"email":"customer@example.com"}`)
	request.Header.Set(adminOriginHeader, "https://evil.example.com")
	request.Header.Set(adminAuthHeader, "Bearer valid-admin-token")
	response := httptest.NewRecorder()

	// Step 2: Origin 不一致は 403 とし、account creation handler や session validator へ到達させない。
	router.ServeHTTP(response, request)

	if response.Code != stdhttp.StatusForbidden {
		t.Fatalf("expected disallowed Origin to return 403, got %d body=%s", response.Code, response.Body.String())
	}
	if validator.calls != 0 {
		t.Fatalf("expected disallowed Origin to stop before validator, got calls=%d", validator.calls)
	}
	assertAdminSecurityHeaders(t, response)
}

func TestAdminMutationRejectsCrossSiteFetchMetadataBeforeSessionValidation(t *testing.T) {
	t.Parallel()

	// Step 1: Origin は許可済みでも Fetch Metadata が cross-site の mutation を作り、Bearer 検証前に拒否される境界を固定する。
	validator := &stubOperatorSessionValidator{contextToReturn: validOperatorSessionContext()}
	router := newAdminTestRouter(validator)
	request := newAdminJSONRequest(stdhttp.MethodPost, "/api/v1/accounts", `{"email":"customer@example.com"}`)
	request.Header.Set(adminOriginHeader, "https://admin.example.com")
	request.Header.Set(adminFetchSiteHeader, "cross-site")
	request.Header.Set(adminAuthHeader, "Bearer valid-admin-token")
	response := httptest.NewRecorder()

	// Step 2: Fetch Metadata 不一致は 403 とし、account creation handler や session validator へ到達させない。
	router.ServeHTTP(response, request)

	if response.Code != stdhttp.StatusForbidden {
		t.Fatalf("expected cross-site Fetch Metadata to return 403, got %d body=%s", response.Code, response.Body.String())
	}
	if validator.calls != 0 {
		t.Fatalf("expected cross-site Fetch Metadata to stop before validator, got calls=%d", validator.calls)
	}
	assertAdminSecurityHeaders(t, response)
}

func TestAdminCookieSettingPreAuthOriginFetchMetadataAndCORS(t *testing.T) {
	t.Parallel()

	t.Run("[ADMIN-AUTH-BE-S059] Admin cookie-setting pre-auth rejects disallowed Origin before Set-Cookie", func(t *testing.T) {
		// Step 1: passkey finish は session Cookie を発行し得るため、許可外 Origin で handler 到達前に止まる状態を作る。
		validator := &stubOperatorSessionValidator{contextToReturn: validOperatorSessionContext()}
		auth := &stubAdminOperatorAuth{sessionResult: validAdminAuthSessionResult()}
		verifier := &stubAdminOperatorPasskeyVerifier{credentialHandle: "verified-credential-handle"}
		router := newAdminTestRouterWithAuthAndPasskeyVerifier(validator, auth, verifier)
		request := newAdminJSONRequest(stdhttp.MethodPost, "/api/v1/auth/passkey/finish", `{"requestId":"01B7X9BN4X2Y3Z4A5B6C7D8E9F","credentialMode":"cookie","credential":{"id":"credential-id","rawId":"credential-handle","type":"public-key","response":{"authenticatorData":"auth-data","clientDataJSON":"client-data","signature":"signature"}}}`)
		request.Header.Set(adminOriginHeader, "https://evil.example.com")
		response := httptest.NewRecorder()

		// Step 2: Origin guard が先に 403 を返し、session Cookie・application service・session validator のいずれにも進まないことを検証する。
		router.ServeHTTP(response, request)

		if response.Code != stdhttp.StatusForbidden {
			t.Fatalf("expected disallowed Origin pre-auth finish to return 403, got %d body=%s", response.Code, response.Body.String())
		}
		if auth.finishInput.CredentialHandle != "" || validator.calls != 0 {
			t.Fatalf("expected disallowed Origin to stop before finish/session validation, auth=%+v calls=%d", auth.finishInput, validator.calls)
		}
		if cookies := response.Header().Values("Set-Cookie"); len(cookies) != 0 {
			t.Fatalf("expected disallowed Origin to avoid Set-Cookie, got %q", cookies)
		}
		assertAdminSecurityHeaders(t, response)
	})

	t.Run("[ADMIN-AUTH-BE-S061] Admin pre-auth CORS preflight is limited to the Admin origin", func(t *testing.T) {
		// Step 1: browser の Cookie-setting preflight と同じ Origin / requested method を付け、CORS が auth middleware より前に処理できる request を作る。
		validator := &stubOperatorSessionValidator{contextToReturn: validOperatorSessionContext()}
		router := newAdminTestRouter(validator)
		request := httptest.NewRequest(stdhttp.MethodOptions, "/api/v1/auth/passkey/finish", nil)
		request.Header.Set(adminOriginHeader, "https://admin.example.com")
		request.Header.Set("Access-Control-Request-Method", stdhttp.MethodPost)
		response := httptest.NewRecorder()

		// Step 2: preflight は bearer/session/CSRF を要求せず、Admin origin だけを credentialed CORS として許可することを確認する。
		router.ServeHTTP(response, request)

		if response.Header().Get("Access-Control-Allow-Origin") != "https://admin.example.com" {
			t.Fatalf("expected Admin CORS origin, got %q", response.Header().Get("Access-Control-Allow-Origin"))
		}
		if response.Header().Get("Access-Control-Allow-Credentials") != "true" {
			t.Fatalf("expected credentialed Admin CORS policy, got %q", response.Header().Get("Access-Control-Allow-Credentials"))
		}
		if validator.calls != 0 {
			t.Fatalf("expected CORS preflight to stop before session validator, got calls=%d", validator.calls)
		}
		assertAdminSecurityHeaders(t, response)
	})
}

func TestAdminMutationValidatesSessionAndPermissionWithoutCSRF(t *testing.T) {
	t.Parallel()

	// Step 1: 許可済み Origin / bearer を渡し、middleware が CSRF header を要求せず permission を validator へ渡すことを確認する。
	validator := &stubOperatorSessionValidator{contextToReturn: validOperatorSessionContext()}
	router := newAdminTestRouter(validator)
	request := newAdminJSONRequest(stdhttp.MethodPost, "/api/v1/accounts", `{"email":"customer@example.com"}`)
	request.Header.Set(adminOriginHeader, "https://admin.example.com")
	request.Header.Set(adminAuthHeader, "Bearer valid-admin-token")
	response := httptest.NewRecorder()

	// Step 2: validator 成功後は未実装 handler の 503 まで進むため、副作用実装なしで middleware 境界だけを検証できる。
	router.ServeHTTP(response, request)

	if response.Code != stdhttp.StatusServiceUnavailable {
		t.Fatalf("expected generated handler to fail closed with 503, got %d body=%s", response.Code, response.Body.String())
	}
	if validator.mutationInput.AccessToken != "valid-admin-token" || validator.mutationInput.Permission != "accounts:create" {
		t.Fatalf("expected mutation validator to receive bearer and accounts:create permission without CSRF, got input=%+v", validator.mutationInput)
	}
	assertAdminSecurityHeaders(t, response)
}

func TestCreateAdminAccountMapsTransportDTOToApplicationDTO(t *testing.T) {
	t.Parallel()

	// Step 1: viewer 相当の operator context を返す middleware stub と、成功する account creation use case stub を注入する。
	operatorContext := validOperatorSessionContext()
	operatorContext.OperatorRole = string(domain.OperatorRoleViewer)
	validator := &stubOperatorSessionValidator{contextToReturn: operatorContext}
	creator := &stubAdminAccountCreator{created: validCreatedAdminAccount()}
	router := newAdminTestRouterWithAccountCreator(validator, creator)
	request := newAdminJSONRequest(stdhttp.MethodPost, "/api/v1/accounts", `{"email":"customer@example.com"}`)
	request.Header.Set(adminOriginHeader, "https://admin.example.com")
	request.Header.Set(adminAuthHeader, "Bearer valid-admin-token")
	response := httptest.NewRecorder()

	// Step 2: request を実行し、handler が role を自前判定せず application use case に DTO を渡して 201 response を生成することを確認する。
	router.ServeHTTP(response, request)

	if response.Code != stdhttp.StatusCreated {
		t.Fatalf("expected handler to return 201 from application result, got %d body=%s", response.Code, response.Body.String())
	}
	if creator.calls != 1 {
		t.Fatalf("expected application account creation to be called once, got %d", creator.calls)
	}
	if creator.input.Email != "customer@example.com" || creator.input.OperatorRole != string(domain.OperatorRoleViewer) || creator.input.PasskeyRegistrationState != string(domain.OperatorPasskeyRegistrationRegistered) {
		t.Fatalf("expected handler to pass transport and operator context to application use case, got %+v", creator.input)
	}
	assertCreateAccountResponse(t, response, creator.input.RequestID)
	assertAdminSecurityHeaders(t, response)
}

func TestNewRouterInjectsOperatorSessionValidatorForAccountCreation(t *testing.T) {
	t.Parallel()

	// Step 1: exported Dependencies DTO だけで validator と account creation use case を注入し、production runtime と同じ public composition 経路を検査する。
	validator := &stubOperatorSessionValidator{contextToReturn: validOperatorSessionContext()}
	creator := &stubAdminAccountCreator{created: validCreatedAdminAccount()}
	cfg := config.Config{}
	cfg.Admin.Domain = "https://admin.example.com"
	router := NewRouter(cfg, Dependencies{OperatorSessions: validator, AccountCreation: creator})
	request := newAdminJSONRequest(stdhttp.MethodPost, "/api/v1/accounts", `{"email":"customer@example.com"}`)
	request.Header.Set(adminOriginHeader, "https://admin.example.com")
	request.Header.Set(adminAuthHeader, "Bearer valid-admin-token")
	response := httptest.NewRecorder()

	// Step 2: request を実行し、nil validator による 503 ではなく account creation handler/use case まで到達することを確認する。
	router.ServeHTTP(response, request)

	if response.Code != stdhttp.StatusCreated {
		t.Fatalf("expected exported dependency composition to reach account creation use case, got %d body=%s", response.Code, response.Body.String())
	}
	if validator.mutationInput.Permission != "accounts:create" {
		t.Fatalf("expected account creation route to request accounts:create validation, got %+v", validator.mutationInput)
	}
	if creator.calls != 1 {
		t.Fatalf("expected account creation use case to be called once, got %d", creator.calls)
	}
	assertAdminSecurityHeaders(t, response)
}

func TestCreateAdminAccountMapsApplicationForbidden(t *testing.T) {
	t.Parallel()

	// Step 1: application account creation use case が forbidden を返す状況を作り、handler の責務を error mapping だけに限定する。
	validator := &stubOperatorSessionValidator{contextToReturn: validOperatorSessionContext()}
	creator := &stubAdminAccountCreator{errToReturn: accountsapplication.ErrAccountCreationForbidden}
	router := newAdminTestRouterWithAccountCreator(validator, creator)
	request := newAdminJSONRequest(stdhttp.MethodPost, "/api/v1/accounts", `{"email":"customer@example.com"}`)
	request.Header.Set(adminOriginHeader, "https://admin.example.com")
	request.Header.Set(adminAuthHeader, "Bearer valid-admin-token")
	response := httptest.NewRecorder()

	// Step 2: handler は RBAC を再評価せず、application error を 403 へ写像する。
	router.ServeHTTP(response, request)

	if response.Code != stdhttp.StatusForbidden {
		t.Fatalf("expected application forbidden to return 403, got %d body=%s", response.Code, response.Body.String())
	}
	if creator.calls != 1 {
		t.Fatalf("expected application account creation to be called once, got %d", creator.calls)
	}
	assertAdminSecurityHeaders(t, response)
}

func TestCreateAdminAccountMapsDuplicateEmail(t *testing.T) {
	t.Parallel()

	// Step 1: application account creation use case が duplicate email を返す状況を作り、HTTP 409 のみ handler で決める。
	validator := &stubOperatorSessionValidator{contextToReturn: validOperatorSessionContext()}
	creator := &stubAdminAccountCreator{errToReturn: accountsapplication.ErrAccountDuplicateEmail}
	router := newAdminTestRouterWithAccountCreator(validator, creator)
	request := newAdminJSONRequest(stdhttp.MethodPost, "/api/v1/accounts", `{"email":"customer@example.com"}`)
	request.Header.Set(adminOriginHeader, "https://admin.example.com")
	request.Header.Set(adminAuthHeader, "Bearer valid-admin-token")
	response := httptest.NewRecorder()

	// Step 2: handler は duplicate 判定を自前で行わず、application error を 409 と安定 error code へ写像する。
	router.ServeHTTP(response, request)

	if response.Code != stdhttp.StatusConflict {
		t.Fatalf("expected duplicate email to return 409, got %d body=%s", response.Code, response.Body.String())
	}
	if creator.calls != 1 {
		t.Fatalf("expected application account creation to be called once, got %d", creator.calls)
	}
	assertOperationError(t, response, adminDuplicateEmailMessage)
	assertAdminSecurityHeaders(t, response)
}

// [ADMIN-CONSOLE-BE-S083] 範囲外の limit は Admin backend で 400 になり、repository query は実行されない。
func TestListAdminAccountsRejectsOutOfRangeLimitBeforeRepository(t *testing.T) {
	t.Parallel()

	// Step 1: real application search service と fake repository を使い、HTTP query が use case validation を通る状態を作る。
	validator := &stubOperatorSessionValidator{contextToReturn: validOperatorSessionContext()}
	repository := &stubAccountSearchRepository{}
	searcher, err := accountsapplication.NewAccountSearchService(repository)
	if err != nil {
		t.Fatalf("new admin account search service: %v", err)
	}
	router := newAdminTestRouterWithAccountSearcher(validator, searcher)
	request := httptest.NewRequest(stdhttp.MethodGet, "/api/v1/accounts?limit=0", nil)
	request.Header.Set(adminAuthHeader, "Bearer valid-admin-token")
	response := httptest.NewRecorder()

	// Step 2: request を実行し、middleware 通過後に application validation error が 400 response へ写像されることを確認する。
	router.ServeHTTP(response, request)

	if response.Code != stdhttp.StatusBadRequest {
		t.Fatalf("expected invalid limit to return 400, got %d body=%s", response.Code, response.Body.String())
	}
	if repository.calls != 0 {
		t.Fatalf("repository search calls = %d, want 0", repository.calls)
	}
	assertOperationError(t, response, adminAccountSearchInvalidMessage)
	assertAdminSecurityHeaders(t, response)
}

func TestAdminPreAuthPasskeyStartRequiresOriginButNotSessionCSRF(t *testing.T) {
	t.Parallel()

	// Step 1: passkey start は session 発行前 flow なので、許可済み Origin だけを付け bearer なしで送る。
	validator := &stubOperatorSessionValidator{contextToReturn: validOperatorSessionContext()}
	router := newAdminTestRouter(validator)
	request := newAdminJSONRequest(stdhttp.MethodPost, "/api/v1/auth/passkey/start", `{"identifier":"admin@example.com"}`)
	request.Header.Set(adminOriginHeader, "https://admin.example.com")
	response := httptest.NewRecorder()

	// Step 2: middleware は session bearer 不在では止めず、未実装 handler の fail-close response まで到達させる。
	router.ServeHTTP(response, request)

	if response.Code != stdhttp.StatusServiceUnavailable {
		t.Fatalf("expected pre-auth handler to fail closed with 503, got %d body=%s", response.Code, response.Body.String())
	}
	if validator.calls != 0 {
		t.Fatalf("expected pre-auth route to skip session validator, got calls=%d", validator.calls)
	}
	assertAdminSecurityHeaders(t, response)
}

func TestAdminPreAuthPasskeyStartRejectsMissingOrigin(t *testing.T) {
	t.Parallel()

	// Step 1: session-bound CSRF 例外 route でも unsafe method の Origin は必須にし、cross-site pre-auth request を拒否する。
	validator := &stubOperatorSessionValidator{contextToReturn: validOperatorSessionContext()}
	router := newAdminTestRouter(validator)
	request := newAdminJSONRequest(stdhttp.MethodPost, "/api/v1/auth/passkey/start", `{"identifier":"admin@example.com"}`)
	response := httptest.NewRecorder()

	// Step 2: Origin 不在を 403 にし、passkey challenge handler や validator に到達させない。
	router.ServeHTTP(response, request)

	if response.Code != stdhttp.StatusForbidden {
		t.Fatalf("expected missing Origin to return 403, got %d body=%s", response.Code, response.Body.String())
	}
	if validator.calls != 0 {
		t.Fatalf("expected missing Origin to stop before validator, got calls=%d", validator.calls)
	}
	assertAdminSecurityHeaders(t, response)
}

func TestAdminPasskeyStartMapsTransportDTOToApplicationService(t *testing.T) {
	t.Parallel()

	// Step 1: passkey start 用 auth service stub を注入し、pre-auth route が session validator を使わず application service に委譲する状態を作る。
	validator := &stubOperatorSessionValidator{contextToReturn: validOperatorSessionContext()}
	auth := &stubAdminOperatorAuth{challenge: validAdminPasskeyChallenge()}
	router := newAdminTestRouterWithPasskeyAuth(validator, auth)
	request := newAdminJSONRequest(stdhttp.MethodPost, "/api/v1/auth/passkey/start", `{"identifier":"admin@example.com"}`)
	request.Header.Set(adminOriginHeader, "https://admin.example.com")
	response := httptest.NewRecorder()

	// Step 2: request を実行し、handler が identifier を application DTO に写像して WebAuthn response DTO を返すことを検証する。
	router.ServeHTTP(response, request)

	if response.Code != stdhttp.StatusOK {
		t.Fatalf("expected passkey start to return 200, got %d body=%s", response.Code, response.Body.String())
	}
	if validator.calls != 0 {
		t.Fatalf("expected pre-auth passkey start to skip session validator, got calls=%d", validator.calls)
	}
	if auth.startInput.Identifier != "admin@example.com" {
		t.Fatalf("expected identifier to reach application service, got %+v", auth.startInput)
	}
	assertPasskeyStartResponse(t, response)
	assertAdminSecurityHeaders(t, response)
}

func TestAdminPasskeyFinishIssuesSessionCookieWithoutBodyExposure(t *testing.T) {
	t.Parallel()

	// Step 1: session 発行結果を返す auth service stub を注入し、finish handler の DTO 変換と Cookie 発行を観測する。
	validator := &stubOperatorSessionValidator{contextToReturn: validOperatorSessionContext()}
	auth := &stubAdminOperatorAuth{sessionResult: validAdminAuthSessionResult()}
	verifier := &stubAdminOperatorPasskeyVerifier{credentialHandle: "verified-credential-handle"}
	router := newAdminTestRouterWithAuthAndPasskeyVerifier(validator, auth, verifier)
	request := newAdminJSONRequest(stdhttp.MethodPost, "/api/v1/auth/passkey/finish", `{"requestId":"01B7X9BN4X2Y3Z4A5B6C7D8E9F","credentialMode":"cookie","credential":{"id":"credential-id","rawId":"credential-handle","type":"public-key","response":{"authenticatorData":"auth-data","clientDataJSON":"client-data","signature":"signature"}}}`)
	request.Header.Set(adminOriginHeader, "https://admin.example.com")
	response := httptest.NewRecorder()

	// Step 2: request を実行し、assertion は verifier へ渡し、検証済み credential handle だけが auth service へ渡ることを確認する。
	router.ServeHTTP(response, request)

	if response.Code != stdhttp.StatusOK {
		t.Fatalf("expected passkey finish to return 200, got %d body=%s", response.Code, response.Body.String())
	}
	if verifier.challengeID != "" || verifier.credential.RawID != "credential-handle" {
		t.Fatalf("expected raw assertion to reach passkey verifier, got %+v", verifier)
	}
	if auth.finishInput.ChallengeID != "01B7X9BN4X2Y3Z4A5B6C7D8E9F" || auth.finishInput.CredentialHandle != "verified-credential-handle" {
		t.Fatalf("expected challenge and credential handle to reach application service, got %+v", auth.finishInput)
	}
	assertAdminAuthSessionResponse(t, response, "01B7X9BN4X2Y3Z4A5B6C7D8E9F")
	assertAdminRefreshCookieSet(t, response, "admin-refresh-cookie-value")
	assertAdminSecurityHeaders(t, response)
}

func TestAdminPasskeyFinishRequiresVerifiedCredentialHandle(t *testing.T) {
	t.Parallel()

	// Step 1: auth service は注入するが verifier は注入せず、rawId だけで session 発行に進まない fail-closed 状態を作る。
	validator := &stubOperatorSessionValidator{contextToReturn: validOperatorSessionContext()}
	auth := &stubAdminOperatorAuth{sessionResult: validAdminAuthSessionResult()}
	router := newAdminTestRouterWithPasskeyAuth(validator, auth)
	request := newAdminJSONRequest(stdhttp.MethodPost, "/api/v1/auth/passkey/finish", `{"requestId":"01B7X9BN4X2Y3Z4A5B6C7D8E9F","credentialMode":"cookie","credential":{"id":"credential-id","rawId":"credential-handle","type":"public-key","response":{"authenticatorData":"auth-data","clientDataJSON":"client-data","signature":"signature"}}}`)
	request.Header.Set(adminOriginHeader, "https://admin.example.com")
	response := httptest.NewRecorder()

	// Step 2: request を実行し、verifier 不在では auth service に credential handle を渡さず 503 で停止することを検証する。
	router.ServeHTTP(response, request)

	if response.Code != stdhttp.StatusServiceUnavailable {
		t.Fatalf("expected passkey finish without verifier to return 503, got %d body=%s", response.Code, response.Body.String())
	}
	if auth.finishInput.CredentialHandle != "" {
		t.Fatalf("expected raw credential handle not to reach auth service without verifier, got %+v", auth.finishInput)
	}
	assertAdminSecurityHeaders(t, response)
}

func TestAdminIssuanceSubjectPayloadScenarioTitles(t *testing.T) {
	t.Parallel()

	t.Run("[AUTH-BE-S094] Admin adapter returns explicit operator subject payload only", func(t *testing.T) {
		// Step 1: Admin Cookie mode login を実行し、HTTP adapter が Admin generated response の operator field を明示 subject として出す経路を観測する。
		validator := &stubOperatorSessionValidator{contextToReturn: validOperatorSessionContext()}
		auth := &stubAdminOperatorAuth{sessionResult: validAdminAuthSessionResult()}
		verifier := &stubAdminOperatorPasskeyVerifier{credentialHandle: "verified-credential-handle"}
		router := newAdminTestRouterWithAuthAndPasskeyVerifier(validator, auth, verifier)
		request := newAdminJSONRequest(stdhttp.MethodPost, "/api/v1/auth/passkey/finish", `{"requestId":"01B7X9BN4X2Y3Z4A5B6C7D8E9F","credentialMode":"cookie","credential":{"id":"credential-id","rawId":"credential-handle","type":"public-key","response":{"authenticatorData":"auth-data","clientDataJSON":"client-data","signature":"signature"}}}`)
		request.Header.Set(adminOriginHeader, "https://admin.example.com")
		response := httptest.NewRecorder()

		// Step 2: response は operator subject payload を含み、Product account payload を混入しないことを境界 evidence として固定する。
		router.ServeHTTP(response, request)

		if response.Code != stdhttp.StatusOK {
			t.Fatalf("expected admin login to return 200, got %d body=%s", response.Code, response.Body.String())
		}
		assertAdminAuthSessionResponse(t, response, "01B7X9BN4X2Y3Z4A5B6C7D8E9F")
	})

}

func TestAdminIssuanceCredentialModeScenarios(t *testing.T) {
	t.Parallel()

	t.Run("[ADMIN-AUTH-BE-S074] Admin Cookie mode login returns accessToken body and path-scoped refresh Cookie", func(t *testing.T) {
		// Step 1: passkey finish を cookie mode で実行し、body secret と Cookie 属性の分離を観測する。
		validator := &stubOperatorSessionValidator{contextToReturn: validOperatorSessionContext()}
		auth := &stubAdminOperatorAuth{sessionResult: validAdminAuthSessionResult()}
		verifier := &stubAdminOperatorPasskeyVerifier{credentialHandle: "verified-credential-handle"}
		router := newAdminTestRouterWithAuthAndPasskeyVerifier(validator, auth, verifier)
		request := newAdminJSONRequest(stdhttp.MethodPost, "/api/v1/auth/passkey/finish", `{"requestId":"01B7X9BN4X2Y3Z4A5B6C7D8E9F","credentialMode":"cookie","credential":{"id":"credential-id","rawId":"credential-handle","type":"public-key","response":{"authenticatorData":"auth-data","clientDataJSON":"client-data","signature":"signature"}}}`)
		request.Header.Set(adminOriginHeader, "https://admin.example.com")
		response := httptest.NewRecorder()

		// Step 2: Cookie mode は refreshToken を body に出さず、Admin context refresh path に限定した Set-Cookie を出す。
		router.ServeHTTP(response, request)

		if response.Code != stdhttp.StatusOK {
			t.Fatalf("expected cookie mode login to return 200, got %d body=%s", response.Code, response.Body.String())
		}
		assertAdminAuthSessionResponse(t, response, "01B7X9BN4X2Y3Z4A5B6C7D8E9F")
		assertAdminRefreshCookieSet(t, response, "admin-refresh-cookie-value")
	})

	t.Run("[ADMIN-AUTH-BE-S079] Admin Bearer mode login returns body refreshToken without Cookie", func(t *testing.T) {
		// Step 1: passkey finish を bearer mode で実行し、automation client 用 body token と Cookie 不発行を観測する。
		validator := &stubOperatorSessionValidator{contextToReturn: validOperatorSessionContext()}
		auth := &stubAdminOperatorAuth{sessionResult: validAdminAuthSessionResult()}
		verifier := &stubAdminOperatorPasskeyVerifier{credentialHandle: "verified-credential-handle"}
		router := newAdminTestRouterWithAuthAndPasskeyVerifier(validator, auth, verifier)
		request := newAdminJSONRequest(stdhttp.MethodPost, "/api/v1/auth/passkey/finish", `{"requestId":"01B7X9BN4X2Y3Z4A5B6C7D8E9F","credentialMode":"bearer","credential":{"id":"credential-id","rawId":"credential-handle","type":"public-key","response":{"authenticatorData":"auth-data","clientDataJSON":"client-data","signature":"signature"}}}`)
		request.Header.Set(adminOriginHeader, "https://admin.example.com")
		response := httptest.NewRecorder()

		// Step 2: Bearer mode は refreshToken を body に返し、Admin refresh Cookie を設定しない。
		router.ServeHTTP(response, request)

		if response.Code != stdhttp.StatusOK {
			t.Fatalf("expected bearer mode login to return 200, got %d body=%s", response.Code, response.Body.String())
		}
		assertAdminBearerAuthSessionResponse(t, response, "01B7X9BN4X2Y3Z4A5B6C7D8E9F")
	})

	t.Run("[ADMIN-AUTH-BE-S064] Admin setup finish Cookie mode keeps Secure Lax context Path", func(t *testing.T) {
		// Step 1: initial setup finish を cookie mode で実行し、login 以外の issuance flow でも同じ Cookie 属性を使うことを検証する。
		validator := &stubOperatorSessionValidator{contextToReturn: validOperatorSessionContext()}
		setupper := &stubAdminOperatorSetupper{sessionResult: validAdminAuthSessionResult()}
		router := newAdminTestRouterWithSetupper(validator, setupper)
		request := newAdminJSONRequest(stdhttp.MethodPost, "/api/v1/auth/setup/finish", `{"email":"admin@example.com","displayName":"Admin","bootstrapSecret":"bootstrap-secret","requestId":"01B7X9BN4X2Y3Z4A5B6C7D8E9F","credentialMode":"cookie","credential":{"id":"credential-id","rawId":"credential-handle","type":"public-key","response":{"clientDataJSON":"client-data","attestationObject":"attestation"}}}`)
		request.Header.Set(adminOriginHeader, "https://admin.example.com")
		response := httptest.NewRecorder()

		// Step 2: setup 完了時も refreshToken body exposure なしで Secure/Lax/context Path の Cookie だけを発行する。
		router.ServeHTTP(response, request)

		if response.Code != stdhttp.StatusOK {
			t.Fatalf("expected setup finish to return 200, got %d body=%s", response.Code, response.Body.String())
		}
		if setupper.finishInitialInput.BootstrapSecret != "bootstrap-secret" {
			t.Fatalf("expected setup finish input to reach application service, got %+v", setupper.finishInitialInput)
		}
		assertAdminAuthSessionResponse(t, response, "01B7X9BN4X2Y3Z4A5B6C7D8E9F")
		assertAdminRefreshCookieSet(t, response, "admin-refresh-cookie-value")
	})

	t.Run("[ADMIN-AUTH-BE-S079] Admin operator-setup Bearer mode returns body refreshToken without Cookie", func(t *testing.T) {
		// Step 1: operator setup finish を bearer mode で実行し、setup flow でも automation client 用 response を選べることを検証する。
		validator := &stubOperatorSessionValidator{contextToReturn: validOperatorSessionContext()}
		setupper := &stubAdminOperatorSetupper{sessionResult: validAdminAuthSessionResult()}
		router := newAdminTestRouterWithSetupper(validator, setupper)
		request := newAdminJSONRequest(stdhttp.MethodPost, "/api/v1/auth/operator-setup/finish", `{"setupToken":"setup-token","requestId":"01B7X9BN4X2Y3Z4A5B6C7D8E9F","credentialMode":"bearer","credential":{"id":"credential-id","rawId":"credential-handle","type":"public-key","response":{"clientDataJSON":"client-data","attestationObject":"attestation"}}}`)
		request.Header.Set(adminOriginHeader, "https://admin.example.com")
		response := httptest.NewRecorder()

		// Step 2: Bearer mode operator setup は refreshToken を body に返し、Set-Cookie を出さない。
		router.ServeHTTP(response, request)

		if response.Code != stdhttp.StatusOK {
			t.Fatalf("expected operator setup bearer finish to return 200, got %d body=%s", response.Code, response.Body.String())
		}
		if setupper.finishOperatorInput.SetupToken != "setup-token" {
			t.Fatalf("expected operator setup input to reach application service, got %+v", setupper.finishOperatorInput)
		}
		assertAdminBearerAuthSessionResponse(t, response, "01B7X9BN4X2Y3Z4A5B6C7D8E9F")
	})
}

func TestAdminIssuanceRejectsInvalidRefreshContextCommandScenario(t *testing.T) {
	t.Parallel()

	unsafeCommands := []struct {
		name   string
		mutate func(*adminauth.OperatorRefreshCookieCommand)
	}{
		{name: "missing auth context", mutate: func(command *adminauth.OperatorRefreshCookieCommand) { command.AuthContextID = "" }},
		{name: "invalid auth context", mutate: func(command *adminauth.OperatorRefreshCookieCommand) { command.AuthContextID = "not-a-ulid" }},
	}

	for _, tt := range unsafeCommands {
		tt := tt
		t.Run("[ADMIN-AUTH-BE-S065] Admin Cookie mode issuance rejects invalid refresh context command "+tt.name, func(t *testing.T) {
			// Step 1: application service が Cookie 属性ではなく不正な auth context selector を返す状況を作り、adapter が fail-close できるかを検査する。
			validator := &stubOperatorSessionValidator{contextToReturn: validOperatorSessionContext()}
			unsafeResult := validAdminAuthSessionResult()
			tt.mutate(&unsafeResult.RefreshCookie)
			auth := &stubAdminOperatorAuth{sessionResult: unsafeResult}
			verifier := &stubAdminOperatorPasskeyVerifier{credentialHandle: "verified-credential-handle"}
			router := newAdminTestRouterWithAuthAndPasskeyVerifier(validator, auth, verifier)
			request := newAdminJSONRequest(stdhttp.MethodPost, "/api/v1/auth/passkey/finish", `{"requestId":"01B7X9BN4X2Y3Z4A5B6C7D8E9F","credentialMode":"cookie","credential":{"id":"credential-id","rawId":"credential-handle","type":"public-key","response":{"authenticatorData":"auth-data","clientDataJSON":"client-data","signature":"signature"}}}`)
			request.Header.Set(adminOriginHeader, "https://admin.example.com")
			response := httptest.NewRecorder()

			// Step 2: 不正な context selector では Cookie Path を構築せず 503 に倒し、Admin browser session を半端に返さないことを確認する。
			router.ServeHTTP(response, request)

			if response.Code != stdhttp.StatusServiceUnavailable {
				t.Fatalf("expected invalid refresh context command to fail closed with 503, got %d body=%s", response.Code, response.Body.String())
			}
			assertNoSetCookie(t, response)
		})
	}
}

func TestGetCurrentAdminOperatorCallsApplicationService(t *testing.T) {
	t.Parallel()

	// Step 1: protected current route に session validator と auth service stub を注入し、handler が application service へ current operator 取得を委譲できる状態にする。
	validator := &stubOperatorSessionValidator{contextToReturn: validOperatorSessionContext()}
	auth := &stubAdminOperatorAuth{operator: validAdminOperatorDTO()}
	router := newAdminTestRouterWithAuth(validator, auth)
	request := httptest.NewRequest(stdhttp.MethodGet, "/api/v1/auth/operator/current", nil)
	request.Header.Set(adminAuthHeader, "Bearer valid-admin-token")
	response := httptest.NewRecorder()

	// Step 2: request を実行し、middleware 通過後に accessToken だけが application service へ渡ることを検証する。
	router.ServeHTTP(response, request)

	if response.Code != stdhttp.StatusOK {
		t.Fatalf("expected current operator to return 200, got %d body=%s", response.Code, response.Body.String())
	}
	if auth.currentInput.AccessToken != "valid-admin-token" {
		t.Fatalf("expected current operator access token, got %+v", auth.currentInput)
	}
	assertCurrentOperatorResponse(t, response)
	assertAdminSecurityHeaders(t, response)
}

func TestRefreshAdminOperatorSessionIssuesNewCSRFAndCookie(t *testing.T) {
	t.Parallel()

	// Step 1: refresh route は pre-auth 扱いのため session validator なしで Cookie から application service へ rotation 入力を渡す。
	validator := &stubOperatorSessionValidator{contextToReturn: validOperatorSessionContext()}
	auth := &stubAdminOperatorAuth{sessionResult: validAdminAuthSessionResult()}
	router := newAdminTestRouterWithAuth(validator, auth)
	request := newAdminJSONRequest(stdhttp.MethodPost, "/api/v1/auth/contexts/01B7X9BN4X2Y3Z4A5B6C7D8E9F/refresh", `{}`)
	request.Header.Set(adminOriginHeader, "https://admin.example.com")
	request.Header.Set("Cookie", adminRefreshCookieName+"=old-refresh-cookie-value")
	response := httptest.NewRecorder()

	// Step 2: request を実行し、古い refresh Cookie が application service に渡り、新しい CSRF と Cookie が返ることを確認する。
	router.ServeHTTP(response, request)

	if response.Code != stdhttp.StatusOK {
		t.Fatalf("expected refresh to return 200, got %d body=%s", response.Code, response.Body.String())
	}
	if auth.refreshInput.AuthContextID != "01B7X9BN4X2Y3Z4A5B6C7D8E9F" || auth.refreshInput.RefreshTokenValue != "old-refresh-cookie-value" || validator.calls != 0 {
		t.Fatalf("expected refresh cookie input without session validator, input=%+v calls=%d", auth.refreshInput, validator.calls)
	}
	assertAdminAuthSessionResponse(t, response, "")
	assertAdminRefreshCookieSet(t, response, "admin-refresh-cookie-value")
	assertAdminSecurityHeaders(t, response)
}

func TestRefreshAdminOperatorSessionRejectsMissingOriginBeforeCookieRotation(t *testing.T) {
	t.Parallel()

	// Step 1: context refresh は session-bound CSRF 例外だが Cookie rotation を伴う unsafe method なので、Origin が無い request を作る。
	validator := &stubOperatorSessionValidator{contextToReturn: validOperatorSessionContext()}
	auth := &stubAdminOperatorAuth{sessionResult: validAdminAuthSessionResult()}
	router := newAdminTestRouterWithAuth(validator, auth)
	request := newAdminJSONRequest(stdhttp.MethodPost, "/api/v1/auth/contexts/01B7X9BN4X2Y3Z4A5B6C7D8E9F/refresh", `{}`)
	request.Header.Set("Cookie", adminRefreshCookieName+"=old-refresh-cookie-value")
	response := httptest.NewRecorder()

	// Step 2: middleware が handler より前に 403 へ止め、refresh Cookie を application service へ渡さないことを検証する。
	router.ServeHTTP(response, request)

	if response.Code != stdhttp.StatusForbidden {
		t.Fatalf("expected missing Origin refresh to return 403, got %d body=%s", response.Code, response.Body.String())
	}
	if auth.refreshInput.RefreshTokenValue != "" || validator.calls != 0 {
		t.Fatalf("expected missing Origin to stop before refresh/service validation, input=%+v calls=%d", auth.refreshInput, validator.calls)
	}
	assertAdminSecurityHeaders(t, response)
}

func TestRefreshAdminOperatorSessionRejectsDisallowedOriginBeforeCookieRotation(t *testing.T) {
	t.Parallel()

	// Step 1: 許可外 Origin 付きの context refresh を作り、path parameter route でも Origin guard が適用されることを確認する。
	validator := &stubOperatorSessionValidator{contextToReturn: validOperatorSessionContext()}
	auth := &stubAdminOperatorAuth{sessionResult: validAdminAuthSessionResult()}
	router := newAdminTestRouterWithAuth(validator, auth)
	request := newAdminJSONRequest(stdhttp.MethodPost, "/api/v1/auth/contexts/01B7X9BN4X2Y3Z4A5B6C7D8E9F/refresh", `{}`)
	request.Header.Set(adminOriginHeader, "https://evil.example.com")
	request.Header.Set("Cookie", adminRefreshCookieName+"=old-refresh-cookie-value")
	response := httptest.NewRecorder()

	// Step 2: 許可外 Origin は Cookie の有無に関係なく 403 とし、rotation service を呼ばないことを検証する。
	router.ServeHTTP(response, request)

	if response.Code != stdhttp.StatusForbidden {
		t.Fatalf("expected disallowed Origin refresh to return 403, got %d body=%s", response.Code, response.Body.String())
	}
	if auth.refreshInput.RefreshTokenValue != "" || validator.calls != 0 {
		t.Fatalf("expected disallowed Origin to stop before refresh/service validation, input=%+v calls=%d", auth.refreshInput, validator.calls)
	}
	assertAdminSecurityHeaders(t, response)
}

func TestAdminContextRefreshCredentialScenarios(t *testing.T) {
	t.Parallel()

	t.Run("[ADMIN-AUTH-BE-S081] Admin Cookie refresh rotates path-scoped refresh Cookie", func(t *testing.T) {
		// Step 1: Cookie mode refresh に path authContextId と HttpOnly Cookie だけを提示する。
		validator := &stubOperatorSessionValidator{contextToReturn: validOperatorSessionContext()}
		auth := &stubAdminOperatorAuth{sessionResult: validAdminAuthSessionResult()}
		router := newAdminTestRouterWithAuth(validator, auth)
		request := newAdminJSONRequest(stdhttp.MethodPost, "/api/v1/auth/contexts/01B7X9BN4X2Y3Z4A5B6C7D8E9F/refresh", `{}`)
		request.Header.Set(adminOriginHeader, "https://admin.example.com")
		request.Header.Set("Cookie", adminRefreshCookieName+"=old-refresh-cookie-value")
		response := httptest.NewRecorder()

		// Step 2: handler は Cookie credential と path context を application service へ渡し、body refreshToken を返さない。
		router.ServeHTTP(response, request)

		if response.Code != stdhttp.StatusOK {
			t.Fatalf("expected cookie refresh to return 200, got %d body=%s", response.Code, response.Body.String())
		}
		if auth.refreshInput.AuthContextID != "01B7X9BN4X2Y3Z4A5B6C7D8E9F" || auth.refreshInput.RefreshTokenValue != "old-refresh-cookie-value" {
			t.Fatalf("expected cookie refresh credential input, got %+v", auth.refreshInput)
		}
		assertAdminAuthSessionResponse(t, response, "")
		assertAdminRefreshCookieSet(t, response, "admin-refresh-cookie-value")
	})

	t.Run("[ADMIN-AUTH-BE-S084] Admin Bearer refresh rotates body refreshToken without Cookie", func(t *testing.T) {
		// Step 1: Bearer mode refresh に body refreshToken だけを提示し、Cookie を送らない。
		validator := &stubOperatorSessionValidator{contextToReturn: validOperatorSessionContext()}
		auth := &stubAdminOperatorAuth{sessionResult: validAdminAuthSessionResult()}
		router := newAdminTestRouterWithAuth(validator, auth)
		request := newAdminJSONRequest(stdhttp.MethodPost, "/api/v1/auth/contexts/01B7X9BN4X2Y3Z4A5B6C7D8E9F/refresh", `{"credentialMode":"bearer","refreshToken":"old-bearer-refresh-token"}`)
		request.Header.Set(adminOriginHeader, "https://admin.example.com")
		response := httptest.NewRecorder()

		// Step 2: handler は body refreshToken を application service へ渡し、新しい refreshToken を body に返す。
		router.ServeHTTP(response, request)

		if response.Code != stdhttp.StatusOK {
			t.Fatalf("expected bearer refresh to return 200, got %d body=%s", response.Code, response.Body.String())
		}
		if auth.refreshInput.RefreshTokenValue != "old-bearer-refresh-token" {
			t.Fatalf("expected bearer refresh token input, got %+v", auth.refreshInput)
		}
		assertAdminBearerAuthSessionResponse(t, response, "")
	})

	t.Run("[ADMIN-AUTH-BE-S082] Admin Bearer refresh rejects Authorization header", func(t *testing.T) {
		// Step 1: body refreshToken と Authorization header を同時に付け、accessToken を refresh credential として使わせない。
		validator := &stubOperatorSessionValidator{contextToReturn: validOperatorSessionContext()}
		auth := &stubAdminOperatorAuth{sessionResult: validAdminAuthSessionResult()}
		router := newAdminTestRouterWithAuth(validator, auth)
		request := newAdminJSONRequest(stdhttp.MethodPost, "/api/v1/auth/contexts/01B7X9BN4X2Y3Z4A5B6C7D8E9F/refresh", `{"credentialMode":"bearer","refreshToken":"old-bearer-refresh-token"}`)
		request.Header.Set(adminOriginHeader, "https://admin.example.com")
		request.Header.Set(adminAuthHeader, "Bearer access-token-must-not-refresh")
		response := httptest.NewRecorder()

		// Step 2: Authorization header がある refresh request は service 到達前に 401 へ止まる。
		router.ServeHTTP(response, request)

		if response.Code != stdhttp.StatusUnauthorized {
			t.Fatalf("expected Authorization refresh to return 401, got %d body=%s", response.Code, response.Body.String())
		}
		if auth.refreshInput.RefreshTokenValue != "" || validator.calls != 0 {
			t.Fatalf("expected Authorization refresh to stop before service/validator, input=%+v calls=%d", auth.refreshInput, validator.calls)
		}
		assertNoSetCookie(t, response)
	})

	t.Run("[ADMIN-AUTH-BE-S083] Admin refresh rejects Cookie and body refreshToken ambiguity", func(t *testing.T) {
		// Step 1: Cookie mode と Bearer mode の credential を同時に送信し、曖昧な rotation request を作る。
		validator := &stubOperatorSessionValidator{contextToReturn: validOperatorSessionContext()}
		auth := &stubAdminOperatorAuth{sessionResult: validAdminAuthSessionResult()}
		router := newAdminTestRouterWithAuth(validator, auth)
		request := newAdminJSONRequest(stdhttp.MethodPost, "/api/v1/auth/contexts/01B7X9BN4X2Y3Z4A5B6C7D8E9F/refresh", `{"credentialMode":"bearer","refreshToken":"old-bearer-refresh-token"}`)
		request.Header.Set(adminOriginHeader, "https://admin.example.com")
		request.Header.Set("Cookie", adminRefreshCookieName+"=old-refresh-cookie-value")
		response := httptest.NewRecorder()

		// Step 2: exactly-one 違反は service 到達前に 401 へ止まり、新 credential を発行しない。
		router.ServeHTTP(response, request)

		if response.Code != stdhttp.StatusUnauthorized {
			t.Fatalf("expected ambiguous refresh to return 401, got %d body=%s", response.Code, response.Body.String())
		}
		if auth.refreshInput.RefreshTokenValue != "" || validator.calls != 0 {
			t.Fatalf("expected ambiguous refresh to stop before service/validator, input=%+v calls=%d", auth.refreshInput, validator.calls)
		}
		assertNoSetCookie(t, response)
	})

	t.Run("[ADMIN-AUTH-BE-S085] Admin invalid refresh token does not issue a new credential", func(t *testing.T) {
		// Step 1: application service が invalid/reuse refresh を拒否する状況を作り、handler が Set-Cookie を出さないことを確認する。
		validator := &stubOperatorSessionValidator{contextToReturn: validOperatorSessionContext()}
		auth := &stubAdminOperatorAuth{sessionResult: validAdminAuthSessionResult(), errToReturn: adminauth.ErrOperatorAuthUnauthenticated}
		router := newAdminTestRouterWithAuth(validator, auth)
		request := newAdminJSONRequest(stdhttp.MethodPost, "/api/v1/auth/contexts/01B7X9BN4X2Y3Z4A5B6C7D8E9F/refresh", `{}`)
		request.Header.Set(adminOriginHeader, "https://admin.example.com")
		request.Header.Set("Cookie", adminRefreshCookieName+"=replayed-refresh-cookie-value")
		response := httptest.NewRecorder()

		// Step 2: invalid/reuse refresh は 401 に畳まれ、新しい accessToken / Cookie を発行しない。
		router.ServeHTTP(response, request)

		if response.Code != stdhttp.StatusUnauthorized {
			t.Fatalf("expected invalid refresh to return 401, got %d body=%s", response.Code, response.Body.String())
		}
		assertNoSetCookie(t, response)
	})
}

func TestLogoutAdminOperatorClearsRefreshCookie(t *testing.T) {
	t.Parallel()

	// Step 1: logout 用 auth service stub と mutation validator を注入し、Bearer/RBAC 検証後に revoke service へ委譲する経路を作る。
	validator := &stubOperatorSessionValidator{contextToReturn: validOperatorSessionContext()}
	auth := &stubAdminOperatorAuth{logoutCookie: adminauth.OperatorRefreshCookieCommand{AuthContextID: "01B7X9BN4X2Y3Z4A5B6C7D8E9F", Clear: true}}
	router := newAdminTestRouterWithAuth(validator, auth)
	request := newAdminJSONRequest(stdhttp.MethodPost, "/api/v1/auth/operator/logout", `{}`)
	request.Header.Set(adminOriginHeader, "https://admin.example.com")
	request.Header.Set(adminAuthHeader, "Bearer valid-admin-token")
	response := httptest.NewRecorder()

	// Step 2: request を実行し、handler が session revoke を application service へ委譲し、Cookie 削除 header を返すことを検証する。
	router.ServeHTTP(response, request)

	if response.Code != stdhttp.StatusOK {
		t.Fatalf("expected logout to return 200, got %d body=%s", response.Code, response.Body.String())
	}
	if auth.logoutInput.AccessToken != "valid-admin-token" || validator.mutationInput.Permission != "operators:logout" {
		t.Fatalf("expected logout access token and operators:logout validation, input=%+v validator=%+v", auth.logoutInput, validator.mutationInput)
	}
	assertAdminRefreshCookieCleared(t, response)
	assertLogoutResponse(t, response)
	assertAdminSecurityHeaders(t, response)
}

func TestAdminConsoleRejectsProductBearerScenario(t *testing.T) {
	t.Parallel()

	t.Run("[ADMIN-CONSOLE-BE-S058] Product bearer token cannot authorize Admin account search", func(t *testing.T) {
		// Step 1: Product account claim 形状の bearer と Admin account search use case を用意し、Admin 管理 API が Product token で開かないことを検査する。
		signer := newAdminBoundarySignVerifier(t)
		productBearer := signedProductBearerForAdminBoundaryTest(t, signer)
		validator := &serviceBackedOperatorSessionValidator{auth: newAdminAuthServiceForBoundaryTest(t, signer)}
		repository := &stubAccountSearchRepository{}
		searcher, err := accountsapplication.NewAccountSearchService(repository)
		if err != nil {
			t.Fatalf("new account search service: %v", err)
		}
		router := newAdminTestRouterWithAccountSearcher(validator, searcher)
		request := httptest.NewRequest(stdhttp.MethodGet, "/api/v1/accounts", nil)
		request.Header.Set(adminAuthHeader, "Bearer "+productBearer)
		response := httptest.NewRecorder()

		// Step 2: session validation で 401 になり、account search repository へ到達しないことを確認する。
		router.ServeHTTP(response, request)

		if response.Code != stdhttp.StatusUnauthorized {
			t.Fatalf("expected Product bearer account search to return 401, got %d body=%s", response.Code, response.Body.String())
		}
		if repository.calls != 0 {
			t.Fatalf("expected rejected Product bearer not to reach account search, got calls=%d", repository.calls)
		}
	})
}

func TestAdminConsoleRejectsRefreshCookieMutationScenario(t *testing.T) {
	t.Parallel()

	t.Run("[ADMIN-CONSOLE-BE-S094] Admin refresh Cookie cannot authorize account mutation", func(t *testing.T) {
		// Step 1: account mutation に refresh Cookie だけを提示し、operator accessToken 不在の境界を作る。
		validator := &stubOperatorSessionValidator{contextToReturn: validOperatorSessionContext()}
		creator := &stubAdminAccountCreator{created: validCreatedAdminAccount()}
		router := newAdminTestRouterWithAccountCreator(validator, creator)
		request := newAdminJSONRequest(stdhttp.MethodPost, "/api/v1/accounts", `{"email":"customer@example.com"}`)
		request.Header.Set(adminOriginHeader, "https://admin.example.com")
		request.Header.Set("Cookie", adminRefreshCookieName+"=refresh-cookie-value")
		response := httptest.NewRecorder()

		// Step 2: refresh Cookie は認可材料にならず、validator と account creation use case の前で 401 になることを確認する。
		router.ServeHTTP(response, request)

		if response.Code != stdhttp.StatusUnauthorized {
			t.Fatalf("expected refresh-cookie-only mutation to return 401, got %d body=%s", response.Code, response.Body.String())
		}
		if validator.calls != 0 || creator.calls != 0 {
			t.Fatalf("expected refresh-cookie-only mutation to stop before validator/use case, validator=%d creator=%d", validator.calls, creator.calls)
		}
	})
}

func TestAdminConsoleRBACInputScenario(t *testing.T) {
	t.Parallel()

	t.Run("[ADMIN-CONSOLE-BE-S095] RBAC input is built from validated operator session only", func(t *testing.T) {
		// Step 1: 有効な bearer に加えて Cookie と X-Auth-Context-Id を混ぜ、handler 入力が検証済み operator context だけから作られるかを確認する。
		validator := &stubOperatorSessionValidator{contextToReturn: validOperatorSessionContext()}
		creator := &stubAdminAccountCreator{created: validCreatedAdminAccount()}
		router := newAdminTestRouterWithAccountCreator(validator, creator)
		request := newAdminJSONRequest(stdhttp.MethodPost, "/api/v1/accounts", `{"email":"customer@example.com"}`)
		request.Header.Set(adminOriginHeader, "https://admin.example.com")
		request.Header.Set(adminAuthHeader, "Bearer valid-admin-token")
		request.Header.Set("Cookie", adminRefreshCookieName+"=ignored-refresh-cookie")
		request.Header.Set("X-Auth-Context-Id", "ignored-context")
		response := httptest.NewRecorder()

		// Step 2: account creation input と validator input が operator/session/permission だけで構成されることを確認する。
		router.ServeHTTP(response, request)

		if response.Code != stdhttp.StatusCreated {
			t.Fatalf("expected account creation to return 201, got %d body=%s", response.Code, response.Body.String())
		}
		if validator.mutationInput.AccessToken != "valid-admin-token" || validator.mutationInput.Permission != "accounts:create" {
			t.Fatalf("expected validator to use bearer and permission only, got %+v", validator.mutationInput)
		}
		if creator.input.OperatorID != "01ARZ3NDEKTSV4RRFFQ69G5FAV" || creator.input.OperatorRole != "admin" || creator.input.Email != "customer@example.com" {
			t.Fatalf("expected account creation input from validated operator context and body only, got %+v", creator.input)
		}
	})
}

func TestAdminLogoutScenarioTitle(t *testing.T) {
	t.Parallel()

	t.Run("[ADMIN-AUTH-BE-S086] Admin logout revokes active operator session and clears scoped refresh Cookie", func(t *testing.T) {
		// Step 1: logout 用 auth service stub と mutation validator を注入し、Bearer operator accessToken から対象 session を選ぶ状態を作る。
		validator := &stubOperatorSessionValidator{contextToReturn: validOperatorSessionContext()}
		auth := &stubAdminOperatorAuth{logoutCookie: adminauth.OperatorRefreshCookieCommand{AuthContextID: "01B7X9BN4X2Y3Z4A5B6C7D8E9F", Clear: true}}
		router := newAdminTestRouterWithAuth(validator, auth)
		request := newAdminJSONRequest(stdhttp.MethodPost, "/api/v1/auth/operator/logout", `{}`)
		request.Header.Set(adminOriginHeader, "https://admin.example.com")
		request.Header.Set(adminAuthHeader, "Bearer valid-admin-token")
		response := httptest.NewRecorder()

		// Step 2: logout use case が bearer だけを受け取り、対象 refresh Cookie path の削除 command が response header に出ることを確認する。
		router.ServeHTTP(response, request)

		if response.Code != stdhttp.StatusOK {
			t.Fatalf("expected logout to return 200, got %d body=%s", response.Code, response.Body.String())
		}
		if auth.logoutInput.AccessToken != "valid-admin-token" || validator.mutationInput.Permission != "operators:logout" {
			t.Fatalf("expected logout access token and operators:logout validation, input=%+v validator=%+v", auth.logoutInput, validator.mutationInput)
		}
		assertAdminRefreshCookieCleared(t, response)
		assertLogoutResponse(t, response)
	})
}

func TestAdminOperatorSetupRemainsFailClosed(t *testing.T) {
	t.Parallel()

	// Step 1: auth service を注入しても operator setup はこの task の非対象なので、既存の 503 fail-closed handler のままにする。
	validator := &stubOperatorSessionValidator{contextToReturn: validOperatorSessionContext()}
	auth := &stubAdminOperatorAuth{sessionResult: validAdminAuthSessionResult(), challenge: validAdminPasskeyChallenge()}
	router := newAdminTestRouterWithAuth(validator, auth)
	request := newAdminJSONRequest(stdhttp.MethodPost, "/api/v1/auth/operator-setup/start", `{"setupToken":"setup-token"}`)
	request.Header.Set(adminOriginHeader, "https://admin.example.com")
	response := httptest.NewRecorder()

	// Step 2: request を実行し、setup use case を呼ばず 503 のまま止まることを明示的に検証する。
	router.ServeHTTP(response, request)

	if response.Code != stdhttp.StatusServiceUnavailable {
		t.Fatalf("expected operator setup to remain 503, got %d body=%s", response.Code, response.Body.String())
	}
	if auth.startInput.Identifier != "" || auth.finishInput.CredentialHandle != "" {
		t.Fatalf("expected operator setup not to call auth-only service methods, auth=%+v", auth)
	}
	assertAdminSecurityHeaders(t, response)
}

func newAdminTestRouter(validator operatorSessionValidator) stdhttp.Handler {
	// Step 1: Admin runtime validation 済み相当の domain を渡し、Origin 比較を production と同じ設定値ベースで行う。
	cfg := config.Config{}
	cfg.Admin.Domain = "https://admin.example.com"
	return newRouterWithDependencies(cfg, adminRouterDependencies{operatorSessions: validator})
}

func newAdminTestRouterWithAuth(validator operatorSessionValidator, auth adminOperatorAuthenticator) stdhttp.Handler {
	// Step 1: auth handler だけを差し替え、router composition の他要素は production と同じ Admin domain 設定で動かす。
	cfg := config.Config{}
	cfg.Admin.Domain = "https://admin.example.com"
	return newRouterWithDependencies(cfg, adminRouterDependencies{operatorSessions: validator, operatorAuth: auth})
}

func newAdminTestRouterWithPasskeyAuth(validator operatorSessionValidator, auth adminOperatorPasskeyAuthenticator) stdhttp.Handler {
	// Step 1: passkey login handler だけを差し替え、session lifecycle handler とは別 dependency として動かす。
	cfg := config.Config{}
	cfg.Admin.Domain = "https://admin.example.com"
	return newRouterWithDependencies(cfg, adminRouterDependencies{operatorSessions: validator, operatorPasskeyAuth: auth})
}

func newAdminTestRouterWithAuthAndPasskeyVerifier(validator operatorSessionValidator, auth adminOperatorPasskeyAuthenticator, verifier adminOperatorPasskeyVerifier) stdhttp.Handler {
	// Step 1: passkey finish handler の WebAuthn verifier seam だけを追加で差し替え、raw credential から直接 session 発行できないことを検査可能にする。
	cfg := config.Config{}
	cfg.Admin.Domain = "https://admin.example.com"
	return newRouterWithDependencies(cfg, adminRouterDependencies{operatorSessions: validator, operatorPasskeyAuth: auth, operatorPasskeys: verifier})
}

func newAdminTestRouterWithSetupper(validator operatorSessionValidator, setupper adminOperatorSetupper) stdhttp.Handler {
	// Step 1: initial setup / operator setup handler だけを差し替え、credential mode ごとの response 境界を DB なしで検証できるようにする。
	cfg := config.Config{}
	cfg.Admin.Domain = "https://admin.example.com"
	return newRouterWithDependencies(cfg, adminRouterDependencies{operatorSessions: validator, operatorSetup: setupper})
}

func newAdminTestRouterWithAccountCreator(validator operatorSessionValidator, creator adminAccountCreator) stdhttp.Handler {
	// Step 1: account creation handler が application account creation use case を呼ぶ経路だけを追加で差し替える。
	cfg := config.Config{}
	cfg.Admin.Domain = "https://admin.example.com"
	return newRouterWithDependencies(cfg, adminRouterDependencies{operatorSessions: validator, accountCreation: creator})
}

func newAdminTestRouterWithAccountSearcher(validator operatorSessionValidator, searcher adminAccountSearcher) stdhttp.Handler {
	// Step 1: account search handler が application account search use case を呼ぶ経路だけを追加で差し替える。
	cfg := config.Config{}
	cfg.Admin.Domain = "https://admin.example.com"
	return newRouterWithDependencies(cfg, adminRouterDependencies{operatorSessions: validator, accountSearch: searcher})
}

func newAdminJSONRequest(method string, path string, body string) *stdhttp.Request {
	// Step 1: generated binding が JSON body として解釈できる request を組み立て、middleware 以外の 400 を避ける。
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	return request
}

func newAdminBoundarySignVerifier(t *testing.T) adminauth.JSONSignVerifier {
	t.Helper()

	// Step 1: domain JWT primitive を application/auth capability adapter 経由で作り、後続 helper と Admin auth service の両方へ同じ検証境界を渡す。
	signer, err := adminauth.NewTokenJSONSignVerifier([]byte("admin-api-product-bearer-boundary-test-secret"))
	if err != nil {
		t.Fatalf("create admin boundary signer: %v", err)
	}
	return signer
}

func signedProductBearerForAdminBoundaryTest(t *testing.T, signer adminauth.JSONSignVerifier) string {
	t.Helper()

	// Step 1: Product AccountAuth payload と同じ `status` claim を含め、Admin OperatorAuth payload が必要とする `role` / `active` claim を含めない。
	payload, err := json.Marshal(map[string]any{
		"sub":    "01ARZ3NDEKTSV4RRFFQ69G5FAV",
		"sid":    "01B7X9BN4X2Y3Z4A5B6C7D8E9F",
		"jti":    "01B7X9BN4X2Y3Z4A5B6C7D8E9G",
		"status": "active",
		"iat":    int64(1_700_000_000),
		"exp":    int64(1_700_000_900),
	})
	if err != nil {
		t.Fatalf("marshal product bearer payload: %v", err)
	}

	// Step 2: 署名済み Product bearer 文字列を返し、HTTP test では Authorization header の bearer 本体として使う。
	token, err := signer.SignJSON(payload)
	if err != nil {
		t.Fatalf("sign product bearer payload: %v", err)
	}
	return token
}

func newAdminAuthServiceForBoundaryTest(t *testing.T, signer adminauth.JSONSignVerifier) *adminauth.OperatorSessionService {
	t.Helper()

	// Step 1: Product bearer の payload 検証だけを観測するため、repository / store / secret / ID port は deterministic な最小実装で埋める。
	service, err := adminauth.NewOperatorSessionService(
		adminauth.OperatorSessionDependencies{
			Operators:       adminBoundaryOperatorRepository{},
			RefreshSessions: adminBoundaryOperatorSessionStore{},
			Signer:          signer,
			TokenGenerator:  adminBoundarySecretGenerator{},
			IDGenerator:     adminBoundaryIDGenerator{},
			Clock:           func() time.Time { return time.Unix(1_700_000_000, 0).UTC() },
		},
		adminauth.OperatorSessionConfig{OperatorAccessTokenTTL: 15 * time.Minute, OperatorRefreshSessionTTL: time.Hour, OperatorRefreshCookieLifetime: 30 * time.Minute, WebAuthnRPID: "admin.example.com"},
	)
	if err != nil {
		t.Fatalf("create admin auth service for boundary test: %v", err)
	}
	return service
}

type serviceBackedOperatorSessionValidator struct {
	auth         *adminauth.OperatorSessionService
	calls        int
	currentInput operatorSessionValidationInput
}

func (v *serviceBackedOperatorSessionValidator) ValidateOperatorSession(ctx context.Context, input operatorSessionValidationInput) (operatorSessionContext, error) {
	// Step 1: HTTP middleware から渡された bearer を記録し、テストが Product bearer を実 validator に通したことを確認できるようにする。
	v.calls++
	v.currentInput = input

	// Step 2: Admin auth service の CurrentOperator へ委譲し、Product-shaped token が Admin OperatorAuth payload として復元できないことを実装経路で検証する。
	operator, err := v.auth.CurrentOperator(ctx, adminauth.CurrentOperatorInput{AccessToken: input.AccessToken})
	if err != nil {
		return operatorSessionContext{}, err
	}

	// Step 3: 成功時だけ operator context へ変換する。今回の Product bearer test ではここへ到達しないため、到達した場合は middleware が fail-open している証拠になる。
	return operatorSessionContext{OperatorID: operator.ID, OperatorEmail: operator.Email, OperatorRole: operator.Role, OperatorActive: operator.Active}, nil
}

type adminBoundaryOperatorRepository struct{}

func (adminBoundaryOperatorRepository) FindOperatorByCredential(context.Context, string) (adminauth.OperatorSnapshot, error) {
	// Step 1: Product bearer rejection test では credential lookup へ到達しないため、到達時も有効な Operator snapshot で副作用なく返す。
	return adminauth.OperatorSnapshot{ID: "01ARZ3NDEKTSV4RRFFQ69G5FAV", Email: "admin@example.com", Role: "admin", Active: true, PasskeyRegistrationState: string(domain.OperatorPasskeyRegistrationRegistered)}, nil
}

func (adminBoundaryOperatorRepository) FindOperatorByID(context.Context, string) (adminauth.OperatorSnapshot, error) {
	// Step 1: Product bearer rejection test では operator lookup へ到達しないため、到達時も有効な Operator snapshot で副作用なく返す。
	return adminauth.OperatorSnapshot{ID: "01ARZ3NDEKTSV4RRFFQ69G5FAV", Email: "admin@example.com", Role: "admin", Active: true, PasskeyRegistrationState: string(domain.OperatorPasskeyRegistrationRegistered)}, nil
}

type adminBoundaryOperatorSessionStore struct{}

func (adminBoundaryOperatorSessionStore) Save(context.Context, adminauth.OperatorSessionRecord, time.Duration) error {
	// Step 1: Product bearer rejection test では session 保存を行わないため、副作用なしで成功扱いにする。
	return nil
}

func (adminBoundaryOperatorSessionStore) Get(context.Context, string) (adminauth.OperatorSessionRecord, error) {
	// Step 1: Product bearer rejection test では payload validation で止まるため、到達した場合も有効 record を返して境界外要因で落とさない。
	return adminauth.OperatorSessionRecord{SessionID: "01B7X9BN4X2Y3Z4A5B6C7D8E9F", OperatorID: "01ARZ3NDEKTSV4RRFFQ69G5FAV", RefreshTokenHash: "refresh-hash", RoleSnapshot: "admin", ActiveSnapshot: true, IssuedAt: time.Unix(1_700_000_000, 0).UTC(), ExpiresAt: time.Unix(1_700_003_600, 0).UTC()}, nil
}

func (adminBoundaryOperatorSessionStore) Rotate(context.Context, string, string, adminauth.OperatorSessionRecord, time.Duration) error {
	// Step 1: Product bearer rejection test では rotation を行わないため、副作用なしで成功扱いにする。
	return nil
}

func (adminBoundaryOperatorSessionStore) Revoke(context.Context, string, string) error {
	// Step 1: Product bearer rejection test では revoke を行わないため、副作用なしで成功扱いにする。
	return nil
}

type adminBoundarySecretGenerator struct{}

func (adminBoundarySecretGenerator) NewToken() (string, error) {
	// Step 1: Product bearer rejection test では secret 発行へ到達しないため、deterministic な非空値を返す。
	return "admin-boundary-secret", nil
}

type adminBoundaryIDGenerator struct{}

func (adminBoundaryIDGenerator) Next() (string, error) {
	// Step 1: Product bearer rejection test では ID 発行へ到達しないため、ULID 形式の deterministic 値を返す。
	return "01B7X9BN4X2Y3Z4A5B6C7D8E9G", nil
}

func validOperatorSessionContext() operatorSessionContext {
	// Step 1: middleware が Gin/request context へ設定する operator/session 値を deterministic に返す。
	return operatorSessionContext{
		OperatorID:                       "01ARZ3NDEKTSV4RRFFQ69G5FAV",
		OperatorEmail:                    "admin@example.com",
		OperatorRole:                     "admin",
		OperatorActive:                   true,
		OperatorPasskeyRegistrationState: string(domain.OperatorPasskeyRegistrationRegistered),
		SessionID:                        "01B7X9BN4X2Y3Z4A5B6C7D8E9F",
	}
}

func assertAdminSecurityHeaders(t *testing.T, response *httptest.ResponseRecorder) {
	t.Helper()

	// Step 1: Cache-Control と browser hardening headers をまとめて検査し、Admin API response の security baseline drift を検知する。
	expectedHeaders := map[string]string{
		"Cache-Control":             noStoreValue,
		"Content-Security-Policy":   adminSecurityCSP,
		"Strict-Transport-Security": adminSecurityHSTS,
		"Referrer-Policy":           adminSecurityReferrerPolicy,
		"X-Content-Type-Options":    "nosniff",
		"X-Frame-Options":           "DENY",
	}
	for header, expected := range expectedHeaders {
		if actual := response.Header().Get(header); actual != expected {
			t.Fatalf("expected %s %q, got %q", header, expected, actual)
		}
	}
}

func assertCreateAccountResponse(t *testing.T, response *httptest.ResponseRecorder, requestID string) {
	t.Helper()

	// Step 1: success body を generic JSON として読み、generated DTO 変換で必須 field が失われていないことを検証する。
	var body map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode create account response: %v", err)
	}
	account, ok := body["account"].(map[string]any)
	if !ok {
		t.Fatalf("expected account object in response, got %#v", body["account"])
	}

	// Step 2: response の requestId は application DTO に渡した correlation ID と一致し、account summary は use case 結果だけから作られる。
	if body["requestId"] != requestID || body["auditEventId"] != "01B7X9BN4X2Y3Z4A5B6C7D8E9F" {
		t.Fatalf("expected response correlation IDs, got %#v", body)
	}
	if account["accountId"] != "01ARZ3NDEKTSV4RRFFQ69G5FAV" || account["email"] != "customer@example.com" || account["status"] != "active" {
		t.Fatalf("expected account summary from application result, got %#v", account)
	}
}

func assertOperationError(t *testing.T, response *httptest.ResponseRecorder, expected string) {
	t.Helper()

	// Step 1: operation error body は安定 error と requestId だけを持ち、入力値や内部詳細を含まないことを検査する。
	var body map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode operation error response: %v", err)
	}
	if body["error"] != expected || body["requestId"] == "" {
		t.Fatalf("expected operation error %q with requestId, got %#v", expected, body)
	}
}

func assertPasskeyStartResponse(t *testing.T, response *httptest.ResponseRecorder) {
	t.Helper()

	// Step 1: start response body を JSON として読み、application challenge と WebAuthn optional fields が transport DTO に写像されたことを確認する。
	var body map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode passkey start response: %v", err)
	}
	if body["challenge"] != "admin-challenge" || body["rpId"] != "admin.example.com" || body["userVerification"] != "required" || body["requestId"] == "" {
		t.Fatalf("expected passkey start fields, got %#v", body)
	}
	allow, ok := body["allowCredentials"].([]any)
	if !ok || len(allow) != 1 {
		t.Fatalf("expected one allowCredentials entry, got %#v", body["allowCredentials"])
	}
}

func assertAdminAuthSessionResponse(t *testing.T, response *httptest.ResponseRecorder, expectedRequestID string) {
	t.Helper()

	// Step 1: session response は accessToken/operator を含み、refreshToken 平文や Cookie command 構造を含まないことを確認する。
	bodyText := response.Body.String()
	if strings.Contains(bodyText, "admin-refresh-cookie-value") || strings.Contains(bodyText, "refreshToken") || strings.Contains(bodyText, "RefreshCookie") {
		t.Fatalf("expected response body not to expose refresh cookie value, got %s", bodyText)
	}
	var body map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode admin auth session response: %v", err)
	}
	if body["accessToken"] != "admin-access-token" || body["sessionId"] != "01B7X9BN4X2Y3Z4A5B6C7D8E9F" || body["authContextId"] != "01B7X9BN4X2Y3Z4A5B6C7D8E9F" {
		t.Fatalf("expected auth session fields, got %#v", body)
	}
	if expectedRequestID != "" && body["requestId"] != expectedRequestID {
		t.Fatalf("expected requestId %q, got %#v", expectedRequestID, body)
	}
	if _, ok := body["account"]; ok {
		t.Fatalf("expected Admin auth session response not to contain Product account payload, got %#v", body["account"])
	}
	operator, ok := body["operator"].(map[string]any)
	if !ok || operator["operatorId"] != "01ARZ3NDEKTSV4RRFFQ69G5FAV" || operator["role"] != "admin" {
		t.Fatalf("expected operator profile in auth session response, got %#v", body["operator"])
	}
}

func assertAdminBearerAuthSessionResponse(t *testing.T, response *httptest.ResponseRecorder, expectedRequestID string) {
	t.Helper()

	// Step 1: Bearer mode response は automation client 用の refreshToken を body に含め、Cookie command 構造は含めないことを検査する。
	bodyText := response.Body.String()
	if !strings.Contains(bodyText, `"refreshToken":"admin-refresh-cookie-value"`) || strings.Contains(bodyText, "RefreshCookie") {
		t.Fatalf("expected bearer response body to expose only refreshToken field, got %s", bodyText)
	}
	var body map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode admin bearer auth session response: %v", err)
	}
	if body["credentialMode"] != "bearer" || body["accessToken"] != "admin-access-token" || body["refreshToken"] != "admin-refresh-cookie-value" || body["sessionId"] != "01B7X9BN4X2Y3Z4A5B6C7D8E9F" {
		t.Fatalf("expected bearer auth session fields, got %#v", body)
	}
	if expectedRequestID != "" && body["requestId"] != expectedRequestID {
		t.Fatalf("expected requestId %q, got %#v", expectedRequestID, body)
	}
	if _, ok := body["account"]; ok {
		t.Fatalf("expected Admin bearer session response not to contain Product account payload, got %#v", body["account"])
	}
	operator, ok := body["operator"].(map[string]any)
	if !ok || operator["operatorId"] != "01ARZ3NDEKTSV4RRFFQ69G5FAV" || operator["role"] != "admin" {
		t.Fatalf("expected operator profile in bearer auth session response, got %#v", body["operator"])
	}
	assertNoSetCookie(t, response)
}

func assertNoSetCookie(t *testing.T, response *httptest.ResponseRecorder) {
	t.Helper()

	// Step 1: Bearer mode や拒否 response が ambient Cookie credential を作らないことを Set-Cookie header の不在で固定する。
	if cookies := response.Header().Values("Set-Cookie"); len(cookies) != 0 {
		t.Fatalf("expected no Set-Cookie header, got %q", cookies)
	}
}

func assertCurrentOperatorResponse(t *testing.T, response *httptest.ResponseRecorder) {
	t.Helper()

	// Step 1: current response body が operator profile だけを含み、session secret を含まないことを検査する。
	var body map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode current operator response: %v", err)
	}
	operator, ok := body["operator"].(map[string]any)
	if !ok || operator["email"] != "admin@example.com" || operator["active"] != true {
		t.Fatalf("expected current operator profile, got %#v", body)
	}
	if strings.Contains(response.Body.String(), "admin-refresh-cookie-value") {
		t.Fatalf("expected current response not to expose refresh cookie, got %s", response.Body.String())
	}
}

func assertAdminRefreshCookieSet(t *testing.T, response *httptest.ResponseRecorder, expectedValue string) {
	t.Helper()

	// Step 1: Set-Cookie header に HttpOnly/Secure/SameSite 属性付きの Admin refresh Cookie が含まれることを確認する。
	for _, header := range response.Header().Values("Set-Cookie") {
		if strings.Contains(header, adminRefreshCookieName+"="+expectedValue) && strings.Contains(header, "HttpOnly") && strings.Contains(header, "Secure") && strings.Contains(header, "SameSite=Lax") && strings.Contains(header, "Path=/api/v1/auth/contexts/01B7X9BN4X2Y3Z4A5B6C7D8E9F/refresh") {
			return
		}
	}
	t.Fatalf("expected %s Set-Cookie header, got %q", adminRefreshCookieName, response.Header().Values("Set-Cookie"))
}

func assertAdminRefreshCookieCleared(t *testing.T, response *httptest.ResponseRecorder) {
	t.Helper()

	// Step 1: logout response が refresh Cookie を Max-Age=0 で削除し、HttpOnly/Secure 属性を保つことを確認する。
	for _, header := range response.Header().Values("Set-Cookie") {
		if strings.Contains(header, adminRefreshCookieName+"=") && strings.Contains(header, "Max-Age=0") && strings.Contains(header, "HttpOnly") && strings.Contains(header, "Secure") && strings.Contains(header, "Path=/api/v1/auth/contexts/01B7X9BN4X2Y3Z4A5B6C7D8E9F/refresh") {
			return
		}
	}
	t.Fatalf("expected cleared %s Set-Cookie header, got %q", adminRefreshCookieName, response.Header().Values("Set-Cookie"))
}

func assertLogoutResponse(t *testing.T, response *httptest.ResponseRecorder) {
	t.Helper()

	// Step 1: logout body は revoke 成功と requestId だけを返し、token や Cookie value を含めないことを確認する。
	var body map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode logout response: %v", err)
	}
	if body["revoked"] != true || body["requestId"] == "" || strings.Contains(response.Body.String(), "admin-refresh-cookie-value") {
		t.Fatalf("expected logout revoked response without secret, got %#v", body)
	}
}

type stubOperatorSessionValidator struct {
	calls           int
	currentInput    operatorSessionValidationInput
	mutationInput   operatorSessionValidationInput
	contextToReturn operatorSessionContext
	errToReturn     error
}

type stubAdminAccountCreator struct {
	calls       int
	input       accountsapplication.CreateAccountInput
	created     accountsapplication.CreatedAccount
	errToReturn error
}

type stubAccountSearchRepository struct {
	calls       int
	detailCalls int
	query       accountsapplication.AccountSearchQuery
}

type stubAdminOperatorAuth struct {
	startInput     adminauth.StartOperatorPasskeyInput
	finishInput    adminauth.FinishOperatorPasskeyInput
	refreshInput   adminauth.RefreshOperatorSessionInput
	currentInput   adminauth.CurrentOperatorInput
	authorizeInput adminauth.AuthorizeOperatorSessionInput
	logoutInput    adminauth.LogoutOperatorInput
	challenge      adminauth.OperatorPasskeyChallenge
	sessionResult  adminauth.OperatorSessionResult
	operator       adminauth.OperatorDTO
	logoutCookie   adminauth.OperatorRefreshCookieCommand
	errToReturn    error
}

type stubAdminOperatorSetupper struct {
	startInitialInput   operatorsapplication.InitialSetupStartInput
	finishInitialInput  operatorsapplication.InitialSetupFinishInput
	startOperatorInput  operatorsapplication.SetupStartInput
	finishOperatorInput operatorsapplication.SetupFinishInput
	createInput         operatorsapplication.CreateOperatorInput
	challenge           operatorsapplication.SetupChallengeResult
	sessionResult       adminauth.OperatorSessionResult
	createdOperator     operatorsapplication.CreatedOperator
	errToReturn         error
}

type stubAdminOperatorPasskeyVerifier struct {
	challengeID      string
	credential       adminauth.WebAuthnAssertionCredentialDTO
	credentialHandle string
	errToReturn      error
}

func (s *stubAdminAccountCreator) CreateAccount(_ context.Context, input accountsapplication.CreateAccountInput) (accountsapplication.CreatedAccount, error) {
	// Step 1: handler が渡した application DTO を記録し、transport mapping と operator context mapping をテストで観測できるようにする。
	s.calls++
	s.input = input

	// Step 2: error 注入時は application use case の失敗としてそのまま返し、handler の HTTP mapping を独立して検証する。
	if s.errToReturn != nil {
		return accountsapplication.CreatedAccount{}, s.errToReturn
	}

	// Step 3: 成功時は request correlation を保持した結果を返し、handler の response DTO 変換を検証できるようにする。
	created := s.created
	created.RequestID = input.RequestID
	return created, nil
}

func (s *stubAccountSearchRepository) SearchAccounts(_ context.Context, query accountsapplication.AccountSearchQuery) (accountsapplication.AccountSearchRepositoryResult, error) {
	// Step 1: application validation 後にだけ呼ばれる repository fake として、呼び出し回数と検証済み query を記録する。
	s.calls++
	s.query = query

	// Step 2: S083 の invalid limit test では呼ばれない想定だが、正常系で使われても空結果を安全に返せるようにする。
	return accountsapplication.AccountSearchRepositoryResult{}, nil
}

func (s *stubAccountSearchRepository) FindAccountByID(_ context.Context, accountID string) (accountsapplication.AccountSummaryRecord, error) {
	// Step 1: detail repository fake として呼び出し回数を記録し、handler/service の wiring を検証可能にする。
	s.detailCalls++

	// Step 2: 空 ID は対象不在として返し、実 repository と同じ stable error に畳む。
	if accountID == "" {
		return accountsapplication.AccountSummaryRecord{}, accountsapplication.ErrAccountSearchNotFound
	}

	// Step 3: deterministic read model を返し、detail route の response conversion を DB なしで実行できるようにする。
	return accountsapplication.AccountSummaryRecord{AccountID: accountID, Email: "customer@example.com", Status: "active", PasskeyCount: 1, CreatedAt: time.Date(2026, 5, 26, 12, 0, 0, 0, time.UTC)}, nil
}

func validCreatedAdminAccount() accountsapplication.CreatedAccount {
	// Step 1: handler success response の期待値として使う deterministic な application DTO を組み立てる。
	return accountsapplication.CreatedAccount{
		AccountID:    "01ARZ3NDEKTSV4RRFFQ69G5FAV",
		Email:        "customer@example.com",
		Status:       "active",
		Locale:       "ja-JP",
		PasskeyCount: 0,
		AuditID:      "01B7X9BN4X2Y3Z4A5B6C7D8E9F",
		CreatedAt:    time.Date(2026, 5, 26, 12, 0, 0, 0, time.UTC),
	}
}

func validAdminPasskeyChallenge() adminauth.OperatorPasskeyChallenge {
	// Step 1: passkey start response の expected value として、application service が返す WebAuthn challenge DTO を作る。
	return adminauth.OperatorPasskeyChallenge{
		ChallengeID:     "01B7X9BN4X2Y3Z4A5B6C7D8E9F",
		Challenge:       "admin-challenge",
		WebAuthnRPID:    "admin.example.com",
		WebAuthnOptions: []byte(`{"publicKey":{"allowCredentials":[{"id":"credential-id","type":"public-key","transports":["internal"]}],"timeout":60000,"userVerification":"required"}}`),
	}
}

func validAdminAuthSessionResult() adminauth.OperatorSessionResult {
	// Step 1: auth session response と Set-Cookie header の expected value として、refresh Cookie command 付き DTO を組み立てる。
	return adminauth.OperatorSessionResult{
		AccessToken: "admin-access-token",
		Operator:    validAdminOperatorDTO(),
		SessionID:   "01B7X9BN4X2Y3Z4A5B6C7D8E9F",
		ExpiresAt:   time.Date(2026, 5, 26, 12, 15, 0, 0, time.UTC),
		RefreshCookie: adminauth.OperatorRefreshCookieCommand{
			AuthContextID: "01B7X9BN4X2Y3Z4A5B6C7D8E9F",
			Value:         "admin-refresh-cookie-value",
			MaxAge:        30 * time.Minute,
		},
	}
}

func validAdminOperatorDTO() adminauth.OperatorDTO {
	// Step 1: current/session response 用に Admin operator の application DTO を deterministic に返す。
	return adminauth.OperatorDTO{ID: "01ARZ3NDEKTSV4RRFFQ69G5FAV", Email: "admin@example.com", Role: "admin", Active: true}
}

func (s *stubAdminOperatorAuth) StartOperatorPasskey(_ context.Context, input adminauth.StartOperatorPasskeyInput) (adminauth.OperatorPasskeyChallenge, error) {
	// Step 1: handler が渡した identifier を記録し、error 注入があれば application service failure として返す。
	s.startInput = input
	if s.errToReturn != nil {
		return adminauth.OperatorPasskeyChallenge{}, s.errToReturn
	}
	return s.challenge, nil
}

func (s *stubAdminOperatorAuth) FinishOperatorPasskey(_ context.Context, input adminauth.FinishOperatorPasskeyInput) (adminauth.OperatorSessionResult, error) {
	// Step 1: handler が渡した challenge / credential handle を記録し、session 発行結果を返す。
	s.finishInput = input
	if s.errToReturn != nil {
		return adminauth.OperatorSessionResult{}, s.errToReturn
	}
	return s.sessionResult, nil
}

func (s *stubAdminOperatorAuth) RefreshOperatorSession(_ context.Context, input adminauth.RefreshOperatorSessionInput) (adminauth.OperatorSessionResult, error) {
	// Step 1: handler が Cookie から読んだ refreshToken を記録し、rotation 結果 DTO を返す。
	s.refreshInput = input
	if s.errToReturn != nil {
		return adminauth.OperatorSessionResult{}, s.errToReturn
	}
	return s.sessionResult, nil
}

func (s *stubAdminOperatorAuth) CurrentOperator(_ context.Context, input adminauth.CurrentOperatorInput) (adminauth.OperatorDTO, error) {
	// Step 1: handler が header から抽出した accessToken を記録し、current operator DTO を返す。
	s.currentInput = input
	if s.errToReturn != nil {
		return adminauth.OperatorDTO{}, s.errToReturn
	}
	return s.operator, nil
}

func (s *stubAdminOperatorAuth) AuthorizeOperatorSession(_ context.Context, input adminauth.AuthorizeOperatorSessionInput) (adminauth.OperatorAuthorizationDecision, error) {
	// Step 1: middleware が渡した bearer と permission を記録し、HTTP adapter が role matrix を持たず auth application へ委譲することを検証可能にする。
	s.authorizeInput = input
	if s.errToReturn != nil {
		return adminauth.OperatorAuthorizationDecision{}, s.errToReturn
	}

	// Step 2: 成功時は current operator DTO と同じ primitive を許可済み decision として返し、handler context 変換を deterministic にする。
	operator := s.operator
	if operator.SessionID == "" {
		operator.SessionID = "01B7X9BN4X2Y3Z4A5B6C7D8E9F"
	}
	return adminauth.OperatorAuthorizationDecision{Operator: operator, SessionID: operator.SessionID, Permission: input.Permission, Allowed: true}, nil
}

func (s *stubAdminOperatorAuth) LogoutOperator(_ context.Context, input adminauth.LogoutOperatorInput) (adminauth.OperatorRefreshCookieCommand, error) {
	// Step 1: handler が header から抽出した accessToken を記録し、Cookie clear command を返す。
	s.logoutInput = input
	if s.errToReturn != nil {
		return adminauth.OperatorRefreshCookieCommand{}, s.errToReturn
	}
	return s.logoutCookie, nil
}

func (s *stubAdminOperatorSetupper) StartInitialSetup(_ context.Context, input operatorsapplication.InitialSetupStartInput) (operatorsapplication.SetupChallengeResult, error) {
	// Step 1: handler が bootstrap / identity DTO を application 境界へ渡した事実を記録する。
	s.startInitialInput = input
	if s.errToReturn != nil {
		return operatorsapplication.SetupChallengeResult{}, s.errToReturn
	}
	return s.challenge, nil
}

func (s *stubAdminOperatorSetupper) FinishInitialSetup(_ context.Context, input operatorsapplication.InitialSetupFinishInput) (adminauth.OperatorSessionResult, error) {
	// Step 1: handler が attestation DTO を application 境界へ渡した事実を記録し、session 発行結果を返す。
	s.finishInitialInput = input
	if s.errToReturn != nil {
		return adminauth.OperatorSessionResult{}, s.errToReturn
	}
	return s.sessionResult, nil
}

func (s *stubAdminOperatorSetupper) StartOperatorSetup(_ context.Context, input operatorsapplication.SetupStartInput) (operatorsapplication.SetupChallengeResult, error) {
	// Step 1: handler が setup token を application 境界へ渡した事実を記録する。
	s.startOperatorInput = input
	if s.errToReturn != nil {
		return operatorsapplication.SetupChallengeResult{}, s.errToReturn
	}
	return s.challenge, nil
}

func (s *stubAdminOperatorSetupper) FinishOperatorSetup(_ context.Context, input operatorsapplication.SetupFinishInput) (adminauth.OperatorSessionResult, error) {
	// Step 1: handler が setup token と attestation DTO を application 境界へ渡した事実を記録し、session 発行結果を返す。
	s.finishOperatorInput = input
	if s.errToReturn != nil {
		return adminauth.OperatorSessionResult{}, s.errToReturn
	}
	return s.sessionResult, nil
}

func (s *stubAdminOperatorSetupper) CreateOperator(_ context.Context, input operatorsapplication.CreateOperatorInput) (operatorsapplication.CreatedOperator, error) {
	// Step 1: handler が検証済み acting operator context と作成 request を application 境界へ渡した事実を記録する。
	s.createInput = input
	if s.errToReturn != nil {
		return operatorsapplication.CreatedOperator{}, s.errToReturn
	}
	created := s.createdOperator
	created.RequestID = input.RequestID
	return created, nil
}

func (s *stubAdminOperatorPasskeyVerifier) VerifyOperatorPasskey(_ context.Context, challengeID string, credential adminauth.WebAuthnAssertionCredentialDTO) (string, error) {
	// Step 1: handler が渡した challenge と assertion DTO を記録し、検証済み credential handle だけを auth service に渡せるよう返す。
	s.challengeID = challengeID
	s.credential = credential
	if s.errToReturn != nil {
		return "", s.errToReturn
	}
	return s.credentialHandle, nil
}

func (s *stubOperatorSessionValidator) ValidateOperatorSession(_ context.Context, input operatorSessionValidationInput) (operatorSessionContext, error) {
	// Step 1: 呼び出し回数と入力を記録し、テストが route 種別ごとの session/RBAC 要求を検査できるようにする。
	s.calls++
	if input.Permission != "" {
		s.mutationInput = input
	} else {
		s.currentInput = input
	}

	// Step 2: エラーを注入された場合はそのまま返し、middleware の error mapping だけを独立して検証できるようにする。
	if s.errToReturn != nil {
		return operatorSessionContext{}, s.errToReturn
	}

	// Step 3: 成功時は deterministic な operator/session context を返し、handler 未実装状態でも middleware 通過を観測できるようにする。
	return s.contextToReturn, nil
}

func TestAdminOperatorContextIsBoundAfterSessionValidation(t *testing.T) {
	t.Parallel()

	// Step 1: middleware 単体 router を組み、後続 handler から Gin context と request.Context に設定された operator/session 情報を観測する。
	validator := &stubOperatorSessionValidator{contextToReturn: validOperatorSessionContext()}
	router := ginlessAdminContextTestRouter(validator)
	request := httptest.NewRequest(stdhttp.MethodGet, "/api/v1/auth/operator/current", nil)
	request.Header.Set(adminAuthHeader, "Bearer valid-admin-token")
	response := httptest.NewRecorder()

	// Step 2: 後続 handler が context 値を JSON 化できることを確認し、middleware が operator session 境界を作る証拠にする。
	router.ServeHTTP(response, request)

	if response.Code != stdhttp.StatusOK {
		t.Fatalf("expected context observer to return 200, got %d body=%s", response.Code, response.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode context observer response: %v", err)
	}
	if body[adminContextKeyOperatorID] != "01ARZ3NDEKTSV4RRFFQ69G5FAV" || body[adminContextKeySessionID] != "01B7X9BN4X2Y3Z4A5B6C7D8E9F" {
		t.Fatalf("expected operator/session context, got %#v", body)
	}
}

func ginlessAdminContextTestRouter(validator operatorSessionValidator) stdhttp.Handler {
	// Step 1: generated handler の 503 に依存せず middleware の context binding だけを観測するため、同じ current path に test 専用 handler を置く。
	router := gin.New()
	router.Use(adminSecurityHeadersMiddleware())
	router.Use(adminAuthMiddleware(config.Config{Admin: config.AdminRuntimeConfig{Domain: "https://admin.example.com"}}, validator))
	router.GET("/api/v1/auth/operator/current", func(c *gin.Context) {
		// Step 2: Gin context と request.Context の両方から値を読み、将来の handler がどちらの境界でも利用できることを検査する。
		c.JSON(stdhttp.StatusOK, map[string]any{
			adminContextKeyOperatorID:                       c.GetString(adminContextKeyOperatorID),
			adminContextKeySessionID:                        c.Request.Context().Value(adminContextValueKey(adminContextKeySessionID)),
			adminContextKeyOperatorRole:                     c.GetString(adminContextKeyOperatorRole),
			adminContextKeyOperatorPasskeyRegistrationState: c.GetString(adminContextKeyOperatorPasskeyRegistrationState),
		})
	})
	return router
}
