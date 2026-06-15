package admin

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	stdhttp "net/http"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	sharedhttp "www-template/packages/backend/internal/adapter/http/shared"
	accountsapplication "www-template/packages/backend/internal/application/accounts"
	adminauth "www-template/packages/backend/internal/application/auth"
	conceptauth "www-template/packages/backend/internal/application/auth"
	operatorsapplication "www-template/packages/backend/internal/application/operators"
	"www-template/packages/backend/internal/generated/adminopenapi"
	"www-template/packages/backend/internal/platform/config"
	"www-template/packages/backend/internal/platform/id"
)

const noStoreValue = "no-store"
const fallbackRequestID = "01ARZ3NDEKTSV4RRFFQ69G5FAV"
const adminInvalidRequestBodyMessage = "invalid request body"
const adminAccountSearchInvalidMessage = "invalid account search query"
const adminDuplicateEmailMessage = "duplicate_email"
const adminAuthInvalidRequestMessage = "invalid auth request"
const adminMinimumAuthenticatorMessage = "minimum_authenticator_required"
const adminRefreshCookieName = "admin_refresh_token"
const adminSetupUnavailableMessage = "setup_unavailable"

// NewRouter は Admin API 専用の Gin router を依存注入済みの状態で構築する。
//
// 役割:
//   - Product router の NewRouter(cfg, deps) と同じ constructor 形式に統一し、依存必須の design policy を Admin にも適用する。
//   - cfg は trusted proxy と production mode の判定に使う設定値である。
//   - dependencies は runtime composition が構築した Admin application service 群であり、nil field は該当 route を fail-close にする。
//   - 戻り値は Admin generated bindings だけを登録した HTTP handler であり、Product generated bindings や Product handler は登録しない。
//
// 使用例:
//
//	router := admin.NewRouter(cfg, admin.Dependencies{...})
func NewRouter(cfg config.Config, dependencies Dependencies) *gin.Engine {
	// Step 1: exported DTO を既存の package-local dependency table へ変換し、未接続 route の fail-close 仕様は維持する。
	return newRouterWithDependencies(cfg, adminRouterDependencies{operatorSessions: dependencies.OperatorSessions, operatorAuth: dependencies.OperatorAuth, operatorPasskeyAuth: dependencies.OperatorPasskeyAuth, operatorPasskeys: dependencies.OperatorPasskeyVerifier, operatorSetup: dependencies.OperatorSetup, operatorPasskeyManagement: dependencies.OperatorPasskeyManagement, accountCreation: dependencies.AccountCreation, accountSearch: dependencies.AccountSearch})
}

func newRouterWithDependencies(cfg config.Config, dependencies adminRouterDependencies) *gin.Engine {
	// Step 1: production では Gin の release mode を使い、Product router と同じ runtime noise 削減方針を保つ。
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Step 2: Admin 専用 Gin engine を新規作成し、Product router の middleware / handler table を共有しない。
	router := gin.New()
	sharedhttp.EnableStrictHandlerRequestContextFallback(router)
	_ = router.SetTrustedProxies(cfg.TrustedProxyCIDRs)
	router.Use(gin.Recovery())
	router.Use(adminSecurityHeadersMiddleware())
	if allowedOrigins := adminCORSAllowedOrigins(cfg); len(allowedOrigins) > 0 {
		// Step 3: CORS は認証 middleware より前に配置し、Admin Cookie 発行・refresh 用の preflight を bearer/session 検証へ進ませない。
		router.Use(cors.New(cors.Config{
			AllowCredentials: true,
			AllowHeaders:     []string{"Content-Type", "Authorization", "traceparent", "tracestate", "baggage"},
			AllowMethods:     []string{"GET", "POST", "DELETE", "OPTIONS"},
			AllowOrigins:     allowedOrigins,
			MaxAge:           12 * time.Hour,
		}))
	}
	// Step 3.5: OTel middleware は認証 middleware より前に配置し、Admin API の全 route で tracing を有効にする。
	// Product router と同じ構造で OTel を適用し、Admin/Product の observability 非対称を解消する。
	router.Use(otelMiddleware())
	router.Use(adminAuthMiddleware(cfg, dependencies.operatorSessions))

	// Step 4: health check は Admin router 自身が所有し、Admin runtime が stdlib mux へ fallback しないようにする。
	router.GET("/health", func(c *gin.Context) {
		c.JSON(stdhttp.StatusOK, gin.H{
			"status":    "ok",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		})
	})

	// Step 5: Admin generated bindings だけから strict handler を作成し、Admin route registration をこの package に閉じる。
	strictHandler := adminopenapi.NewStrictHandler(strictServer{operatorAuth: dependencies.operatorAuth, operatorPasskeyAuth: dependencies.operatorPasskeyAuth, operatorSetup: dependencies.operatorSetup, operatorPasskeys: dependencies.operatorPasskeys, operatorPasskeyManagement: dependencies.operatorPasskeyManagement, accountCreation: dependencies.accountCreation, accountSearch: dependencies.accountSearch}, nil)
	adminopenapi.RegisterHandlersWithOptions(router, strictHandler, adminopenapi.GinServerOptions{})

	// Step 6: 未定義 path は 404 JSON に正規化し、Admin route table へ Product path が混入していないことを検査しやすくする。
	router.NoRoute(func(c *gin.Context) {
		c.JSON(stdhttp.StatusNotFound, gin.H{
			"error": "not found",
			"path":  c.Request.URL.Path,
		})
	})

	return router
}

func adminCORSAllowedOrigins(cfg config.Config) []string {
	// Step 1: Admin CORS の許可元は Product の allowed_origins ではなく Admin runtime domain だけに限定する。
	origin, ok := sharedhttp.NormalizeOrigin(cfg.Admin.Domain)
	if !ok {
		// Step 2: Admin domain が未設定または不正なテスト/未検証 config では CORS middleware を登録せず、runtime validation 側の fail-close に委ねる。
		return nil
	}

	// Step 3: gin-contrib/cors へ canonical origin を渡し、path/query/fragment 付き Origin を許可しない比較規則と揃える。
	return []string{origin}
}

type adminAccountCreator interface {
	CreateAccount(ctx context.Context, input accountsapplication.CreateAccountInput) (accountsapplication.CreatedAccount, error)
}

type adminAccountSearcher interface {
	SearchAccounts(ctx context.Context, input accountsapplication.AccountSearchInput) (accountsapplication.AccountSearchResult, error)
	GetAccount(ctx context.Context, input accountsapplication.AccountDetailInput) (accountsapplication.AccountDetailResult, error)
}

// AccountCreator は Admin account 作成 handler が呼び出す application 境界である。
//
// 役割:
//   - runtime composition から account creation use case を注入し、HTTP adapter が repository や OpenSearch projector を直接生成しないようにする。
//   - CreateAccount は generated DTO ではなく adminapplication の入力/出力 DTO だけを扱う。
//   - nil の場合は Admin mutation を 503 で fail-close し、監査なし account 作成を防ぐ。
type AccountCreator = adminAccountCreator

// AccountSearcher は Admin account 一覧 handler が呼び出す application 境界である。
//
// 役割:
//   - runtime composition から account search use case を注入し、HTTP adapter が GORM や SQL へ依存しない境界を保つ。
//   - SearchAccounts は pagination/input validation を application 層へ委譲する。
//   - nil の場合は Admin read model を 503 で fail-close する。
type AccountSearcher = adminAccountSearcher

// Dependencies は Admin HTTP router へ runtime composition 済み use case を注入するための DTO である。
//
// 役割:
//   - Product router/container と共有せず、Admin binary 専用の application service だけを渡す。
//   - OperatorSessions は middleware が protected route の accessToken/CSRF/session を検証するための dependency である。
//   - OperatorPasskeyAuth は Admin passkey login の challenge/finish outer flow を扱う dependency である。
//   - OperatorPasskeyVerifier は finish handler が WebAuthn assertion を検証済み credential handle へ変換する dependency である。
//   - OperatorPasskeyManagement は Admin operator 自身の passkey 一覧・削除 use case を表す。
//   - AccountCreation は audit projection を含む account mutation use case、AccountSearch は account read model use case を表す。
//   - setup handler 用 dependency は後続 task で別 field として追加されるため、この DTO では setup token 登録 use case は公開しない。
type Dependencies struct {
	OperatorSessions          OperatorSessionValidator
	OperatorAuth              OperatorAuthenticator
	OperatorPasskeyAuth       OperatorPasskeyAuthenticator
	OperatorPasskeyVerifier   OperatorPasskeyVerifier
	OperatorSetup             OperatorSetupper
	OperatorPasskeyManagement OperatorPasskeyManager
	AccountCreation           AccountCreator
	AccountSearch             AccountSearcher
}

type adminOperatorAuthenticator interface {
	RefreshOperatorSession(ctx context.Context, input adminauth.RefreshOperatorSessionInput) (adminauth.OperatorSessionResult, error)
	CurrentOperator(ctx context.Context, input adminauth.CurrentOperatorInput) (adminauth.OperatorDTO, error)
	LogoutOperator(ctx context.Context, input adminauth.LogoutOperatorInput) (adminauth.OperatorRefreshCookieCommand, error)
}

type adminOperatorPasskeyAuthenticator interface {
	StartOperatorPasskey(ctx context.Context, input adminauth.StartOperatorPasskeyInput) (adminauth.OperatorPasskeyChallenge, error)
	FinishOperatorPasskey(ctx context.Context, input adminauth.FinishOperatorPasskeyInput) (adminauth.OperatorSessionResult, error)
}

// OperatorAuthenticator は Admin auth handler が呼び出す application 境界である。
//
// 役割:
//   - runtime composition から Admin OperatorAuth service を注入し、HTTP adapter が token signer や session store を直接生成しないようにする。
//   - Product auth service ではなく Admin operator 専用 DTO だけを扱う。
//   - nil の場合は該当 auth route を 503 で fail-close する。
type OperatorAuthenticator = adminOperatorAuthenticator

// OperatorPasskeyAuthenticator は Admin passkey login handler が呼び出す application 境界である。
//
// 役割:
//   - WebAuthn challenge 開始と検証済み credential からの session 発行 outer flow を受け持つ。
//   - session refresh/current/logout lifecycle と別 dependency にし、HTTP adapter が責務混在した service を要求しないようにする。
//   - nil の場合は passkey login route を 503 で fail-close する。
type OperatorPasskeyAuthenticator = adminOperatorPasskeyAuthenticator

type adminOperatorSetupper interface {
	StartInitialSetup(ctx context.Context, input operatorsapplication.InitialSetupStartInput) (operatorsapplication.SetupChallengeResult, error)
	FinishInitialSetup(ctx context.Context, input operatorsapplication.InitialSetupFinishInput) (adminauth.OperatorSessionResult, error)
	StartOperatorSetup(ctx context.Context, input operatorsapplication.SetupStartInput) (operatorsapplication.SetupChallengeResult, error)
	FinishOperatorSetup(ctx context.Context, input operatorsapplication.SetupFinishInput) (adminauth.OperatorSessionResult, error)
	CreateOperator(ctx context.Context, input operatorsapplication.CreateOperatorInput) (operatorsapplication.CreatedOperator, error)
}

// OperatorSetupper は Admin setup handler が呼び出す application 境界である。
//
// 役割:
//   - runtime composition から initial setup / operator setup use case を注入する。
//   - setup token や bootstrap secret の検証を handler に置かず、application/domain/repository へ委譲する。
//   - nil の場合は setup route を 503 で fail-close し、未検証 credential 登録を防ぐ。
type OperatorSetupper = adminOperatorSetupper

type adminOperatorPasskeyVerifier interface {
	VerifyOperatorPasskey(ctx context.Context, challengeID string, credential adminauth.WebAuthnAssertionCredentialDTO) (string, error)
}

// OperatorPasskeyVerifier は Admin passkey finish handler が WebAuthn assertion を検証する境界である。
//
// 役割:
//   - HTTP DTO から取り出した assertion を署名検証し、検証済み credential handle だけを passkey login service へ渡す。
//   - handler が raw credential handle を信用して session 発行へ進むことを防ぐ。
//   - nil の場合は passkey finish route を 503 で fail-close する。
type OperatorPasskeyVerifier = adminOperatorPasskeyVerifier

type adminOperatorPasskeyManager interface {
	ListOperatorPasskeys(ctx context.Context, input adminauth.ListOperatorPasskeysInput) (adminauth.OperatorPasskeyListResult, error)
	DeleteOperatorPasskey(ctx context.Context, input adminauth.DeleteOperatorPasskeyInput) error
}

// OperatorPasskeyManager は Admin operator passkey 管理 handler が呼び出す application 境界である。
//
// 役割:
//   - runtime composition から Admin operator passkey list/delete use case を注入する。
//   - Product passkey repository や package-local BFF route を使わず、Admin backend の same-origin API に閉じる。
//   - nil の場合は passkey 管理 route を 503 で fail-close する。
type OperatorPasskeyManager = adminOperatorPasskeyManager

type strictServer struct {
	operatorAuth              adminOperatorAuthenticator
	operatorPasskeyAuth       adminOperatorPasskeyAuthenticator
	operatorSetup             adminOperatorSetupper
	operatorPasskeys          adminOperatorPasskeyVerifier
	operatorPasskeyManagement adminOperatorPasskeyManager
	accountCreation           adminAccountCreator
	accountSearch             adminAccountSearcher
}

func (s strictServer) ListAdminAccounts(ctx context.Context, request adminopenapi.ListAdminAccountsRequestObject) (adminopenapi.ListAdminAccountsResponseObject, error) {
	// Step 1: 追跡 ID を先に生成し、validation error と成功 response の correlation を一致させる。
	requestID := nextAdminRequestID()

	// Step 2: account search use case 未接続時は read model を公開せず、Admin API を fail-close にする。
	if s.accountSearch == nil {
		return adminopenapi.ListAdminAccounts503JSONResponse{Body: authFailureResponseWithRequestID(adminopenapi.InternalError, requestID), Headers: adminopenapi.ListAdminAccounts503ResponseHeaders{CacheControl: noStoreValue}}, nil
	}

	// Step 3: generated query params を application DTO へ詰め替え、pagination validation は application use case に委譲する。
	result, err := s.accountSearch.SearchAccounts(ctx, accountSearchInput(request.Params, requestID))
	if err != nil {
		return listAdminAccountsFailure(requestID, err), nil
	}

	// Step 4: application read model を Admin OpenAPI response DTO へ変換し、repository/generated 型を相互に漏らさない。
	return adminopenapi.ListAdminAccounts200JSONResponse{Body: adminAccountListResponse(result), Headers: adminopenapi.ListAdminAccounts200ResponseHeaders{CacheControl: noStoreValue}}, nil
}

func (s strictServer) CreateAdminAccount(ctx context.Context, request adminopenapi.CreateAdminAccountRequestObject) (adminopenapi.CreateAdminAccountResponseObject, error) {
	// Step 1: 追跡 ID は handler 境界で一度だけ生成し、application DTO と HTTP response の correlation を一致させる。
	requestID := nextAdminRequestID()

	// Step 2: generated binding が通常は JSON body を必ず設定するが、strict server の直接呼び出しでも nil body は 400 に正規化する。
	if request.Body == nil {
		return adminopenapi.CreateAdminAccount400JSONResponse{Body: authOperationError(requestID, adminInvalidRequestBodyMessage), Headers: adminopenapi.CreateAdminAccount400ResponseHeaders{CacheControl: noStoreValue}}, nil
	}

	// Step 3: account creation use case 未接続時は mutation を実行せず、監査なし作成を防ぐため fail-close にする。
	if s.accountCreation == nil {
		return adminopenapi.CreateAdminAccount503JSONResponse{Body: authFailureResponseWithRequestID(adminopenapi.InternalError, requestID), Headers: adminopenapi.CreateAdminAccount503ResponseHeaders{CacheControl: noStoreValue}}, nil
	}

	// Step 4: middleware が検証済み session から束縛した operator context と request body を application DTO へ変換し、generated 型や Gin 型を use case に渡さない。
	input, ok := accountCreationInput(ctx, *request.Body, requestID)
	if !ok {
		return adminopenapi.CreateAdminAccount401JSONResponse{Body: authFailureResponseWithRequestID(adminopenapi.Unauthenticated, requestID), Headers: adminopenapi.CreateAdminAccount401ResponseHeaders{CacheControl: noStoreValue}}, nil
	}

	// Step 5: email 正規化、Account 初期状態、RBAC、audit は application/domain の責務として use case へ委譲し、handler は HTTP 写像だけを担う。
	created, err := s.accountCreation.CreateAccount(ctx, input)
	if err != nil {
		return createAdminAccountFailure(requestID, err), nil
	}

	// Step 6: application DTO を Admin OpenAPI response DTO へ変換し、domain/repository 型を transport 層に漏らさない。
	return adminopenapi.CreateAdminAccount201JSONResponse{Body: adminCreateAccountResponse(created), Headers: adminopenapi.CreateAdminAccount201ResponseHeaders{CacheControl: noStoreValue}}, nil
}

func (s strictServer) GetAdminAccount(ctx context.Context, request adminopenapi.GetAdminAccountRequestObject) (adminopenapi.GetAdminAccountResponseObject, error) {
	// Step 1: detail response と error response 用の追跡 ID を生成し、一覧 API と同じ correlation 形式に揃える。
	requestID := nextAdminRequestID()

	// Step 2: account search/detail use case 未接続時は read model を公開せず、Admin API を fail-close にする。
	if s.accountSearch == nil {
		return adminopenapi.GetAdminAccount503JSONResponse{Body: authFailureResponseWithRequestID(adminopenapi.InternalError, requestID), Headers: adminopenapi.GetAdminAccount503ResponseHeaders{CacheControl: noStoreValue}}, nil
	}

	// Step 3: generated path parameter を application DTO へ詰め替え、永続化 query は application/repository へ委譲する。
	result, err := s.accountSearch.GetAccount(ctx, accountsapplication.AccountDetailInput{AccountID: string(request.AccountId), RequestID: requestID})
	if err != nil {
		return getAdminAccountFailure(requestID, err), nil
	}

	// Step 4: application read model を Admin OpenAPI response DTO へ変換し、repository/generated 型を相互に漏らさない。
	return adminopenapi.GetAdminAccount200JSONResponse{Body: adminAccountDetailResponse(result), Headers: adminopenapi.GetAdminAccount200ResponseHeaders{CacheControl: noStoreValue}}, nil
}

func (s strictServer) FinishAdminInitialSetup(ctx context.Context, request adminopenapi.FinishAdminInitialSetupRequestObject) (adminopenapi.FinishAdminInitialSetupResponseObject, error) {
	// Step 1: setup 完了用の request ID は body の ceremony ID と一致させ、error response でも correlation を維持する。
	requestID := nextAdminRequestID()
	if request.Body != nil && request.Body.RequestId != "" {
		requestID = request.Body.RequestId
	}
	if request.Body == nil {
		return adminopenapi.FinishAdminInitialSetup400JSONResponse{Body: authOperationError(requestID, adminInvalidRequestBodyMessage), Headers: adminopenapi.FinishAdminInitialSetup400ResponseHeaders{CacheControl: noStoreValue}}, nil
	}
	if s.operatorSetup == nil {
		return adminopenapi.FinishAdminInitialSetup503JSONResponse{Body: authFailureResponseWithRequestID(adminopenapi.InternalError, requestID), Headers: adminopenapi.FinishAdminInitialSetup503ResponseHeaders{CacheControl: noStoreValue}}, nil
	}

	// Step 2: credentialMode は handler で transport 出力境界だけに使い、session 発行 rule は application use case へ閉じる。
	mode, ok := adminSessionCredentialMode(request.Body.CredentialMode)
	if !ok {
		return adminopenapi.FinishAdminInitialSetup400JSONResponse{Body: authOperationError(requestID, adminInvalidRequestBodyMessage), Headers: adminopenapi.FinishAdminInitialSetup400ResponseHeaders{CacheControl: noStoreValue}}, nil
	}

	// Step 3: bootstrap secret と WebAuthn attestation は application use case に渡し、handler は token/hash/challenge 判定を行わない。
	result, err := s.operatorSetup.FinishInitialSetup(ctx, operatorsapplication.InitialSetupFinishInput{Email: string(request.Body.Email), DisplayName: request.Body.DisplayName, BootstrapSecret: request.Body.BootstrapSecret, RequestID: request.Body.RequestId, Credential: adminAttestationCredentialDTO(request.Body.Credential)})
	if err != nil {
		return finishAdminInitialSetupFailure(requestID, err), nil
	}
	if !adminSessionResultHasSecrets(result) {
		return adminopenapi.FinishAdminInitialSetup503JSONResponse{Body: authFailureResponseWithRequestID(adminopenapi.InternalError, requestID), Headers: adminopenapi.FinishAdminInitialSetup503ResponseHeaders{CacheControl: noStoreValue}}, nil
	}
	body, responseErr := adminAuthSessionUnionResponse(result, requestID, mode)
	if responseErr != nil {
		return adminopenapi.FinishAdminInitialSetup503JSONResponse{Body: authFailureResponseWithRequestID(adminopenapi.InternalError, requestID), Headers: adminopenapi.FinishAdminInitialSetup503ResponseHeaders{CacheControl: noStoreValue}}, nil
	}
	if !applyAdminRefreshCredential(ctx, result.RefreshCookie, mode) {
		return adminopenapi.FinishAdminInitialSetup503JSONResponse{Body: authFailureResponseWithRequestID(adminopenapi.InternalError, requestID), Headers: adminopenapi.FinishAdminInitialSetup503ResponseHeaders{CacheControl: noStoreValue}}, nil
	}
	return adminopenapi.FinishAdminInitialSetup200JSONResponse{Body: body, Headers: adminopenapi.FinishAdminInitialSetup200ResponseHeaders{CacheControl: noStoreValue}}, nil
}

func (s strictServer) StartAdminInitialSetup(ctx context.Context, request adminopenapi.StartAdminInitialSetupRequestObject) (adminopenapi.StartAdminInitialSetupResponseObject, error) {
	// Step 1: 初回 setup ceremony ID を ULID として生成し、WebAuthn session lookup と response requestId に同じ値を使う。
	requestID := nextAdminRequestID()
	if request.Body == nil {
		return adminopenapi.StartAdminInitialSetup400JSONResponse{Body: authOperationError(requestID, adminInvalidRequestBodyMessage), Headers: adminopenapi.StartAdminInitialSetup400ResponseHeaders{CacheControl: noStoreValue}}, nil
	}
	if s.operatorSetup == nil {
		return adminopenapi.StartAdminInitialSetup503JSONResponse{Body: authFailureResponseWithRequestID(adminopenapi.InternalError, requestID), Headers: adminopenapi.StartAdminInitialSetup503ResponseHeaders{CacheControl: noStoreValue}}, nil
	}

	// Step 2: bootstrap 検証、operator 0 件確認、challenge 発行を application use case へ委譲する。
	challenge, err := s.operatorSetup.StartInitialSetup(ctx, operatorsapplication.InitialSetupStartInput{Email: string(request.Body.Email), DisplayName: request.Body.DisplayName, BootstrapSecret: request.Body.BootstrapSecret, RequestID: requestID})
	if err != nil {
		return startAdminInitialSetupFailure(requestID, err), nil
	}
	body, err := adminPasskeyAddStartResponse(challenge)
	if err != nil {
		return adminopenapi.StartAdminInitialSetup503JSONResponse{Body: authFailureResponseWithRequestID(adminopenapi.InternalError, requestID), Headers: adminopenapi.StartAdminInitialSetup503ResponseHeaders{CacheControl: noStoreValue}}, nil
	}
	return adminopenapi.StartAdminInitialSetup200JSONResponse{Body: body, Headers: adminopenapi.StartAdminInitialSetup200ResponseHeaders{CacheControl: noStoreValue}}, nil
}

func (s strictServer) FinishAdminOperatorSetup(ctx context.Context, request adminopenapi.FinishAdminOperatorSetupRequestObject) (adminopenapi.FinishAdminOperatorSetupResponseObject, error) {
	// Step 1: operator setup 用 request ID は start response と同じ body.requestId を優先する。
	requestID := nextAdminRequestID()
	if request.Body != nil && request.Body.RequestId != "" {
		requestID = request.Body.RequestId
	}
	if request.Body == nil {
		return adminopenapi.FinishAdminOperatorSetup400JSONResponse{Body: authOperationError(requestID, adminInvalidRequestBodyMessage), Headers: adminopenapi.FinishAdminOperatorSetup400ResponseHeaders{CacheControl: noStoreValue}}, nil
	}
	if s.operatorSetup == nil {
		return adminopenapi.FinishAdminOperatorSetup503JSONResponse{Body: authFailureResponseWithRequestID(adminopenapi.InternalError, requestID), Headers: adminopenapi.FinishAdminOperatorSetup503ResponseHeaders{CacheControl: noStoreValue}}, nil
	}

	// Step 2: credentialMode は handler で transport 出力境界だけに使い、session 発行 rule は application use case へ閉じる。
	mode, ok := adminSessionCredentialMode(request.Body.CredentialMode)
	if !ok {
		return adminopenapi.FinishAdminOperatorSetup400JSONResponse{Body: authOperationError(requestID, adminInvalidRequestBodyMessage), Headers: adminopenapi.FinishAdminOperatorSetup400ResponseHeaders{CacheControl: noStoreValue}}, nil
	}

	// Step 3: setup token consumption と attestation verification は application use case に集約する。
	result, err := s.operatorSetup.FinishOperatorSetup(ctx, operatorsapplication.SetupFinishInput{SetupToken: request.Body.SetupToken, RequestID: request.Body.RequestId, Credential: adminAttestationCredentialDTO(request.Body.Credential)})
	if err != nil {
		return finishAdminOperatorSetupFailure(requestID, err), nil
	}
	if !adminSessionResultHasSecrets(result) {
		return adminopenapi.FinishAdminOperatorSetup503JSONResponse{Body: authFailureResponseWithRequestID(adminopenapi.InternalError, requestID), Headers: adminopenapi.FinishAdminOperatorSetup503ResponseHeaders{CacheControl: noStoreValue}}, nil
	}
	body, responseErr := adminAuthSessionUnionResponse(result, requestID, mode)
	if responseErr != nil {
		return adminopenapi.FinishAdminOperatorSetup503JSONResponse{Body: authFailureResponseWithRequestID(adminopenapi.InternalError, requestID), Headers: adminopenapi.FinishAdminOperatorSetup503ResponseHeaders{CacheControl: noStoreValue}}, nil
	}
	if !applyAdminRefreshCredential(ctx, result.RefreshCookie, mode) {
		return adminopenapi.FinishAdminOperatorSetup503JSONResponse{Body: authFailureResponseWithRequestID(adminopenapi.InternalError, requestID), Headers: adminopenapi.FinishAdminOperatorSetup503ResponseHeaders{CacheControl: noStoreValue}}, nil
	}
	return adminopenapi.FinishAdminOperatorSetup200JSONResponse{Body: body, Headers: adminopenapi.FinishAdminOperatorSetup200ResponseHeaders{CacheControl: noStoreValue}}, nil
}

func (s strictServer) StartAdminOperatorSetup(ctx context.Context, request adminopenapi.StartAdminOperatorSetupRequestObject) (adminopenapi.StartAdminOperatorSetupResponseObject, error) {
	// Step 1: operator setup ceremony ID を ULID として生成し、WebAuthn session lookup と response requestId に同じ値を使う。
	requestID := nextAdminRequestID()
	if request.Body == nil {
		return adminopenapi.StartAdminOperatorSetup400JSONResponse{Body: authOperationError(requestID, adminInvalidRequestBodyMessage), Headers: adminopenapi.StartAdminOperatorSetup400ResponseHeaders{CacheControl: noStoreValue}}, nil
	}
	if s.operatorSetup == nil {
		return adminopenapi.StartAdminOperatorSetup503JSONResponse{Body: authFailureResponseWithRequestID(adminopenapi.InternalError, requestID), Headers: adminopenapi.StartAdminOperatorSetup503ResponseHeaders{CacheControl: noStoreValue}}, nil
	}

	// Step 2: setup token 検証と challenge 発行を application use case に委譲し、handler は token 状態を区別しない。
	challenge, err := s.operatorSetup.StartOperatorSetup(ctx, operatorsapplication.SetupStartInput{SetupToken: request.Body.SetupToken, RequestID: requestID})
	if err != nil {
		return startAdminOperatorSetupFailure(requestID, err), nil
	}
	body, err := adminPasskeyAddStartResponse(challenge)
	if err != nil {
		return adminopenapi.StartAdminOperatorSetup503JSONResponse{Body: authFailureResponseWithRequestID(adminopenapi.InternalError, requestID), Headers: adminopenapi.StartAdminOperatorSetup503ResponseHeaders{CacheControl: noStoreValue}}, nil
	}
	return adminopenapi.StartAdminOperatorSetup200JSONResponse{Body: body, Headers: adminopenapi.StartAdminOperatorSetup200ResponseHeaders{CacheControl: noStoreValue}}, nil
}

func (s strictServer) CreateAdminOperator(ctx context.Context, request adminopenapi.CreateAdminOperatorRequestObject) (adminopenapi.CreateAdminOperatorResponseObject, error) {
	// Step 1: operator 作成 request ごとの追跡 ID を生成し、audit correlation と response requestId を一致させる。
	requestID := nextAdminRequestID()
	if request.Body == nil {
		return adminopenapi.CreateAdminOperator400JSONResponse{Body: authOperationError(requestID, adminInvalidRequestBodyMessage), Headers: adminopenapi.CreateAdminOperator400ResponseHeaders{CacheControl: noStoreValue}}, nil
	}
	if s.operatorSetup == nil {
		return adminopenapi.CreateAdminOperator503JSONResponse{Body: authFailureResponseWithRequestID(adminopenapi.InternalError, requestID), Headers: adminopenapi.CreateAdminOperator503ResponseHeaders{CacheControl: noStoreValue}}, nil
	}

	// Step 2: middleware が検証済み session から束縛した operator context と request body を application DTO へ変換し、handler に RBAC rule を置かない。
	input, ok := operatorCreationInput(ctx, *request.Body, requestID)
	if !ok {
		return adminopenapi.CreateAdminOperator401JSONResponse{Body: authFailureResponseWithRequestID(adminopenapi.Unauthenticated, requestID), Headers: adminopenapi.CreateAdminOperator401ResponseHeaders{CacheControl: noStoreValue}}, nil
	}
	created, err := s.operatorSetup.CreateOperator(ctx, input)
	if err != nil {
		return createAdminOperatorFailure(requestID, err), nil
	}

	// Step 3: setup token 平文を response に含めず、operator summary、delivery status、audit correlation だけを返す。
	return adminopenapi.CreateAdminOperator201JSONResponse{Body: adminCreateOperatorResponse(created), Headers: adminopenapi.CreateAdminOperator201ResponseHeaders{CacheControl: noStoreValue}}, nil
}

func (s strictServer) GetCurrentAdminOperator(ctx context.Context, _ adminopenapi.GetCurrentAdminOperatorRequestObject) (adminopenapi.GetCurrentAdminOperatorResponseObject, error) {
	// Step 1: current operator request ごとの追跡 ID を先に生成し、成功/失敗 response の correlation を揃える。
	requestID := nextAdminRequestID()

	// Step 2: application auth service 未接続では operator profile を公開せず、Admin auth boundary を fail-closed にする。
	if s.operatorAuth == nil {
		return adminopenapi.GetCurrentAdminOperator503JSONResponse{Body: authFailureResponseWithRequestID(adminopenapi.InternalError, requestID), Headers: adminopenapi.GetCurrentAdminOperator503ResponseHeaders{CacheControl: noStoreValue}}, nil
	}

	// Step 3: Authorization header から transport token だけを抽出し、署名/session/snapshot 検証は application service へ委譲する。
	accessToken, ok := adminAccessTokenFromContext(ctx)
	if !ok {
		return adminopenapi.GetCurrentAdminOperator401JSONResponse{Body: authFailureResponseWithRequestID(adminopenapi.Unauthenticated, requestID), Headers: adminopenapi.GetCurrentAdminOperator401ResponseHeaders{CacheControl: noStoreValue}}, nil
	}
	operator, err := s.operatorAuth.CurrentOperator(ctx, adminauth.CurrentOperatorInput{AccessToken: accessToken})
	if err != nil {
		return currentAdminOperatorFailure(requestID, err), nil
	}

	// Step 4: application DTO を generated response DTO へ詰め替え、role/active の意味づけは handler で再判定しない。
	return adminopenapi.GetCurrentAdminOperator200JSONResponse{Body: adminCurrentOperatorResponse(operator, requestID), Headers: adminopenapi.GetCurrentAdminOperator200ResponseHeaders{CacheControl: noStoreValue}}, nil
}

func (s strictServer) LogoutAdminOperator(ctx context.Context, _ adminopenapi.LogoutAdminOperatorRequestObject) (adminopenapi.LogoutAdminOperatorResponseObject, error) {
	// Step 1: logout response と error response 用の追跡 ID を生成し、CSRF/header 検証済み request の監査性を保つ。
	requestID := nextAdminRequestID()

	// Step 2: logout は session revoke 副作用を持つため、application auth service 未接続なら副作用なしで 503 にする。
	if s.operatorAuth == nil {
		return adminopenapi.LogoutAdminOperator503JSONResponse{Body: authFailureResponseWithRequestID(adminopenapi.InternalError, requestID), Headers: adminopenapi.LogoutAdminOperator503ResponseHeaders{CacheControl: noStoreValue}}, nil
	}

	// Step 3: middleware が検証した bearer を再利用して、revoke 対象 session の特定は application service へ委譲する。
	accessToken, ok := adminAccessTokenFromContext(ctx)
	if !ok {
		return adminopenapi.LogoutAdminOperator401JSONResponse{Body: authFailureResponseWithRequestID(adminopenapi.Unauthenticated, requestID), Headers: adminopenapi.LogoutAdminOperator401ResponseHeaders{CacheControl: noStoreValue}}, nil
	}
	command, err := s.operatorAuth.LogoutOperator(ctx, adminauth.LogoutOperatorInput{AccessToken: accessToken})
	if err != nil {
		return logoutAdminOperatorFailure(requestID, err), nil
	}
	if !setAdminRefreshCookie(ctx, command) {
		return adminopenapi.LogoutAdminOperator503JSONResponse{Body: authFailureResponseWithRequestID(adminopenapi.InternalError, requestID), Headers: adminopenapi.LogoutAdminOperator503ResponseHeaders{CacheControl: noStoreValue}}, nil
	}

	// Step 4: refresh Cookie 削除指示を header に出したうえで、body には revoke 成否と requestId だけを返す。
	return adminopenapi.LogoutAdminOperator200JSONResponse{Body: adminopenapi.WWWTemplateLogoutResponse{RequestId: requestID, Revoked: true}, Headers: adminopenapi.LogoutAdminOperator200ResponseHeaders{CacheControl: noStoreValue}}, nil
}

func (s strictServer) ListAdminOperatorPasskeys(ctx context.Context, _ adminopenapi.ListAdminOperatorPasskeysRequestObject) (adminopenapi.ListAdminOperatorPasskeysResponseObject, error) {
	// Step 1: passkey 一覧 response と error response 用の追跡 ID を生成し、Bearer 検証済み request の監査性を保つ。
	requestID := nextAdminRequestID()

	// Step 2: passkey 管理 use case 未接続では credential metadata を公開せず、Admin auth boundary を fail-closed にする。
	if s.operatorPasskeyManagement == nil {
		return adminopenapi.ListAdminOperatorPasskeys503JSONResponse{Body: authFailureResponseWithRequestID(adminopenapi.InternalError, requestID), Headers: adminopenapi.ListAdminOperatorPasskeys503ResponseHeaders{CacheControl: noStoreValue}}, nil
	}

	// Step 3: middleware が検証済み session から束縛した OperatorID だけを使い、Product account ID や request body 由来の所有者指定を受け付けない。
	operatorID, ok := contextString(ctx, adminContextKeyOperatorID)
	if !ok {
		return adminopenapi.ListAdminOperatorPasskeys401JSONResponse{Body: authFailureResponseWithRequestID(adminopenapi.Unauthenticated, requestID), Headers: adminopenapi.ListAdminOperatorPasskeys401ResponseHeaders{CacheControl: noStoreValue}}, nil
	}

	// Step 4: application service に Admin operator ID だけを渡し、保存層選択と非秘匿 DTO 化を use case に委譲する。
	result, err := s.operatorPasskeyManagement.ListOperatorPasskeys(ctx, adminauth.ListOperatorPasskeysInput{OperatorID: operatorID})
	if err != nil {
		return listAdminOperatorPasskeysFailure(requestID, err), nil
	}

	// Step 5: application DTO を generated Admin DTO へ詰め替え、credential handle や public key を response に含めない。
	return adminopenapi.ListAdminOperatorPasskeys200JSONResponse{Body: adminOperatorPasskeyListResponse(result, requestID), Headers: adminopenapi.ListAdminOperatorPasskeys200ResponseHeaders{CacheControl: noStoreValue}}, nil
}

func (s strictServer) DeleteAdminOperatorPasskey(ctx context.Context, request adminopenapi.DeleteAdminOperatorPasskeyRequestObject) (adminopenapi.DeleteAdminOperatorPasskeyResponseObject, error) {
	// Step 1: passkey 削除 response と error response 用の追跡 ID を生成し、Bearer 検証済み request の監査性を保つ。
	requestID := nextAdminRequestID()

	// Step 2: passkey 管理 use case 未接続では credential を削除せず、Admin auth boundary を fail-closed にする。
	if s.operatorPasskeyManagement == nil {
		return adminopenapi.DeleteAdminOperatorPasskey503JSONResponse{Body: authFailureResponseWithRequestID(adminopenapi.InternalError, requestID), Headers: adminopenapi.DeleteAdminOperatorPasskey503ResponseHeaders{CacheControl: noStoreValue}}, nil
	}

	// Step 3: middleware が検証済み session から束縛した OperatorID だけを使い、他 Operator の credential 削除を防ぐ。
	operatorID, ok := contextString(ctx, adminContextKeyOperatorID)
	if !ok {
		return adminopenapi.DeleteAdminOperatorPasskey401JSONResponse{Body: authFailureResponseWithRequestID(adminopenapi.Unauthenticated, requestID), Headers: adminopenapi.DeleteAdminOperatorPasskey401ResponseHeaders{CacheControl: noStoreValue}}, nil
	}

	// Step 4: path の credential ID と検証済み OperatorID を application DTO へ渡し、最後の 1 件削除 rule は domain/application に委譲する。
	if err := s.operatorPasskeyManagement.DeleteOperatorPasskey(ctx, adminauth.DeleteOperatorPasskeyInput{OperatorID: operatorID, PasskeyID: string(request.Id)}); err != nil {
		return deleteAdminOperatorPasskeyFailure(requestID, err), nil
	}

	// Step 5: 削除成功時は body を返さず、no-store header だけを付与する。
	return adminopenapi.DeleteAdminOperatorPasskey204Response{Headers: adminopenapi.DeleteAdminOperatorPasskey204ResponseHeaders{CacheControl: noStoreValue}}, nil
}

func (s strictServer) RefreshAdminOperatorSession(ctx context.Context, request adminopenapi.RefreshAdminOperatorSessionRequestObject) (adminopenapi.RefreshAdminOperatorSessionResponseObject, error) {
	// Step 1: refresh / CSRF 再発行用の request ID を生成し、Cookie 欠落時も non-secret response に限定する。
	requestID := nextAdminRequestID()

	// Step 2: refresh rotation service 未接続なら Cookie を読んでも使わず、session 発行を fail-closed にする。
	if s.operatorAuth == nil {
		return adminopenapi.RefreshAdminOperatorSession503JSONResponse{Body: authFailureResponseWithRequestID(adminopenapi.InternalError, requestID), Headers: adminopenapi.RefreshAdminOperatorSession503ResponseHeaders{CacheControl: noStoreValue}}, nil
	}

	// Step 3: context refresh は accessToken Authorization header を refresh credential として扱わず、同時提示時も fail-close にする。
	if adminAuthorizationHeaderPresent(ctx) {
		return adminopenapi.RefreshAdminOperatorSession401JSONResponse{Body: authFailureResponseWithRequestID(adminopenapi.Unauthenticated, requestID), Headers: adminopenapi.RefreshAdminOperatorSession401ResponseHeaders{CacheControl: noStoreValue}}, nil
	}

	// Step 4: Cookie mode と Bearer mode の refresh credential を exactly-one で抽出し、ambiguous request を rotation 前に拒否する。
	refreshCredential, mode, ok := adminRefreshCredential(ctx, request.Body)
	if !ok {
		return adminopenapi.RefreshAdminOperatorSession401JSONResponse{Body: authFailureResponseWithRequestID(adminopenapi.Unauthenticated, requestID), Headers: adminopenapi.RefreshAdminOperatorSession401ResponseHeaders{CacheControl: noStoreValue}}, nil
	}
	result, err := s.operatorAuth.RefreshOperatorSession(ctx, adminauth.RefreshOperatorSessionInput{AuthContextID: request.AuthContextId, RefreshTokenValue: refreshCredential})
	if err != nil {
		return refreshAdminOperatorSessionFailure(requestID, err), nil
	}
	if !adminSessionResultHasSecrets(result) {
		return adminopenapi.RefreshAdminOperatorSession503JSONResponse{Body: authFailureResponseWithRequestID(adminopenapi.InternalError, requestID), Headers: adminopenapi.RefreshAdminOperatorSession503ResponseHeaders{CacheControl: noStoreValue}}, nil
	}

	// Step 5: credential mode ごとの body shape を生成し、Cookie mode だけ refreshToken 平文を body から除外する。
	body, responseErr := adminContextRefreshUnionResponse(result, requestID, mode)
	if responseErr != nil {
		return adminopenapi.RefreshAdminOperatorSession503JSONResponse{Body: authFailureResponseWithRequestID(adminopenapi.InternalError, requestID), Headers: adminopenapi.RefreshAdminOperatorSession503ResponseHeaders{CacheControl: noStoreValue}}, nil
	}
	if !applyAdminRefreshCredential(ctx, result.RefreshCookie, mode) {
		return adminopenapi.RefreshAdminOperatorSession503JSONResponse{Body: authFailureResponseWithRequestID(adminopenapi.InternalError, requestID), Headers: adminopenapi.RefreshAdminOperatorSession503ResponseHeaders{CacheControl: noStoreValue}}, nil
	}
	return adminopenapi.RefreshAdminOperatorSession200JSONResponse{Body: body, Headers: adminopenapi.RefreshAdminOperatorSession200ResponseHeaders{CacheControl: noStoreValue}}, nil
}

func (s strictServer) FinishAdminPasskeyAuthentication(ctx context.Context, request adminopenapi.FinishAdminPasskeyAuthenticationRequestObject) (adminopenapi.FinishAdminPasskeyAuthenticationResponseObject, error) {
	// Step 1: body がない場合の error correlation 用に、先に fallback request ID を生成する。
	requestID := nextAdminRequestID()
	if request.Body != nil && request.Body.RequestId != "" {
		requestID = request.Body.RequestId
	}

	// Step 2: generated binding の通常経路外で nil body が来ても、application service に空 credential を渡さず 400 にする。
	if request.Body == nil {
		return adminopenapi.FinishAdminPasskeyAuthentication400JSONResponse{Body: authOperationError(requestID, adminInvalidRequestBodyMessage), Headers: adminopenapi.FinishAdminPasskeyAuthentication400ResponseHeaders{CacheControl: noStoreValue}}, nil
	}

	// Step 3: Admin passkey login service または WebAuthn verifier 未接続では credential を処理せず、session/Cookie 発行を fail-closed にする。
	if s.operatorPasskeyAuth == nil || s.operatorPasskeys == nil {
		return adminopenapi.FinishAdminPasskeyAuthentication503JSONResponse{Body: authFailureResponseWithRequestID(adminopenapi.InternalError, requestID), Headers: adminopenapi.FinishAdminPasskeyAuthentication503ResponseHeaders{CacheControl: noStoreValue}}, nil
	}

	// Step 4: assertion credential は verifier に渡して署名/challenge/user verification を確認し、検証済み handle だけを auth service に渡す。
	// requestId は追跡 ID のため challenge lookup key としては使わず、provider が clientDataJSON 内の challenge から session selector を自己解決する。
	credentialHandle, err := s.operatorPasskeys.VerifyOperatorPasskey(ctx, "", adminAssertionCredentialDTO(request.Body.Credential))
	if err != nil {
		return finishAdminPasskeyAuthenticationFailure(requestID, err), nil
	}

	// Step 5: credentialMode は handler で transport 出力境界だけに使い、session 発行 rule は application use case へ閉じる。
	mode, ok := adminSessionCredentialMode(request.Body.CredentialMode)
	if !ok {
		return adminopenapi.FinishAdminPasskeyAuthentication400JSONResponse{Body: authOperationError(requestID, adminInvalidRequestBodyMessage), Headers: adminopenapi.FinishAdminPasskeyAuthentication400ResponseHeaders{CacheControl: noStoreValue}}, nil
	}

	// Step 6: 検証済み credential handle だけを application DTO へ写像し、operator/session rule は service に委譲する。
	result, err := s.operatorPasskeyAuth.FinishOperatorPasskey(ctx, adminauth.FinishOperatorPasskeyInput{ChallengeID: request.Body.RequestId, CredentialHandle: credentialHandle})
	if err != nil {
		return finishAdminPasskeyAuthenticationFailure(requestID, err), nil
	}
	if !adminSessionResultHasSecrets(result) {
		return adminopenapi.FinishAdminPasskeyAuthentication503JSONResponse{Body: authFailureResponseWithRequestID(adminopenapi.InternalError, requestID), Headers: adminopenapi.FinishAdminPasskeyAuthentication503ResponseHeaders{CacheControl: noStoreValue}}, nil
	}

	// Step 7: credential mode ごとの body shape を生成し、Cookie mode だけ refreshToken 平文を body から除外する。
	body, responseErr := adminAuthSessionUnionResponse(result, requestID, mode)
	if responseErr != nil {
		return adminopenapi.FinishAdminPasskeyAuthentication503JSONResponse{Body: authFailureResponseWithRequestID(adminopenapi.InternalError, requestID), Headers: adminopenapi.FinishAdminPasskeyAuthentication503ResponseHeaders{CacheControl: noStoreValue}}, nil
	}
	if !applyAdminRefreshCredential(ctx, result.RefreshCookie, mode) {
		return adminopenapi.FinishAdminPasskeyAuthentication503JSONResponse{Body: authFailureResponseWithRequestID(adminopenapi.InternalError, requestID), Headers: adminopenapi.FinishAdminPasskeyAuthentication503ResponseHeaders{CacheControl: noStoreValue}}, nil
	}
	return adminopenapi.FinishAdminPasskeyAuthentication200JSONResponse{Body: body, Headers: adminopenapi.FinishAdminPasskeyAuthentication200ResponseHeaders{CacheControl: noStoreValue}}, nil
}

func (s strictServer) StartAdminPasskeyAuthentication(ctx context.Context, request adminopenapi.StartAdminPasskeyAuthenticationRequestObject) (adminopenapi.StartAdminPasskeyAuthenticationResponseObject, error) {
	// Step 1: passkey challenge request ごとの request ID を生成し、body 不備や service error でも同じ形の response にする。
	requestID := nextAdminRequestID()

	// Step 2: nil body は transport DTO 不備として 400 にし、identifier の業務的扱いは provider/application 側へ残す。
	if request.Body == nil {
		return adminopenapi.StartAdminPasskeyAuthentication400JSONResponse{Body: authOperationError(requestID, adminInvalidRequestBodyMessage), Headers: adminopenapi.StartAdminPasskeyAuthentication400ResponseHeaders{CacheControl: noStoreValue}}, nil
	}

	// Step 3: challenge provider を持つ application service が未接続なら、認証 ceremony を開始せず fail-closed にする。
	if s.operatorPasskeyAuth == nil {
		return adminopenapi.StartAdminPasskeyAuthentication503JSONResponse{Body: authFailureResponseWithRequestID(adminopenapi.InternalError, requestID), Headers: adminopenapi.StartAdminPasskeyAuthentication503ResponseHeaders{CacheControl: noStoreValue}}, nil
	}

	// Step 4: handler は identifier を正規化せず application DTO へ渡し、存在確認や challenge 発行 rule を service/provider に委譲する。
	challenge, err := s.operatorPasskeyAuth.StartOperatorPasskey(ctx, adminauth.StartOperatorPasskeyInput{Identifier: request.Body.Identifier})
	if err != nil {
		return startAdminPasskeyAuthenticationFailure(requestID, err), nil
	}
	// Step 5: requestId は追跡 ID のまま返し、WebAuthn session selector は browser が返す clientDataJSON.challenge から verifier が復元する。
	body := adminopenapi.WWWTemplatePasskeyStartResponse{RequestId: requestID, Challenge: challenge.Challenge, RpId: challenge.WebAuthnRPID, UserVerification: "required"}
	applyAdminWebAuthnLoginOptions(&body, challenge.WebAuthnOptions)
	if body.RequestId == "" || body.Challenge == "" || body.RpId == "" {
		return adminopenapi.StartAdminPasskeyAuthentication503JSONResponse{Body: authFailureResponseWithRequestID(adminopenapi.InternalError, requestID), Headers: adminopenapi.StartAdminPasskeyAuthentication503ResponseHeaders{CacheControl: noStoreValue}}, nil
	}

	// Step 6: generated response DTO だけを返し、session secret は start response に含めない。
	return adminopenapi.StartAdminPasskeyAuthentication200JSONResponse{Body: body, Headers: adminopenapi.StartAdminPasskeyAuthentication200ResponseHeaders{CacheControl: noStoreValue}}, nil
}

type adminWebAuthnLoginOptionsJSON struct {
	PublicKey struct {
		AllowCredentials []struct {
			ID         string   `json:"id"`
			Type       string   `json:"type"`
			Transports []string `json:"transports,omitempty"`
		} `json:"allowCredentials,omitempty"`
		Timeout          *int64 `json:"timeout,omitempty"`
		UserVerification string `json:"userVerification,omitempty"`
	} `json:"publicKey"`
}

type adminWebAuthnRegistrationOptionsJSON struct {
	PublicKey struct {
		RP struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"rp"`
		User struct {
			ID          any    `json:"id"`
			Name        string `json:"name"`
			DisplayName string `json:"displayName"`
		} `json:"user"`
		PubKeyCredParams []struct {
			Type string `json:"type"`
			Alg  int32  `json:"alg"`
		} `json:"pubKeyCredParams,omitempty"`
		ExcludeCredentials []struct {
			ID         string   `json:"id"`
			Type       string   `json:"type"`
			Transports []string `json:"transports,omitempty"`
		} `json:"excludeCredentials,omitempty"`
		AuthenticatorSelection struct {
			RequireResidentKey *bool  `json:"requireResidentKey,omitempty"`
			ResidentKey        string `json:"residentKey,omitempty"`
			UserVerification   string `json:"userVerification,omitempty"`
		} `json:"authenticatorSelection"`
		Attestation string `json:"attestation,omitempty"`
		Timeout     *int64 `json:"timeout,omitempty"`
	} `json:"publicKey"`
}

func adminPasskeyAddStartResponse(challenge operatorsapplication.SetupChallengeResult) (adminopenapi.WWWTemplatePasskeyAddStartResponse, error) {
	// Step 1: WebAuthn provider が生成した JSON options を response DTO へ fail-closed に写像する。
	body := adminopenapi.WWWTemplatePasskeyAddStartResponse{RequestId: challenge.RequestID, Challenge: challenge.Challenge}
	options, err := adminRegistrationOptionsFromChallenge(challenge.OptionsJSON)
	if err != nil {
		return adminopenapi.WWWTemplatePasskeyAddStartResponse{}, err
	}
	if err := applyAdminRegistrationRequiredOptions(&body, options); err != nil {
		return adminopenapi.WWWTemplatePasskeyAddStartResponse{}, err
	}
	applyAdminRegistrationExcludedCredentials(&body, options)
	applyAdminRegistrationOptionalOptions(&body, options)
	return body, nil
}

func adminRegistrationOptionsFromChallenge(optionsJSON []byte) (adminWebAuthnRegistrationOptionsJSON, error) {
	// Step 1: WebAuthn provider の options JSON は provider 側が作る正規形式だけを受け入れ、空や破損 JSON は setup ceremony を止める。
	var options adminWebAuthnRegistrationOptionsJSON
	if len(optionsJSON) == 0 || json.Unmarshal(optionsJSON, &options) != nil {
		return adminWebAuthnRegistrationOptionsJSON{}, errors.New("invalid webauthn registration options")
	}
	return options, nil
}

func applyAdminRegistrationRequiredOptions(body *adminopenapi.WWWTemplatePasskeyAddStartResponse, options adminWebAuthnRegistrationOptionsJSON) error {
	// Step 1: registration に必須の RP/User/credential parameter が欠けている場合は、部分的な challenge を browser へ返さず fail-close にする。
	pk := options.PublicKey
	if pk.RP.ID == "" || pk.RP.Name == "" || pk.User.Name == "" || pk.User.DisplayName == "" || len(pk.PubKeyCredParams) == 0 {
		return errors.New("incomplete webauthn registration options")
	}
	userID, ok := pk.User.ID.(string)
	if !ok || userID == "" {
		return errors.New("missing webauthn user id")
	}
	body.RpId = pk.RP.ID
	body.RpName = pk.RP.Name
	body.User = adminopenapi.WWWTemplateWebAuthnUserEntity{Id: userID, Name: pk.User.Name, DisplayName: pk.User.DisplayName}
	body.PubKeyCredParams = make([]adminopenapi.WWWTemplateWebAuthnCredentialParameter, 0, len(pk.PubKeyCredParams))
	for _, param := range pk.PubKeyCredParams {
		body.PubKeyCredParams = append(body.PubKeyCredParams, adminopenapi.WWWTemplateWebAuthnCredentialParameter{Type: param.Type, Alg: param.Alg})
	}

	// Step 2: Admin operator login も discoverable credential 前提のため、registration options が resident key を必須化していなければ fail-close にする。
	if pk.AuthenticatorSelection.RequireResidentKey == nil || !*pk.AuthenticatorSelection.RequireResidentKey {
		return errors.New("webauthn options require discoverable credential")
	}
	if pk.AuthenticatorSelection.ResidentKey != "required" {
		return errors.New("webauthn options missing required residentKey")
	}
	if pk.AuthenticatorSelection.UserVerification != "required" {
		return errors.New("webauthn options missing required userVerification")
	}

	// Step 3: Browser API が同じ authenticatorSelection を復元できるよう、contract の literal fields として response body に固定値を設定する。
	body.RequireResidentKey = true
	body.ResidentKey = "required"
	body.UserVerification = adminopenapi.WWWTemplatePasskeyAddStartResponseUserVerificationRequired
	return nil
}

func applyAdminRegistrationExcludedCredentials(body *adminopenapi.WWWTemplatePasskeyAddStartResponse, options adminWebAuthnRegistrationOptionsJSON) {
	// Step 1: provider が既存 credential を除外対象として返した場合だけ、transport DTO に詰め替えて browser の重複登録防止へ渡す。
	pk := options.PublicKey
	if len(pk.ExcludeCredentials) > 0 {
		descriptors := make([]adminopenapi.WWWTemplateWebAuthnCredentialDescriptor, 0, len(pk.ExcludeCredentials))
		for _, credential := range pk.ExcludeCredentials {
			descriptor := adminopenapi.WWWTemplateWebAuthnCredentialDescriptor{Id: credential.ID, Type: credential.Type}
			if len(credential.Transports) > 0 {
				transports := credential.Transports
				descriptor.Transports = &transports
			}
			descriptors = append(descriptors, descriptor)
		}
		body.ExcludeCredentials = &descriptors
	}
}

func applyAdminRegistrationOptionalOptions(body *adminopenapi.WWWTemplatePasskeyAddStartResponse, options adminWebAuthnRegistrationOptionsJSON) {
	// Step 1: attestation と timeout は browser hint であり、provider から明示された場合だけ response DTO に反映する。
	pk := options.PublicKey
	if pk.Attestation != "" {
		body.Attestation = &pk.Attestation
	}
	if pk.Timeout != nil {
		body.Timeout = pk.Timeout
	}
}

func applyAdminWebAuthnLoginOptions(resp *adminopenapi.WWWTemplatePasskeyStartResponse, optionsJSON []byte) {
	// Step 1: provider 由来の optional WebAuthn JSON が無い場合は、必須 field だけの response として返す。
	if len(optionsJSON) == 0 {
		return
	}
	var options adminWebAuthnLoginOptionsJSON
	if err := json.Unmarshal(optionsJSON, &options); err != nil {
		return
	}

	// Step 2: allowCredentials は transport DTO の配列へ詰め替え、credential ID の意味や検証は WebAuthn provider 側に残す。
	if len(options.PublicKey.AllowCredentials) > 0 {
		descriptors := make([]adminopenapi.WWWTemplateWebAuthnCredentialDescriptor, 0, len(options.PublicKey.AllowCredentials))
		for _, credential := range options.PublicKey.AllowCredentials {
			descriptor := adminopenapi.WWWTemplateWebAuthnCredentialDescriptor{Id: credential.ID, Type: credential.Type}
			if len(credential.Transports) > 0 {
				transports := credential.Transports
				descriptor.Transports = &transports
			}
			descriptors = append(descriptors, descriptor)
		}
		resp.AllowCredentials = &descriptors
	}

	// Step 3: timeout と userVerification は browser ceremony hint として response DTO へ写像する。
	if options.PublicKey.Timeout != nil {
		resp.Timeout = options.PublicKey.Timeout
	}
	if options.PublicKey.UserVerification != "" {
		resp.UserVerification = "required"
	}
}

func adminAssertionCredentialDTO(credential adminopenapi.WWWTemplateWebAuthnAssertionCredential) adminauth.WebAuthnAssertionCredentialDTO {
	// Step 1: optional userHandle / authenticatorAttachment を pointer から primitive string へ安全に写像し、nil を空値として扱う。
	userHandle := ""
	if credential.Response.UserHandle != nil {
		userHandle = *credential.Response.UserHandle
	}
	authenticatorAttachment := ""
	if credential.AuthenticatorAttachment != nil {
		authenticatorAttachment = *credential.AuthenticatorAttachment
	}

	// Step 2: generated WebAuthn assertion DTO を verifier 用 application DTO に詰め替え、handler では署名検証結果を自前判定しない。
	return adminauth.WebAuthnAssertionCredentialDTO{
		ID:                      credential.Id,
		RawID:                   credential.RawId,
		Type:                    credential.Type,
		AuthenticatorAttachment: authenticatorAttachment,
		Response: adminauth.WebAuthnAssertionResponseDTO{
			ClientDataJSON:    credential.Response.ClientDataJSON,
			AuthenticatorData: credential.Response.AuthenticatorData,
			Signature:         credential.Response.Signature,
			UserHandle:        userHandle,
		},
	}
}

func adminAttestationCredentialDTO(credential adminopenapi.WWWTemplateWebAuthnAttestationCredential) adminauth.OperatorWebAuthnAttestationCredential {
	// Step 1: optional transports / authenticatorAttachment を nil 安全に primitive DTO へ写像する。
	transports := []string(nil)
	if credential.Response.Transports != nil {
		transports = *credential.Response.Transports
	}
	authenticatorAttachment := ""
	if credential.AuthenticatorAttachment != nil {
		authenticatorAttachment = *credential.AuthenticatorAttachment
	}

	// Step 2: generated DTO を Admin application の WebAuthn attestation DTO へ詰め替え、handler では検証しない。
	return adminauth.OperatorWebAuthnAttestationCredential{ID: credential.Id, RawID: credential.RawId, Type: credential.Type, AuthenticatorAttachment: authenticatorAttachment, Response: adminauth.OperatorWebAuthnAttestationResponse{ClientDataJSON: credential.Response.ClientDataJSON, AttestationObject: credential.Response.AttestationObject, Transports: transports}}
}

func authFailureResponseWithRequestID(classification adminopenapi.WWWTemplateAuthFailureClassification, requestID string) adminopenapi.WWWTemplateAuthFailureResponse {
	// Step 1: request ごとの correlation ID と stable classification だけを返し、内部 error や権限判定理由を公開しない。
	return adminopenapi.WWWTemplateAuthFailureResponse{Error: classification, RequestId: requestID}
}

func authOperationError(requestID string, message string) adminopenapi.WWWTemplateAuthOperationErrorResponse {
	// Step 1: operation error は UI が分岐できる安定 message と request ID だけに限定し、入力値や内部例外を含めない。
	return adminopenapi.WWWTemplateAuthOperationErrorResponse{RequestId: requestID, Error: message}
}

func accountCreationInput(ctx context.Context, body adminopenapi.WWWTemplateCreateAccountRequest, requestID string) (accountsapplication.CreateAccountInput, bool) {
	// Step 1: middleware が request.Context に保存した primitive 値だけを読み、handler 内に RBAC role map を置かない。
	operatorID, ok := contextString(ctx, adminContextKeyOperatorID)
	if !ok {
		return accountsapplication.CreateAccountInput{}, false
	}
	operatorEmail, ok := contextString(ctx, adminContextKeyOperatorEmail)
	if !ok {
		return accountsapplication.CreateAccountInput{}, false
	}
	operatorRole, ok := contextString(ctx, adminContextKeyOperatorRole)
	if !ok {
		return accountsapplication.CreateAccountInput{}, false
	}
	passkeyState, ok := contextString(ctx, adminContextKeyOperatorPasskeyRegistrationState)
	if !ok {
		return accountsapplication.CreateAccountInput{}, false
	}
	operatorActive, ok := contextBool(ctx, adminContextKeyOperatorActive)
	if !ok {
		return accountsapplication.CreateAccountInput{}, false
	}

	// Step 2: raw email は正規化せず application DTO へ渡し、AccountEmail domain object だけが入力 rule を所有する状態に保つ。
	return accountsapplication.CreateAccountInput{
		Email:                    string(body.Email),
		RequestID:                requestID,
		OperatorID:               operatorID,
		OperatorEmail:            operatorEmail,
		OperatorRole:             operatorRole,
		OperatorActive:           operatorActive,
		PasskeyRegistrationState: passkeyState,
	}, true
}

func operatorCreationInput(ctx context.Context, body adminopenapi.AdminCreateOperatorRequest, requestID string) (operatorsapplication.CreateOperatorInput, bool) {
	// Step 1: middleware が検証済み session から保存した acting operator snapshot を読み、Product account 情報を operator 管理に混入させない。
	operatorID, ok := contextString(ctx, adminContextKeyOperatorID)
	if !ok {
		return operatorsapplication.CreateOperatorInput{}, false
	}
	operatorEmail, ok := contextString(ctx, adminContextKeyOperatorEmail)
	if !ok {
		return operatorsapplication.CreateOperatorInput{}, false
	}
	operatorRole, ok := contextString(ctx, adminContextKeyOperatorRole)
	if !ok {
		return operatorsapplication.CreateOperatorInput{}, false
	}
	passkeyState, ok := contextString(ctx, adminContextKeyOperatorPasskeyRegistrationState)
	if !ok {
		return operatorsapplication.CreateOperatorInput{}, false
	}
	operatorActive, ok := contextBool(ctx, adminContextKeyOperatorActive)
	if !ok {
		return operatorsapplication.CreateOperatorInput{}, false
	}

	// Step 2: 作成対象 email/role は raw DTO のまま application に渡し、domain.OperatorEmail/OperatorRole だけが validation rule を所有する状態に保つ。
	return operatorsapplication.CreateOperatorInput{Email: string(body.Email), Role: string(body.Role), RequestID: requestID, OperatorID: operatorID, OperatorEmail: operatorEmail, OperatorRole: operatorRole, OperatorActive: operatorActive, PasskeyRegistrationState: passkeyState}, true
}

func accountSearchInput(params adminopenapi.ListAdminAccountsParams, requestID string) accountsapplication.AccountSearchInput {
	// Step 1: optional query parameter は nil 安全に primitive DTO へ移し、handler では範囲や長さを判断しない。
	input := accountsapplication.AccountSearchInput{Limit: params.Limit, RequestID: requestID}
	if params.Email != nil {
		input.Email = *params.Email
	}
	if params.Cursor != nil {
		input.Cursor = *params.Cursor
	}

	// Step 2: application use case が validation と default limit を所有できるよう、raw query snapshot をそのまま返す。
	return input
}

func contextString(ctx context.Context, key string) (string, bool) {
	// Step 1: context key は package-local 型に限定し、外部 middleware の同名 string key と衝突しないようにする。
	value, ok := ctx.Value(adminContextValueKey(key)).(string)
	return value, ok && value != ""
}

func contextBool(ctx context.Context, key string) (bool, bool) {
	// Step 1: bool 値は false も有効な operator state なので、型 assertion の成否だけを返す。
	value, ok := ctx.Value(adminContextValueKey(key)).(bool)
	return value, ok
}

func adminAccessTokenFromContext(ctx context.Context) (string, bool) {
	// Step 1: generated strict handler が渡す Gin context だけを header source として扱い、外部 context value に token を置かない。
	return sharedhttp.BearerTokenFromContext(ctx, adminAuthHeader)
}

func adminAuthorizationHeaderPresent(ctx context.Context) bool {
	// Step 1: generated strict handler が渡す Gin context だけを header source として扱い、refresh endpoint で bearer accessToken の混入を検出する。
	return sharedhttp.AuthorizationHeaderPresent(ctx, adminAuthHeader)
}

func adminRefreshCookieValue(ctx context.Context) (string, bool) {
	// Step 1: refreshToken は HttpOnly Cookie からだけ取り出し、request body や header 由来の値を rotation 材料にしない。
	value, ok := sharedhttp.CookieValueFromContext(ctx, adminRefreshCookieName)
	if !ok {
		return "", false
	}

	// Step 2: Cookie 平文は application service に渡す直前だけ保持し、response body へは戻さない。
	return value, true
}

type adminCredentialMode string

const (
	adminCredentialModeCookie adminCredentialMode = "cookie"
	adminCredentialModeBearer adminCredentialMode = "bearer"
)

func adminSessionCredentialMode(mode adminopenapi.WWWTemplateCredentialMode) (adminCredentialMode, bool) {
	// Step 1: generated enum の cookie / bearer だけを許可し、空値や未知値では session credential の露出先を決めない。
	switch mode {
	case adminopenapi.WWWTemplateCredentialModeCookie:
		return adminCredentialModeCookie, true
	case adminopenapi.WWWTemplateCredentialModeBearer:
		return adminCredentialModeBearer, true
	default:
		return "", false
	}
}

func adminRefreshCredential(ctx context.Context, body *adminopenapi.RefreshAdminOperatorSessionJSONRequestBody) (string, adminCredentialMode, bool) {
	// Step 1: Cookie mode の HttpOnly refresh credential を先に読み取り、body credential との同時提示を検出できる状態にする。
	cookieValue, hasCookie := adminRefreshCookieValue(ctx)

	// Step 2: Bearer mode body は refreshToken が存在する場合だけ候補にし、Cookie mode の空 JSON body は Cookie credential の邪魔をさせない。
	bodyValue := ""
	if body != nil {
		bodyValue = strings.TrimSpace(body.RefreshToken)
		if bodyValue != "" && body.CredentialMode != adminopenapi.WWWTemplateBearerContextRefreshRequestCredentialModeBearer {
			return "", "", false
		}
	}
	hasBody := bodyValue != ""

	// Step 3: Cookie と body の exactly-one を強制し、ambiguous / missing credential で rotation へ進まない。
	if hasCookie == hasBody {
		return "", "", false
	}
	if hasCookie {
		return cookieValue, adminCredentialModeCookie, true
	}
	return bodyValue, adminCredentialModeBearer, true
}

func applyAdminRefreshCredential(ctx context.Context, command adminauth.OperatorRefreshCookieCommand, mode adminCredentialMode) bool {
	// Step 1: Cookie mode は application が返した command を Set-Cookie header に反映し、body に refreshToken 平文を置かない。
	if mode == adminCredentialModeCookie {
		return setAdminRefreshCookie(ctx, command)
	}

	// Step 2: Bearer mode は refreshToken を response body にだけ返すため、Cookie command に必要な secret があることだけを確認する。
	return mode == adminCredentialModeBearer && strings.TrimSpace(command.Value) != ""
}

func setAdminRefreshCookie(ctx context.Context, command adminauth.OperatorRefreshCookieCommand) bool {
	// Step 1: generated strict handler から Gin context が渡らない場合は Set-Cookie を出せないため、session 発行を失敗扱いにする。
	ginContext, ok := ctx.(*gin.Context)
	if !ok {
		return false
	}

	// Step 2: Admin refresh Cookie の Path は HTTP adapter が auth context selector から構築し、application 層から Cookie 属性を受け取らない。
	path, err := sharedhttp.BuildRefreshPath(command.AuthContextID)
	if err != nil {
		return false
	}

	// Step 3: Admin refresh Cookie は強権限 operator session の長寿命 credential なので、SameSite=Lax / Secure / HttpOnly を adapter 境界で固定する。
	ginContext.SetSameSite(stdhttp.SameSiteLaxMode)
	value := command.Value
	if command.Clear {
		value = ""
	}
	ginContext.SetCookie(adminRefreshCookieName, value, adminRefreshCookieMaxAge(command), path, "", true, true)
	return true
}

func adminRefreshCookieMaxAge(command adminauth.OperatorRefreshCookieCommand) int {
	// Step 1: Clear command は発行時と同じ path/name で即時削除するため、Gin SetCookie の削除値 -1 に変換する。
	if command.Clear {
		return -1
	}
	if command.MaxAge <= 0 {
		return 0
	}

	// Step 2: 秒未満の正 TTL は即時失効させないよう 1 秒へ丸め、通常値は秒単位の Max-Age に変換する。
	seconds := int(command.MaxAge.Seconds())
	if seconds < 1 {
		return 1
	}
	return seconds
}

func adminSessionResultHasSecrets(result adminauth.OperatorSessionResult) bool {
	// Step 1: body 用 access token と Cookie 用 refreshToken のいずれかが空なら、半端な session を browser へ返さず fail-closed にする。
	return result.AccessToken != "" && result.SessionID != "" && result.RefreshCookie.AuthContextID != "" && result.RefreshCookie.Value != ""
}

func adminAuthSessionUnionResponse(result adminauth.OperatorSessionResult, requestID string, mode adminCredentialMode) (adminopenapi.AdminOperatorAuthSessionResponse, error) {
	// Step 1: credential mode に対応する具体 DTO を oneOf union に格納し、Cookie body と Bearer body の secret 境界を分ける。
	var union adminopenapi.AdminOperatorAuthSessionResponse
	if mode == adminCredentialModeBearer {
		body, err := adminBearerAuthSessionResponse(result, requestID)
		if err != nil {
			return adminopenapi.AdminOperatorAuthSessionResponse{}, err
		}
		if err := union.FromAdminBearerOperatorSessionResponse(body); err != nil {
			return adminopenapi.AdminOperatorAuthSessionResponse{}, err
		}
		return union, nil
	}
	body, err := adminAuthSessionResponse(result, requestID)
	if err != nil {
		return adminopenapi.AdminOperatorAuthSessionResponse{}, err
	}
	if err := union.FromAdminOperatorSessionResponse(body); err != nil {
		return adminopenapi.AdminOperatorAuthSessionResponse{}, err
	}
	return union, nil
}

func adminContextRefreshUnionResponse(result adminauth.OperatorSessionResult, requestID string, mode adminCredentialMode) (adminopenapi.AdminOperatorContextRefreshResponse, error) {
	// Step 1: credential mode に対応する context refresh DTO を oneOf union に格納し、Bearer automation response と Cookie response を混在させない。
	var union adminopenapi.AdminOperatorContextRefreshResponse
	if mode == adminCredentialModeBearer {
		body, err := adminBearerContextRefreshResponse(result, requestID)
		if err != nil {
			return adminopenapi.AdminOperatorContextRefreshResponse{}, err
		}
		if err := union.FromAdminBearerContextRefreshResponse(body); err != nil {
			return adminopenapi.AdminOperatorContextRefreshResponse{}, err
		}
		return union, nil
	}
	body, err := adminContextRefreshResponse(result, requestID)
	if err != nil {
		return adminopenapi.AdminOperatorContextRefreshResponse{}, err
	}
	if err := union.FromAdminContextRefreshResponse(body); err != nil {
		return adminopenapi.AdminOperatorContextRefreshResponse{}, err
	}
	return union, nil
}

func adminBearerAuthSessionResponse(result adminauth.OperatorSessionResult, requestID string) (adminopenapi.AdminBearerOperatorSessionResponse, error) {
	// Step 1: Bearer automation response 用にも Operator subject payload を明示生成し、Admin artifact の operator field だけへ写像する。
	operatorProfile, err := adminOperatorProfileForSession(result.Operator, result.SessionID)
	if err != nil {
		return adminopenapi.AdminBearerOperatorSessionResponse{}, err
	}

	// Step 2: Bearer automation response は refreshToken を body に含め、Cookie command と context index hint は出さない。
	return adminopenapi.AdminBearerOperatorSessionResponse{
		RequestId:      requestID,
		AccessToken:    result.AccessToken,
		RefreshToken:   result.RefreshCookie.Value,
		SessionId:      result.SessionID,
		AuthContextId:  result.SessionID,
		ExpiresAt:      result.ExpiresAt,
		Operator:       operatorProfile,
		CredentialMode: adminopenapi.AdminBearerOperatorSessionResponseCredentialModeBearer,
	}, nil
}

func adminAuthSessionResponse(result adminauth.OperatorSessionResult, requestID string) (adminopenapi.AdminOperatorSessionResponse, error) {
	// Step 1: application session DTO から Operator subject payload を構築し、response の operator field を Admin 専用 subject として固定する。
	operatorProfile, err := adminOperatorProfileForSession(result.Operator, result.SessionID)
	if err != nil {
		return adminopenapi.AdminOperatorSessionResponse{}, err
	}

	// Step 2: browser-readable な値だけを generated DTO へ写像し、RefreshCookie field は意図的に body へ入れない。
	// Admin operator session は同じ session selector を refresh context selector として使い、Cookie-only contract に合わせる。
	return adminopenapi.AdminOperatorSessionResponse{
		RequestId:               requestID,
		AccessToken:             result.AccessToken,
		SessionId:               result.SessionID,
		AuthContextId:           result.SessionID,
		ExpiresAt:               result.ExpiresAt,
		Operator:                operatorProfile,
		CredentialMode:          adminopenapi.AdminOperatorSessionResponseCredentialModeCookie,
		ClearCookieCommands:     []adminopenapi.WWWTemplateCookieClearCommand{},
		ContextIndexUpdateHints: []adminopenapi.WWWTemplateContextIndexUpdateHint{},
	}, nil
}

func adminContextRefreshResponse(result adminauth.OperatorSessionResult, requestID string) (adminopenapi.AdminContextRefreshResponse, error) {
	// Step 1: refresh 成功時も Operator subject payload を明示生成し、Cookie mode context refresh の operator field へだけ反映する。
	operatorProfile, err := adminOperatorProfileForSession(result.Operator, result.SessionID)
	if err != nil {
		return adminopenapi.AdminContextRefreshResponse{}, err
	}

	// Step 2: operator refreshToken 平文は body に入れず、Cookie mode context refresh union として accessToken だけを返す。
	return adminopenapi.AdminContextRefreshResponse{
		RequestId:               requestID,
		AccessToken:             result.AccessToken,
		SessionId:               result.SessionID,
		AuthContextId:           result.SessionID,
		ExpiresAt:               result.ExpiresAt,
		Operator:                operatorProfile,
		CredentialMode:          adminopenapi.AdminContextRefreshResponseCredentialModeCookie,
		ClearCookieCommands:     []adminopenapi.WWWTemplateCookieClearCommand{},
		ContextIndexUpdateHints: []adminopenapi.WWWTemplateContextIndexUpdateHint{},
	}, nil
}

func adminBearerContextRefreshResponse(result adminauth.OperatorSessionResult, requestID string) (adminopenapi.AdminBearerContextRefreshResponse, error) {
	// Step 1: Bearer automation refresh response でも Operator subject payload を明示生成し、Product account payload と混同しない。
	operatorProfile, err := adminOperatorProfileForSession(result.Operator, result.SessionID)
	if err != nil {
		return adminopenapi.AdminBearerContextRefreshResponse{}, err
	}

	// Step 2: Bearer automation refresh response は新しい refreshToken を body に返し、Set-Cookie には依存しない。
	return adminopenapi.AdminBearerContextRefreshResponse{
		RequestId:      requestID,
		AccessToken:    result.AccessToken,
		RefreshToken:   result.RefreshCookie.Value,
		SessionId:      result.SessionID,
		AuthContextId:  result.SessionID,
		ExpiresAt:      result.ExpiresAt,
		Operator:       operatorProfile,
		CredentialMode: adminopenapi.AdminBearerContextRefreshResponseCredentialModeBearer,
	}, nil
}

func adminCurrentOperatorResponse(operator adminauth.OperatorDTO, requestID string) adminopenapi.AdminCurrentOperatorResponse {
	// Step 1: current operator response は session secret を含めず、profile と requestId だけに限定する。
	return adminopenapi.AdminCurrentOperatorResponse{RequestId: requestID, Operator: adminOperatorProfile(operator)}
}

func adminOperatorPasskeyListResponse(result adminauth.OperatorPasskeyListResult, requestID string) adminopenapi.AdminOperatorPasskeyListResponse {
	// Step 1: application DTO の件数に合わせて response slice を確保し、非秘匿 field だけを OpenAPI DTO へ変換する。
	passkeys := make([]adminopenapi.AdminOperatorPasskeyItem, 0, len(result.Passkeys))
	for _, passkey := range result.Passkeys {
		passkeys = append(passkeys, adminOperatorPasskeyItem(passkey))
	}

	// Step 2: requestId と passkey 要約だけを返し、credential handle や public key を response body に含めない。
	return adminopenapi.AdminOperatorPasskeyListResponse{RequestId: requestID, Passkeys: passkeys}
}

func adminOperatorPasskeyItem(passkey adminauth.OperatorPasskeyCredential) adminopenapi.AdminOperatorPasskeyItem {
	// Step 1: optional lastUsedAt は nil の場合に response から省略し、未使用 credential を空時刻で誤表現しない。
	var lastUsedAt *time.Time
	if passkey.LastUsedAt != nil {
		value := passkey.LastUsedAt.UTC()
		lastUsedAt = &value
	}

	// Step 2: 管理に必要な credential ID と時刻だけを generated DTO へ詰め替える。
	return adminopenapi.AdminOperatorPasskeyItem{Id: adminopenapi.WWWTemplateUlidId(passkey.ID), CreatedAt: passkey.CreatedAt, LastUsedAt: lastUsedAt}
}

func adminOperatorProfile(operator adminauth.OperatorDTO) adminopenapi.AdminOperatorProfile {
	// Step 1: Admin Operator application DTO の primitive 値だけを generated profile へ詰め替え、role の許可判定は handler に置かない。
	return adminopenapi.AdminOperatorProfile{
		OperatorId: operator.ID,
		Email:      operator.Email,
		Role:       adminopenapi.AdminOperatorRole(operator.Role),
		Active:     operator.Active,
	}
}

func adminOperatorProfileForSession(operator adminauth.OperatorDTO, sessionID string) (adminopenapi.AdminOperatorProfile, error) {
	// Step 1: application DTO の operator ID と session ID から Admin operator subject payload を作り、HTTP adapter 境界で主体種別を明示する。
	subject, err := conceptauth.NewOperatorSubjectPayload(operator.ID, sessionID)
	if err != nil {
		return adminopenapi.AdminOperatorProfile{}, err
	}

	// Step 2: generated Admin response の operator field には検証済み subject ID を使い、email/role/active は application DTO から値コピーする。
	operator.ID = subject.OperatorID().String()
	return adminOperatorProfile(operator), nil
}

func startAdminPasskeyAuthenticationFailure(requestID string, err error) adminopenapi.StartAdminPasskeyAuthenticationResponseObject {
	// Step 1: challenge 開始の application error を non-secret な HTTP response へ写像し、identifier の存在有無を漏らさない。
	if errors.Is(err, adminauth.ErrOperatorAuthUnavailable) {
		return adminopenapi.StartAdminPasskeyAuthentication503JSONResponse{Body: authFailureResponseWithRequestID(adminopenapi.InternalError, requestID), Headers: adminopenapi.StartAdminPasskeyAuthentication503ResponseHeaders{CacheControl: noStoreValue}}
	}
	return adminopenapi.StartAdminPasskeyAuthentication400JSONResponse{Body: authOperationError(requestID, adminAuthInvalidRequestMessage), Headers: adminopenapi.StartAdminPasskeyAuthentication400ResponseHeaders{CacheControl: noStoreValue}}
}

func finishAdminPasskeyAuthenticationFailure(requestID string, err error) adminopenapi.FinishAdminPasskeyAuthenticationResponseObject {
	// Step 1: passkey finish の application error を generated response 種別へ写像し、credential 検証の詳細は body に出さない。
	switch {
	case errors.Is(err, adminauth.ErrOperatorAuthUnavailable):
		return adminopenapi.FinishAdminPasskeyAuthentication503JSONResponse{Body: authFailureResponseWithRequestID(adminopenapi.InternalError, requestID), Headers: adminopenapi.FinishAdminPasskeyAuthentication503ResponseHeaders{CacheControl: noStoreValue}}
	case errors.Is(err, adminauth.ErrOperatorAuthForbidden):
		return adminopenapi.FinishAdminPasskeyAuthentication403JSONResponse{Body: authFailureResponseWithRequestID(adminopenapi.Unauthenticated, requestID), Headers: adminopenapi.FinishAdminPasskeyAuthentication403ResponseHeaders{CacheControl: noStoreValue}}
	default:
		return adminopenapi.FinishAdminPasskeyAuthentication400JSONResponse{Body: authOperationError(requestID, adminAuthInvalidRequestMessage), Headers: adminopenapi.FinishAdminPasskeyAuthentication400ResponseHeaders{CacheControl: noStoreValue}}
	}
}

func refreshAdminOperatorSessionFailure(requestID string, err error) adminopenapi.RefreshAdminOperatorSessionResponseObject {
	// Step 1: refresh rotation の application error を 401/403/503 に限定し、refreshToken selector や hash 状態を response に出さない。
	switch {
	case errors.Is(err, adminauth.ErrOperatorAuthUnavailable):
		return adminopenapi.RefreshAdminOperatorSession503JSONResponse{Body: authFailureResponseWithRequestID(adminopenapi.InternalError, requestID), Headers: adminopenapi.RefreshAdminOperatorSession503ResponseHeaders{CacheControl: noStoreValue}}
	case errors.Is(err, adminauth.ErrOperatorAuthForbidden):
		return adminopenapi.RefreshAdminOperatorSession403JSONResponse{Body: authFailureResponseWithRequestID(adminopenapi.Unauthenticated, requestID), Headers: adminopenapi.RefreshAdminOperatorSession403ResponseHeaders{CacheControl: noStoreValue}}
	default:
		return adminopenapi.RefreshAdminOperatorSession401JSONResponse{Body: authFailureResponseWithRequestID(adminopenapi.Unauthenticated, requestID), Headers: adminopenapi.RefreshAdminOperatorSession401ResponseHeaders{CacheControl: noStoreValue}}
	}
}

func currentAdminOperatorFailure(requestID string, err error) adminopenapi.GetCurrentAdminOperatorResponseObject {
	// Step 1: current operator の application error を stable auth failure に畳み、token/session/snapshot の詳細を隠す。
	switch {
	case errors.Is(err, adminauth.ErrOperatorAuthUnavailable):
		return adminopenapi.GetCurrentAdminOperator503JSONResponse{Body: authFailureResponseWithRequestID(adminopenapi.InternalError, requestID), Headers: adminopenapi.GetCurrentAdminOperator503ResponseHeaders{CacheControl: noStoreValue}}
	case errors.Is(err, adminauth.ErrOperatorAuthForbidden):
		return adminopenapi.GetCurrentAdminOperator403JSONResponse{Body: authFailureResponseWithRequestID(adminopenapi.Unauthenticated, requestID), Headers: adminopenapi.GetCurrentAdminOperator403ResponseHeaders{CacheControl: noStoreValue}}
	default:
		return adminopenapi.GetCurrentAdminOperator401JSONResponse{Body: authFailureResponseWithRequestID(adminopenapi.Unauthenticated, requestID), Headers: adminopenapi.GetCurrentAdminOperator401ResponseHeaders{CacheControl: noStoreValue}}
	}
}

func logoutAdminOperatorFailure(requestID string, err error) adminopenapi.LogoutAdminOperatorResponseObject {
	// Step 1: logout の application error を stable auth failure に写像し、対象 session の存在有無を露出しない。
	switch {
	case errors.Is(err, adminauth.ErrOperatorAuthUnavailable):
		return adminopenapi.LogoutAdminOperator503JSONResponse{Body: authFailureResponseWithRequestID(adminopenapi.InternalError, requestID), Headers: adminopenapi.LogoutAdminOperator503ResponseHeaders{CacheControl: noStoreValue}}
	case errors.Is(err, adminauth.ErrOperatorAuthForbidden):
		return adminopenapi.LogoutAdminOperator403JSONResponse{Body: authFailureResponseWithRequestID(adminopenapi.Unauthenticated, requestID), Headers: adminopenapi.LogoutAdminOperator403ResponseHeaders{CacheControl: noStoreValue}}
	default:
		return adminopenapi.LogoutAdminOperator401JSONResponse{Body: authFailureResponseWithRequestID(adminopenapi.Unauthenticated, requestID), Headers: adminopenapi.LogoutAdminOperator401ResponseHeaders{CacheControl: noStoreValue}}
	}
}

func listAdminOperatorPasskeysFailure(requestID string, err error) adminopenapi.ListAdminOperatorPasskeysResponseObject {
	// Step 1: passkey 一覧の application error を stable auth failure に畳み、保存層の詳細を隠す。
	switch {
	case errors.Is(err, adminauth.ErrOperatorAuthUnavailable):
		return adminopenapi.ListAdminOperatorPasskeys503JSONResponse{Body: authFailureResponseWithRequestID(adminopenapi.InternalError, requestID), Headers: adminopenapi.ListAdminOperatorPasskeys503ResponseHeaders{CacheControl: noStoreValue}}
	case errors.Is(err, adminauth.ErrOperatorAuthForbidden):
		return adminopenapi.ListAdminOperatorPasskeys403JSONResponse{Body: authFailureResponseWithRequestID(adminopenapi.Unauthenticated, requestID), Headers: adminopenapi.ListAdminOperatorPasskeys403ResponseHeaders{CacheControl: noStoreValue}}
	default:
		return adminopenapi.ListAdminOperatorPasskeys401JSONResponse{Body: authFailureResponseWithRequestID(adminopenapi.Unauthenticated, requestID), Headers: adminopenapi.ListAdminOperatorPasskeys401ResponseHeaders{CacheControl: noStoreValue}}
	}
}

func deleteAdminOperatorPasskeyFailure(requestID string, err error) adminopenapi.DeleteAdminOperatorPasskeyResponseObject {
	// Step 1: passkey 削除の application error を stable HTTP response へ写像し、credential の所有者や存在有無の詳細を隠す。
	switch {
	case errors.Is(err, adminauth.ErrOperatorAuthLastPasskey):
		return adminopenapi.DeleteAdminOperatorPasskey409JSONResponse{Body: authOperationError(requestID, adminMinimumAuthenticatorMessage), Headers: adminopenapi.DeleteAdminOperatorPasskey409ResponseHeaders{CacheControl: noStoreValue}}
	case errors.Is(err, adminauth.ErrOperatorAuthInvalidInput) || errors.Is(err, adminauth.ErrOperatorAuthPasskeyNotFound):
		return adminopenapi.DeleteAdminOperatorPasskey400JSONResponse{Body: authOperationError(requestID, adminAuthInvalidRequestMessage), Headers: adminopenapi.DeleteAdminOperatorPasskey400ResponseHeaders{CacheControl: noStoreValue}}
	case errors.Is(err, adminauth.ErrOperatorAuthForbidden):
		return adminopenapi.DeleteAdminOperatorPasskey403JSONResponse{Body: authFailureResponseWithRequestID(adminopenapi.Unauthenticated, requestID), Headers: adminopenapi.DeleteAdminOperatorPasskey403ResponseHeaders{CacheControl: noStoreValue}}
	case errors.Is(err, adminauth.ErrOperatorAuthUnavailable):
		return adminopenapi.DeleteAdminOperatorPasskey503JSONResponse{Body: authFailureResponseWithRequestID(adminopenapi.InternalError, requestID), Headers: adminopenapi.DeleteAdminOperatorPasskey503ResponseHeaders{CacheControl: noStoreValue}}
	default:
		return adminopenapi.DeleteAdminOperatorPasskey401JSONResponse{Body: authFailureResponseWithRequestID(adminopenapi.Unauthenticated, requestID), Headers: adminopenapi.DeleteAdminOperatorPasskey401ResponseHeaders{CacheControl: noStoreValue}}
	}
}

func startAdminInitialSetupFailure(requestID string, err error) adminopenapi.StartAdminInitialSetupResponseObject {
	// Step 1: bootstrap state の詳細を秘匿しつつ、既存 operator ありだけは UI が setup 画面を閉じられる conflict として返す。
	switch {
	case errors.Is(err, operatorsapplication.ErrOperatorConflict):
		return adminopenapi.StartAdminInitialSetup409JSONResponse{Body: authOperationError(requestID, adminSetupUnavailableMessage), Headers: adminopenapi.StartAdminInitialSetup409ResponseHeaders{CacheControl: noStoreValue}}
	case errors.Is(err, operatorsapplication.ErrOperatorInvalidInput):
		return adminopenapi.StartAdminInitialSetup400JSONResponse{Body: authOperationError(requestID, adminInvalidRequestBodyMessage), Headers: adminopenapi.StartAdminInitialSetup400ResponseHeaders{CacheControl: noStoreValue}}
	case errors.Is(err, operatorsapplication.ErrOperatorForbidden):
		return adminopenapi.StartAdminInitialSetup403JSONResponse{Body: authFailureResponseWithRequestID(adminopenapi.Unauthenticated, requestID), Headers: adminopenapi.StartAdminInitialSetup403ResponseHeaders{CacheControl: noStoreValue}}
	default:
		return adminopenapi.StartAdminInitialSetup503JSONResponse{Body: authFailureResponseWithRequestID(adminopenapi.InternalError, requestID), Headers: adminopenapi.StartAdminInitialSetup503ResponseHeaders{CacheControl: noStoreValue}}
	}
}

func finishAdminInitialSetupFailure(requestID string, err error) adminopenapi.FinishAdminInitialSetupResponseObject {
	// Step 1: finish でも bootstrap/token/challenge 詳細を response に出さず、stable HTTP response へ畳む。
	switch {
	case errors.Is(err, operatorsapplication.ErrOperatorConflict):
		return adminopenapi.FinishAdminInitialSetup409JSONResponse{Body: authOperationError(requestID, adminSetupUnavailableMessage), Headers: adminopenapi.FinishAdminInitialSetup409ResponseHeaders{CacheControl: noStoreValue}}
	case errors.Is(err, operatorsapplication.ErrOperatorInvalidInput):
		return adminopenapi.FinishAdminInitialSetup400JSONResponse{Body: authOperationError(requestID, adminInvalidRequestBodyMessage), Headers: adminopenapi.FinishAdminInitialSetup400ResponseHeaders{CacheControl: noStoreValue}}
	case errors.Is(err, operatorsapplication.ErrOperatorForbidden):
		return adminopenapi.FinishAdminInitialSetup403JSONResponse{Body: authFailureResponseWithRequestID(adminopenapi.Unauthenticated, requestID), Headers: adminopenapi.FinishAdminInitialSetup403ResponseHeaders{CacheControl: noStoreValue}}
	default:
		return adminopenapi.FinishAdminInitialSetup503JSONResponse{Body: authFailureResponseWithRequestID(adminopenapi.InternalError, requestID), Headers: adminopenapi.FinishAdminInitialSetup503ResponseHeaders{CacheControl: noStoreValue}}
	}
}

func startAdminOperatorSetupFailure(requestID string, err error) adminopenapi.StartAdminOperatorSetupResponseObject {
	// Step 1: setup token の invalid/expired/consumed/registered を区別せず、non-revealing response へ畳む。
	switch {
	case errors.Is(err, operatorsapplication.ErrOperatorInvalidInput):
		return adminopenapi.StartAdminOperatorSetup400JSONResponse{Body: authOperationError(requestID, adminInvalidRequestBodyMessage), Headers: adminopenapi.StartAdminOperatorSetup400ResponseHeaders{CacheControl: noStoreValue}}
	case errors.Is(err, operatorsapplication.ErrOperatorForbidden):
		return adminopenapi.StartAdminOperatorSetup403JSONResponse{Body: authFailureResponseWithRequestID(adminopenapi.Unauthenticated, requestID), Headers: adminopenapi.StartAdminOperatorSetup403ResponseHeaders{CacheControl: noStoreValue}}
	default:
		return adminopenapi.StartAdminOperatorSetup503JSONResponse{Body: authFailureResponseWithRequestID(adminopenapi.InternalError, requestID), Headers: adminopenapi.StartAdminOperatorSetup503ResponseHeaders{CacheControl: noStoreValue}}
	}
}

func finishAdminOperatorSetupFailure(requestID string, err error) adminopenapi.FinishAdminOperatorSetupResponseObject {
	// Step 1: setup finish の token/challenge/passkey 詳細を隠し、UI には安定分類だけを返す。
	switch {
	case errors.Is(err, operatorsapplication.ErrOperatorInvalidInput):
		return adminopenapi.FinishAdminOperatorSetup400JSONResponse{Body: authOperationError(requestID, adminInvalidRequestBodyMessage), Headers: adminopenapi.FinishAdminOperatorSetup400ResponseHeaders{CacheControl: noStoreValue}}
	case errors.Is(err, operatorsapplication.ErrOperatorForbidden):
		return adminopenapi.FinishAdminOperatorSetup403JSONResponse{Body: authFailureResponseWithRequestID(adminopenapi.Unauthenticated, requestID), Headers: adminopenapi.FinishAdminOperatorSetup403ResponseHeaders{CacheControl: noStoreValue}}
	default:
		return adminopenapi.FinishAdminOperatorSetup503JSONResponse{Body: authFailureResponseWithRequestID(adminopenapi.InternalError, requestID), Headers: adminopenapi.FinishAdminOperatorSetup503ResponseHeaders{CacheControl: noStoreValue}}
	}
}

func createAdminAccountFailure(requestID string, err error) adminopenapi.CreateAdminAccountResponseObject {
	// Step 1: application account creation の abstract error を stable HTTP response へ写像し、domain/repository の詳細を body に出さない。
	switch {
	case errors.Is(err, accountsapplication.ErrAccountCreationInvalidInput):
		return adminopenapi.CreateAdminAccount400JSONResponse{Body: authOperationError(requestID, adminInvalidRequestBodyMessage), Headers: adminopenapi.CreateAdminAccount400ResponseHeaders{CacheControl: noStoreValue}}
	case errors.Is(err, accountsapplication.ErrAccountCreationForbidden):
		return adminopenapi.CreateAdminAccount403JSONResponse{Body: authFailureResponseWithRequestID(adminopenapi.Unauthenticated, requestID), Headers: adminopenapi.CreateAdminAccount403ResponseHeaders{CacheControl: noStoreValue}}
	case errors.Is(err, accountsapplication.ErrAccountDuplicateEmail):
		return adminopenapi.CreateAdminAccount409JSONResponse{Body: authOperationError(requestID, adminDuplicateEmailMessage), Headers: adminopenapi.CreateAdminAccount409ResponseHeaders{CacheControl: noStoreValue}}
	default:
		return adminopenapi.CreateAdminAccount503JSONResponse{Body: authFailureResponseWithRequestID(adminopenapi.InternalError, requestID), Headers: adminopenapi.CreateAdminAccount503ResponseHeaders{CacheControl: noStoreValue}}
	}
}

func createAdminOperatorFailure(requestID string, err error) adminopenapi.CreateAdminOperatorResponseObject {
	// Step 1: operator 作成 use case の抽象 error を stable HTTP response に写像し、setup token や delivery 失敗詳細を body に出さない。
	switch {
	case errors.Is(err, operatorsapplication.ErrOperatorInvalidInput):
		return adminopenapi.CreateAdminOperator400JSONResponse{Body: authOperationError(requestID, adminInvalidRequestBodyMessage), Headers: adminopenapi.CreateAdminOperator400ResponseHeaders{CacheControl: noStoreValue}}
	case errors.Is(err, operatorsapplication.ErrOperatorForbidden):
		return adminopenapi.CreateAdminOperator403JSONResponse{Body: authFailureResponseWithRequestID(adminopenapi.Unauthenticated, requestID), Headers: adminopenapi.CreateAdminOperator403ResponseHeaders{CacheControl: noStoreValue}}
	case errors.Is(err, operatorsapplication.ErrOperatorConflict):
		return adminopenapi.CreateAdminOperator409JSONResponse{Body: authOperationError(requestID, adminDuplicateEmailMessage), Headers: adminopenapi.CreateAdminOperator409ResponseHeaders{CacheControl: noStoreValue}}
	default:
		return adminopenapi.CreateAdminOperator503JSONResponse{Body: authFailureResponseWithRequestID(adminopenapi.InternalError, requestID), Headers: adminopenapi.CreateAdminOperator503ResponseHeaders{CacheControl: noStoreValue}}
	}
}

func listAdminAccountsFailure(requestID string, err error) adminopenapi.ListAdminAccountsResponseObject {
	// Step 1: account search の pagination/input error を 400 stable validation error へ写像し、内部理由を body に含めない。
	if errors.Is(err, accountsapplication.ErrAccountSearchInvalidInput) {
		return adminopenapi.ListAdminAccounts400JSONResponse{Body: authOperationError(requestID, adminAccountSearchInvalidMessage), Headers: adminopenapi.ListAdminAccounts400ResponseHeaders{CacheControl: noStoreValue}}
	}

	// Step 2: repository や依存欠落などの failure は 503 に畳み、DB error 詳細を外部へ出さない。
	return adminopenapi.ListAdminAccounts503JSONResponse{Body: authFailureResponseWithRequestID(adminopenapi.InternalError, requestID), Headers: adminopenapi.ListAdminAccounts503ResponseHeaders{CacheControl: noStoreValue}}
}

func getAdminAccountFailure(requestID string, err error) adminopenapi.GetAdminAccountResponseObject {
	// Step 1: 対象不在は stable 404 へ畳み、DB の record existence 以外の詳細は body に出さない。
	if errors.Is(err, accountsapplication.ErrAccountSearchNotFound) {
		return adminopenapi.GetAdminAccount404JSONResponse{Body: authOperationError(requestID, "account_not_found"), Headers: adminopenapi.GetAdminAccount404ResponseHeaders{CacheControl: noStoreValue}}
	}

	// Step 2: validation や repository failure は internal failure として fail-closed にし、詳細理由を隠す。
	return adminopenapi.GetAdminAccount503JSONResponse{Body: authFailureResponseWithRequestID(adminopenapi.InternalError, requestID), Headers: adminopenapi.GetAdminAccount503ResponseHeaders{CacheControl: noStoreValue}}
}

func adminCreateAccountResponse(created accountsapplication.CreatedAccount) adminopenapi.WWWTemplateCreateAccountResponse {
	// Step 1: application が返した primitive snapshot だけを OpenAPI DTO に詰め替え、status などの業務値はここで再判定しない。
	return adminopenapi.WWWTemplateCreateAccountResponse{
		RequestId:    created.RequestID,
		AuditEventId: created.AuditID,
		Account: adminopenapi.WWWTemplateAccountSummary{
			AccountId:    created.AccountID,
			Email:        created.Email,
			Status:       adminopenapi.WWWTemplateAccountStatus(created.Status),
			CreatedAt:    created.CreatedAt,
			PasskeyCount: created.PasskeyCount,
		},
	}
}

func adminCreateOperatorResponse(created operatorsapplication.CreatedOperator) adminopenapi.AdminCreateOperatorResponse {
	// Step 1: application DTO を generated response DTO へ詰め替え、setup token 平文や delivery 先の詳細は返さない。
	return adminopenapi.AdminCreateOperatorResponse{RequestId: created.RequestID, AuditEventId: created.AuditID, DeliveryStatus: adminopenapi.AdminSetupTokenDeliveryStatus(created.DeliveryStatus), Operator: adminOperatorProfile(created.Operator)}
}

func adminAccountListResponse(result accountsapplication.AccountSearchResult) adminopenapi.WWWTemplateAccountListResponse {
	// Step 1: application DTO の件数に合わせて response slice を確保し、値コピーだけで OpenAPI DTO へ変換する。
	accounts := make([]adminopenapi.WWWTemplateAccountSummary, 0, len(result.Accounts))
	for _, account := range result.Accounts {
		accounts = append(accounts, adminopenapi.WWWTemplateAccountSummary{
			AccountId:    account.AccountID,
			Email:        account.Email,
			Status:       adminopenapi.WWWTemplateAccountStatus(account.Status),
			PasskeyCount: account.PasskeyCount,
			CreatedAt:    account.CreatedAt,
		})
	}

	// Step 2: opaque cursor は存在する場合だけ pointer 化し、空値を response に出さない。
	var nextCursor *string
	if result.NextCursor != "" {
		nextCursor = &result.NextCursor
	}

	// Step 3: requestId と read model だけを返し、検索条件や内部 query を response body に含めない。
	return adminopenapi.WWWTemplateAccountListResponse{Accounts: accounts, NextCursor: nextCursor, RequestId: result.RequestID}
}

func adminAccountDetailResponse(result accountsapplication.AccountDetailResult) adminopenapi.WWWTemplateAccountDetailResponse {
	// Step 1: detail でも一覧と同じ account summary DTO を使い、画面間で表示値の意味を揃える。
	return adminopenapi.WWWTemplateAccountDetailResponse{RequestId: result.RequestID, Account: adminopenapi.WWWTemplateAccountSummary{AccountId: result.Account.AccountID, Email: result.Account.Email, Status: adminopenapi.WWWTemplateAccountStatus(result.Account.Status), PasskeyCount: result.Account.PasskeyCount, CreatedAt: result.Account.CreatedAt}}
}

func nextAdminRequestID() string {
	// Step 1: 監査・response correlation 用に ULID を発行し、entropy failure 時だけ既存の fallback ID で fail-safe に応答を継続する。
	requestID, err := id.NewULID(time.Now().UTC(), rand.Reader)
	if err != nil {
		return fallbackRequestID
	}

	// Step 2: 正常に生成できた canonical ULID を application DTO と response の両方へ渡す。
	return requestID
}
