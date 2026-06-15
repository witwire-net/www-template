package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
)

const modulePath = "www-template/packages/backend"

var migrationFilePattern = regexp.MustCompile(`^\d{6}_[a-z0-9_]+\.(up|down)\.sql$`)

// allowedInternalImports は層間の依存方向を定義する。
// 新しい feature を増やしてもこのマップに変更は不要。
var allowedInternalImports = map[string][]string{
	"cmd":              {"app"},
	"app":              {"platform", "adapter-http", "adapter-postgres", "adapter-valkey", "adapter-webauthn", "adapter-mailer", "application", "domain"},
	"platform":         {"platform"},
	"domain":           {"domain"},
	"application":      {"domain", "platform", "application"},
	"adapter-http":     {"generated", "application", "platform", "domain"},
	"adapter-postgres": {"domain", "application", "platform"},
	"adapter-valkey":   {"domain", "application", "platform"},
	"adapter-webauthn": {"domain", "application", "platform"},
	"adapter-mailer":   {"domain", "application", "platform"},
	"generated":        {},
}

// allowedExternalImports は各層が使ってよい外部ライブラリを定義する。
var allowedExternalImports = map[string][]string{
	"app": {},
	"adapter-http": {
		"github.com/gin-contrib/cors",
		"github.com/gin-gonic/gin",
		"github.com/oapi-codegen/runtime/types",
		"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin",
	},
	"adapter-postgres": {
		"gorm.io/driver/postgres",
		"gorm.io/gorm",
		// GORM ロガー設定に使用。record not found のノイズ抑制と SQL パラメータの露出防止に必要。
		"gorm.io/gorm/logger",
	},
	"adapter-valkey": {
		"github.com/redis/go-redis/v9",
	},
	"adapter-webauthn": {
		"github.com/go-webauthn/webauthn/protocol",
		"github.com/go-webauthn/webauthn/webauthn",
	},
	"platform": {
		"golang.org/x/crypto/bcrypt",
		"github.com/oklog/ulid/v2",
		"github.com/pelletier/go-toml/v2",
		"go.opentelemetry.io/contrib/instrumentation/runtime",
		"go.opentelemetry.io/otel",
		// Trace event の属性を platform/observability 内で生成し、application/app 層へ OTel 型を漏らさないために許可する。
		"go.opentelemetry.io/otel/attribute",
		"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc",
		"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc",
		"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc",
		"go.opentelemetry.io/otel/log",
		"go.opentelemetry.io/otel/propagation",
		"go.opentelemetry.io/otel/sdk/log",
		"go.opentelemetry.io/otel/sdk/metric",
		"go.opentelemetry.io/otel/sdk/resource",
		"go.opentelemetry.io/otel/sdk/trace",
		"go.opentelemetry.io/otel/semconv/v1.26.0",
		"go.opentelemetry.io/otel/trace",
	},
}

var routeSelectors = map[string]struct{}{
	"Any":     {},
	"DELETE":  {},
	"GET":     {},
	"Group":   {},
	"HEAD":    {},
	"Match":   {},
	"OPTIONS": {},
	"PATCH":   {},
	"POST":    {},
	"PUT":     {},
}

// allowedPackageNames は配置パスと許可される package 名の対応を定義する。
// これにより「ディレクトリに適当なファイルを置いて層を偽装する」回避を防ぐ。
var allowedPackageNames = []struct {
	pathPattern string // 前方一致または正規表現
	isRegex     bool
	packageName string
}{
	{pathPattern: "cmd/admin-api/", isRegex: false, packageName: "main"},
	{pathPattern: "cmd/api/", isRegex: false, packageName: "main"},
	{pathPattern: "cmd/", isRegex: false, packageName: "main"},
	{pathPattern: "internal/app/", isRegex: false, packageName: "app"},
	{pathPattern: "internal/platform/config/", isRegex: false, packageName: "config"},
	{pathPattern: "internal/platform/secret/", isRegex: false, packageName: "secret"},
	{pathPattern: "internal/platform/observability/", isRegex: false, packageName: "observability"},
	{pathPattern: "internal/platform/health/", isRegex: false, packageName: "health"},
	{pathPattern: "internal/platform/id/", isRegex: false, packageName: "id"},
	{pathPattern: "internal/generated/adminopenapi/", isRegex: false, packageName: "adminopenapi"},
	{pathPattern: "internal/generated/openapi/", isRegex: false, packageName: "openapi"},
	{pathPattern: "tools/analyzers/", isRegex: false, packageName: "main"},
	{pathPattern: "internal/domain/", isRegex: false, packageName: "domain"},
	{pathPattern: "internal/application/auth/", isRegex: false, packageName: "auth"},
	{pathPattern: "internal/application/accounts/", isRegex: false, packageName: "accounts"},
	{pathPattern: "internal/application/audit/", isRegex: false, packageName: "audit"},
	{pathPattern: "internal/application/operators/", isRegex: false, packageName: "operators"},
	// internal/adapter/http/ は Product/Admin/shared subtree を先に判定し、legacy flat package は最後に残す。
	{pathPattern: "internal/adapter/http/product/", isRegex: false, packageName: "product"},
	{pathPattern: "internal/adapter/http/admin/", isRegex: false, packageName: "admin"},
	{pathPattern: "internal/adapter/http/shared/", isRegex: false, packageName: "shared"},
	{pathPattern: "internal/adapter/http/", isRegex: false, packageName: "http"},
	// internal/adapter/postgres/ は単一 package で Product/Admin surface 分離を持たない。
	{pathPattern: "internal/adapter/postgres/", isRegex: false, packageName: "postgres"},
	// internal/adapter/valkey/ は Product/Admin の session namespace 分離を package 名でも固定する。
	{pathPattern: "internal/adapter/valkey/product/", isRegex: false, packageName: "product"},
	{pathPattern: "internal/adapter/valkey/admin/", isRegex: false, packageName: "admin"},
	{pathPattern: "internal/adapter/valkey/", isRegex: false, packageName: "valkey"},
	// internal/adapter/webauthn/
	{pathPattern: "internal/adapter/webauthn/", isRegex: false, packageName: "webauthn"},
	// internal/adapter/mailer/
	{pathPattern: "internal/adapter/mailer/", isRegex: false, packageName: "mailer"},
}

var usecaseDomainTouchPrefixes = []string{"Create", "Update", "Change", "Rename", "Set", "Add"}

var forbiddenPortTypeImportPrefixes = []string{
	modulePath + "/internal/generated",
	modulePath + "/internal/adapter/http",
	modulePath + "/internal/adapter",
	"github.com/gin-contrib/cors",
	"github.com/gin-gonic/gin",
	"github.com/oapi-codegen/runtime/types",
	"gorm.io/",
}

// domainImportPattern は layer-axis domain パッケージの import path にマッチする。
var domainImportPattern = regexp.MustCompile(`^` + regexp.QuoteMeta(modulePath+"/internal/domain") + `$`)

func main() {
	flag.Parse()
	root := "."
	if flag.NArg() > 0 {
		root = flag.Arg(0)
	}

	violations, err := collectViolations(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "guardrails: %v\n", err)
		os.Exit(1)
	}

	if len(violations) > 0 {
		sort.Strings(violations)
		for _, violation := range violations {
			fmt.Fprintln(os.Stderr, violation)
		}
		os.Exit(1)
	}
}

func collectViolations(root string) ([]string, error) {
	fileSet := token.NewFileSet()
	violations := make([]string, 0)

	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		relativePath := filepath.ToSlash(path)
		if entry.IsDir() {
			if entry.Name() == ".git" || entry.Name() == "bin" || entry.Name() == "dist" {
				return filepath.SkipDir
			}
			return nil
		}

		if strings.HasSuffix(relativePath, ".go") {
			violations = append(violations, verifyGoFilePlacement(relativePath)...)
			violations = append(violations, verifyPostgresAdapterFlatLayout(relativePath)...)
			violations = append(violations, verifyApplicationConceptOwnership(relativePath)...)

			if strings.HasPrefix(relativePath, "internal/generated/") {
				violations = append(violations, verifyGeneratedFile(relativePath)...)
				return nil
			}

			file, err := parser.ParseFile(fileSet, path, nil, parser.ParseComments)
			if err != nil {
				return fmt.Errorf("parse %s: %w", path, err)
			}

			violations = append(violations, checkPackageName(relativePath, file)...)
			violations = append(violations, checkImports(relativePath, file)...)
			violations = append(violations, checkAutoMigrate(relativePath, file)...)
			violations = append(violations, checkCoreSideEffects(relativePath, file)...)
			violations = append(violations, checkDomainCompositeLiterals(relativePath, file)...)
			violations = append(violations, checkErrorStringMatching(relativePath, file)...)
			violations = append(violations, checkForbiddenCalls(relativePath, file)...)
			violations = append(violations, checkForbiddenHostUsage(relativePath, file)...)
			violations = append(violations, checkPostgresRepositorySQLConstruction(relativePath, file)...)
			violations = append(violations, checkHTTPDomainBoundary(relativePath, file)...)
			violations = append(violations, checkPortPurity(relativePath, file)...)
			violations = append(violations, checkRoutePolicy(relativePath, file)...)
			violations = append(violations, checkUnitTestDeterminism(relativePath, file)...)
			violations = append(violations, checkUsecaseExportedAPIBoundary(relativePath, file)...)
			violations = append(violations, checkWriteUsecasesTouchDomain(relativePath, file)...)
			violations = append(violations, checkUsecaseInlineBusinessValidation(relativePath, file)...)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	violations = append(violations, verifyMigrationFiles(root)...)

	return violations, nil
}

func verifyGoFilePlacement(path string) []string {
	allowedPrefixes := []string{
		"cmd/admin-api/",
		"cmd/api/",
		"internal/app/",
		"internal/platform/",
		"internal/adapter/",
		"internal/generated/",
		"tools/analyzers/",
	}

	for _, prefix := range allowedPrefixes {
		if strings.HasPrefix(path, prefix) {
			return nil
		}
	}

	// flat な internal/domain または internal/application を許可
	if strings.HasPrefix(path, "internal/domain/") || strings.HasPrefix(path, "internal/application/") {
		return nil
	}

	return []string{fmt.Sprintf("%s: go files must live under cmd/api, cmd/admin-api, internal/app, internal/platform, internal/domain, internal/application, internal/adapter, or internal/generated", path)}
}

// verifyPostgresAdapterFlatLayout は Postgres adapter の Product/Admin surface subtree の再発を検査する。
// 単一 postgres package 化後、postgres/product/ や postgres/admin/ の再作成を拒否し、
// DB 境界は schema/table/role/grant で守ることを強制する。
func verifyPostgresAdapterFlatLayout(path string) []string {
	forbiddenPrefixes := []string{
		"internal/adapter/postgres/product/",
		"internal/adapter/postgres/admin/",
	}
	for _, prefix := range forbiddenPrefixes {
		if strings.HasPrefix(path, prefix) {
			return []string{fmt.Sprintf("%s: Postgres adapter must be a single flat package; use schema/table/role/grant for DB boundaries instead of %s", path, prefix)}
		}
	}
	return nil
}

// verifyApplicationConceptOwnership は application 層に Product/Admin surface owner を復活させないための配置検査である。
// Product/Admin は HTTP adapter、binary、generated artifact の境界用語に限定し、application owner は auth などの concept package に固定する。
func verifyApplicationConceptOwnership(path string) []string {
	// Step 1: production Go source だけを対象にし、既存のテスト fixture や削除済み directory の有無ではなく実装 owner の復活を検査する。
	if isTestFile(path) {
		return nil
	}

	// Step 2: internal/application 直下の flat owner は責務が曖昧な legacy bucket になるため、production source としての復活を拒否する。
	applicationPrefix := "internal/application/"
	if strings.HasPrefix(path, applicationPrefix) {
		remainder := strings.TrimPrefix(path, applicationPrefix)
		segment, _, hasNestedPath := strings.Cut(remainder, "/")
		if !hasNestedPath {
			return []string{fmt.Sprintf("%s: root internal/application production files are forbidden; application owners must be concept packages such as auth, accounts, operators, or audit", path)}
		}

		// Step 3: Product/Admin/shared/unknown application subtrees are not concept owners; binary, HTTP adapter, generated artifacts, and explicit concept packages own those responsibilities.
		switch segment {
		case "auth", "accounts", "operators", "audit":
			return nil
		default:
			return []string{fmt.Sprintf("%s: application owners must be concept packages such as auth, accounts, operators, or audit; Product/Admin belong to binary, HTTP adapter, and generated artifact boundaries", path)}
		}
	}

	// Step 4: application 層以外の production source はこの検査の対象外にする。
	return nil
}

func verifyGeneratedFile(path string) []string {
	violations := make([]string, 0)
	if !strings.HasSuffix(path, ".gen.go") {
		violations = append(violations, fmt.Sprintf("%s: internal/generated may only contain *.gen.go files", path))
		return violations
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return []string{fmt.Sprintf("%s: read generated file: %v", path, err)}
	}

	if !strings.Contains(string(content), "Code generated by") {
		violations = append(violations, fmt.Sprintf("%s: generated file must keep the codegen header", path))
	}

	return violations
}

func verifyMigrationFiles(root string) []string {
	migrationsDir := filepath.Join(root, "db", "migrations")
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return []string{fmt.Sprintf("db/migrations: %v", err)}
	}

	pairs := make(map[string]map[string]bool)
	violations := make([]string, 0)
	for _, entry := range entries {
		if entry.IsDir() {
			violations = append(violations, fmt.Sprintf("db/migrations/%s: nested directories are not allowed", entry.Name()))
			continue
		}

		name := entry.Name()
		if !migrationFilePattern.MatchString(name) {
			violations = append(violations, fmt.Sprintf("db/migrations/%s: migration files must match 000001_name.(up|down).sql", name))
			continue
		}

		base, direction := splitMigrationName(name)
		if pairs[base] == nil {
			pairs[base] = map[string]bool{}
		}
		pairs[base][direction] = true
	}

	for base, directions := range pairs {
		if !directions["up"] || !directions["down"] {
			violations = append(violations, fmt.Sprintf("db/migrations/%s: every migration must have both .up.sql and .down.sql", base))
		}
	}

	return violations
}

func splitMigrationName(name string) (string, string) {
	if strings.HasSuffix(name, ".up.sql") {
		return strings.TrimSuffix(name, ".up.sql"), "up"
	}

	if strings.HasSuffix(name, ".down.sql") {
		return strings.TrimSuffix(name, ".down.sql"), "down"
	}

	base, direction, _ := strings.Cut(name, ".")
	return base, direction
}

func checkImports(path string, file *ast.File) []string {
	layer, _ := layerFromPath(path)
	if layer == "generated" || layer == "" {
		return nil
	}

	violations := make([]string, 0)
	for _, imported := range file.Imports {
		importPath := strings.Trim(imported.Path.Value, "\"")

		if strings.HasPrefix(importPath, modulePath+"/") {
			targetLayer, _ := layerFromPath(strings.TrimPrefix(importPath, modulePath+"/"))
			if targetLayer == "" {
				violations = append(violations, fmt.Sprintf("%s: internal import %s does not map to an allowed backend layer", path, importPath))
				continue
			}

			// HTTP adapter subtree 同士の import は shared helper への片方向依存だけに限定する。
			// これにより Product/Admin transport が相互に handler や generated 接続を共有する抜け道を塞ぐ。
			if layer == "adapter-http" && targetLayer == "adapter-http" {
				if !isAllowedHTTPAdapterBoundaryImport(path, importPath) {
					violations = append(violations, fmt.Sprintf("%s: HTTP adapter surface must not import %s", path, importPath))
				}
				continue
			}

			// Valkey adapter subtree 同士の import は Product/Admin から共通 root helper への依存だけに限定する。
			// これにより Product/Admin session namespace を混在させず、観測 hook などの中立実装だけを共有できる。
			if layer == "adapter-valkey" && targetLayer == "adapter-valkey" {
				if !isAllowedValkeyAdapterBoundaryImport(path, importPath) {
					violations = append(violations, fmt.Sprintf("%s: Valkey adapter surface must not import %s", path, importPath))
				}
				continue
			}

			// Product/Admin application subtree と persistence adapter は surface を越えた application import を禁止する。
			// 認証 primitive は domain/application auth concept に集約し、shared wrapper への依存を canonical path にしない。
			if targetLayer == "application" && !isAllowedApplicationBoundaryImport(path, importPath) {
				violations = append(violations, fmt.Sprintf("%s: %s surface must not import %s", path, layer, importPath))
				continue
			}

			// Product generated bindings は Product HTTP adapter だけに閉じ込める。
			// ここで早期に surface を判定し、application / cmd / Admin adapter からの横断 import を fail-closed にする。
			if importPath == modulePath+"/internal/generated/openapi" && !isProductHTTPAdapterPath(path) {
				violations = append(violations, fmt.Sprintf("%s: only Product HTTP adapter may import internal/generated/openapi", path))
				continue
			}

			// Admin generated bindings は Admin HTTP adapter だけに閉じ込める。
			// Product binary や既存の flat Product adapter が Admin operation へ到達可能になる経路を拒否する。
			if importPath == modulePath+"/internal/generated/adminopenapi" && !isAdminHTTPAdapterPath(path) {
				violations = append(violations, fmt.Sprintf("%s: only Admin HTTP adapter may import internal/generated/adminopenapi", path))
				continue
			}

			if !slices.Contains(allowedInternalImports[layer], targetLayer) {
				violations = append(violations, fmt.Sprintf("%s: %s must not import %s", path, layer, importPath))
			}
			continue
		}

		if !isExternalImport(importPath) {
			continue
		}

		if strings.HasPrefix(importPath, "gorm.io/") && layer != "adapter-postgres" {
			violations = append(violations, fmt.Sprintf("%s: gorm imports are only allowed in adapter-postgres", path))
			continue
		}

		allowedExternal := allowedExternalImports[layer]
		if len(allowedExternal) == 0 || !slices.Contains(allowedExternal, importPath) {
			violations = append(violations, fmt.Sprintf("%s: %s must not import external package %s", path, layer, importPath))
		}
	}

	return violations
}

// isAllowedHTTPAdapterBoundaryImport は HTTP adapter subtree 間の許可依存だけを判定する。
// Product/Admin は shared helper を使えるが、互いの handler package や legacy flat package へは依存できない。
func isAllowedHTTPAdapterBoundaryImport(sourcePath string, importPath string) bool {
	// Step 1: source file と import target の surface を取り出し、path 文字列の比較を一箇所に集約する。
	sourceSurface := httpAdapterSurfaceFromPath(sourcePath)
	targetSurface := httpAdapterSurfaceFromImport(importPath)

	// Step 2: 同一 surface 内の補助 package は許可し、将来 product/internal や admin/internal を増やす余地を残す。
	if sourceSurface == targetSurface {
		return true
	}

	// Step 3: Product/Admin から shared への一方向依存だけを許可し、shared から具体 surface への逆依存は拒否する。
	return (sourceSurface == "product" || sourceSurface == "admin") && targetSurface == "shared"
}

// isAllowedValkeyAdapterBoundaryImport は Valkey adapter subtree 間の許可依存だけを判定する。
// Product/Admin は共通 root helper を使えるが、互いの store package へは依存できない。
func isAllowedValkeyAdapterBoundaryImport(sourcePath string, importPath string) bool {
	// Step 1: source file と import target の Valkey surface を取り出し、Product/Admin/root の関係を固定する。
	sourceSurface := persistenceAdapterSurfaceFromPath(sourcePath, "valkey")
	targetSurface := surfaceFromRelativePath(strings.TrimPrefix(importPath, modulePath+"/"), "internal/adapter/valkey")

	// Step 2: 同一 surface 内の補助 package は許可し、将来 product/internal 等を増やす余地を残す。
	if sourceSurface == targetSurface {
		return true
	}

	// Step 3: Product/Admin から共通 root helper への片方向依存だけを許可し、root から具体 surface への逆依存は拒否する。
	return (sourceSurface == "product" || sourceSurface == "admin") && targetSurface == "legacy"
}

// isAllowedApplicationBoundaryImport は application package への import が surface 境界を越えていないか判定する。
// app layer は runtime composition owner のため任意 surface を組み立てられるが、application/adapter 内部では最小限に制限する。
func isAllowedApplicationBoundaryImport(sourcePath string, importPath string) bool {
	// Step 1: layer を先に判定し、application 境界が意味を持つ source だけを追加検査の対象にする。
	sourceLayer, _ := layerFromPath(sourcePath)
	targetSurface := applicationSurfaceFromImport(importPath)

	// Step 2: internal/app は Product/Admin runtime composition を所有するため、surface-specific application の import を許可する。
	if sourceLayer == "app" {
		return true
	}

	// Step 3: application source は Product/Admin/shared/legacy の相互境界を直接守る。
	if sourceLayer == "application" {
		return isAllowedApplicationSourceDependency(sourcePath, importPath, targetSurface)
	}

	// Step 4: transport / persistence adapters は自身の surface に対応する application ports/use cases だけを参照できる。
	if sourceLayer == "adapter-http" {
		// Step 4-a: Admin hosted-service adapter は application owner を surface 名ではなく capability package として呼び出す。
		// 許可対象を HTTP adapter に限定し、Admin auth application から account/operator owner へ逆流する抜け道を作らない。
		if httpAdapterSurfaceFromPath(sourcePath) == "admin" && isAdminConceptApplicationSurface(targetSurface) {
			return true
		}

		// Step 4-a: 既存 Product API の use case はまだ internal/application 直下に残っているため、Product HTTP adapter から legacy Product application への参照だけを移行互換として許可する。
		if httpAdapterSurfaceFromPath(sourcePath) == "product" && targetSurface == "legacy" {
			return true
		}
		return isAllowedSurfaceApplicationDependency(httpAdapterSurfaceFromPath(sourcePath), targetSurface)
	}
	if sourceLayer == "adapter-postgres" {
		// Step 4-b: 単一 postgres package 内の operator repository は operators concept port への依存を許可する。
		if sourcePath == "internal/adapter/postgres/operator_repository.go" && targetSurface == "operators" {
			return true
		}
		// Step 4-c: 単一 postgres package は Product/Admin surface subtree を持たないため、
		// surface ではなく application concept への直接依存を許可する。
		// schema/table/role/grant 境界で security を守り、package path による Product/Admin 分離は行わない。
		if isAllowedPostgresAdapterApplicationDependency(targetSurface) {
			return true
		}
		return isAllowedSurfaceApplicationDependency(persistenceAdapterSurfaceFromPath(sourcePath, "postgres"), targetSurface)
	}
	if sourceLayer == "adapter-valkey" {
		return isAllowedSurfaceApplicationDependency(persistenceAdapterSurfaceFromPath(sourcePath, "valkey"), targetSurface)
	}

	// Step 5: それ以外の layer は既存の layer-level allowlist で拒否/許可を判断するため、ここでは追加制限しない。
	return true
}

// isAllowedApplicationSourceDependency は application layer 内の concept/surface 間依存を source path 単位で判定する。
// Admin auth surface から account/operator/audit owner へ依存する経路は許可せず、必要な setup session 発行だけを operators concept から Admin auth boundary へ通す。
func isAllowedApplicationSourceDependency(sourcePath string, importPath string, targetSurface string) bool {
	// Step 1: source surface を path から取り出し、concept package 追加時も surface 判定を一箇所に集約する。
	sourceSurface := applicationSurfaceFromPath(sourcePath)

	// Step 2: accounts concept は audit completion DTO だけを使い、Admin surface へ戻らず account mutation を完結させる。
	if sourceSurface == "accounts" && targetSurface == "audit" {
		return true
	}

	// Step 3: operators concept は account ID generator と audit completion を共有し、setup 完了後の session 発行だけ operator auth lifecycle concept へ委譲する。
	if sourceSurface == "operators" && (targetSurface == "accounts" || targetSurface == "audit" || importPath == modulePath+"/internal/application/auth") {
		return true
	}

	// Step 4: 上記以外は surface 共通規則へ委譲し、Admin auth から business capability owner へ依存できない状態を保つ。
	return isAllowedSurfaceApplicationDependency(sourceSurface, targetSurface)
}

// isAdminConceptApplicationSurface は Admin HTTP adapter が呼び出す capability owner package だけを列挙する。
// ここへ追加する package は Admin hosted-service 経由の use case owner であることを review 対象にする。
func isAdminConceptApplicationSurface(targetSurface string) bool {
	// Step 1: explicit allowlist にして未知 surface を legacy 扱いで偶然許可しない。
	switch targetSurface {
	case "accounts", "operators", "audit":
		return true
	default:
		return false
	}
}

// isAllowedSurfaceApplicationDependency は surface ごとの application 依存を判定する。
// Product/Admin は auth concept を使えるが相互 import と legacy root application への逆流は拒否する。
func isAllowedSurfaceApplicationDependency(sourceSurface string, targetSurface string) bool {
	// Step 1: 同一 surface は許可し、Product->Product や Admin->Admin の use case 分割を妨げない。
	if sourceSurface == targetSurface {
		return true
	}

	// Step 2: Product HTTP surface から Product AccountSetting use case owner への依存を許可し、root application bucket へ DTO を戻らず accounts concept を直接呼ぶ。
	if sourceSurface == "product" && targetSurface == "accounts" {
		return true
	}

	// Step 3: Admin persistence adapter は accounts/audit application DTO を直接参照する。
	// 単一 postgres package 内の account_management_repository と operator_audit_repository が Admin-owned として application DTO を import する経路を許可する。
	if sourceSurface == "admin" && (targetSurface == "accounts" || targetSurface == "audit") {
		return true
	}

	// Step 4: Product/Admin から canonical auth concept への依存を許可し、surface 固有 use case が root legacy DTO へ戻らず explicit subject payload を使えるようにする。
	if (sourceSurface == "product" || sourceSurface == "admin") && targetSurface == "auth" {
		return true
	}

	// Step 4: root legacy application は移行中の facade として canonical auth concept だけを参照できるが、Product/Admin surface package へは逆流させない。
	if sourceSurface == "legacy" && targetSurface == "auth" {
		return true
	}

	// Step 5: Product/Admin から shared への依存だけを許可し、中立 DTO の再利用を surface 境界内に収める。
	return (sourceSurface == "product" || sourceSurface == "admin") && targetSurface == "shared"
}

// isAllowedPostgresAdapterApplicationDependency は単一 postgres package が参照できる application concept を判定する。
// 単一 postgres package は Product/Admin surface subtree を持たないため、schema/table/role/grant 境界で守られる application concept への直接依存を許可する。
func isAllowedPostgresAdapterApplicationDependency(targetSurface string) bool {
	switch targetSurface {
	case "accounts", "operators", "audit", "auth":
		return true
	default:
		return false
	}
}

// applicationSurfaceFromPath は application 配下のファイル配置から surface 名を返す。
// legacy は既存 flat package を段階的に守るための名前で、新規 Product/Admin subtree とは混ぜない。
func applicationSurfaceFromPath(path string) string {
	return surfaceFromRelativePath(path, "internal/application")
}

// applicationSurfaceFromImport は application import path から surface 名を返す。
// module path を剥がした後の判定に統一し、alias import の有無に左右されない境界検査にする。
func applicationSurfaceFromImport(importPath string) string {
	return surfaceFromRelativePath(strings.TrimPrefix(importPath, modulePath+"/"), "internal/application")
}

// httpAdapterSurfaceFromPath は HTTP adapter 配下のファイル配置から Product/Admin/shared/legacy を判定する。
// legacy flat adapter は現行 Product router の移行前配置として扱い、Admin/shared とは分離する。
func httpAdapterSurfaceFromPath(path string) string {
	return surfaceFromRelativePath(path, "internal/adapter/http")
}

// httpAdapterSurfaceFromImport は HTTP adapter import path から surface 名を返す。
// Product/Admin/shared の import 境界を package path だけで安定して検査する。
func httpAdapterSurfaceFromImport(importPath string) string {
	return surfaceFromRelativePath(strings.TrimPrefix(importPath, modulePath+"/"), "internal/adapter/http")
}

// persistenceAdapterSurfaceFromPath は Postgres/Valkey adapter 配置から surface 名を返す。
// adapterKind は "postgres" または "valkey" を渡し、将来の surface subtree を同じ規則で判定する。
func persistenceAdapterSurfaceFromPath(path string, adapterKind string) string {
	return surfaceFromRelativePath(path, "internal/adapter/"+adapterKind)
}

// surfaceFromRelativePath は base directory 直下の最初の segment を surface として抽出する。
// product/admin/shared 以外は legacy として扱い、意図しない未知 surface の自動許可を避ける。
func surfaceFromRelativePath(path string, base string) string {
	// Step 1: base 自体または base 直下の flat file は legacy とし、既存 package と新 surface を混同しない。
	trimmedBase := strings.TrimSuffix(base, "/")
	if path == trimmedBase || !strings.HasPrefix(path, trimmedBase+"/") {
		return "legacy"
	}

	// Step 2: base からの相対 path の最初の segment を取り出し、surface subtree かどうかを判定する。
	remainder := strings.TrimPrefix(path, trimmedBase+"/")
	segment, _, _ := strings.Cut(remainder, "/")
	switch segment {
	case "product", "admin", "auth", "shared", "accounts", "audit", "operators":
		return segment
	default:
		return "legacy"
	}
}

// isProductHTTPAdapterPath は Product Go bindings を import してよい HTTP adapter の path だけを許可する。
// 4.7 以後は product subtree だけを Product surface とみなし、legacy flat HTTP adapter からの generated import を拒否する。
func isProductHTTPAdapterPath(path string) bool {
	return strings.HasPrefix(path, "internal/adapter/http/product/")
}

// isAdminHTTPAdapterPath は Admin Go bindings を import してよい HTTP adapter の path だけを許可する。
// Admin bindings は強権限 operation を含むため、flat Product adapter や Product binary には絶対に露出させない。
func isAdminHTTPAdapterPath(path string) bool {
	return strings.HasPrefix(path, "internal/adapter/http/admin/")
}

func checkPackageName(path string, file *ast.File) []string {
	violations := make([]string, 0)

	// AST から package 名を取得
	pkgName := file.Name.Name

	// テストファイルの場合は "xxx_test" または "xxx" を許可
	if isTestFile(path) {
		basePkg := strings.TrimSuffix(pkgName, "_test")
		if !checkPackageNameViolation(path, basePkg) {
			// 基本パッケージ名が許可されている → OK
			return nil
		}
		// 基本パッケージ名が許可されていない → 違反
		violations = append(violations, fmt.Sprintf("%s: test package name %q does not match allowed package for this directory layout", path, pkgName))
		return violations
	}

	if checkPackageNameViolation(path, pkgName) {
		violations = append(violations, fmt.Sprintf("%s: package name %q does not match allowed package for this directory layout", path, pkgName))
	}

	return violations
}

// checkPackageNameViolation は指定パッケージ名が配置パスに許可されているか判定する。
// 違反があれば true を返す。
func checkPackageNameViolation(path, pkgName string) bool {
	for _, rule := range allowedPackageNames {
		if rule.isRegex {
			if matched, _ := regexp.MatchString(rule.pathPattern, path); matched {
				return pkgName != rule.packageName
			}
		} else {
			if strings.HasPrefix(path, rule.pathPattern) {
				return pkgName != rule.packageName
			}
		}
	}

	// どのルールにもマッチしない場合は不明な配置なので違反とみなさない（verifyGoFilePlacement で別途検査）
	return false
}

func checkAutoMigrate(path string, file *ast.File) []string {
	violations := make([]string, 0)
	ast.Inspect(file, func(node ast.Node) bool {
		selector, ok := node.(*ast.SelectorExpr)
		if ok && selector.Sel != nil && selector.Sel.Name == "AutoMigrate" {
			violations = append(violations, fmt.Sprintf("%s: AutoMigrate is banned; use golang-migrate SQL files", path))
		}
		return true
	})
	return violations
}

func checkCoreSideEffects(path string, file *ast.File) []string {
	layer, _ := layerFromPath(path)
	if isTestFile(path) || (layer != "domain" && layer != "application") {
		return nil
	}

	imports := importAliases(file)
	violations := make([]string, 0)
	ast.Inspect(file, func(node ast.Node) bool {
		callExpr, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}

		selector, ok := callExpr.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}

		ident, ok := selector.X.(*ast.Ident)
		if !ok {
			return true
		}

		importPath, ok := imports[ident.Name]
		if !ok {
			return true
		}

		switch importPath {
		case "time":
			if selector.Sel.Name == "Now" {
				violations = append(violations, fmt.Sprintf("%s: %s must not call time.Now directly; inject a clock from the outer layer", path, layer))
			}
		case "os":
			if selector.Sel.Name == "Getenv" || selector.Sel.Name == "LookupEnv" || selector.Sel.Name == "Environ" {
				violations = append(violations, fmt.Sprintf("%s: %s must not read environment variables directly; pass configuration in from the outer layer", path, layer))
			}
		case "log", "log/slog", "math/rand", "math/rand/v2":
			violations = append(violations, fmt.Sprintf("%s: %s must not call %s directly; keep side effects in outer layers", path, layer, importPath))
		}

		return true
	})

	return violations
}

func checkDomainCompositeLiterals(path string, file *ast.File) []string {
	layer, _ := layerFromPath(path)
	if isTestFile(path) || layer == "domain" {
		return nil
	}

	imports := importAliases(file)
	violations := make([]string, 0)
	ast.Inspect(file, func(node ast.Node) bool {
		literal, ok := node.(*ast.CompositeLit)
		if !ok {
			return true
		}

		selector, ok := literal.Type.(*ast.SelectorExpr)
		if !ok {
			return true
		}

		ident, ok := selector.X.(*ast.Ident)
		if !ok {
			return true
		}

		importPath, ok := imports[ident.Name]
		if !ok {
			return true
		}

		// internal/{feature}/domain の import かどうかをチェック
		if !domainImportPattern.MatchString(importPath) {
			return true
		}

		violations = append(violations, fmt.Sprintf("%s: construct domain.%s via domain constructors or reconstitution helpers instead of composite literals", path, selector.Sel.Name))
		return true
	})

	return violations
}

func checkErrorStringMatching(path string, file *ast.File) []string {
	imports := importAliases(file)
	violations := make([]string, 0)
	ast.Inspect(file, func(node ast.Node) bool {
		switch n := node.(type) {
		case *ast.BinaryExpr:
			if (n.Op == token.EQL || n.Op == token.NEQ) && (isErrorStringCall(n.X) || isErrorStringCall(n.Y)) {
				violations = append(violations, fmt.Sprintf("%s: do not branch on err.Error(); compare typed errors with errors.Is instead", path))
			}
		case *ast.CallExpr:
			selector, ok := n.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}

			ident, ok := selector.X.(*ast.Ident)
			if !ok || imports[ident.Name] != "strings" {
				return true
			}

			if selector.Sel.Name != "Contains" && selector.Sel.Name != "HasPrefix" && selector.Sel.Name != "HasSuffix" && selector.Sel.Name != "EqualFold" {
				return true
			}

			for _, arg := range n.Args {
				if isErrorStringCall(arg) {
					violations = append(violations, fmt.Sprintf("%s: do not branch on err.Error(); compare typed errors with errors.Is instead", path))
					break
				}
			}
		}

		return true
	})

	return violations
}

func checkHTTPDomainBoundary(path string, file *ast.File) []string {
	layer, _ := layerFromPath(path)
	if isTestFile(path) || layer != "adapter-http" {
		return nil
	}

	imports := importAliases(file)
	violations := make([]string, 0)
	ast.Inspect(file, func(node ast.Node) bool {
		selector, ok := node.(*ast.SelectorExpr)
		if !ok {
			return true
		}

		ident, ok := selector.X.(*ast.Ident)
		if !ok {
			return true
		}

		importPath, ok := imports[ident.Name]
		if !ok {
			return true
		}

		// internal/{feature}/domain の import かどうか
		if !domainImportPattern.MatchString(importPath) {
			return true
		}

		if strings.HasSuffix(selector.Sel.Name, "Repository") || strings.HasSuffix(selector.Sel.Name, "Port") {
			return true
		}

		violations = append(violations, fmt.Sprintf("%s: adapter-http must not depend on domain.%s directly; map transport DTOs to application DTOs instead", path, selector.Sel.Name))
		return true
	})

	return violations
}

func checkPortPurity(path string, file *ast.File) []string {
	layer, _ := layerFromPath(path)
	if isTestFile(path) || (layer != "domain" && layer != "application") {
		return nil
	}

	imports := importAliases(file)
	violations := make([]string, 0)
	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
			continue
		}

		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}

			interfaceType, ok := typeSpec.Type.(*ast.InterfaceType)
			if !ok {
				continue
			}

			for _, method := range interfaceType.Methods.List {
				funcType, ok := method.Type.(*ast.FuncType)
				if ok {
					if containsForbiddenPortType(funcType.Params, imports) || containsForbiddenPortType(funcType.Results, imports) {
						violations = append(violations, fmt.Sprintf("%s: %s interface %s must not depend on transport or persistence types", path, layer, typeSpec.Name.Name))
					}
					continue
				}

				if typeExprContainsForbiddenPortType(method.Type, imports) {
					violations = append(violations, fmt.Sprintf("%s: %s interface %s must not depend on transport or persistence types", path, layer, typeSpec.Name.Name))
				}
			}
		}
	}

	return violations
}

func checkUnitTestDeterminism(path string, file *ast.File) []string {
	if !isTestFile(path) {
		return nil
	}

	imports := importAliases(file)
	violations := make([]string, 0)
	ast.Inspect(file, func(node ast.Node) bool {
		callExpr, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}

		selector, ok := callExpr.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}

		ident, ok := selector.X.(*ast.Ident)
		if !ok {
			return true
		}

		importPath, ok := imports[ident.Name]
		if !ok {
			return true
		}

		if importPath == "time" && selector.Sel.Name == "Sleep" {
			violations = append(violations, fmt.Sprintf("%s: unit tests must not call time.Sleep; use deterministic synchronization instead", path))
		}

		if importPath == "net/http" {
			switch selector.Sel.Name {
			case "Get", "Head", "Post", "PostForm":
				violations = append(violations, fmt.Sprintf("%s: unit tests must not perform real network requests via net/http", path))
			}
		}

		return true
	})

	return violations
}

func checkUsecaseExportedAPIBoundary(path string, file *ast.File) []string {
	layer, _ := layerFromPath(path)
	if isTestFile(path) || layer != "application" {
		return nil
	}

	imports := importAliases(file)
	violations := make([]string, 0)
	for _, decl := range file.Decls {
		funcDecl, ok := decl.(*ast.FuncDecl)
		if !ok || !funcDecl.Name.IsExported() || strings.HasPrefix(funcDecl.Name.Name, "New") {
			continue
		}

		if containsForbiddenUsecaseDomainType(funcDecl.Type.Params, imports) || containsForbiddenUsecaseDomainType(funcDecl.Type.Results, imports) {
			violations = append(violations, fmt.Sprintf("%s: exported application API %s must use application DTOs instead of domain entities or commands", path, funcDecl.Name.Name))
		}
	}

	return violations
}

func checkWriteUsecasesTouchDomain(path string, file *ast.File) []string {
	layer, _ := layerFromPath(path)
	if isTestFile(path) || layer != "application" {
		return nil
	}

	imports := importAliases(file)
	violations := make([]string, 0)
	for _, decl := range file.Decls {
		funcDecl, ok := decl.(*ast.FuncDecl)
		if !ok || !requiresDirectDomainTouch(funcDecl) {
			continue
		}

		if !functionBodyTouchesDomain(funcDecl.Body, imports) {
			violations = append(violations, fmt.Sprintf("%s: exported write application %s must call into domain directly so business rules cannot bypass the domain layer", path, funcDecl.Name.Name))
		}
	}

	return violations
}

func checkUsecaseInlineBusinessValidation(path string, file *ast.File) []string {
	layer, _ := layerFromPath(path)
	if isTestFile(path) || layer != "application" {
		return nil
	}

	imports := importAliases(file)
	violations := make([]string, 0)
	for _, decl := range file.Decls {
		funcDecl, ok := decl.(*ast.FuncDecl)
		if !ok || !requiresDirectDomainTouch(funcDecl) {
			continue
		}

		paramNames := nonContextParameterNames(funcDecl.Type.Params, imports)
		if len(paramNames) == 0 || funcDecl.Body == nil {
			continue
		}

		if hasInlineUsecaseValidation(funcDecl.Body, imports, paramNames) {
			violations = append(violations, fmt.Sprintf("%s: exported write application %s must delegate trimming and validation to domain instead of validating request fields inline", path, funcDecl.Name.Name))
		}
	}

	return violations
}

func checkRoutePolicy(path string, file *ast.File) []string {
	layer, _ := layerFromPath(path)
	if layer != "adapter-http" {
		return nil
	}

	violations := make([]string, 0)
	ast.Inspect(file, func(node ast.Node) bool {
		callExpr, ok := node.(*ast.CallExpr)
		if !ok || len(callExpr.Args) == 0 {
			return true
		}

		selector, ok := callExpr.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}

		if _, ok := routeSelectors[selector.Sel.Name]; !ok {
			return true
		}

		literal, ok := callExpr.Args[0].(*ast.BasicLit)
		if !ok || literal.Kind != token.STRING {
			violations = append(violations, fmt.Sprintf("%s: non-generated Gin routes must use string literal paths", path))
			return true
		}

		routePath, err := strconv.Unquote(literal.Value)
		if err != nil {
			violations = append(violations, fmt.Sprintf("%s: invalid route literal %s", path, literal.Value))
			return true
		}

		if selector.Sel.Name == "Group" {
			if routePath != "/api/v1" && !strings.HasPrefix(routePath, "/api/v1/") {
				violations = append(violations, fmt.Sprintf("%s: custom Gin groups must live under /api/v1/*, got %s", path, routePath))
			}
			return true
		}

		if routePath == "/health" || routePath == "/api/v1" || strings.HasPrefix(routePath, "/api/v1/") {
			return true
		}

		violations = append(violations, fmt.Sprintf("%s: non-generated Gin routes must be /health or /api/v1/*, got %s", path, routePath))
		return true
	})

	return violations
}

func checkForbiddenCalls(path string, file *ast.File) []string {
	violations := make([]string, 0)
	ast.Inspect(file, func(node ast.Node) bool {
		callExpr, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}

		switch fun := callExpr.Fun.(type) {
		case *ast.Ident:
			if fun.Name == "print" || fun.Name == "println" {
				violations = append(violations, fmt.Sprintf("%s: print and println are banned; use structured logging or error returns", path))
			}
		case *ast.SelectorExpr:
			if ident, ok := fun.X.(*ast.Ident); ok && ident.Name == "fmt" && strings.HasPrefix(fun.Sel.Name, "Print") {
				violations = append(violations, fmt.Sprintf("%s: fmt.Print* is banned; use structured logging or error returns", path))
			}
		}

		return true
	})

	return violations
}

func checkForbiddenHostUsage(path string, file *ast.File) []string {
	violations := make([]string, 0)
	ast.Inspect(file, func(node ast.Node) bool {
		switch n := node.(type) {
		case *ast.SelectorExpr:
			if n.Sel == nil || n.Sel.Name != "Host" {
				return true
			}

			if innerSelector, ok := n.X.(*ast.SelectorExpr); ok && innerSelector.Sel != nil {
				if innerSelector.Sel.Name == "Request" || innerSelector.Sel.Name == "URL" {
					violations = append(violations, fmt.Sprintf("%s: host-derived URL composition is banned; do not read Request.Host or URL.Host", path))
				}
			}
		case *ast.CallExpr:
			selector, ok := n.Fun.(*ast.SelectorExpr)
			if !ok || selector.Sel == nil {
				return true
			}

			if selector.Sel.Name != "Get" && selector.Sel.Name != "GetHeader" && selector.Sel.Name != "Values" {
				return true
			}

			if len(n.Args) == 0 {
				return true
			}

			literal, ok := n.Args[0].(*ast.BasicLit)
			if !ok || literal.Kind != token.STRING {
				return true
			}

			value, err := strconv.Unquote(literal.Value)
			if err != nil {
				return true
			}

			if value == "Host" || value == "X-Forwarded-Host" || value == "X-Original-Host" {
				violations = append(violations, fmt.Sprintf("%s: host-derived URL composition is banned; do not read host headers", path))
			}
		}

		return true
	})

	return violations
}

// checkPostgresRepositorySQLConstruction は Postgres adapter の repository 実装における unsafe SQL construction を検査する。
// 単一 postgres package 化後も Product/Admin repository 全体に対して Raw/Exec と動的 SQL fragment を拒否する。
func checkPostgresRepositorySQLConstruction(path string, file *ast.File) []string {
	// Step 1: Postgres adapter の repository 実装だけに限定し、database.go など DB open/ping のみの file は対象外にする。
	if !isPostgresRepositoryPath(path) || isTestFile(path) {
		return nil
	}

	// Step 2: GORM の SQL fragment を受け取る method を AST で検査し、raw SQL と動的 SQL 文字列を拒否する。
	violations := make([]string, 0)
	ast.Inspect(file, func(node ast.Node) bool {
		callExpr, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}

		selector, ok := callExpr.Fun.(*ast.SelectorExpr)
		if !ok || selector.Sel == nil {
			return true
		}

		// Step 3: Raw/Exec は query 全体を文字列で受けるため、Postgres repository では使用自体を禁止する。
		if selector.Sel.Name == "Raw" || selector.Sel.Name == "Exec" {
			violations = append(violations, fmt.Sprintf("%s: unsafe SQL construction is banned in Postgres repositories; use GORM parameter binding instead of %s", path, selector.Sel.Name))
			return true
		}

		// Step 4: Where/Order などの SQL fragment 引数は静的文字列だけに限定し、変数経由の動的 SQL も拒否する。
		if isGormSQLFragmentMethod(selector.Sel.Name) && len(callExpr.Args) > 0 && !isStaticStringLiteral(callExpr.Args[0]) {
			violations = append(violations, fmt.Sprintf("%s: unsafe SQL construction is banned in Postgres repositories; SQL fragments must be static string literals with bound parameters", path))
		}

		return true
	})

	// Step 5: 検出した violation を返し、pnpm lint 経由の guardrail で unsafe query を fail-closed にする。
	return violations
}

// isPostgresRepositoryPath は Postgres adapter の repository file かどうかを判定する。
// internal/adapter/postgres/ 配下の production Go file を広く対象とし、database.go など DB open/ping のみの file は除外する。
// 将来の repository 追加でも SQL construction guardrail の対象外にならないよう、明示列挙ではなく prefix + 除外パターンで判定する。
func isPostgresRepositoryPath(path string) bool {
	// Step 1: internal/adapter/postgres/ 配下でない file は対象外。
	if !strings.HasPrefix(path, "internal/adapter/postgres/") {
		return false
	}

	// Step 2: database.go は DB open/ping のみで SQL construction を行わないため除外する。
	if path == "internal/adapter/postgres/database.go" {
		return false
	}

	// Step 3: production Go file だけを対象にし、test file は checkPostgresRepositorySQLConstruction 側で除外する。
	return strings.HasSuffix(path, ".go")
}

func isGormSQLFragmentMethod(methodName string) bool {
	// Step 1: GORM で SQL fragment を受け取る代表 method だけを対象にし、Create/Find など安全な API を誤検知しない。
	switch methodName {
	case "Where", "Or", "Not", "Order", "Joins", "Having", "Select", "Table", "Group":
		return true
	default:
		return false
	}
}

func isStaticStringLiteral(expr ast.Expr) bool {
	// Step 1: SQL fragment の第1引数はソース上の string literal だけを許可し、変数・連結・formatting call を安全側で拒否する。
	literal, ok := expr.(*ast.BasicLit)
	return ok && literal.Kind == token.STRING
}

func containsForbiddenPortType(fields *ast.FieldList, imports map[string]string) bool {
	if fields == nil {
		return false
	}

	for _, field := range fields.List {
		if typeExprContainsForbiddenPortType(field.Type, imports) {
			return true
		}
	}

	return false
}

func typeExprContainsForbiddenPortType(expr ast.Expr, imports map[string]string) bool {
	found := false
	walkType(expr, func(candidate ast.Expr) bool {
		selector, ok := candidate.(*ast.SelectorExpr)
		if !ok {
			return false
		}

		ident, ok := selector.X.(*ast.Ident)
		if !ok {
			return false
		}

		importPath, ok := imports[ident.Name]
		if !ok {
			return false
		}

		for _, prefix := range forbiddenPortTypeImportPrefixes {
			if strings.HasPrefix(importPath, prefix) {
				found = true
				return true
			}
		}

		return false
	})

	return found
}

func containsForbiddenUsecaseDomainType(fields *ast.FieldList, imports map[string]string) bool {
	if fields == nil {
		return false
	}

	for _, field := range fields.List {
		found := false
		walkType(field.Type, func(expr ast.Expr) bool {
			selector, ok := expr.(*ast.SelectorExpr)
			if !ok {
				return false
			}

			ident, ok := selector.X.(*ast.Ident)
			if !ok {
				return false
			}

			importPath, ok := imports[ident.Name]
			if !ok {
				return false
			}

			if !domainImportPattern.MatchString(importPath) {
				return false
			}

			if strings.HasSuffix(selector.Sel.Name, "Repository") || strings.HasSuffix(selector.Sel.Name, "Port") {
				return false
			}
			if isAllowedApplicationBoundaryDomainType(selector.Sel.Name) {
				return false
			}

			found = true
			return true
		})

		if found {
			return true
		}
	}

	return false
}

func isAllowedApplicationBoundaryDomainType(typeName string) bool {
	switch typeName {
	case "AccountID", "AccountLocale", "TokenKind", "WebAuthnCredentialData", "WebAuthnStoredCredential", "PasskeyCredential":
		return true
	default:
		return false
	}
}

func functionBodyTouchesDomain(body *ast.BlockStmt, imports map[string]string) bool {
	if body == nil {
		return false
	}

	found := false
	ast.Inspect(body, func(node ast.Node) bool {
		selector, ok := node.(*ast.SelectorExpr)
		if !ok {
			return true
		}

		ident, ok := selector.X.(*ast.Ident)
		if !ok {
			return true
		}

		importPath, ok := imports[ident.Name]
		if !ok {
			return true
		}

		if domainImportPattern.MatchString(importPath) {
			found = true
			return false
		}

		return true
	})

	return found
}

func hasInlineUsecaseValidation(body *ast.BlockStmt, imports map[string]string, paramNames map[string]struct{}) bool {
	if body == nil {
		return false
	}

	found := false
	ast.Inspect(body, func(node ast.Node) bool {
		if found {
			return false
		}

		switch n := node.(type) {
		case *ast.BinaryExpr:
			if isStringEqualityCheck(n) && (exprReferencesParameters(n.X, paramNames) || exprReferencesParameters(n.Y, paramNames)) {
				found = true
				return false
			}
		case *ast.CallExpr:
			if isInlineValidationCall(n, imports, paramNames) {
				found = true
				return false
			}
		}

		return true
	})

	return found
}

func isInlineValidationCall(callExpr *ast.CallExpr, imports map[string]string, paramNames map[string]struct{}) bool {
	selector, ok := callExpr.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}

	ident, ok := selector.X.(*ast.Ident)
	if !ok {
		return false
	}

	importPath := imports[ident.Name]
	switch importPath {
	case "strings":
		if selector.Sel.Name != "TrimSpace" && selector.Sel.Name != "Trim" && selector.Sel.Name != "TrimLeft" && selector.Sel.Name != "TrimRight" {
			return false
		}
	case "regexp":
		if selector.Sel.Name != "Match" && selector.Sel.Name != "MatchReader" && selector.Sel.Name != "MatchString" && selector.Sel.Name != "Compile" && selector.Sel.Name != "CompilePOSIX" && selector.Sel.Name != "MustCompile" && selector.Sel.Name != "MustCompilePOSIX" {
			return false
		}
	default:
		return false
	}

	for _, arg := range callExpr.Args {
		if exprReferencesParameters(arg, paramNames) {
			return true
		}
	}

	return false
}

func exprReferencesParameters(expr ast.Expr, paramNames map[string]struct{}) bool {
	found := false
	ast.Inspect(expr, func(node ast.Node) bool {
		ident, ok := node.(*ast.Ident)
		if !ok {
			return true
		}

		if _, ok := paramNames[ident.Name]; ok {
			found = true
			return false
		}

		return true
	})

	return found
}

func isStringEqualityCheck(expr *ast.BinaryExpr) bool {
	if expr.Op != token.EQL && expr.Op != token.NEQ {
		return false
	}

	return isEmptyStringLiteral(expr.X) || isEmptyStringLiteral(expr.Y)
}

func isEmptyStringLiteral(expr ast.Expr) bool {
	literal, ok := expr.(*ast.BasicLit)
	return ok && literal.Kind == token.STRING && literal.Value == `""`
}

func nonContextParameterNames(fields *ast.FieldList, imports map[string]string) map[string]struct{} {
	paramNames := make(map[string]struct{})
	if fields == nil {
		return paramNames
	}

	for _, field := range fields.List {
		if isContextType(field.Type, imports) {
			continue
		}

		for _, name := range field.Names {
			paramNames[name.Name] = struct{}{}
		}
	}

	return paramNames
}

func isContextType(expr ast.Expr, imports map[string]string) bool {
	selector, ok := expr.(*ast.SelectorExpr)
	if !ok {
		return false
	}

	ident, ok := selector.X.(*ast.Ident)
	if !ok {
		return false
	}

	return imports[ident.Name] == "context" && selector.Sel.Name == "Context"
}

func requiresDirectDomainTouch(funcDecl *ast.FuncDecl) bool {
	if funcDecl == nil || funcDecl.Name == nil || !funcDecl.Name.IsExported() || funcDecl.Body == nil {
		return false
	}

	for _, prefix := range usecaseDomainTouchPrefixes {
		if strings.HasPrefix(funcDecl.Name.Name, prefix) {
			return true
		}
	}

	return false
}

func importAliases(file *ast.File) map[string]string {
	aliases := make(map[string]string, len(file.Imports))
	for _, imported := range file.Imports {
		importPath := strings.Trim(imported.Path.Value, "\"")
		if imported.Name != nil {
			aliases[imported.Name.Name] = importPath
			continue
		}

		_, base := filepath.Split(importPath)
		aliases[base] = importPath
	}

	return aliases
}

func isErrorStringCall(expr ast.Expr) bool {
	callExpr, ok := expr.(*ast.CallExpr)
	if !ok || len(callExpr.Args) != 0 {
		return false
	}

	selector, ok := callExpr.Fun.(*ast.SelectorExpr)
	return ok && selector.Sel != nil && selector.Sel.Name == "Error"
}

func isTestFile(path string) bool {
	return strings.HasSuffix(path, "_test.go")
}

func walkType(expr ast.Expr, visit func(ast.Expr) bool) {
	if expr == nil {
		return
	}

	if visit(expr) {
		return
	}

	switch n := expr.(type) {
	case *ast.ArrayType:
		walkType(n.Elt, visit)
	case *ast.ChanType:
		walkType(n.Value, visit)
	case *ast.Ellipsis:
		walkType(n.Elt, visit)
	case *ast.FuncType:
		if n.Params != nil {
			for _, field := range n.Params.List {
				walkType(field.Type, visit)
			}
		}
		if n.Results != nil {
			for _, field := range n.Results.List {
				walkType(field.Type, visit)
			}
		}
	case *ast.IndexExpr:
		walkType(n.X, visit)
		walkType(n.Index, visit)
	case *ast.IndexListExpr:
		walkType(n.X, visit)
		for _, index := range n.Indices {
			walkType(index, visit)
		}
	case *ast.InterfaceType:
		for _, field := range n.Methods.List {
			walkType(field.Type, visit)
		}
	case *ast.MapType:
		walkType(n.Key, visit)
		walkType(n.Value, visit)
	case *ast.ParenExpr:
		walkType(n.X, visit)
	case *ast.SelectorExpr:
		walkType(n.X, visit)
	case *ast.StarExpr:
		walkType(n.X, visit)
	case *ast.StructType:
		for _, field := range n.Fields.List {
			walkType(field.Type, visit)
		}
	}
}

func isExternalImport(importPath string) bool {
	firstSegment, _, _ := strings.Cut(importPath, "/")
	return strings.Contains(firstSegment, ".")
}

// layerFromPath はファイルパスから層名と feature 名を抽出する。
// 配置規約に基づき、feature 名に依存せず機械的に判定する。
func layerFromPath(path string) (layer string, feature string) {
	switch {
	case strings.HasPrefix(path, "cmd/") || path == "cmd":
		return "cmd", ""
	case strings.HasPrefix(path, "internal/app/") || path == "internal/app":
		return "app", ""
	case strings.HasPrefix(path, "internal/platform/") || path == "internal/platform":
		return "platform", ""
	case strings.HasPrefix(path, "internal/generated/") || path == "internal/generated":
		return "generated", ""
	case strings.HasPrefix(path, "internal/adapter/http/") || path == "internal/adapter/http":
		return "adapter-http", ""
	case strings.HasPrefix(path, "internal/adapter/postgres/") || path == "internal/adapter/postgres":
		return "adapter-postgres", ""
	case strings.HasPrefix(path, "internal/adapter/valkey/") || path == "internal/adapter/valkey":
		return "adapter-valkey", ""
	case strings.HasPrefix(path, "internal/adapter/webauthn/") || path == "internal/adapter/webauthn":
		return "adapter-webauthn", ""
	case strings.HasPrefix(path, "internal/adapter/mailer/") || path == "internal/adapter/mailer":
		return "adapter-mailer", ""
	case strings.HasPrefix(path, "internal/domain/") || path == "internal/domain":
		return "domain", ""
	case strings.HasPrefix(path, "internal/application/") || path == "internal/application":
		return "application", ""
	}

	return "", ""
}
