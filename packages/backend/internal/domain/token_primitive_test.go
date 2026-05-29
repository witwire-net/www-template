package domain

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
	"time"
)

// [AUTH-BE-S068] Neutral token primitive が HS256 JWT payload を意味づけせず署名・検証できることを検証する。
func TestNeutralTokenPrimitiveSignsAndVerifiesJWT(t *testing.T) {
	t.Parallel()

	// 固定 secret を使い、署名・検証の両方が同じ primitive 境界で完結することを確認する。
	secret := []byte("neutral-token-primitive-secret-at-least-32-bytes")

	// payload は上位層が意味づける JSON object として渡し、この primitive は claim の意味を解釈しない。
	payload := []byte(`{"sub":"01ARZ3NDEKTSV4RRFFQ69G5FAV","jti":"01ARZ3NDEKTSV4RRFFQ69G5FAX"}`)

	// JWT を生成し、compact serialization の文字列を受け取る。
	tokenString, err := SignTokenJWT(payload, secret)
	if err != nil {
		t.Fatalf("sign token jwt: %v", err)
	}

	// 同じ secret で検証し、payload がそのまま戻ることを確認する。
	verifiedPayload, err := VerifyTokenJWT(tokenString, secret)
	if err != nil {
		t.Fatalf("verify token jwt: %v", err)
	}
	if !bytes.Equal(verifiedPayload, payload) {
		t.Fatalf("payload mismatch: got %s, want %s", verifiedPayload, payload)
	}
}

// [AUTH-BE-S068] Neutral token primitive が JWT 署名不一致と不正 payload を拒否することを検証する。
func TestNeutralTokenPrimitiveRejectsInvalidJWTInputs(t *testing.T) {
	t.Parallel()

	// 正常 token を作るための secret と、検証失敗を起こす別 secret を用意する。
	secret := []byte("neutral-token-primitive-secret-at-least-32-bytes")
	wrongSecret := []byte("neutral-token-primitive-wrong-secret-32-bytes")
	payload := []byte(`{"jti":"01ARZ3NDEKTSV4RRFFQ69G5FAX"}`)

	// 署名済み token を作成し、別 secret では検証できないことを確認する。
	tokenString, err := SignTokenJWT(payload, secret)
	if err != nil {
		t.Fatalf("sign token jwt: %v", err)
	}
	_, err = VerifyTokenJWT(tokenString, wrongSecret)
	if !errors.Is(err, ErrInvalidSignature) {
		t.Fatalf("expected ErrInvalidSignature, got %v", err)
	}

	// JSON object ではない payload は署名前に拒否されることを確認する。
	_, err = SignTokenJWT([]byte(`"not-object"`), secret)
	if !errors.Is(err, ErrInvalidTokenPayload) {
		t.Fatalf("expected ErrInvalidTokenPayload, got %v", err)
	}
}

// [AUTH-BE-S068] Neutral token primitive が opaque token を hash 化し、平文比較を constant-time 境界へ寄せることを検証する。
func TestNeutralTokenPrimitiveHashesOpaqueToken(t *testing.T) {
	t.Parallel()

	// 平文 token を保存用 hash へ変換する。
	hash, err := HashOpaqueToken("opaque-secret-value")
	if err != nil {
		t.Fatalf("hash opaque token: %v", err)
	}

	// 同じ平文 token は hash と一致する。
	if !hash.Matches("opaque-secret-value") {
		t.Fatal("expected opaque token hash to match original token")
	}

	// 異なる平文 token は一致しない。
	if hash.Matches("different-secret-value") {
		t.Fatal("expected opaque token hash to reject different token")
	}

	// 空 token は保存用 hash として拒否される。
	_, err = HashOpaqueToken("   ")
	if !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken, got %v", err)
	}
}

// [AUTH-BE-S068] Neutral token primitive が ULID と JTI の形式だけを検証することを確認する。
func TestNeutralTokenPrimitiveValidatesULIDAndJTI(t *testing.T) {
	t.Parallel()

	// token 系 ULID は前後空白を除去したうえで検証済み値になる。
	id, err := NewTokenULID(" 01ARZ3NDEKTSV4RRFFQ69G5FAV ")
	if err != nil {
		t.Fatalf("new token ulid: %v", err)
	}
	if id.String() != "01ARZ3NDEKTSV4RRFFQ69G5FAV" {
		t.Fatalf("unexpected token ulid: %s", id.String())
	}

	// jti も同じ ULID 規則で検証される。
	jti, err := NewTokenJTI("01ARZ3NDEKTSV4RRFFQ69G5FAX")
	if err != nil {
		t.Fatalf("new token jti: %v", err)
	}
	if jti.String() != "01ARZ3NDEKTSV4RRFFQ69G5FAX" {
		t.Fatalf("unexpected token jti: %s", jti.String())
	}

	// ULID でない値は既存の auth id error で拒否される。
	_, err = NewTokenJTI("not-a-ulid")
	if !errors.Is(err, ErrInvalidAuthID) {
		t.Fatalf("expected ErrInvalidAuthID, got %v", err)
	}
}

// [AUTH-BE-S068] Neutral token primitive が TTL と cookie lifetime の大小関係だけを検証することを確認する。
func TestNeutralTokenPrimitiveValidatesTTL(t *testing.T) {
	t.Parallel()

	// 正の TTL は value object として生成できる。
	ttl, err := ValidateTokenTTL(15 * time.Minute)
	if err != nil {
		t.Fatalf("validate token ttl: %v", err)
	}
	if ttl.Duration() != 15*time.Minute {
		t.Fatalf("unexpected ttl: %s", ttl.Duration())
	}

	// 失効時刻は呼び出し側が渡した発行時刻だけから決まる。
	issuedAt := time.Date(2026, 5, 24, 12, 0, 0, 0, time.FixedZone("JST", 9*60*60))
	expiresAt := ttl.ExpiresAt(issuedAt)
	if !expiresAt.Equal(issuedAt.UTC().Add(15 * time.Minute)) {
		t.Fatalf("unexpected expiresAt: %s", expiresAt)
	}

	// cookie lifetime が TTL 以下なら許可される。
	if err := ValidateTokenCookieLifetime(10*time.Minute, ttl); err != nil {
		t.Fatalf("validate token cookie lifetime: %v", err)
	}

	// 0 以下の TTL は拒否される。
	_, err = ValidateTokenTTL(0)
	if !errors.Is(err, ErrInvalidTokenTTL) {
		t.Fatalf("expected ErrInvalidTokenTTL, got %v", err)
	}

	// TTL を超える cookie lifetime は拒否される。
	err = ValidateTokenCookieLifetime(20*time.Minute, ttl)
	if !errors.Is(err, ErrInvalidTokenCookieLifetime) {
		t.Fatalf("expected ErrInvalidTokenCookieLifetime, got %v", err)
	}
}

// [AUTH-BE-S068] Neutral token primitive が利用者種別の切替語彙を識別子・文字列・switch に持たないことを静的検証する。
func TestNeutralTokenPrimitiveHasNoAccountOperatorDomainSwitch(t *testing.T) {
	t.Parallel()

	// この test file の場所から検証対象の primitive file を特定する。
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve current test file")
	}
	targetFile := filepath.Join(filepath.Dir(currentFile), "token_primitive.go")

	// Go parser で AST 化し、comment ではなく実際の識別子・文字列・構文だけを確認する。
	fileSet := token.NewFileSet()
	parsedFile, err := parser.ParseFile(fileSet, targetFile, nil, 0)
	if err != nil {
		t.Fatalf("parse token primitive: %v", err)
	}

	// domain 切替や権限・状態判定を示す語彙が primitive の実装に入っていないことを検証する。
	forbiddenTerms := []string{"account", "operator", "rbac", "role", "status", "issuer", "audience", "identitydomain"}

	// AST を走査し、識別子・文字列 literal・switch 文を禁止対象として検査する。
	ast.Inspect(parsedFile, func(node ast.Node) bool {
		switch typedNode := node.(type) {
		case *ast.Ident:
			assertNoForbiddenTokenPrimitiveTerm(t, "identifier", typedNode.Name, forbiddenTerms)
		case *ast.BasicLit:
			assertNoForbiddenTokenPrimitiveTerm(t, "literal", typedNode.Value, forbiddenTerms)
		case *ast.SwitchStmt:
			t.Fatalf("token_primitive.go must not contain switch statements; found at %s", fileSet.Position(typedNode.Pos()))
		case *ast.TypeSwitchStmt:
			t.Fatalf("token_primitive.go must not contain type switch statements; found at %s", fileSet.Position(typedNode.Pos()))
		}

		return true
	})
}

// assertNoForbiddenTokenPrimitiveTerm は静的検査対象の文字列に禁止語彙が含まれないことを確認する。
func assertNoForbiddenTokenPrimitiveTerm(t *testing.T, kind string, value string, forbiddenTerms []string) {
	t.Helper()

	// 大文字小文字の違いで禁止語彙がすり抜けないよう lowercase へ正規化する。
	normalizedValue := strings.ToLower(value)

	// すべての禁止語彙について部分一致を調べる。
	for _, term := range forbiddenTerms {
		if strings.Contains(normalizedValue, term) {
			t.Fatalf("token_primitive.go contains forbidden %s term %q in %q", kind, term, value)
		}
	}
}
