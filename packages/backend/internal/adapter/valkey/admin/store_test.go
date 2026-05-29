package admin

import (
	"strings"
	"testing"

	"github.com/redis/go-redis/v9"

	"www-template/packages/backend/internal/platform/config"
)

func TestStoreUsesSeparateLogicalDBOnSharedInfrastructure(t *testing.T) {
	t.Parallel()

	// Step 1: Product と Admin が同じ Valkey endpoint を使い、DB path だけを分ける代表 URL を固定する。
	productOptions, err := redis.ParseURL("redis://valkey:6379/0")
	if err != nil {
		t.Fatalf("parse product valkey url: %v", err)
	}
	store, err := NewStore(config.ValkeyConfig{URL: "redis://valkey:6379/1"})
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Step 2: Redis client option に同じ infrastructure address と別 logical DB が保持されることを確認する。
	adminOptions := store.client.Options()
	if adminOptions.Addr != productOptions.Addr {
		t.Fatalf("expected shared Valkey infrastructure %q, got %q", productOptions.Addr, adminOptions.Addr)
	}
	if adminOptions.DB == productOptions.DB {
		t.Fatalf("expected separate logical DB, both use %d", adminOptions.DB)
	}
	if adminOptions.DB != 1 || productOptions.DB != 0 {
		t.Fatalf("expected admin DB 1 and product DB 0, got admin=%d product=%d", adminOptions.DB, productOptions.DB)
	}
}

func TestStoreKeysUseAdminPrefixOnly(t *testing.T) {
	t.Parallel()

	// Step 1: [OpenSpec Task 4.46] admin-prefixed key namespace の追跡点として、誤って Product 用 prefix が渡されても Admin store は key namespace を `admin:*` に固定する。
	store, err := NewStore(config.ValkeyConfig{URL: "redis://valkey:6379/1", KeyPrefix: "product"})
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Step 2: [OpenSpec Task 4.46] admin-prefixed keys only の挙動として、session key が `admin:*` で始まり、Product prefix や環境 prefix を先頭へ混入しないことを検証する。
	key := store.key("auth", "operator-session", "session-1")
	if key != "admin:auth:operator-session:session-1" {
		t.Fatalf("expected admin-only key prefix, got %q", key)
	}
	if strings.HasPrefix(key, "product:") || strings.Contains(key, ":product:") {
		t.Fatalf("admin key must not contain product namespace, got %q", key)
	}
}
