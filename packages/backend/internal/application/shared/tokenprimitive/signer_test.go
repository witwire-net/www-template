package application

import (
	"bytes"
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	domain "www-template/packages/backend/internal/domain"
)

// [AUTH-BE-S065] shared tokenprimitive が signer/verifier を中立 interface として合成できることを確認する。
func TestJWTSignVerifierSignsAndVerifiesPayload(t *testing.T) {
	t.Parallel()

	// 固定 secret を使い、同じ helper instance が署名と検証の両方を提供することを確認する。
	secret := []byte("application-shared-tokenprimitive-secret-at-least-32-bytes")
	helper, err := NewJWTSignVerifier(secret)
	if err != nil {
		t.Fatalf("new jwt sign verifier: %v", err)
	}

	// helper を interface として扱い、呼び出し元が具体型に依存しなくてもよいことを確認する。
	var signVerifier JSONSignVerifier = helper
	payload := []byte(`{"jti":"01ARZ3NDEKTSV4RRFFQ69G5FAX"}`)

	// payload を署名済み token に変換する。
	tokenString, err := signVerifier.SignJSON(payload)
	if err != nil {
		t.Fatalf("sign json: %v", err)
	}

	// 署名済み token を検証し、payload がそのまま戻ることを確認する。
	verifiedPayload, err := signVerifier.VerifyJSON(tokenString)
	if err != nil {
		t.Fatalf("verify json: %v", err)
	}
	if !bytes.Equal(verifiedPayload, payload) {
		t.Fatalf("payload mismatch: got %s, want %s", verifiedPayload, payload)
	}
}

// [AUTH-BE-S065] shared tokenprimitive の関数 API が method API と同じ署名検証境界を使うことを確認する。
func TestSignAndVerifyJSONFunctions(t *testing.T) {
	t.Parallel()

	// 一回限りの関数 API で使う secret と payload を用意する。
	secret := []byte("application-shared-tokenprimitive-secret-at-least-32-bytes")
	payload := []byte(`{"jti":"01ARZ3NDEKTSV4RRFFQ69G5FAY"}`)

	// 関数 API で署名し、同じ関数 API で検証する。
	tokenString, err := SignJSON(payload, secret)
	if err != nil {
		t.Fatalf("sign json: %v", err)
	}
	verifiedPayload, err := VerifyJSON(tokenString, secret)
	if err != nil {
		t.Fatalf("verify json: %v", err)
	}

	// 検証済み payload が入力 payload と一致することを確認する。
	if !bytes.Equal(verifiedPayload, payload) {
		t.Fatalf("payload mismatch: got %s, want %s", verifiedPayload, payload)
	}
}

// [AUTH-BE-S065] shared tokenprimitive が secret、payload、署名の不正を domain error として返すことを確認する。
func TestJWTSignVerifierRejectsInvalidInputs(t *testing.T) {
	t.Parallel()

	// 空 secret は helper 生成時点で拒否される。
	_, err := NewJWTSignVerifier(nil)
	if !errors.Is(err, domain.ErrInvalidSecret) {
		t.Fatalf("expected ErrInvalidSecret, got %v", err)
	}

	// 正常 helper を作り、不正 payload と署名不一致の検証に使う。
	secret := []byte("application-shared-tokenprimitive-secret-at-least-32-bytes")
	helper, err := NewJWTSignVerifier(secret)
	if err != nil {
		t.Fatalf("new jwt sign verifier: %v", err)
	}

	// JSON object ではない payload は署名前に拒否される。
	_, err = helper.SignJSON([]byte(`"not-object"`))
	if !errors.Is(err, domain.ErrInvalidTokenPayload) {
		t.Fatalf("expected ErrInvalidTokenPayload, got %v", err)
	}

	// 別 secret で作った token は署名不一致として拒否される。
	tokenString, err := SignJSON([]byte(`{"jti":"01ARZ3NDEKTSV4RRFFQ69G5FAZ"}`), []byte("different-tokenprimitive-secret-at-least-32-bytes"))
	if err != nil {
		t.Fatalf("sign json with different secret: %v", err)
	}
	_, err = helper.VerifyJSON(tokenString)
	if !errors.Is(err, domain.ErrInvalidSignature) {
		t.Fatalf("expected ErrInvalidSignature, got %v", err)
	}
}

// [AUTH-BE-S065] shared tokenprimitive の production code が呼び出し元種別や権限の分岐語彙を持たないことを静的検証する。
func TestSharedTokenPrimitiveHasNoDomainSpecificSwitch(t *testing.T) {
	t.Parallel()

	// この test file の場所から検証対象 directory を特定する。
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve current test file")
	}
	targetDirectory := filepath.Dir(currentFile)

	// production file だけを AST として検査し、test の説明文は対象外にする。
	productionFiles := []string{
		filepath.Join(targetDirectory, "signer.go"),
		filepath.Join(targetDirectory, "ttl.go"),
	}

	// 実装に混ぜてはいけない呼び出し元固有の語彙を定義する。
	forbiddenTerms := []string{"account", "operator", "rbac", "role", "status", "issuer", "audience", "identitydomain"}

	// 各 production file を AST 走査し、識別子・文字列 literal・switch 文を検査する。
	fileSet := token.NewFileSet()
	for _, targetFile := range productionFiles {
		parsedFile, err := parser.ParseFile(fileSet, targetFile, nil, 0)
		if err != nil {
			t.Fatalf("parse shared tokenprimitive file %s: %v", targetFile, err)
		}
		assertNoDomainSpecificTokenPrimitiveNode(t, fileSet, parsedFile, forbiddenTerms)
	}
}

// assertNoDomainSpecificTokenPrimitiveNode は AST に禁止語彙や switch 文が含まれないことを確認する。
func assertNoDomainSpecificTokenPrimitiveNode(t *testing.T, fileSet *token.FileSet, parsedFile *ast.File, forbiddenTerms []string) {
	t.Helper()

	// AST を走査し、comment ではなく実際の識別子・文字列・分岐構文だけを検査する。
	ast.Inspect(parsedFile, func(node ast.Node) bool {
		switch typedNode := node.(type) {
		case *ast.Ident:
			assertNoDomainSpecificTokenPrimitiveTerm(t, "identifier", typedNode.Name, forbiddenTerms)
		case *ast.BasicLit:
			assertNoDomainSpecificTokenPrimitiveTerm(t, "literal", typedNode.Value, forbiddenTerms)
		case *ast.SwitchStmt:
			t.Fatalf("shared tokenprimitive must not contain switch statements; found at %s", fileSet.Position(typedNode.Pos()))
		case *ast.TypeSwitchStmt:
			t.Fatalf("shared tokenprimitive must not contain type switch statements; found at %s", fileSet.Position(typedNode.Pos()))
		}

		return true
	})
}

// assertNoDomainSpecificTokenPrimitiveTerm は文字列に禁止語彙が含まれないことを確認する。
func assertNoDomainSpecificTokenPrimitiveTerm(t *testing.T, kind string, value string, forbiddenTerms []string) {
	t.Helper()

	// 大文字小文字の差で検査がすり抜けないよう lowercase へ正規化する。
	normalizedValue := strings.ToLower(value)

	// すべての禁止語彙について部分一致を調べる。
	for _, term := range forbiddenTerms {
		if strings.Contains(normalizedValue, term) {
			t.Fatalf("shared tokenprimitive contains forbidden %s term %q in %q", kind, term, value)
		}
	}
}
