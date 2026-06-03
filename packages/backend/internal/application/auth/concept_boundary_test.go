package auth

import (
	"embed"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

const (
	legacyProductAuthApplicationImportPath = "www-template/packages/backend/internal/application/product/auth"
	legacyAdminAuthApplicationImportPath   = "www-template/packages/backend/internal/application/admin/auth"
	conceptAuthApplicationImportPath       = "www-template/packages/backend/internal/application/auth"
	rootAuthApplicationImportPath          = "www-template/packages/backend/internal/application"
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

// productAdminAuthApplicationSourceFiles は Account/Operator auth application source を test binary に固定して埋め込む。
// 実行時 path を外部入力から受け取らず、検査対象を backend-owned auth application boundary に限定する。
//
//go:embed *.go
var productAdminAuthApplicationSourceFiles embed.FS

// TestAuthConceptApplicationImportBoundary は Account/Operator auth lifecycle が concept package 境界で分離されることを検証する。
func TestAuthConceptApplicationImportBoundary(t *testing.T) {
	t.Parallel()

	// Step 1: 単一 auth concept package が旧 account/operator subpackage import に戻らないことを検証する。
	t.Run("[AUTH-BE-S096] auth lifecycle does not import legacy split lifecycle packages", func(t *testing.T) {
		t.Parallel()
		assertEmbeddedPackageDoesNotImport(t, "", conceptAuthApplicationImportPath+"/account")
		assertEmbeddedPackageDoesNotImport(t, "", conceptAuthApplicationImportPath+"/operator")
	})

	// Step 2: removed authorization package が production Go source を持たないことを検証する。
	t.Run("[AUTH-BE-S096] removed authorization package has no production Go sources", func(t *testing.T) {
		t.Parallel()
		assertNoProductionGoFilesUnder(t, "../authorization")
	})

	// Step 3: legacy Product/Admin auth owner directory が production Go source を持たないことを filesystem で検証する。
	t.Run("[AUTH-BE-S093] legacy surface auth owner directories have no production Go sources", func(t *testing.T) {
		t.Parallel()
		assertNoProductionGoFilesUnder(t, "../product/auth")
		assertNoProductionGoFilesUnder(t, "../admin/auth")
	})
}

// TestAuthConceptSubjectPayloadBoundary は Account/Operator subject payload を switch-based shared lifecycle に畳み込めないことを検証する。
func TestAuthConceptSubjectPayloadBoundary(t *testing.T) {
	t.Parallel()

	// Step 1: Account auth application が Admin/Product 切替用 selector を受け取らず、Account 境界に閉じることを固定する。
	t.Run("[AUTH-BE-S067] account auth application has no auth domain switch selector", func(t *testing.T) {
		t.Parallel()
		assertEmbeddedSourceHasNoDomainSwitchSelector(t, "account_session_service.go")
	})

	// Step 2: Operator auth application が Product/Admin 切替用 selector を受け取らず、Operator 境界に閉じることを固定する。
	t.Run("[AUTH-BE-S067] operator auth application has no auth domain switch selector", func(t *testing.T) {
		t.Parallel()
		assertEmbeddedSourceHasNoDomainSwitchSelector(t, "operator_session_service.go")
	})

	// Step 3: Account/Operator auth application が legacy root application TokenService を import して共有 service 化しないことを固定する。
	t.Run("[AUTH-BE-S067] account and operator auth applications do not import root token service", func(t *testing.T) {
		t.Parallel()
		assertEmbeddedPackageDoesNotImport(t, "", rootAuthApplicationImportPath)
	})

	// Step 4: shared application package が token primitive wrapper として復活しないことを固定する。
	t.Run("[AUTH-BE-S067] shared application owns no auth lifecycle token service", func(t *testing.T) {
		t.Parallel()
		assertNoProductionGoFilesUnder(t, "../shared")
	})

	// Step 5: token lifecycle 操作が hosted service 名や Account/Operator の switch 文で意味を切り替えないことを固定する。
	t.Run("[AUTH-BE-S096] token lifecycle operations do not collapse subject decisions", func(t *testing.T) {
		t.Parallel()
		assertEmbeddedPackageHasNoDomainSwitchingAuthOperation(t, "")
	})
}

// TestCanonicalAuthConceptPackageBoundary は canonical auth lifecycle concept が surface package から明示利用されることを検証する。
func TestCanonicalAuthConceptPackageBoundary(t *testing.T) {
	t.Parallel()

	// Step 1: canonical auth concept package が Account/Operator subject payload と refresh credential hash を所有することを固定する。
	t.Run("[AUTH-BE-S093] auth concept package owns subject and refresh hash helpers", func(t *testing.T) {
		t.Parallel()
		assertEmbeddedSourceContains(t, "subject.go", []string{"AccountSubjectPayload", "OperatorSubjectPayload", "NewAccountSubjectPayload", "NewOperatorSubjectPayload"})
		assertEmbeddedSourceContains(t, "refresh.go", []string{"HashRefreshCredential", "domain.HashOpaqueToken"})
		assertNoProductionGoFilesUnder(t, "cookie.go")
	})

	// Step 2: Account/Operator lifecycle service が単一 auth concept package 内の helper を直接呼ぶことを固定する。
	t.Run("[AUTH-BE-S097] account and operator lifecycle services call canonical helpers", func(t *testing.T) {
		t.Parallel()
		assertEmbeddedSourceContains(t, "account_session_service.go", []string{"NewAccountSubjectPayload", "HashRefreshCredential"})
		assertEmbeddedSourceContains(t, "operator_session_service.go", []string{"NewOperatorSubjectPayload", "domain.EnsureRefreshContext"})
	})

	// Step 3: backend production source が legacy Product/Admin auth owner import を保持しないことを AST import で検証する。
	t.Run("[AUTH-BE-S100] production callers do not import legacy surface auth owners", func(t *testing.T) {
		t.Parallel()
		assertBackendProductionSourcesDoNotImport(t, []string{legacyProductAuthApplicationImportPath, legacyAdminAuthApplicationImportPath})
	})

	// Step 4: root legacy token service が production source から削除され、独自 lifecycle primitive へ戻れないことを確認する。
	t.Run("[AUTH-BE-S100] root token service is removed", func(t *testing.T) {
		t.Parallel()
		assertNoProductionGoFilesUnder(t, "../token_service.go")
	})

	// Step 5: Product production container が root token service ではなく canonical account lifecycle を構築して HTTP adapter へ渡すことを検証する。
	t.Run("[AUTH-BE-S100] product production container wires canonical account lifecycle", func(t *testing.T) {
		t.Parallel()
		assertContainerSourceContains(t, []string{"productauth.NewAccountSessionService", "productvalkey.NewAccountRefreshSessionStore", "productvalkey.NewAccountSessionMetadataStore", "productauth.NewProductContextRefreshService(productLifecycle)", "productauth.NewProductSessionService(productLifecycle)", "productauth.NewProductAuthService(stateRepo, accountRepo, recoverySender, rejectingInvitationPasskeyRegistrar{}, productLifecycle"})
		assertContainerSourceDoesNotContain(t, []string{"productauth.NewTokenService(", "productauth.NewSessionService("})
	})

	// Step 6: Operator application error が Admin surface 名に戻らず、concept-owned な Operator 語彙だけで公開されることを検証する。
	t.Run("[AUTH-BE-S100] operator auth errors do not use admin surface vocabulary", func(t *testing.T) {
		t.Parallel()
		assertBackendProductionASTDoesNotContain(t, []string{"err" + "adminauth"})
	})
}

// TestProductAdminAuthEligibilityOwnershipBoundary は Product/Admin それぞれの eligibility 判断が専用 domain object に残っていることを検証する。
func TestProductAdminAuthEligibilityOwnershipBoundary(t *testing.T) {
	t.Parallel()

	// Step 1: Account auth は AccountAuth domain constructor と method で token eligibility を検証することを固定する。
	t.Run("[AUTH-BE-S069] product account auth domain owns account token eligibility", func(t *testing.T) {
		t.Parallel()
		assertEmbeddedSourceContains(t, "account_session_refresh.go", []string{"NewAccountAccessTokenClaims", "CanRotate", "EnsureEligible"})
		assertEmbeddedSourceContains(t, "account_session_service.go", []string{"NewAccountRefreshSession"})
	})

	// Step 2: Operator auth は OperatorAuth domain constructor と method で operator eligibility / permission を検証することを固定する。
	t.Run("[AUTH-BE-S070] operator auth domain owns operator token eligibility", func(t *testing.T) {
		t.Parallel()
		assertEmbeddedSourceContains(t, "operator_session_service.go", []string{"NewOperatorAuthSession", "NewOperatorAccessTokenClaims", "ValidateAccess"})
	})

	// Step 3: Operator session lifecycle は passkey outer flow を保持せず、login ceremony は専用 service に分離されることを固定する。
	t.Run("[AUTH-BE-S070] operator session lifecycle does not own passkey login outer flow", func(t *testing.T) {
		t.Parallel()
		assertEmbeddedSourceDoesNotContain(t, "operator_session_service.go", []string{"StartOperatorPasskey", "FinishOperatorPasskey", "OperatorPasskeyChallengeProvider", "Challenges"})
		assertEmbeddedSourceContains(t, "operator_passkey_login_service.go", []string{"OperatorPasskeyLoginService", "StartOperatorPasskey", "FinishOperatorPasskey", "OperatorSessionIssuer"})
	})

	// Step 4: Product Account session lifecycle は passkey credential handle を直接 login 入力にせず、outer flow から確定済み AccountID だけを受け取ることを固定する。
	t.Run("[AUTH-BE-S069] product account session lifecycle does not own passkey login outer flow", func(t *testing.T) {
		t.Parallel()
		assertEmbeddedSourceDoesNotContain(t, "account_session_service.go", []string{"LoginWithPasskey", "LoginWithPasskeyInput"})
		assertEmbeddedSourceContains(t, "account_session_service.go", []string{"IssueAccountSession", "IssueAccountSessionInput"})
	})
}

// TestAuthConceptCanonicalPrimitiveOwnershipBoundary は token primitive を domain、context path を HTTP adapter へ寄せていることを検証する。
func TestAuthConceptCanonicalPrimitiveOwnershipBoundary(t *testing.T) {
	t.Parallel()

	// Step 1: legacy root TokenService が削除され、独自 hash/path 実装へ戻らないことを固定する。
	t.Run("[AUTH-BE-S098] legacy token service is absent", func(t *testing.T) {
		t.Parallel()
		assertNoProductionGoFilesUnder(t, "../token_service.go")
	})

	// Step 2: Product concept package の refresh hash は domain primitive へ直接委譲し、domain/application の二重 hash 規則を作らないことを固定する。
	t.Run("[AUTH-BE-S095] product refresh lifecycle delegates opaque token hashing", func(t *testing.T) {
		t.Parallel()
		assertEmbeddedSourceContains(t, "refresh.go", []string{"domain.HashOpaqueToken"})
	})

	// Step 3: Operator concept package の refresh Cookie path は application/auth DTO helper へ委譲せず、HTTP adapter が transport path を所有することを固定する。
	t.Run("[ADMIN-AUTH-BE-S081] operator refresh lifecycle leaves context path construction to HTTP adapter", func(t *testing.T) {
		t.Parallel()
		assertEmbeddedSourceContains(t, "operator_session_service.go", []string{"domain.EnsureRefreshContext"})
		assertEmbeddedSourceDoesNotContain(t, "operator_session_service.go", []string{"BuildRefreshPath", "NewRefreshCookieClearCommand", "SameSite", "HTTPOnly", "Secure"})
		assertEmbeddedSourceDoesNotContain(t, "operator_session_service.go", []string{"\"/api/v1/auth/contexts/\" +"})
	})

	// Step 4: Operator logout の Cookie clear command が transport 属性を持たず、auth context selector と削除意図だけを返すことを固定する。
	t.Run("[AUTH-BE-S095] operator logout returns selector-only clear command", func(t *testing.T) {
		t.Parallel()
		assertEmbeddedSourceContains(t, "operator_session_service.go", []string{"AuthContextID", "Clear"})
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

func assertEmbeddedSourceHasNoDomainSwitchSelector(t *testing.T, path string) {
	t.Helper()

	// Step 1: 単一 auth package 内の明示 service file だけを読み、Account/Operator service 間の責務語彙を個別に検査する。
	source, err := productAdminAuthApplicationSourceFiles.ReadFile(path)
	if err != nil {
		t.Fatalf("read embedded auth source %s: %v", path, err)
	}

	// Step 2: AST を comment なしで解析し、実コード内に discriminator / surface switch 語彙がないことを固定する。
	fileSet := token.NewFileSet()
	parsedFile := parseEmbeddedAuthBoundaryFile(t, fileSet, path, source)
	assertASTHasNoForbiddenTerm(t, fileSet, parsedFile, authDomainSwitchSelectorTerms)
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

func assertEmbeddedSourceContains(t *testing.T, path string, requiredTerms []string) {
	t.Helper()

	// Step 1: go:embed された固定 source を読み、検査対象が実行時の外部 path に依存しないようにする。
	source, err := productAdminAuthApplicationSourceFiles.ReadFile(path)
	if err != nil {
		t.Fatalf("read embedded auth source %s: %v", path, err)
	}
	sourceText := string(source)

	// Step 2: 必須語彙が source に存在することを確認し、owner package への委譲が消えた場合に失敗させる。
	for _, term := range requiredTerms {
		if !strings.Contains(sourceText, term) {
			t.Fatalf("%s must contain %q to preserve canonical auth concept ownership", path, term)
		}
	}
}

func assertEmbeddedSourceDoesNotContain(t *testing.T, path string, forbiddenTerms []string) {
	t.Helper()

	// Step 1: go:embed された固定 source を読み、作業ツリー外の内容で判定しないようにする。
	source, err := productAdminAuthApplicationSourceFiles.ReadFile(path)
	if err != nil {
		t.Fatalf("read embedded auth source %s: %v", path, err)
	}
	sourceText := string(source)

	// Step 2: 重複実装の兆候になる語彙が存在しないことを確認し、shared concept からの逸脱を防ぐ。
	for _, term := range forbiddenTerms {
		if strings.Contains(sourceText, term) {
			t.Fatalf("%s must not contain %q because canonical auth concept ownership is elsewhere", path, term)
		}
	}
}

func assertContainerSourceContains(t *testing.T, requiredTerms []string) {
	t.Helper()

	// Step 1: production caller migration は runtime composition file そのものを証跡にするため、固定相対 path の source を読む。
	source, err := os.ReadFile("../../app/product_container.go")
	if err != nil {
		t.Fatalf("read production source ../../app/product_container.go: %v", err)
	}
	sourceText := string(source)

	// Step 2: canonical lifecycle wiring に必要な語彙が欠けた場合、helper/type-only evidence として失敗させる。
	for _, term := range requiredTerms {
		if !strings.Contains(sourceText, term) {
			t.Fatalf("../../app/product_container.go must contain %q to prove production caller migration to canonical lifecycle", term)
		}
	}
}

func assertContainerSourceDoesNotContain(t *testing.T, forbiddenTerms []string) {
	t.Helper()

	// Step 1: old owner/caller absence はコメントではなく production source 文字列から検出し、container の直接 caller 復活を拒否する。
	source, err := os.ReadFile("../../app/product_container.go")
	if err != nil {
		t.Fatalf("read production source ../../app/product_container.go: %v", err)
	}
	sourceText := string(source)

	// Step 2: root TokenService caller が container に残る場合は 10.2 の caller migration 未完了として失敗させる。
	for _, term := range forbiddenTerms {
		if strings.Contains(sourceText, term) {
			t.Fatalf("../../app/product_container.go must not contain %q because production caller must use canonical auth lifecycle", term)
		}
	}
}

func assertBackendProductionASTDoesNotContain(t *testing.T, forbiddenTerms []string) {
	t.Helper()

	// Step 1: backend/internal 配下の production Go source を固定 root から走査し、生成物・test fixture・外部入力を検査対象から外す。
	backendInternalRoot := filepath.Clean("../..")
	err := filepath.WalkDir(backendInternalRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		// Step 2: directory、test、generated 以外の production Go source だけを error vocabulary guardrail の対象にする。
		if entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") || strings.Contains(path, string(filepath.Separator)+"generated"+string(filepath.Separator)) {
			return nil
		}

		// Step 3: Go parser に固定 root 配下の source を読ませ、識別子・literal として旧 surface error 語彙が使われないことを確認する。
		fileSet := token.NewFileSet()
		parsedFile, parseErr := parser.ParseFile(fileSet, path, nil, 0)
		if parseErr != nil {
			return parseErr
		}
		assertASTHasNoForbiddenTerm(t, fileSet, parsedFile, forbiddenTerms)
		return nil
	})
	if err != nil {
		t.Fatalf("walk backend production sources for legacy auth error vocabulary: %v", err)
	}
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

func assertNoProductionGoFilesUnder(t *testing.T, relativeDir string) {
	t.Helper()

	// Step 1: legacy owner directory が存在しない場合は、production owner source がないため成功とする。
	_, err := os.Stat(relativeDir)
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		t.Fatalf("stat legacy auth owner directory %s: %v", relativeDir, err)
	}

	// Step 2: directory が残っている場合は再帰的に走査し、nested source による旧 owner 復活も拒否する。
	walkErr := filepath.WalkDir(relativeDir, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		if strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, "_test.go") {
			t.Fatalf("legacy auth owner directory %s must not contain production Go source %s", relativeDir, path)
		}
		return nil
	})
	if walkErr != nil {
		t.Fatalf("walk legacy auth owner directory %s: %v", relativeDir, walkErr)
	}
}

func assertBackendProductionSourcesDoNotImport(t *testing.T, forbiddenImportPaths []string) {
	t.Helper()

	// Step 1: application package から backend/internal 配下を固定して走査し、runtime 入力ではなく repository source を検査する。
	backendInternalRoot := filepath.Clean("../..")
	err := filepath.WalkDir(backendInternalRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		// Step 2: directory、test、generated 以外の production Go source だけを caller migration の証跡にする。
		if entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") || strings.Contains(path, string(filepath.Separator)+"generated"+string(filepath.Separator)) {
			return nil
		}

		// Step 3: Go import 宣言を AST で解析し、コメントや文字列ではなく production import だけを見る。
		// parser.ParseFile に固定 root 配下の path を渡し、任意入力から file を読む処理をこの test に持ち込まない。
		fileSet := token.NewFileSet()
		file, parseErr := parser.ParseFile(fileSet, path, nil, parser.ImportsOnly)
		if parseErr != nil {
			return parseErr
		}
		for _, importSpec := range file.Imports {
			importPath, unquoteErr := strconv.Unquote(importSpec.Path.Value)
			if unquoteErr != nil {
				return unquoteErr
			}
			for _, forbiddenImportPath := range forbiddenImportPaths {
				if importPath == forbiddenImportPath {
					t.Fatalf("production source %s must not import legacy auth owner %s", path, forbiddenImportPath)
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk backend production sources for legacy auth imports: %v", err)
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

	// Step 1: switch subtree に hosted service または Account/Operator の両側語彙が揃う場合、単一 service の subject 分岐として拒否する。
	if nodeContainsTermPair(switchNode, []string{"product", "account"}, []string{"admin", "operator"}) || nodeContainsForbiddenTerm(switchNode, authDomainSwitchSelectorTerms) {
		t.Fatalf("auth lifecycle operation must not collapse account/operator subject decisions; found at %s", fileSet.Position(switchNode.Pos()))
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
