package application

import (
	"embed"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"strconv"
	"strings"
	"testing"
)

const (
	productAuthApplicationImportPath = "www-template/packages/backend/internal/application/product/auth"
	adminAuthApplicationImportPath   = "www-template/packages/backend/internal/application/admin/auth"
	rootAuthApplicationImportPath    = "www-template/packages/backend/internal/application"
)

var authDomainSwitchSelectorTerms = []string{
	"identitydomain",
	"identity_domain",
	"authdomain",
	"auth_domain",
	"tokendomain",
	"token_domain",
	"domainkind",
	"domain_kind",
	"domaintype",
	"domain_type",
	"domainselector",
	"domain_selector",
	"domaindiscriminator",
	"domain_discriminator",
	"authsurface",
	"auth_surface",
	"tokensurface",
	"token_surface",
	"productadmin",
	"product_admin",
	"adminproduct",
	"admin_product",
	"accountoperator",
	"account_operator",
	"operatoraccount",
	"operator_account",
}

// productAdminAuthApplicationSourceFiles は Product/Admin auth application と shared application source を test binary に固定して埋め込む。
// 実行時 path を外部入力から受け取らず、検査対象を backend-owned auth application boundary に限定する。
//
//go:embed product/auth/*.go admin/auth/*.go shared
var productAdminAuthApplicationSourceFiles embed.FS

// TestProductAdminAuthApplicationImportBoundary は Product/Admin auth application の相互 import 禁止を検証する。
func TestProductAdminAuthApplicationImportBoundary(t *testing.T) {
	t.Parallel()

	// Step 1: Product auth application 側の production source だけを走査し、Admin auth application import がないことを検証する。
	t.Run("[AUTH-BE-S071] product auth application does not import admin auth application", func(t *testing.T) {
		t.Parallel()
		assertEmbeddedPackageDoesNotImport(t, "product/auth/", adminAuthApplicationImportPath)
	})

	// Step 2: Admin auth application 側の production source だけを走査し、Product auth application import がないことを検証する。
	t.Run("[AUTH-BE-S072] admin auth application does not import product auth application", func(t *testing.T) {
		t.Parallel()
		assertEmbeddedPackageDoesNotImport(t, "admin/auth/", productAuthApplicationImportPath)
	})
}

// TestProductAdminAuthDomainSeparationBoundary は Product/Admin auth domain を単一 switch-based token service に戻せないことを検証する。
func TestProductAdminAuthDomainSeparationBoundary(t *testing.T) {
	t.Parallel()

	// Step 1: Product auth application が Admin/Product 切替用 selector を受け取らず、Product account 境界に閉じることを固定する。
	t.Run("[AUTH-BE-S067] product auth application has no auth domain switch selector", func(t *testing.T) {
		t.Parallel()
		assertEmbeddedPackageHasNoDomainSwitchSelector(t, "product/auth/")
	})

	// Step 2: Admin auth application が Product/Admin 切替用 selector を受け取らず、Admin operator 境界に閉じることを固定する。
	t.Run("[AUTH-BE-S067] admin auth application has no auth domain switch selector", func(t *testing.T) {
		t.Parallel()
		assertEmbeddedPackageHasNoDomainSwitchSelector(t, "admin/auth/")
	})

	// Step 3: Product/Admin auth application が legacy root application TokenService を import して共有 service 化しないことを固定する。
	t.Run("[AUTH-BE-S067] product and admin auth applications do not import root token service", func(t *testing.T) {
		t.Parallel()
		assertEmbeddedPackageDoesNotImport(t, "product/auth/", rootAuthApplicationImportPath)
		assertEmbeddedPackageDoesNotImport(t, "admin/auth/", rootAuthApplicationImportPath)
	})

	// Step 4: shared application package が Issue/Refresh/Revoke を持つ共有 token service へ成長しないことを固定する。
	t.Run("[AUTH-BE-S067] shared application owns no auth lifecycle token service", func(t *testing.T) {
		t.Parallel()
		assertSharedAuthPackageHasNoLifecycleService(t)
	})

	// Step 5: token lifecycle 操作が Product/Admin または Account/Operator の switch 文で意味を切り替えないことを固定する。
	t.Run("[AUTH-BE-S067] token lifecycle operations do not switch product admin domains", func(t *testing.T) {
		t.Parallel()
		assertEmbeddedPackageHasNoDomainSwitchingAuthOperation(t, "product/auth/")
		assertEmbeddedPackageHasNoDomainSwitchingAuthOperation(t, "admin/auth/")
		assertEmbeddedPackageHasNoDomainSwitchingAuthOperation(t, "shared/")
	})
}

func assertEmbeddedPackageDoesNotImport(t *testing.T, packagePrefix string, forbiddenImportPath string) {
	t.Helper()

	// Step 1: go:embed 済み source tree を使い、runtime filesystem へ依存しない固定入力だけを検査する。
	err := fs.WalkDir(productAdminAuthApplicationSourceFiles, ".", func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		// Step 2: 対象 package prefix の production Go source だけを import graph guardrail の対象にする。
		if shouldSkipAuthBoundarySource(path, entry, packagePrefix) {
			return nil
		}

		// Step 3: 埋め込み済み source を読み、go/parser が import 宣言だけを AST として解釈できる形で渡す。
		source, readErr := productAdminAuthApplicationSourceFiles.ReadFile(path)
		if readErr != nil {
			return readErr
		}

		// Step 4: import 宣言に禁止 package が現れた場合は、境界崩壊として test を即時失敗させる。
		assertSourceDoesNotImport(t, path, source, forbiddenImportPath)
		return nil
	})
	if err != nil {
		t.Fatalf("walk embedded auth application sources: %v", err)
	}
}

func assertEmbeddedPackageHasNoDomainSwitchSelector(t *testing.T, packagePrefix string) {
	t.Helper()

	// Step 1: production source だけを固定入力として走査し、test fixture の説明文による誤検知を避ける。
	walkEmbeddedAuthBoundarySources(t, packagePrefix, func(path string, source []byte) {
		// Step 2: AST を comment なしで解析し、識別子と literal の実コードだけを検査する。
		fileSet := token.NewFileSet()
		parsedFile := parseEmbeddedAuthBoundaryFile(t, fileSet, path, source)

		// Step 3: identityDomain などの切替 selector 語彙が入った時点で境界崩壊として失敗させる。
		assertASTHasNoForbiddenTerm(t, fileSet, parsedFile, authDomainSwitchSelectorTerms)
	})
}

func assertSharedAuthPackageHasNoLifecycleService(t *testing.T) {
	t.Helper()

	// Step 1: shared application package は署名・TTL などの中立 primitive だけを持つため、lifecycle service 宣言を禁止する。
	forbiddenServiceTerms := append([]string{"tokenservice", "sharedtokenservice", "shared_token_service"}, authDomainSwitchSelectorTerms...)
	walkEmbeddedAuthBoundarySources(t, "shared/", func(path string, source []byte) {
		fileSet := token.NewFileSet()
		parsedFile := parseEmbeddedAuthBoundaryFile(t, fileSet, path, source)

		// Step 2: shared package に TokenService 型や domain switch selector が宣言された場合は共有 service 化として拒否する。
		assertASTHasNoForbiddenTerm(t, fileSet, parsedFile, forbiddenServiceTerms)

		// Step 3: Issue/Refresh/Revoke などの auth lifecycle operation を shared primitive が直接所有しないことを検証する。
		for _, declaration := range parsedFile.Decls {
			assertDeclarationIsNotSharedLifecycleService(t, fileSet, declaration)
		}
	})
}

func assertEmbeddedPackageHasNoDomainSwitchingAuthOperation(t *testing.T, packagePrefix string) {
	t.Helper()

	// Step 1: Product/Admin/shared の production source を AST として解析し、実装上の switch 文だけを対象にする。
	walkEmbeddedAuthBoundarySources(t, packagePrefix, func(path string, source []byte) {
		fileSet := token.NewFileSet()
		parsedFile := parseEmbeddedAuthBoundaryFile(t, fileSet, path, source)

		// Step 2: token lifecycle 関数の本文に Product/Admin または Account/Operator 切替 switch がないことを検査する。
		for _, declaration := range parsedFile.Decls {
			assertAuthOperationHasNoDomainSwitch(t, fileSet, declaration)
		}
	})
}

func walkEmbeddedAuthBoundarySources(t *testing.T, packagePrefix string, visit func(path string, source []byte)) {
	t.Helper()

	// Step 1: go:embed 済み source tree を walk し、実行時 filesystem の変更や外部 path 入力へ依存しない。
	err := fs.WalkDir(productAdminAuthApplicationSourceFiles, ".", func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		// Step 2: 指定 prefix の production Go source だけを境界検査対象にする。
		if shouldSkipAuthBoundarySource(path, entry, packagePrefix) {
			return nil
		}

		// Step 3: 埋め込み済み source を読み、呼び出し側の静的検査へ渡す。
		source, readErr := productAdminAuthApplicationSourceFiles.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		visit(path, source)
		return nil
	})
	if err != nil {
		t.Fatalf("walk embedded auth boundary sources: %v", err)
	}
}

func shouldSkipAuthBoundarySource(path string, entry fs.DirEntry, packagePrefix string) bool {
	// Step 1: directory と対象外 prefix は検査から除外し、Product/Admin の片側ごとに境界を確認する。
	if entry.IsDir() || !strings.HasPrefix(path, packagePrefix) {
		return true
	}

	// Step 2: test source は検証 fixture として相互 package を参照する可能性があるため、production source 境界から外す。
	return !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go")
}

func assertSourceDoesNotImport(t *testing.T, path string, source []byte, forbiddenImportPath string) {
	t.Helper()

	// Step 1: import 宣言だけを parse し、コメントや文字列 literal による誤検知を避ける。
	fileSet := token.NewFileSet()
	file, err := parser.ParseFile(fileSet, path, source, parser.ImportsOnly)
	if err != nil {
		t.Fatalf("parse auth application source %s: %v", path, err)
	}

	// Step 2: 各 import spec を unquote し、alias の有無に関係なく実 import path を比較する。
	for _, importSpec := range file.Imports {
		importPath, unquoteErr := strconv.Unquote(importSpec.Path.Value)
		if unquoteErr != nil {
			t.Fatalf("unquote import path in %s: %v", path, unquoteErr)
		}
		if importPath == forbiddenImportPath {
			t.Fatalf("%s must not import %s", path, forbiddenImportPath)
		}
	}
}

func parseEmbeddedAuthBoundaryFile(t *testing.T, fileSet *token.FileSet, path string, source []byte) *ast.File {
	t.Helper()

	// Step 1: comment は検査対象から外し、識別子・literal・分岐構文だけを実装として解釈する。
	parsedFile, err := parser.ParseFile(fileSet, path, source, 0)
	if err != nil {
		t.Fatalf("parse auth boundary source %s: %v", path, err)
	}

	// Step 2: 後続の AST guardrail が同じ parse 結果を再利用できるよう返す。
	return parsedFile
}

func assertASTHasNoForbiddenTerm(t *testing.T, fileSet *token.FileSet, parsedFile *ast.File, forbiddenTerms []string) {
	t.Helper()

	// Step 1: AST を走査し、識別子と literal に切替 selector 語彙が含まれないことを検証する。
	ast.Inspect(parsedFile, func(node ast.Node) bool {
		switch typedNode := node.(type) {
		case *ast.Ident:
			assertValueHasNoForbiddenTerm(t, fileSet, typedNode.Pos(), "identifier", typedNode.Name, forbiddenTerms)
		case *ast.BasicLit:
			assertValueHasNoForbiddenTerm(t, fileSet, typedNode.Pos(), "literal", typedNode.Value, forbiddenTerms)
		}

		// Step 2: すべての node を検査し、深い式にある selector も見落とさない。
		return true
	})
}

func assertDeclarationIsNotSharedLifecycleService(t *testing.T, fileSet *token.FileSet, declaration ast.Decl) {
	t.Helper()

	// Step 1: shared application package の function 宣言が lifecycle operation を直接所有しないことを確認する。
	if functionDeclaration, ok := declaration.(*ast.FuncDecl); ok && isSharedTokenLifecycleOperationName(functionDeclaration.Name.Name) {
		t.Fatalf("shared tokenprimitive must not declare auth lifecycle operation %s at %s", functionDeclaration.Name.Name, fileSet.Position(functionDeclaration.Pos()))
	}
}

func assertAuthOperationHasNoDomainSwitch(t *testing.T, fileSet *token.FileSet, declaration ast.Decl) {
	t.Helper()

	// Step 1: function 宣言以外、または本文のない宣言は switch-based operation にならないため対象外にする。
	functionDeclaration, ok := declaration.(*ast.FuncDecl)
	if !ok || functionDeclaration.Body == nil || !isAuthLifecycleOperationName(functionDeclaration.Name.Name) {
		return
	}

	// Step 2: lifecycle function の中に Product/Admin や Account/Operator を切り替える switch 文がないことを検査する。
	ast.Inspect(functionDeclaration.Body, func(node ast.Node) bool {
		switch typedNode := node.(type) {
		case *ast.SwitchStmt:
			assertSwitchDoesNotCollapseAuthDomains(t, fileSet, typedNode)
		case *ast.TypeSwitchStmt:
			assertSwitchDoesNotCollapseAuthDomains(t, fileSet, typedNode)
		}

		// Step 3: nested block も含めて全 switch を検査する。
		return true
	})
}

func assertSwitchDoesNotCollapseAuthDomains(t *testing.T, fileSet *token.FileSet, switchNode ast.Node) {
	t.Helper()

	// Step 1: switch subtree に Product/Admin または Account/Operator の両側語彙が揃う場合、単一 service の domain 分岐として拒否する。
	if nodeContainsTermPair(switchNode, []string{"product", "account"}, []string{"admin", "operator"}) || nodeContainsForbiddenTerm(switchNode, authDomainSwitchSelectorTerms) {
		t.Fatalf("auth lifecycle operation must not switch product/admin auth domains; found at %s", fileSet.Position(switchNode.Pos()))
	}
}

func nodeContainsTermPair(node ast.Node, leftTerms []string, rightTerms []string) bool {
	// Step 1: 片側だけの語彙では Product-only / Admin-only 実装の説明になり得るため、両側が揃った場合だけ崩壊候補にする。
	foundLeft := false
	foundRight := false
	ast.Inspect(node, func(child ast.Node) bool {
		value, ok := authBoundaryNodeValue(child)
		if !ok {
			return true
		}
		foundLeft = foundLeft || containsAnyTerm(value, leftTerms)
		foundRight = foundRight || containsAnyTerm(value, rightTerms)
		return true
	})

	// Step 2: Product/Admin または Account/Operator の両側語彙が同じ switch に入った場合だけ true を返す。
	return foundLeft && foundRight
}

func nodeContainsForbiddenTerm(node ast.Node, forbiddenTerms []string) bool {
	// Step 1: 呼び出し元が fatal 位置を管理できるよう、ここでは真偽値だけを返す。
	found := false
	ast.Inspect(node, func(child ast.Node) bool {
		value, ok := authBoundaryNodeValue(child)
		if !ok {
			return true
		}
		found = found || containsAnyTerm(value, forbiddenTerms)
		return !found
	})

	// Step 2: 禁止 selector 語彙を見つけたかどうかを返す。
	return found
}

func authBoundaryNodeValue(node ast.Node) (string, bool) {
	// Step 1: 実装語彙として意味を持つ identifier と literal だけを抽出する。
	switch typedNode := node.(type) {
	case *ast.Ident:
		return typedNode.Name, true
	case *ast.BasicLit:
		return typedNode.Value, true
	default:
		return "", false
	}
}

func assertValueHasNoForbiddenTerm(t *testing.T, fileSet *token.FileSet, position token.Pos, kind string, value string, forbiddenTerms []string) {
	t.Helper()

	// Step 1: 大文字小文字や snake/camel の差で selector 語彙がすり抜けないよう正規化して比較する。
	normalizedValue := strings.ToLower(value)
	for _, term := range forbiddenTerms {
		if strings.Contains(normalizedValue, term) {
			t.Fatalf("auth boundary contains forbidden domain switch %s term %q in %q at %s", kind, term, value, fileSet.Position(position))
		}
	}
}

func containsAnyTerm(value string, terms []string) bool {
	// Step 1: AST node から得た値を lowercase にし、case difference による抜け道を閉じる。
	normalizedValue := strings.ToLower(value)
	for _, term := range terms {
		if strings.Contains(normalizedValue, term) {
			return true
		}
	}

	// Step 2: どの語彙にも一致しなかった場合だけ false を返す。
	return false
}

func isAuthLifecycleOperationName(name string) bool {
	// Step 1: token/session lifecycle に関わる操作だけを switch guardrail の対象にし、無関係な helper の将来追加余地を残す。
	return containsAnyTerm(name, []string{"issue", "login", "finish", "refresh", "rotate", "revoke", "logout", "validate", "current", "session", "sign", "verify"})
}

func isSharedTokenLifecycleOperationName(name string) bool {
	// Step 1: shared application package が署名・TTL primitive を超えて auth lifecycle use case を所有し始めた場合に拒否する。
	normalizedName := strings.ToLower(name)
	return strings.HasPrefix(normalizedName, "issue") || strings.HasPrefix(normalizedName, "refresh") || strings.HasPrefix(normalizedName, "rotate") || strings.HasPrefix(normalizedName, "revoke") || strings.HasPrefix(normalizedName, "login") || strings.HasPrefix(normalizedName, "logout")
}
