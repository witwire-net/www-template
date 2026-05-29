package http

import (
	"encoding/json"
	"net/url"
	"os"
	"sort"
	"strings"
	"testing"
)

type openAPIContract struct {
	Paths      map[string]openAPIPath `json:"paths"`
	Servers    []openAPIServer        `json:"servers"`
	Tags       []openAPITag           `json:"tags"`
	Components struct {
		SecuritySchemes map[string]struct {
			Scheme string `json:"scheme"`
			Type   string `json:"type"`
		} `json:"securitySchemes"`
	} `json:"components"`
}

type openAPIServer struct {
	URL string `json:"url"`
}

type openAPITag struct {
	Name string `json:"name"`
}

type openAPIPath struct {
	Delete openAPIOperation `json:"delete"`
	Get    openAPIOperation `json:"get"`
	Patch  openAPIOperation `json:"patch"`
	Post   openAPIOperation `json:"post"`
}

type openAPIOperation struct {
	OperationID string                `json:"operationId"`
	Security    []map[string][]string `json:"security"`
	Tags        []string              `json:"tags"`
}

type openAPIRouteSurfaceContract struct {
	Paths map[string]map[string]json.RawMessage `json:"paths"`
}

func TestAppOpenAPIDeclaresBearerSecurity(t *testing.T) {
	t.Parallel()

	// Product OpenAPI の security 宣言だけを検査対象にするため、生成済み Product artifact を読み込む。
	content := readProductOpenAPIArtifact(t)

	var contract openAPIContract
	if err := json.Unmarshal(content, &contract); err != nil {
		t.Fatalf("unmarshal openapi: %v", err)
	}

	bearerAuth, ok := contract.Components.SecuritySchemes["BearerAuth"]
	if !ok {
		t.Fatal("BearerAuth security scheme not found")
	}
	if bearerAuth.Type != "http" || strings.ToLower(bearerAuth.Scheme) != "bearer" {
		t.Fatalf("unexpected BearerAuth scheme: type=%s scheme=%s", bearerAuth.Type, bearerAuth.Scheme)
	}

	assertBearerSecurity(t, contract, "/api/v1/auth/logout", "post")
}

// TestAPIContractBES001ProductArtifactsExcludeAdminOperations は、Product 用に公開される生成物へ Admin operation が混入していないことを検証する。
//
// このテストは [API-CONTRACT-BE-S001] の受け皿であり、Product OpenAPI、Product SDK、Product Go bindings を同じ Admin operation 一覧で検査する。
// 引数は testing.T だけで、戻り値はない。違反がある場合は該当 artifact と token を含む failure として報告する。
func TestAPIContractBES001ProductArtifactsExcludeAdminOperations(t *testing.T) {
	t.Parallel()

	// Step 1: Admin surface に属する operation 名を Product artifact 禁止語彙として固定し、OpenAPI / SDK / Go bindings の検査意味をそろえる。
	adminOperations := []adminOperationName{
		{lowerCamel: "listAdminAccounts", pascal: "ListAdminAccounts"},
		{lowerCamel: "createAdminAccount", pascal: "CreateAdminAccount"},
		{lowerCamel: "getAdminAccount", pascal: "GetAdminAccount"},
		{lowerCamel: "createAdminOperator", pascal: "CreateAdminOperator"},
		{lowerCamel: "finishAdminOperatorSetup", pascal: "FinishAdminOperatorSetup"},
		{lowerCamel: "startAdminOperatorSetup", pascal: "StartAdminOperatorSetup"},
		{lowerCamel: "getCurrentAdminOperator", pascal: "GetCurrentAdminOperator"},
		{lowerCamel: "listAdminOperatorPasskeys", pascal: "ListAdminOperatorPasskeys"},
		{lowerCamel: "deleteAdminOperatorPasskey", pascal: "DeleteAdminOperatorPasskey"},
		{lowerCamel: "logoutAdminOperator", pascal: "LogoutAdminOperator"},
		{lowerCamel: "refreshAdminOperatorSession", pascal: "RefreshAdminOperatorSession"},
		{lowerCamel: "finishAdminPasskeyAuthentication", pascal: "FinishAdminPasskeyAuthentication"},
		{lowerCamel: "startAdminPasskeyAuthentication", pascal: "StartAdminPasskeyAuthentication"},
	}

	// Step 2: Product OpenAPI は operationId と tag を JSON 構造として検査し、Admin route が path 名を変えて混入しても検知できるようにする。
	assertProductOpenAPIExcludesAdminOperations(t, adminOperations)

	// Step 3: Product SDK は Orval が operationId から生成する関数名・URL helper・response type 名を検査し、frontend Product SDK への Admin operation 露出を拒否する。
	assertProductSDKExcludesAdminOperations(t, adminOperations)

	// Step 4: Product Go bindings は oapi-codegen が operation 名から生成する interface / wrapper / request object 名を検査し、Product binary 側の Admin handler 到達口を拒否する。
	assertProductGoBindingsExcludeAdminOperations(t, adminOperations)
}

// TestAPIContractBES002AdminArtifactsExcludeProductOperations は、Admin 用に公開される生成物へ Product operation が混入していないことを検証する。
//
// このテストは [API-CONTRACT-BE-S002] の受け皿であり、Admin OpenAPI、Admin SDK、Admin Go bindings を同じ Product operation 一覧で検査する。
// 引数は testing.T だけで、戻り値はない。違反がある場合は該当 artifact と token を含む failure として報告する。
func TestAPIContractBES002AdminArtifactsExcludeProductOperations(t *testing.T) {
	t.Parallel()

	// Step 1: Product surface に属する operation 名を Admin artifact 禁止語彙として固定し、OpenAPI / SDK / Go bindings の検査意味をそろえる。
	productOperations := []productOperationName{
		{lowerCamel: "getAccountSettings", pascal: "GetAccountSettings"},
		{lowerCamel: "updateAccountSettings", pascal: "UpdateAccountSettings"},
		{lowerCamel: "logout", pascal: "Logout"},
		{lowerCamel: "finishPasskeyAuthentication", pascal: "FinishPasskeyAuthentication"},
		{lowerCamel: "registerPasskey", pascal: "RegisterPasskey"},
		{lowerCamel: "startPasskeyRegistration", pascal: "StartPasskeyRegistration"},
		{lowerCamel: "startPasskeyAuthentication", pascal: "StartPasskeyAuthentication"},
		{lowerCamel: "finishReauthentication", pascal: "FinishReauthentication"},
		{lowerCamel: "startReauthentication", pascal: "StartReauthentication"},
		{lowerCamel: "requestPasskeyRecovery", pascal: "RequestPasskeyRecovery"},
		{lowerCamel: "consumeRecoveryToken", pascal: "ConsumeRecoveryToken"},
		{lowerCamel: "refreshToken", pascal: "RefreshToken"},
		{lowerCamel: "listPasskeys", pascal: "ListPasskeys"},
		{lowerCamel: "finishPasskeyAddition", pascal: "FinishPasskeyAddition"},
		{lowerCamel: "sendDeviceLink", pascal: "SendDeviceLink"},
		{lowerCamel: "startPasskeyAddition", pascal: "StartPasskeyAddition"},
		{lowerCamel: "deletePasskey", pascal: "DeletePasskey"},
		{lowerCamel: "listSessions", pascal: "ListSessions"},
		{lowerCamel: "revokeOtherSessions", pascal: "RevokeOtherSessions"},
		{lowerCamel: "revokeSession", pascal: "RevokeSession"},
		{lowerCamel: "getStatus", pascal: "GetStatus"},
	}

	// Step 2: Admin OpenAPI は operationId と tag を JSON 構造として検査し、Product route が path 名を変えて混入しても検知できるようにする。
	assertAdminOpenAPIExcludesProductOperations(t, productOperations)

	// Step 3: Admin SDK は Orval が operationId から生成する関数名・URL helper・response type 名を検査し、Admin SDK への Product operation 露出を拒否する。
	assertAdminSDKExcludesProductOperations(t, productOperations)

	// Step 4: Admin Go bindings は oapi-codegen が operation 名から生成する interface / wrapper / request object 名を検査し、Admin binary 側の Product handler 到達口を拒否する。
	assertAdminGoBindingsExcludeProductOperations(t, productOperations)
}

// TestAPIContractBES003SurfaceServerURLsAreSeparated は、Product と Admin の OpenAPI が別々の server domain を公開していることを検証する。
//
// このテストは [API-CONTRACT-BE-S003] の受け皿であり、両 surface が同じ `/api/v1/*` path 空間を使っても origin/domain で分離される契約を固定する。
// 引数は testing.T だけで、戻り値はない。server URL が未定義、相対 URL、または同一 domain の場合は契約違反として失敗する。
func TestAPIContractBES003SurfaceServerURLsAreSeparated(t *testing.T) {
	t.Parallel()

	// Step 1: Product OpenAPI の server URL を構造化して読み、Product surface が絶対 URL の domain を公開していることを確認する。
	productDomain := readOpenAPIServerDomain(t, productOpenAPIArtifactPath, readProductOpenAPIArtifact(t))

	// Step 2: Admin OpenAPI の server URL も同じ helper で読み、Product と同じ判定基準で domain を抽出する。
	adminDomain := readOpenAPIServerDomain(t, adminOpenAPIArtifactPath, readAdminOpenAPIArtifact(t))

	// Step 3: 同一 path 空間でも別 origin として運用される契約を守るため、Product/Admin の domain 一致を即座に失敗させる。
	if productDomain == adminDomain {
		t.Fatalf("Product and Admin OpenAPI servers must use distinct domains: product=%s admin=%s", productDomain, adminDomain)
	}
}

// TestAPIContractBES004SharedModelImportsDoNotAddRoutes は、共有 model import が Product/Admin の route surface を増やさないことを検証する。
//
// このテストは [API-CONTRACT-BE-S004] の受け皿であり、共有 model を両 surface から参照しても `paths` に現れる route は各 surface の明示 route だけであることを固定する。
// 引数は testing.T だけで、戻り値はない。期待外の path / method が増えた場合は、共有 model import が route namespace を引き込んだ可能性として失敗する。
func TestAPIContractBES004SharedModelImportsDoNotAddRoutes(t *testing.T) {
	t.Parallel()

	// Step 1: Product surface の許可 route を列挙し、共有 model import によって Admin route や未承認 route が増えた場合に検知できるようにする。
	assertOpenAPIRouteSurfaceEqual(t, productOpenAPIArtifactPath, readProductOpenAPIArtifact(t), []openAPIRouteSurface{
		{path: "/api/v1/account/settings", methods: []string{"get", "patch"}},
		{path: "/api/v1/auth/logout", methods: []string{"post"}},
		{path: "/api/v1/auth/passkey/finish", methods: []string{"post"}},
		{path: "/api/v1/auth/passkey/register", methods: []string{"post"}},
		{path: "/api/v1/auth/passkey/register/start", methods: []string{"post"}},
		{path: "/api/v1/auth/passkey/start", methods: []string{"post"}},
		{path: "/api/v1/auth/reauth/finish", methods: []string{"post"}},
		{path: "/api/v1/auth/reauth/start", methods: []string{"post"}},
		{path: "/api/v1/auth/recovery", methods: []string{"post"}},
		{path: "/api/v1/auth/recovery/consume", methods: []string{"post"}},
		{path: "/api/v1/auth/refresh", methods: []string{"post"}},
		{path: "/api/v1/passkeys", methods: []string{"get"}},
		{path: "/api/v1/passkeys/finish", methods: []string{"post"}},
		{path: "/api/v1/passkeys/send-device-link", methods: []string{"post"}},
		{path: "/api/v1/passkeys/start", methods: []string{"post"}},
		{path: "/api/v1/passkeys/{id}", methods: []string{"delete"}},
		{path: "/api/v1/sessions", methods: []string{"get"}},
		{path: "/api/v1/sessions/others", methods: []string{"delete"}},
		{path: "/api/v1/sessions/{id}", methods: []string{"delete"}},
		{path: "/api/v1/status", methods: []string{"get"}},
	})

	// Step 2: Admin surface の許可 route も同じ方式で列挙し、共有 model import が Product route を Admin artifact へ追加しないことを検証する。
	assertOpenAPIRouteSurfaceEqual(t, adminOpenAPIArtifactPath, readAdminOpenAPIArtifact(t), []openAPIRouteSurface{
		{path: "/api/v1/accounts", methods: []string{"get", "post"}},
		{path: "/api/v1/accounts/{accountId}", methods: []string{"get"}},
		{path: "/api/v1/auth/operator-setup/finish", methods: []string{"post"}},
		{path: "/api/v1/auth/operator-setup/start", methods: []string{"post"}},
		{path: "/api/v1/auth/operator/current", methods: []string{"get"}},
		{path: "/api/v1/auth/operator/logout", methods: []string{"post"}},
		{path: "/api/v1/auth/operator/refresh", methods: []string{"post"}},
		{path: "/api/v1/auth/operators", methods: []string{"post"}},
		{path: "/api/v1/auth/passkey/finish", methods: []string{"post"}},
		{path: "/api/v1/auth/passkey/start", methods: []string{"post"}},
		{path: "/api/v1/auth/passkeys", methods: []string{"get"}},
		{path: "/api/v1/auth/passkeys/{id}", methods: []string{"delete"}},
		{path: "/api/v1/auth/setup/finish", methods: []string{"post"}},
		{path: "/api/v1/auth/setup/start", methods: []string{"post"}},
	})
}

type adminOperationName struct {
	lowerCamel string
	pascal     string
}

type productOperationName struct {
	lowerCamel string
	pascal     string
}

type openAPIRouteSurface struct {
	path    string
	methods []string
}

func assertOpenAPIRouteSurfaceEqual(t *testing.T, artifactPath string, content []byte, expected []openAPIRouteSurface) {
	t.Helper()

	// Step 1: OpenAPI artifact を method key が欠落しない汎用 map として読み込み、components ではなく paths だけを route surface として検査対象にする。
	var contract openAPIRouteSurfaceContract
	if err := json.Unmarshal(content, &contract); err != nil {
		t.Fatalf("unmarshal %s for route surface check: %v", artifactPath, err)
	}

	// Step 2: 実際の paths と期待 paths を比較しやすい map に正規化して、順序差ではなく route 増減だけを検知する。
	actualSurface := collectOpenAPIRouteSurface(contract)
	expectedSurface := mapOpenAPIRouteSurface(expected)

	// Step 3: 期待数と実数を先に比較し、共有 model import による追加または削除が起きたときに概要を即座に示す。
	if len(actualSurface) != len(expectedSurface) {
		t.Fatalf("%s route surface count mismatch: got %d paths %v, want %d paths %v", artifactPath, len(actualSurface), actualSurface, len(expectedSurface), expectedSurface)
	}

	// Step 4: 各 path の存在と method 集合を検査し、path が同じでも method が増えた route surface 拡張を拒否する。
	for actualPath, actualMethods := range actualSurface {
		expectedMethods, ok := expectedSurface[actualPath]
		if !ok {
			t.Fatalf("%s includes unexpected route path %q with methods %v", artifactPath, actualPath, actualMethods)
		}
		if strings.Join(actualMethods, ",") != strings.Join(expectedMethods, ",") {
			t.Fatalf("%s route %q methods mismatch: got %v, want %v", artifactPath, actualPath, actualMethods, expectedMethods)
		}
	}

	// Step 5: 期待 route が欠落した場合も surface 定義の破壊として報告し、追加だけでなく削除も検知する。
	for expectedPath, expectedMethods := range expectedSurface {
		if _, ok := actualSurface[expectedPath]; !ok {
			t.Fatalf("%s is missing expected route path %q with methods %v", artifactPath, expectedPath, expectedMethods)
		}
	}
}

func collectOpenAPIRouteSurface(contract openAPIRouteSurfaceContract) map[string][]string {
	// Step 1: OpenAPI の paths を raw path item として走査し、未知の operation struct 定義に依存せず route method key の増加を検知する。
	routeSurface := make(map[string][]string, len(contract.Paths))
	for path, item := range contract.Paths {
		routeSurface[path] = collectOpenAPIPathMethodKeys(item)
	}

	// Step 2: 呼び出し側が期待値と比較できるよう、path を key、昇順 method slice を value にした map を返す。
	return routeSurface
}

func mapOpenAPIRouteSurface(routeSurfaces []openAPIRouteSurface) map[string][]string {
	// Step 1: 期待 route 定義を map 化し、テスト本体の読みやすい slice 表現と比較処理を分離する。
	mappedSurface := make(map[string][]string, len(routeSurfaces))
	for _, routeSurface := range routeSurfaces {
		methods := append([]string(nil), routeSurface.methods...)
		sort.Strings(methods)
		mappedSurface[routeSurface.path] = methods
	}

	// Step 2: 正規化済みの期待 route surface を返し、method 順序の揺れで失敗しないようにする。
	return mappedSurface
}

func collectOpenAPIPathMethodKeys(pathItem map[string]json.RawMessage) []string {
	// Step 1: OpenAPI 3.0 の正式な HTTP method key をすべて列挙し、put/head/options などの追加も route surface 増加として扱う。
	openAPIHTTPMethods := []string{"delete", "get", "head", "options", "patch", "post", "put", "trace"}
	methods := make([]string, 0, len(openAPIHTTPMethods))
	for _, method := range openAPIHTTPMethods {
		if _, ok := pathItem[method]; ok {
			methods = append(methods, method)
		}
	}

	// Step 2: 期待値比較を安定化するため、method 一覧を辞書順にして返す。
	sort.Strings(methods)
	return methods
}

func assertProductOpenAPIExcludesAdminOperations(t *testing.T, adminOperations []adminOperationName) {
	t.Helper()

	// Step 1: Product OpenAPI を構造化して読み、文字列検索ではなく operationId / tags の契約フィールドを検査する。
	content := readProductOpenAPIArtifact(t)
	var contract openAPIContract
	if err := json.Unmarshal(content, &contract); err != nil {
		t.Fatalf("unmarshal Product OpenAPI for contamination check: %v", err)
	}

	// Step 2: top-level tags は OpenAPI 全体の surface 分類であり、path に operation がなくても Admin 分類の混入を示すため最初に拒否する。
	assertTopLevelTagsAreNotAdmin(t, productOpenAPIArtifactPath, contract.Tags)

	// Step 3: すべての path / method を走査し、Admin operationId または Admin tag が 1 つでも出たら Product surface 混入として失敗させる。
	for path, item := range contract.Paths {
		operationsByMethod := map[string]openAPIOperation{
			"delete": item.Delete,
			"get":    item.Get,
			"patch":  item.Patch,
			"post":   item.Post,
		}
		for method, operation := range operationsByMethod {
			assertOperationIDIsNotAdmin(t, productOpenAPIArtifactPath, path, method, operation.OperationID, adminOperations)
			assertOperationTagsAreNotAdmin(t, productOpenAPIArtifactPath, path, method, operation.Tags)
		}
	}
}

func assertTopLevelTagsAreNotAdmin(t *testing.T, artifactPath string, tags []openAPITag) {
	t.Helper()

	// Step 1: Product OpenAPI の top-level tags を検査し、Admin operation group が path 外の metadata として混入する状態を拒否する。
	for _, tag := range tags {
		if tag.Name == "admin-accounts" || tag.Name == "admin-auth" {
			t.Fatalf("%s includes Admin top-level tag %q", artifactPath, tag.Name)
		}
	}
}

func assertOperationIDIsNotAdmin(t *testing.T, artifactPath string, path string, method string, operationID string, adminOperations []adminOperationName) {
	t.Helper()

	// Step 1: operation が存在しない method のゼロ値は検査対象外にして、OpenAPI path item の未定義 method を許可する。
	if operationID == "" {
		return
	}

	// Step 2: operationId が Admin operation 一覧に一致した場合、どの Product path へ混入したかを failure message に含める。
	for _, adminOperation := range adminOperations {
		if operationID == adminOperation.lowerCamel {
			t.Fatalf("%s includes Admin operationId %q at %s %s", artifactPath, operationID, method, path)
		}
	}
}

func assertOperationTagsAreNotAdmin(t *testing.T, artifactPath string, path string, method string, tags []string) {
	t.Helper()

	// Step 1: Admin route namespace の代表 tag を Product OpenAPI では禁止し、operationId 以外の分類情報からの Admin 混入も検知する。
	for _, tag := range tags {
		if tag == "admin-accounts" || tag == "admin-auth" {
			t.Fatalf("%s includes Admin tag %q at %s %s", artifactPath, tag, method, path)
		}
	}
}

func assertProductSDKExcludesAdminOperations(t *testing.T, adminOperations []adminOperationName) {
	t.Helper()

	// Step 1: Product SDK の生成済み TypeScript を読み、Admin operationId 由来の export がないことだけを検査する。
	content := string(readProductSDKArtifact(t))

	// Step 2: Orval の関数 export / URL helper / response type export の命名を検査し、Product SDK から Admin operation を呼べないことを保証する。
	for _, adminOperation := range adminOperations {
		forbiddenTokens := []string{
			"export const " + adminOperation.lowerCamel,
			"export const get" + adminOperation.pascal + "Url",
			"export type " + adminOperation.lowerCamel,
		}
		assertArtifactExcludesTokens(t, productSDKArtifactPath, content, forbiddenTokens)
	}
}

func assertProductGoBindingsExcludeAdminOperations(t *testing.T, adminOperations []adminOperationName) {
	t.Helper()

	// Step 1: Product Go bindings の生成済み source を読み、Admin operation 名が Go export として存在しないことを検査する。
	content := string(readProductGoBindingsArtifact(t))

	// Step 2: oapi-codegen の interface method / wrapper method / request object type の命名を検査し、Product handler 実装に Admin operation が要求されないことを保証する。
	for _, adminOperation := range adminOperations {
		forbiddenTokens := []string{
			adminOperation.pascal + "(c *gin.Context)",
			"type " + adminOperation.pascal,
			adminOperation.pascal + "RequestObject",
		}
		assertArtifactExcludesTokens(t, productGoBindingsArtifactPath, content, forbiddenTokens)
	}
}

func assertAdminOpenAPIExcludesProductOperations(t *testing.T, productOperations []productOperationName) {
	t.Helper()

	// Step 1: Admin OpenAPI を構造化して読み、文字列検索ではなく operationId / tags の契約フィールドを検査する。
	content := readAdminOpenAPIArtifact(t)
	var contract openAPIContract
	if err := json.Unmarshal(content, &contract); err != nil {
		t.Fatalf("unmarshal Admin OpenAPI for contamination check: %v", err)
	}

	// Step 2: top-level tags は OpenAPI 全体の surface 分類であり、path に operation がなくても Product 分類の混入を示すため最初に拒否する。
	assertTopLevelTagsAreNotProduct(t, adminOpenAPIArtifactPath, contract.Tags)

	// Step 3: すべての path / method を走査し、Product operationId または Product tag が 1 つでも出たら Admin surface 混入として失敗させる。
	for path, item := range contract.Paths {
		operationsByMethod := map[string]openAPIOperation{
			"delete": item.Delete,
			"get":    item.Get,
			"patch":  item.Patch,
			"post":   item.Post,
		}
		for method, operation := range operationsByMethod {
			assertOperationIDIsNotProduct(t, adminOpenAPIArtifactPath, path, method, operation.OperationID, productOperations)
			assertOperationTagsAreNotProduct(t, adminOpenAPIArtifactPath, path, method, operation.Tags)
		}
	}
}

func assertTopLevelTagsAreNotProduct(t *testing.T, artifactPath string, tags []openAPITag) {
	t.Helper()

	// Step 1: Admin OpenAPI の top-level tags を検査し、Product operation group が path 外の metadata として混入する状態を拒否する。
	for _, tag := range tags {
		if isProductOperationTag(tag.Name) {
			t.Fatalf("%s includes Product top-level tag %q", artifactPath, tag.Name)
		}
	}
}

func assertOperationIDIsNotProduct(t *testing.T, artifactPath string, path string, method string, operationID string, productOperations []productOperationName) {
	t.Helper()

	// Step 1: operation が存在しない method のゼロ値は検査対象外にして、OpenAPI path item の未定義 method を許可する。
	if operationID == "" {
		return
	}

	// Step 2: operationId が Product operation 一覧に一致した場合、どの Admin path へ混入したかを failure message に含める。
	for _, productOperation := range productOperations {
		if operationID == productOperation.lowerCamel {
			t.Fatalf("%s includes Product operationId %q at %s %s", artifactPath, operationID, method, path)
		}
	}
}

func assertOperationTagsAreNotProduct(t *testing.T, artifactPath string, path string, method string, tags []string) {
	t.Helper()

	// Step 1: Product route namespace の代表 tag を Admin OpenAPI では禁止し、operationId 以外の分類情報からの Product 混入も検知する。
	for _, tag := range tags {
		if isProductOperationTag(tag) {
			t.Fatalf("%s includes Product tag %q at %s %s", artifactPath, tag, method, path)
		}
	}
}

func isProductOperationTag(tag string) bool {
	// Step 1: Product route namespace で実際に使われる operation tag だけを true にし、Admin read model 内の Product 説明語は誤検知しない。
	return tag == "account-settings" || tag == "app-auth" || tag == "auth" || tag == "status"
}

func assertAdminSDKExcludesProductOperations(t *testing.T, productOperations []productOperationName) {
	t.Helper()

	// Step 1: Admin SDK の生成済み TypeScript を読み、Product operationId 由来の export がないことだけを検査する。
	content := string(readAdminSDKArtifact(t))

	// Step 2: Orval の関数 export / URL helper / response type export の命名を検査し、Admin SDK から Product operation を呼べないことを保証する。
	for _, productOperation := range productOperations {
		forbiddenTokens := []string{
			"export const " + productOperation.lowerCamel + " =",
			"export const get" + productOperation.pascal + "Url",
			"export type " + productOperation.lowerCamel + "Response",
		}
		assertArtifactExcludesTokens(t, adminSDKArtifactPath, content, forbiddenTokens)
	}
}

func assertAdminGoBindingsExcludeProductOperations(t *testing.T, productOperations []productOperationName) {
	t.Helper()

	// Step 1: Admin Go bindings の生成済み source を読み、Product operation 名が Go export として存在しないことを検査する。
	content := string(readAdminGoBindingsArtifact(t))

	// Step 2: oapi-codegen の interface method / wrapper method / request object type の命名を検査し、Admin handler 実装に Product operation が要求されないことを保証する。
	for _, productOperation := range productOperations {
		forbiddenTokens := []string{
			productOperation.pascal + "(c *gin.Context)",
			"type " + productOperation.pascal + "RequestObject",
			productOperation.pascal + "(ctx context.Context, request " + productOperation.pascal + "RequestObject)",
		}
		assertArtifactExcludesTokens(t, adminGoBindingsArtifactPath, content, forbiddenTokens)
	}
}

func assertArtifactExcludesTokens(t *testing.T, artifactPath string, content string, forbiddenTokens []string) {
	t.Helper()

	// Step 1: artifact ごとの禁止 token を個別に照合し、どの生成物にどの surface operation export が混入したかを特定できる failure にする。
	for _, forbiddenToken := range forbiddenTokens {
		if strings.Contains(content, forbiddenToken) {
			t.Fatalf("%s includes forbidden operation token %q", artifactPath, forbiddenToken)
		}
	}
}

func readOpenAPIServerDomain(t *testing.T, artifactPath string, content []byte) string {
	t.Helper()

	// Step 1: OpenAPI artifact を JSON として decode し、servers 配列の有無を契約フィールドとして検査する。
	var contract openAPIContract
	if err := json.Unmarshal(content, &contract); err != nil {
		t.Fatalf("unmarshal %s for server domain check: %v", artifactPath, err)
	}
	if len(contract.Servers) == 0 {
		t.Fatalf("%s must declare at least one OpenAPI server", artifactPath)
	}

	// Step 2: 先頭 server URL を parse し、相対 URL や host のない URL を拒否して domain 分離を観測可能にする。
	serverURL := strings.TrimSpace(contract.Servers[0].URL)
	parsedURL, err := url.Parse(serverURL)
	if err != nil {
		t.Fatalf("%s has invalid OpenAPI server URL %q: %v", artifactPath, serverURL, err)
	}
	if parsedURL.Scheme == "" || parsedURL.Hostname() == "" {
		t.Fatalf("%s must declare an absolute OpenAPI server URL with a domain, got %q", artifactPath, serverURL)
	}

	// Step 3: scheme や port の差ではなく domain の分離を検査するため、hostname だけを小文字化して呼び出し元へ返す。
	return strings.ToLower(parsedURL.Hostname())
}

const (
	productOpenAPIArtifactPath    = "../../../../typespec/openapi/openapi.json"
	productSDKArtifactPath        = "../../../../frontend/api/src/generated/client.ts"
	productGoBindingsArtifactPath = "../../generated/openapi/openapi.gen.go"
	adminOpenAPIArtifactPath      = "../../../../typespec/openapi/admin.openapi.json"
	adminSDKArtifactPath          = "../../../../admin/api/src/generated/client.ts"
	adminGoBindingsArtifactPath   = "../../generated/adminopenapi/openapi.gen.go"
)

func readProductOpenAPIArtifact(t *testing.T) []byte {
	t.Helper()

	// Step 1: Product OpenAPI の固定生成先だけを読み、任意 path 読み込みではなく contract artifact の検査に限定する。
	content, err := os.ReadFile(productOpenAPIArtifactPath)
	if err != nil {
		t.Fatalf("read Product artifact %s: %v", productOpenAPIArtifactPath, err)
	}

	// Step 2: 呼び出し側が JSON decode を行えるように、生 bytes のまま返す。
	return content
}

func readProductSDKArtifact(t *testing.T) []byte {
	t.Helper()

	// Step 1: Product SDK の固定生成先だけを読み、Admin SDK や任意 file を誤って検査対象にしない。
	content, err := os.ReadFile(productSDKArtifactPath)
	if err != nil {
		t.Fatalf("read Product artifact %s: %v", productSDKArtifactPath, err)
	}

	// Step 2: 呼び出し側が TypeScript export token を検査できるように、生 bytes のまま返す。
	return content
}

func readProductGoBindingsArtifact(t *testing.T) []byte {
	t.Helper()

	// Step 1: Product Go bindings の固定生成先だけを読み、Admin bindings の内容と混同しない。
	content, err := os.ReadFile(productGoBindingsArtifactPath)
	if err != nil {
		t.Fatalf("read Product artifact %s: %v", productGoBindingsArtifactPath, err)
	}

	// Step 2: 呼び出し側が Go export token を検査できるように、生 bytes のまま返す。
	return content
}

func readAdminOpenAPIArtifact(t *testing.T) []byte {
	t.Helper()

	// Step 1: Admin OpenAPI の固定生成先だけを読み、任意 path 読み込みではなく contract artifact の検査に限定する。
	content, err := os.ReadFile(adminOpenAPIArtifactPath)
	if err != nil {
		t.Fatalf("read Admin artifact %s: %v", adminOpenAPIArtifactPath, err)
	}

	// Step 2: 呼び出し側が JSON decode を行えるように、生 bytes のまま返す。
	return content
}

func readAdminSDKArtifact(t *testing.T) []byte {
	t.Helper()

	// Step 1: Admin SDK の固定生成先だけを読み、Product SDK や任意 file を誤って検査対象にしない。
	content, err := os.ReadFile(adminSDKArtifactPath)
	if err != nil {
		t.Fatalf("read Admin artifact %s: %v", adminSDKArtifactPath, err)
	}

	// Step 2: 呼び出し側が TypeScript export token を検査できるように、生 bytes のまま返す。
	return content
}

func readAdminGoBindingsArtifact(t *testing.T) []byte {
	t.Helper()

	// Step 1: Admin Go bindings の固定生成先だけを読み、Product bindings の内容と混同しない。
	content, err := os.ReadFile(adminGoBindingsArtifactPath)
	if err != nil {
		t.Fatalf("read Admin artifact %s: %v", adminGoBindingsArtifactPath, err)
	}

	// Step 2: 呼び出し側が Go export token を検査できるように、生 bytes のまま返す。
	return content
}

func assertBearerSecurity(t *testing.T, contract openAPIContract, path string, method string) {
	t.Helper()

	item, ok := contract.Paths[path]
	if !ok {
		t.Fatalf("path %s not found", path)
	}

	security := item.Get.Security
	if method == "post" {
		security = item.Post.Security
	}

	for _, entry := range security {
		if _, ok := entry["BearerAuth"]; ok {
			return
		}
	}

	t.Fatalf("BearerAuth security missing for %s", path)
}
