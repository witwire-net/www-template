package main

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// [ADMIN-CONSOLE-BE-S080] internal/domain import graph が stdlib だけを許可することを検証する。
func TestAdminConsoleBES080DomainImportGraphAllowsPureDomain(t *testing.T) {
	t.Parallel()

	// Step 1: stdlib のみを import する domain file を AST 化し、flat domain が外側 layer に依存しない正常系を表す。
	file := parseGuardrailTestFile(t, `package domain

import "time"

type PureClock struct {
	At time.Time
}
`)

	// Step 2: import guardrail を domain path として評価し、stdlib import が purity violation にならないことを確認する。
	violations := checkImports("internal/domain/pure_clock.go", file)
	if len(violations) != 0 {
		t.Fatalf("expected pure domain imports to pass, got violations: %v", violations)
	}
}

// [ADMIN-CONSOLE-BE-S080] internal/domain import graph が application / adapter / generated / platform import を拒否することを検証する。
func TestAdminConsoleBES080DomainImportGraphRejectsOuterLayers(t *testing.T) {
	t.Parallel()

	// Step 1: domain から見て外側にあたる禁止 layer を列挙し、将来の層追加時にも失敗箇所が読みやすい table にする。
	forbiddenImports := []struct {
		name       string
		importPath string
	}{
		{name: "application", importPath: modulePath + "/internal/application"},
		{name: "adapter", importPath: modulePath + "/internal/adapter/http"},
		{name: "generated", importPath: modulePath + "/internal/generated/openapi"},
		{name: "platform", importPath: modulePath + "/internal/platform/config"},
	}

	for _, forbiddenImport := range forbiddenImports {
		// Step 2: table case ごとに独立した subtest とし、どの layer への逆依存が混入したかを出力名で即座に特定できるようにする。
		t.Run(forbiddenImport.name, func(t *testing.T) {
			t.Parallel()

			// Step 3: 禁止 import を含む最小 Go source を AST 化し、実ファイルを汚さず guardrail の判定だけを検証する。
			file := parseGuardrailTestFile(t, `package domain

import _ "`+forbiddenImport.importPath+`"
`)

			// Step 4: domain file として import guardrail を実行し、外側 layer への依存が必ず violation になることを確認する。
			violations := checkImports("internal/domain/impure_dependency.go", file)
			assertGuardrailViolationMentions(t, violations, forbiddenImport.importPath)
		})
	}
}

// [API-CONTRACT-BE-S005] Product surface の TypeSpec source が Admin route namespace を import できないことを検証する。
func TestAPIContractBES005ProductTypespecRejectsAdminNamespaceImport(t *testing.T) {
	t.Parallel()

	// Step 1: 本物の Product route source を検査し、clean source が境界 script を通ることを確認する。
	checkerPath := repositoryPath(t, "packages/typespec/scripts/check-surface-boundaries.mjs")
	cleanCommand := exec.Command("node", checkerPath) //nolint:gosec // test は repository 内の固定 checker path を実行し、外部入力の command 名を使わない。
	cleanCommand.Dir = repositoryRoot(t)
	if output, err := cleanCommand.CombinedOutput(); err != nil {
		t.Fatalf("expected clean Product TypeSpec surface to pass: %v\n%s", err, string(output))
	}

	// Step 2: Product route source が Admin route file を import する contamination fixture を一時 file に作る。
	fixturePath := filepath.Join(t.TempDir(), "product_imports_admin.tsp")
	fixtureSource := `import "@typespec/http";
import "../../src/routes/admin-v1/accounts.tsp";

using Http;

namespace WWWTemplate.ApiV1;
`
	if err := os.WriteFile(fixturePath, []byte(fixtureSource), 0o600); err != nil {
		t.Fatalf("write TypeSpec contamination fixture: %v", err)
	}

	// Step 3: fixture を明示入力として渡し、Admin route namespace import が contract lint violation になることを固定する。
	contaminatedCommand := exec.Command("node", checkerPath, fixturePath) //nolint:gosec // test は固定 checker に一時 fixture path だけを渡し、shell 展開を使わない。
	contaminatedCommand.Dir = repositoryRoot(t)
	output, err := contaminatedCommand.CombinedOutput()
	if err == nil {
		t.Fatalf("expected Product TypeSpec fixture importing Admin namespace to fail, got success")
	}
	if !bytes.Contains(output, []byte("Product route source must not import Admin route")) {
		t.Fatalf("expected Admin route namespace violation, got output: %s", string(output))
	}
}

// [API-CONTRACT-BE-S006] Product artifact に Admin operation が混入すると codegen check の contamination pattern が検出することを検証する。
func TestAPIContractBES006CodegenCheckRejectsProductAdminOperationFixtures(t *testing.T) {
	t.Parallel()

	// Step 1: pnpm check:codegen が実際に参照する shell script から Product artifact 向けの禁止 pattern を抽出する。
	checkScript := readRepositoryFile(t, "scripts/codegen/check.sh")
	adminOpenAPIPattern := extractShellSingleQuotedVariable(t, checkScript, "admin_openapi_contamination")
	adminTypeScriptPattern := extractShellSingleQuotedVariable(t, checkScript, "admin_typescript_export_contamination")
	adminGoPattern := extractShellSingleQuotedVariable(t, checkScript, "admin_go_export_contamination")

	// Step 2: Product OpenAPI に Admin operationId / tag が混入した fixture を拒否することを確認する。
	assertPatternMatchesFixture(t, adminOpenAPIPattern, `"operationId": "createAdminAccount"`)
	assertPatternMatchesFixture(t, adminOpenAPIPattern, `"tags": ["admin-accounts"]`)

	// Step 3: Product SDK に Admin export が混入した fixture を拒否することを確認する。
	assertPatternMatchesFixture(t, adminTypeScriptPattern, `export type AdminAccountCreateResponse = { requestId: string }`)
	assertPatternMatchesFixture(t, adminTypeScriptPattern, `export const createAdminAccount = () => undefined`)

	// Step 4: Product Go bindings に Admin operation export が混入した fixture を拒否することを確認する。
	assertPatternMatchesFixture(t, adminGoPattern, `type AdminAccountLeak struct {}`)
	assertPatternMatchesFixture(t, adminGoPattern, `func StartAdminOperatorSetup() {}`)
}

// [API-CONTRACT-BE-S007] [ADMIN-CONSOLE-BE-S070] Product binary が Admin generated bindings を import できないことを検証する。
func TestAPIContractBES007ProductBinaryRejectsAdminBindingsImport(t *testing.T) {
	t.Parallel()

	// Step 1: Product binary の main package が Admin bindings を import する最小 AST fixture を作る。
	file := parseGuardrailTestFile(t, `package main

import _ "`+modulePath+`/internal/generated/adminopenapi"
`)

	// Step 2: cmd/api/main.go として import guardrail を実行し、Product binary から Admin bindings への経路が拒否されることを確認する。
	violations := checkImports("cmd/api/main.go", file)
	assertGuardrailViolationMentions(t, violations, modulePath+"/internal/generated/adminopenapi")
}

// [API-CONTRACT-BE-S007] Product bindings は Product HTTP adapter だけが import できることを検証する。
func TestAPIContractBES007ProductBindingsAreOnlyImportedByProductHTTPAdapter(t *testing.T) {
	t.Parallel()

	// Step 1: Product bindings import を含む最小 AST fixture を作り、path ごとの境界判定だけを検証する。
	file := parseGuardrailTestFile(t, `package product

import _ "`+modulePath+`/internal/generated/openapi"
`)

	forbiddenPaths := []string{
		"cmd/api/main.go",
		"internal/app/runtime.go",
		"internal/application/account_service.go",
		"internal/domain/account.go",
		"internal/adapter/http/router.go",
		"internal/adapter/http/admin/router.go",
		"internal/adapter/http/shared/cookie.go",
	}

	// Step 2: Product HTTP adapter subtree 以外から Product bindings への import をすべて拒否し、legacy flat adapter への逆戻りも検出する。
	for _, path := range forbiddenPaths {
		path := path
		t.Run(path, func(t *testing.T) {
			t.Parallel()
			violations := checkImports(path, file)
			assertGuardrailViolationMentions(t, violations, modulePath+"/internal/generated/openapi")
		})
	}

	// Step 3: Product HTTP adapter subtree だけは Product bindings import を許可し、4.7 の物理分離後の正常系を固定する。
	violations := checkImports("internal/adapter/http/product/router.go", file)
	if len(violations) != 0 {
		t.Fatalf("expected Product HTTP adapter to import Product bindings, got violations: %v", violations)
	}
}

// [API-CONTRACT-BE-S008] Admin bindings は Admin HTTP adapter だけが import できることを検証する。
func TestAPIContractBES008AdminBindingsAreOnlyImportedByAdminHTTPAdapter(t *testing.T) {
	t.Parallel()

	// Step 1: Admin bindings import を含む最小 AST fixture を 1 つ作り、path ごとの境界判定だけを比較する。
	file := parseGuardrailTestFile(t, `package http

import _ "`+modulePath+`/internal/generated/adminopenapi"
`)

	forbiddenPaths := []string{
		"cmd/api/main.go",
		"internal/app/runtime.go",
		"internal/application/account_service.go",
		"internal/domain/account.go",
		"internal/adapter/http/router.go",
		"internal/adapter/http/product/router.go",
	}

	// Step 2: Product binary / app / application / domain / Product HTTP adapter からの Admin bindings import をすべて拒否する。
	for _, path := range forbiddenPaths {
		path := path
		t.Run(path, func(t *testing.T) {
			t.Parallel()
			violations := checkImports(path, file)
			assertGuardrailViolationMentions(t, violations, modulePath+"/internal/generated/adminopenapi")
		})
	}

	// Step 3: Admin HTTP adapter subtree だけは Admin bindings import を許可し、正しい surface の clean source を通す。
	violations := checkImports("internal/adapter/http/admin/router.go", file)
	if len(violations) != 0 {
		t.Fatalf("expected Admin HTTP adapter to import Admin bindings, got violations: %v", violations)
	}
}

// [ADMIN-CONSOLE-BE-S072] Admin HTTP adapter が persistence adapter を import できないことを検証する。
func TestAdminConsoleBES072AdminHTTPAdapterRejectsPersistenceAdapters(t *testing.T) {
	t.Parallel()

	// Step 1: Admin HTTP adapter から Postgres / Valkey adapter へ直接依存する禁止 import を table に固定する。
	forbiddenCases := []struct {
		name       string
		importPath string
	}{
		{name: "admin http imports postgres", importPath: modulePath + "/internal/adapter/postgres/admin"},
		{name: "admin http imports valkey", importPath: modulePath + "/internal/adapter/valkey/admin"},
	}

	for _, forbiddenCase := range forbiddenCases {
		forbiddenCase := forbiddenCase
		// Step 2: 禁止 import ごとに最小 fixture を作り、layer allowlist が HTTP→persistence の近道を拒否することを確認する。
		t.Run(forbiddenCase.name, func(t *testing.T) {
			t.Parallel()
			file := parseGuardrailTestFile(t, `package admin

import _ "`+forbiddenCase.importPath+`"
`)
			violations := checkImports("internal/adapter/http/admin/router.go", file)
			assertGuardrailViolationMentions(t, violations, forbiddenCase.importPath)
		})
	}
}

// [ADMIN-CONSOLE-BE-S073] application port が adapter 型や generated 型を公開できないことを検証する。
func TestAdminConsoleBES073ApplicationPortRejectsAdapterAndGeneratedTypes(t *testing.T) {
	t.Parallel()

	// Step 1: application interface の method signature に混入しやすい generated 型と adapter 型を禁止 fixture として固定する。
	forbiddenCases := []struct {
		name       string
		importPath string
		typeName   string
	}{
		{name: "generated binding request", importPath: modulePath + "/internal/generated/adminopenapi", typeName: "CreateAdminAccountRequestObject"},
		{name: "adapter repository record", importPath: modulePath + "/internal/adapter/postgres/admin", typeName: "AccountRecord"},
	}

	for _, forbiddenCase := range forbiddenCases {
		forbiddenCase := forbiddenCase
		// Step 2: port purity guardrail だけを直接実行し、import boundary 以外の偶然の failure で成功扱いにならないようにする。
		t.Run(forbiddenCase.name, func(t *testing.T) {
			t.Parallel()
			file := parseGuardrailTestFile(t, `package application

import dependency "`+forbiddenCase.importPath+`"

type LeakyAdminPort interface {
	Save(record dependency.`+forbiddenCase.typeName+`) error
}
`)
			violations := checkPortPurity("internal/application/admin/leaky_port.go", file)
			assertGuardrailViolationContains(t, violations, "must not depend on transport or persistence types")
		})
	}
}

// [ADMIN-CONSOLE-BE-S072] 4.2 の backend surface 配置が guardrail によって許可されることを検証する。
func TestAdminConsoleBES072BackendSurfacePlacementsAreAllowed(t *testing.T) {
	t.Parallel()

	// Step 1: 4.2 で明示された配置と package 名の対応を table に固定し、将来の分割作業が placement guardrail で止まらないことを保証する。
	allowedFiles := []struct {
		path        string
		packageName string
	}{
		{path: "cmd/admin-api/main.go", packageName: "main"},
		{path: "internal/generated/adminopenapi/openapi.gen.go", packageName: "adminopenapi"},
		{path: "internal/adapter/http/product/router.go", packageName: "product"},
		{path: "internal/adapter/http/admin/router.go", packageName: "admin"},
		{path: "internal/adapter/http/shared/cookie.go", packageName: "shared"},
		{path: "internal/application/product/service.go", packageName: "application"},
		{path: "internal/application/admin/service.go", packageName: "application"},
		{path: "internal/application/shared/tokenprimitive/ttl.go", packageName: "application"},
		{path: "internal/adapter/postgres/product/account_repository.go", packageName: "product"},
		{path: "internal/adapter/postgres/admin/account_repository.go", packageName: "admin"},
		{path: "internal/adapter/valkey/product/account_session_store.go", packageName: "product"},
		{path: "internal/adapter/valkey/admin/operator_session_store.go", packageName: "admin"},
	}

	for _, allowedFile := range allowedFiles {
		allowedFile := allowedFile
		// Step 2: 各配置を subtest 化し、どの package layout が拒否されたかを failure 名で確認できるようにする。
		t.Run(allowedFile.path, func(t *testing.T) {
			t.Parallel()

			// Step 3: generated file は collectViolations 上では package check 前に return するが、package policy 自体も固定して将来の手動配置を防ぐ。
			file := parseGuardrailTestFile(t, "package "+allowedFile.packageName+"\n")
			if violations := verifyGoFilePlacement(allowedFile.path); len(violations) != 0 {
				t.Fatalf("expected placement to pass, got violations: %v", violations)
			}
			if violations := checkPackageName(allowedFile.path, file); len(violations) != 0 {
				t.Fatalf("expected package name to pass, got violations: %v", violations)
			}
		})
	}
}

// [ADMIN-CONSOLE-BE-S075] Product/Admin application subtree の相互 import と legacy root 逆流を拒否することを検証する。
func TestAdminConsoleBES075ApplicationSurfaceImportBoundary(t *testing.T) {
	t.Parallel()

	// Step 1: Product application は shared/tokenprimitive だけを中立 helper として import できることを確認する。
	productImportsShared := parseGuardrailTestFile(t, `package auth

import _ "`+modulePath+`/internal/application/shared/tokenprimitive"
`)
	if violations := checkImports("internal/application/product/auth/service.go", productImportsShared); len(violations) != 0 {
		t.Fatalf("expected Product application to import shared tokenprimitive, got violations: %v", violations)
	}

	// Step 2: Product/Admin application の相互 import と root legacy application への逆流を拒否し、単一 service 共有へ戻れないようにする。
	forbiddenCases := []struct {
		name       string
		sourcePath string
		importPath string
	}{
		{name: "product imports admin", sourcePath: "internal/application/product/auth/service.go", importPath: modulePath + "/internal/application/admin/auth"},
		{name: "admin imports product", sourcePath: "internal/application/admin/auth/service.go", importPath: modulePath + "/internal/application/product/auth"},
		{name: "product imports legacy root", sourcePath: "internal/application/product/auth/service.go", importPath: modulePath + "/internal/application"},
		{name: "shared imports product", sourcePath: "internal/application/shared/tokenprimitive/signer.go", importPath: modulePath + "/internal/application/product/auth"},
	}

	for _, forbiddenCase := range forbiddenCases {
		forbiddenCase := forbiddenCase
		// Step 3: 禁止 case ごとに最小 import fixture を作り、surface 境界の violation が対象 path を含むことを確認する。
		t.Run(forbiddenCase.name, func(t *testing.T) {
			t.Parallel()
			file := parseGuardrailTestFile(t, `package auth

import _ "`+forbiddenCase.importPath+`"
`)
			violations := checkImports(forbiddenCase.sourcePath, file)
			assertGuardrailViolationMentions(t, violations, forbiddenCase.importPath)
		})
	}
}

// [ADMIN-CONSOLE-BE-S074] Account 不変条件を application 内の inline 検証で迂回できないことを検証する。
func TestAdminConsoleBES074AccountInvariantUsecaseBoundary(t *testing.T) {
	t.Parallel()

	// Step 1: Account 作成 use case が domain を直接呼ばない fixture を作り、AccountEmail / lifecycle constructor 迂回を拒否する。
	missingDomainTouch := parseGuardrailTestFile(t, `package application

type AdminCreateAccountInput struct {
	Email string
}

func CreateAccount(input AdminCreateAccountInput) error {
	return nil
}
`)
	missingDomainViolations := checkWriteUsecasesTouchDomain("internal/application/admin/account_creation_service.go", missingDomainTouch)
	assertGuardrailViolationContains(t, missingDomainViolations, "must call into domain directly")

	// Step 2: raw email の trim / 空文字判定を use case に置く fixture を作り、正規化と validation が domain object に残ることを固定する。
	inlineValidation := parseGuardrailTestFile(t, `package application

import "strings"

type AdminCreateAccountInput struct {
	Email string
}

func CreateAccount(input AdminCreateAccountInput) error {
	if strings.TrimSpace(input.Email) == "" {
		return nil
	}
	return nil
}
`)
	inlineValidationViolations := checkUsecaseInlineBusinessValidation("internal/application/admin/account_creation_service.go", inlineValidation)
	assertGuardrailViolationContains(t, inlineValidationViolations, "must delegate trimming and validation to domain")
}

// [ADMIN-CONSOLE-BE-S074] HTTP adapter は shared helper の片方向利用だけを許可し、Product/Admin 相互 import を拒否することを検証する。
func TestAdminConsoleBES074HTTPAdapterSurfaceImportBoundary(t *testing.T) {
	t.Parallel()

	// Step 1: Product HTTP adapter から shared HTTP helper への依存は Cookie などの中立 transport helper 用に許可する。
	productImportsShared := parseGuardrailTestFile(t, `package product

import _ "`+modulePath+`/internal/adapter/http/shared"
`)
	if violations := checkImports("internal/adapter/http/product/router.go", productImportsShared); len(violations) != 0 {
		t.Fatalf("expected Product HTTP adapter to import shared HTTP helper, got violations: %v", violations)
	}

	// Step 2: Product/Admin HTTP adapter の相互 import と shared から concrete surface への逆依存を拒否する。
	forbiddenCases := []struct {
		name       string
		sourcePath string
		importPath string
	}{
		{name: "product imports admin", sourcePath: "internal/adapter/http/product/router.go", importPath: modulePath + "/internal/adapter/http/admin"},
		{name: "admin imports product", sourcePath: "internal/adapter/http/admin/router.go", importPath: modulePath + "/internal/adapter/http/product"},
		{name: "shared imports admin", sourcePath: "internal/adapter/http/shared/cookie.go", importPath: modulePath + "/internal/adapter/http/admin"},
	}

	for _, forbiddenCase := range forbiddenCases {
		forbiddenCase := forbiddenCase
		// Step 3: 禁止 import が layer-level allow で偶然通らないことを、対象 import path を含む violation で確認する。
		t.Run(forbiddenCase.name, func(t *testing.T) {
			t.Parallel()
			file := parseGuardrailTestFile(t, `package product

import _ "`+forbiddenCase.importPath+`"
`)
			violations := checkImports(forbiddenCase.sourcePath, file)
			assertGuardrailViolationMentions(t, violations, forbiddenCase.importPath)
		})
	}
}

// [ADMIN-CONSOLE-BE-S076] Product/Admin persistence adapter は対応する application surface だけを import できることを検証する。
func TestAdminConsoleBES076PersistenceSurfaceImportBoundary(t *testing.T) {
	t.Parallel()

	// Step 1: Product persistence が Product application port を参照する正常系を固定し、subtree 移行後の repository 実装を許可する。
	productImportsProductApplication := parseGuardrailTestFile(t, `package product

import _ "`+modulePath+`/internal/application/product/auth"
`)
	if violations := checkImports("internal/adapter/postgres/product/account_repository.go", productImportsProductApplication); len(violations) != 0 {
		t.Fatalf("expected Product Postgres adapter to import Product application, got violations: %v", violations)
	}

	// Step 2: Product/Admin persistence adapter の application cross import を拒否し、repository 実装が反対 surface の use case/port へ依存しないようにする。
	forbiddenCases := []struct {
		name       string
		sourcePath string
		importPath string
	}{
		{name: "product postgres imports admin", sourcePath: "internal/adapter/postgres/product/account_repository.go", importPath: modulePath + "/internal/application/admin/auth"},
		{name: "admin postgres imports product", sourcePath: "internal/adapter/postgres/admin/account_repository.go", importPath: modulePath + "/internal/application/product/auth"},
		{name: "product valkey imports admin", sourcePath: "internal/adapter/valkey/product/account_session_store.go", importPath: modulePath + "/internal/application/admin/auth"},
		{name: "admin valkey imports product", sourcePath: "internal/adapter/valkey/admin/operator_session_store.go", importPath: modulePath + "/internal/application/product/auth"},
	}

	for _, forbiddenCase := range forbiddenCases {
		forbiddenCase := forbiddenCase
		// Step 3: Postgres/Valkey の両方で surface 逆流を拒否し、永続化 namespace 分離の静的境界にする。
		t.Run(forbiddenCase.name, func(t *testing.T) {
			t.Parallel()
			file := parseGuardrailTestFile(t, `package product

import _ "`+forbiddenCase.importPath+`"
`)
			violations := checkImports(forbiddenCase.sourcePath, file)
			assertGuardrailViolationMentions(t, violations, forbiddenCase.importPath)
		})
	}
}

// [ADMIN-CONSOLE-BE-S084] Admin account search repository の unsafe raw SQL construction を guardrail が拒否することを検証する。
func TestAdminConsoleBES084AdminAccountRepositoryRejectsUnsafeSQLConstruction(t *testing.T) {
	t.Parallel()

	// Step 1: GORM parameter binding を使う clean fixture を確認し、静的 SQL fragment と bound parameter の正常系を固定する。
	cleanRepository := parseGuardrailTestFile(t, `package admin

func search(queryBuilder query, email string) query {
	return queryBuilder.Where("email ILIKE ?", email).Order("created_at DESC, id DESC")
}
`)
	if violations := checkAdminBackendSQLConstruction("internal/adapter/postgres/admin/account_repository.go", cleanRepository); len(violations) != 0 {
		t.Fatalf("expected parameterized query fixture to pass, got violations: %v", violations)
	}

	// Step 2: raw SQL と文字列連結の代表 fixture を拒否し、Admin account search repository が SQL injection 境界を破れないことを確認する。
	unsafeCases := []struct {
		name   string
		source string
	}{
		{name: "raw", source: `package admin

func search(db query, email string) query {
	return db.Raw("SELECT * FROM public.accounts WHERE email = " + email)
}
`},
		{name: "where concatenation", source: `package admin

func search(db query, email string) query {
	return db.Where("email = '" + email + "'")
}
`},
		{name: "fmt sprintf", source: `package admin

import "fmt"

func search(db query, email string) query {
	return db.Where(fmt.Sprintf("email = %q", email))
}
`},
		{name: "variable condition", source: `package admin

func search(db query, email string) query {
	condition := "email = '" + email + "'"
	return db.Where(condition)
}
`},
	}

	for _, unsafeCase := range unsafeCases {
		unsafeCase := unsafeCase
		// Step 3: case ごとに subtest 化し、どの unsafe SQL construction を見逃したかを failure 名で特定できるようにする。
		t.Run(unsafeCase.name, func(t *testing.T) {
			t.Parallel()
			file := parseGuardrailTestFile(t, unsafeCase.source)
			violations := checkAdminBackendSQLConstruction("internal/adapter/postgres/admin/account_repository.go", file)
			assertGuardrailViolationContains(t, violations, "unsafe SQL construction")
		})
	}
}

// [ADMIN-CONSOLE-BE-S090] Admin backend repository 全体の unsafe SQL construction を guardrail が拒否することを検証する。
func TestAdminConsoleBES090AdminBackendRepositoriesRejectUnsafeSQLConstruction(t *testing.T) {
	t.Parallel()

	// Step 1: Admin operator repository でも静的 SQL fragment と bound parameter は許可し、4.59 の guardrail が安全な GORM 利用を妨げないことを固定する。
	cleanRepository := parseGuardrailTestFile(t, `package admin

func find(db query, operatorID string) query {
	return db.Where("id = ?", operatorID).Order("created_at DESC")
}
`)
	if violations := checkAdminBackendSQLConstruction("internal/adapter/postgres/admin/operator_repository.go", cleanRepository); len(violations) != 0 {
		t.Fatalf("expected parameterized Admin backend query fixture to pass, got violations: %v", violations)
	}

	// Step 2: account search 以外の Admin backend repository でも Raw/Exec と動的 SQL fragment を拒否し、将来の repository 追加で抜け道ができないようにする。
	unsafeCases := []struct {
		name   string
		source string
	}{
		{name: "raw", source: `package admin

func find(db query, email string) query {
	return db.Raw("SELECT * FROM admin.operators WHERE email = " + email)
}
`},
		{name: "exec", source: `package admin

func delete(db query, operatorID string) query {
	return db.Exec("DELETE FROM admin.operators WHERE id = " + operatorID)
}
`},
		{name: "dynamic where", source: `package admin

func find(db query, email string) query {
	condition := "email = '" + email + "'"
	return db.Where(condition)
}
`},
		{name: "dynamic group", source: `package admin

func summarize(db query, groupColumn string) query {
	return db.Group(groupColumn)
}
`},
	}

	for _, unsafeCase := range unsafeCases {
		unsafeCase := unsafeCase
		// Step 3: case ごとに subtest 化し、違反種別ごとの fail-open を個別に検出する。
		t.Run(unsafeCase.name, func(t *testing.T) {
			t.Parallel()
			file := parseGuardrailTestFile(t, unsafeCase.source)
			violations := checkAdminBackendSQLConstruction("internal/adapter/postgres/admin/operator_repository.go", file)
			assertGuardrailViolationContains(t, violations, "unsafe SQL construction")
		})
	}
}

// parseGuardrailTestFile は guardrail test 用の Go source を AST に変換する。
func parseGuardrailTestFile(t *testing.T, source string) *ast.File {
	t.Helper()

	// Step 1: test ごとに独立した FileSet を作り、parser error の位置情報を source 内に閉じ込める。
	fileSet := token.NewFileSet()

	// Step 2: import 宣言を含む最小 source を parse し、失敗時は guardrail ではなく fixture の問題として即時停止する。
	file, err := parser.ParseFile(fileSet, "guardrail_fixture.go", source, 0)
	if err != nil {
		t.Fatalf("parse guardrail fixture: %v", err)
	}

	// Step 3: 呼び出し側が checkImports に渡せる AST を返す。
	return file
}

// assertGuardrailViolationMentions は violation 一覧が対象 import path を含むことを検証する。
func assertGuardrailViolationMentions(t *testing.T, violations []string, importPath string) {
	t.Helper()

	// Step 1: violation が 1 件もない場合は、禁止 import を見逃した fail-open として扱う。
	if len(violations) == 0 {
		t.Fatalf("expected violation for %s, got none", importPath)
	}

	// Step 2: guardrail message は full import path または backend module からの相対 path を出すため、両方を証跡候補にする。
	relativeImportPath := strings.TrimPrefix(importPath, modulePath+"/")

	// Step 3: violation message に対象 import path が含まれることを確認し、どの依存を拒否したかを証跡に残す。
	for _, violation := range violations {
		if strings.Contains(violation, importPath) || strings.Contains(violation, relativeImportPath) {
			return
		}
	}

	// Step 4: violation は出たが対象 path に紐づかない場合、別の guardrail failure による偶然の成功を拒否する。
	t.Fatalf("expected violation mentioning %s, got %v", importPath, violations)
}

// assertGuardrailViolationContains は violation 一覧が期待する文言を含むことを検証する。
func assertGuardrailViolationContains(t *testing.T, violations []string, expectedMessage string) {
	t.Helper()

	// Step 1: violation が 1 件もない場合は、禁止 fixture を見逃した fail-open として扱う。
	if len(violations) == 0 {
		t.Fatalf("expected violation containing %q, got none", expectedMessage)
	}

	// Step 2: 対象文言が少なくとも 1 件に含まれることを確認し、別 guardrail による偶然の成功を避ける。
	for _, violation := range violations {
		if strings.Contains(violation, expectedMessage) {
			return
		}
	}

	// Step 3: violation は出たが期待文言に紐づかない場合、検査対象の guardrail が動いていないため失敗にする。
	t.Fatalf("expected violation containing %q, got %v", expectedMessage, violations)
}

// repositoryRoot は test 実行 directory から repository root を探索する。
func repositoryRoot(t *testing.T) string {
	t.Helper()

	// Step 1: 現在の test process の working directory を取得し、backend package 配下から親へ探索を始める。
	currentDirectory, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}

	// Step 2: scripts/codegen/check.sh が存在する directory を repository root とみなし、環境依存の絶対 path を避ける。
	for {
		candidate := filepath.Join(currentDirectory, "scripts", "codegen", "check.sh")
		if _, statErr := os.Stat(candidate); statErr == nil {
			return currentDirectory
		}

		parent := filepath.Dir(currentDirectory)
		if parent == currentDirectory {
			t.Fatalf("repository root with scripts/codegen/check.sh was not found")
		}
		currentDirectory = parent
	}
}

// repositoryPath は repository root からの相対 path を絶対 path に変換する。
func repositoryPath(t *testing.T, relativePath string) string {
	t.Helper()

	// Step 1: test がどの package directory で実行されても同じ repository file を参照できるようにする。
	return filepath.Join(repositoryRoot(t), filepath.FromSlash(relativePath))
}

// readRepositoryFile は repository root からの相対 path で file を読み込む。
func readRepositoryFile(t *testing.T, relativePath string) string {
	t.Helper()

	// Step 1: guardrail が参照する script 本体を読み、test fixture との乖離を防ぐ。
	content, err := os.ReadFile(repositoryPath(t, relativePath))
	if err != nil {
		t.Fatalf("read %s: %v", relativePath, err)
	}

	return string(content)
}

// extractShellSingleQuotedVariable は shell script の単一引用符 variable から pattern 文字列を取り出す。
func extractShellSingleQuotedVariable(t *testing.T, script string, variableName string) string {
	t.Helper()

	// Step 1: scripts/codegen/check.sh の contamination pattern 定義だけを対象にし、実際の check と test の pattern を一致させる。
	pattern := regexp.MustCompile(`(?m)^` + regexp.QuoteMeta(variableName) + `='([^']*)'$`)
	matches := pattern.FindStringSubmatch(script)
	if len(matches) != 2 {
		t.Fatalf("shell variable %s was not found", variableName)
	}

	return matches[1]
}

// assertPatternMatchesFixture は codegen check の禁止 pattern が contamination fixture を検出することを検証する。
func assertPatternMatchesFixture(t *testing.T, pattern string, fixture string) {
	t.Helper()

	// Step 1: grep -E と互換性のある代表 pattern を Go regexp として compile し、fixture に一致するかを確認する。
	compiledPattern, err := regexp.Compile(pattern)
	if err != nil {
		t.Fatalf("compile codegen contamination pattern %q: %v", pattern, err)
	}
	if !compiledPattern.MatchString(fixture) {
		t.Fatalf("expected pattern %q to reject fixture %q", pattern, fixture)
	}
}
