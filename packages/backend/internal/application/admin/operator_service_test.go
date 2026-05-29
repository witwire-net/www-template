package application

import (
	"testing"

	secrethash "www-template/packages/backend/internal/platform/secret"
)

func TestAdminSecretHashUsesBcrypt(t *testing.T) {
	t.Parallel()

	// Step 1: setup token / bootstrap secret として使う平文を bcrypt hash 化し、保存値が平文と一致しないことを確認する。
	hash, err := adminHashSecret("dev-admin-bootstrap-secret")
	if err != nil {
		t.Fatalf("hash admin secret: %v", err)
	}
	if hash == "" || hash == "dev-admin-bootstrap-secret" {
		t.Fatalf("expected non-empty bcrypt hash, got %q", hash)
	}

	// Step 2: platform の bcrypt 形式検査が通る値だけを保存値として扱い、Base64URL digest のような高速 hash へ退行しないことを固定する。
	if !secrethash.IsBcryptHash(hash) {
		t.Fatalf("expected bcrypt hash, got %q", hash)
	}

	// Step 3: 照合は bcrypt.CompareHashAndPassword 経由で成功し、前後空白だけは copy/paste 揺れとして吸収する。
	if !adminSecretMatchesHash(hash, "  dev-admin-bootstrap-secret  ") {
		t.Fatal("expected bcrypt hash to match trimmed secret")
	}
}

func TestAdminSecretMatchRejectsFastDigestHash(t *testing.T) {
	t.Parallel()

	// Step 1: 以前誤って使った Base64URL SHA-256 digest は bcrypt 形式ではないため、同じ平文でも必ず拒否する。
	fastDigestHash := "cGFIBQC2yFy4n7fRpQS_RGruxzrq5UwXJpkxlyLj1QQ"
	if adminSecretMatchesHash(fastDigestHash, "dev-admin-bootstrap-secret") {
		t.Fatal("expected non-bcrypt digest hash to be rejected")
	}
}
