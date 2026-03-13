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

const modulePath = "witwire.net/www-template/packages/backend"

var migrationFilePattern = regexp.MustCompile(`^\d{6}_[a-z0-9_]+\.(up|down)\.sql$`)

var allowedInternalImports = map[string][]string{
	"app":         {"app", "domain", "http", "persistence", "types", "usecases"},
	"cmd":         {"app"},
	"domain":      {"domain", "types"},
	"http":        {"domain", "generated", "http", "types", "usecases"},
	"persistence": {"domain", "persistence", "types"},
	"tools":       {"tools"},
	"types":       {"types"},
	"usecases":    {"domain", "types", "usecases"},
}

var allowedExternalImports = map[string][]string{
	"http":        {"github.com/gin-contrib/cors", "github.com/gin-gonic/gin", "github.com/oapi-codegen/runtime/types"},
	"persistence": {"gorm.io/driver/postgres", "gorm.io/gorm"},
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

var usecaseDomainTouchPrefixes = []string{"Create", "Update", "Change", "Rename", "Set", "Add"}

var forbiddenPortTypeImportPrefixes = []string{
	modulePath + "/internal/generated",
	modulePath + "/internal/http",
	modulePath + "/internal/persistence",
	"github.com/gin-contrib/cors",
	"github.com/gin-gonic/gin",
	"github.com/oapi-codegen/runtime/types",
	"gorm.io/",
}

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

			if strings.HasPrefix(relativePath, "internal/generated/") {
				violations = append(violations, verifyGeneratedFile(relativePath)...)
				return nil
			}

			file, err := parser.ParseFile(fileSet, path, nil, parser.ParseComments)
			if err != nil {
				return fmt.Errorf("parse %s: %w", path, err)
			}

			violations = append(violations, checkImports(relativePath, file)...)
			violations = append(violations, checkAutoMigrate(relativePath, file)...)
			violations = append(violations, checkCoreSideEffects(relativePath, file)...)
			violations = append(violations, checkDomainCompositeLiterals(relativePath, file)...)
			violations = append(violations, checkErrorStringMatching(relativePath, file)...)
			violations = append(violations, checkForbiddenCalls(relativePath, file)...)
			violations = append(violations, checkForbiddenHostUsage(relativePath, file)...)
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
		"cmd/api/",
		"internal/app/",
		"internal/domain/",
		"internal/generated/",
		"internal/http/",
		"internal/persistence/",
		"internal/types/",
		"internal/usecases/",
		"tools/analyzers/",
	}

	for _, prefix := range allowedPrefixes {
		if strings.HasPrefix(path, prefix) {
			return nil
		}
	}

	return []string{fmt.Sprintf("%s: go files must live under cmd/api, internal/*, or tools/analyzers", path)}
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
	layer := layerFromPath(path)
	if layer == "generated" || layer == "" {
		return nil
	}

	violations := make([]string, 0)
	for _, imported := range file.Imports {
		importPath := strings.Trim(imported.Path.Value, "\"")

		if strings.HasPrefix(importPath, modulePath+"/") {
			targetLayer := layerFromImport(importPath)
			if targetLayer == "" {
				violations = append(violations, fmt.Sprintf("%s: internal import %s does not map to an allowed backend layer", path, importPath))
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

		if strings.HasPrefix(importPath, "gorm.io/") && layer != "persistence" {
			violations = append(violations, fmt.Sprintf("%s: gorm imports are only allowed in internal/persistence", path))
			continue
		}

		allowedExternal := allowedExternalImports[layer]
		if len(allowedExternal) == 0 || !slices.Contains(allowedExternal, importPath) {
			violations = append(violations, fmt.Sprintf("%s: %s must not import external package %s", path, layer, importPath))
		}
	}

	return violations
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
	layer := layerFromPath(path)
	if isTestFile(path) || (layer != "domain" && layer != "usecases") {
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
	if isTestFile(path) || layerFromPath(path) == "domain" {
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
		if !ok || imports[ident.Name] != modulePath+"/internal/domain" {
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
	if isTestFile(path) || layerFromPath(path) != "http" {
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
		if !ok || imports[ident.Name] != modulePath+"/internal/domain" {
			return true
		}

		if strings.HasSuffix(selector.Sel.Name, "Repository") || strings.HasSuffix(selector.Sel.Name, "Port") {
			return true
		}

		violations = append(violations, fmt.Sprintf("%s: http must not depend on domain.%s directly; map transport DTOs to usecase DTOs instead", path, selector.Sel.Name))
		return true
	})

	return violations
}

func checkPortPurity(path string, file *ast.File) []string {
	layer := layerFromPath(path)
	if isTestFile(path) || (layer != "domain" && layer != "usecases") {
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
	if isTestFile(path) || layerFromPath(path) != "usecases" {
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
			violations = append(violations, fmt.Sprintf("%s: exported usecase API %s must use usecase DTOs instead of domain entities or commands", path, funcDecl.Name.Name))
		}
	}

	return violations
}

func checkWriteUsecasesTouchDomain(path string, file *ast.File) []string {
	if isTestFile(path) || layerFromPath(path) != "usecases" {
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
			violations = append(violations, fmt.Sprintf("%s: exported write usecase %s must call into internal/domain directly so business rules cannot bypass the domain layer", path, funcDecl.Name.Name))
		}
	}

	return violations
}

func checkUsecaseInlineBusinessValidation(path string, file *ast.File) []string {
	if isTestFile(path) || layerFromPath(path) != "usecases" {
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
			violations = append(violations, fmt.Sprintf("%s: exported write usecase %s must delegate trimming and validation to internal/domain instead of validating request fields inline", path, funcDecl.Name.Name))
		}
	}

	return violations
}

func checkRoutePolicy(path string, file *ast.File) []string {
	if layerFromPath(path) != "http" {
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
			if routePath != "/api/v1/app" && !strings.HasPrefix(routePath, "/api/v1/app/") {
				violations = append(violations, fmt.Sprintf("%s: custom Gin groups must live under /api/v1/app/*, got %s", path, routePath))
			}
			return true
		}

		if routePath == "/health" || routePath == "/api/v1/app" || strings.HasPrefix(routePath, "/api/v1/app/") {
			return true
		}

		violations = append(violations, fmt.Sprintf("%s: non-generated Gin routes must be /health or /api/v1/app/*, got %s", path, routePath))
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
			if !ok || imports[ident.Name] != modulePath+"/internal/domain" {
				return false
			}

			if strings.HasSuffix(selector.Sel.Name, "Repository") || strings.HasSuffix(selector.Sel.Name, "Port") {
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

		if imports[ident.Name] == modulePath+"/internal/domain" {
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

func layerFromImport(importPath string) string {
	return layerFromPath(strings.TrimPrefix(importPath, modulePath+"/"))
}

func layerFromPath(path string) string {
	switch {
	case path == "cmd/api" || strings.HasPrefix(path, "cmd/api/"):
		return "cmd"
	case path == "internal/app" || strings.HasPrefix(path, "internal/app/"):
		return "app"
	case path == "internal/domain" || strings.HasPrefix(path, "internal/domain/"):
		return "domain"
	case path == "internal/generated" || strings.HasPrefix(path, "internal/generated/"):
		return "generated"
	case path == "internal/http" || strings.HasPrefix(path, "internal/http/"):
		return "http"
	case path == "internal/persistence" || strings.HasPrefix(path, "internal/persistence/"):
		return "persistence"
	case path == "internal/types" || strings.HasPrefix(path, "internal/types/"):
		return "types"
	case path == "internal/usecases" || strings.HasPrefix(path, "internal/usecases/"):
		return "usecases"
	case path == "tools/analyzers" || strings.HasPrefix(path, "tools/analyzers/"):
		return "tools"
	default:
		return ""
	}
}
