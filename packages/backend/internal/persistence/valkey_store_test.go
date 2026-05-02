package persistence

import (
	"context"
	"errors"
	"testing"
	"time"

	"www-template/packages/backend/internal/types"
)

func TestValkeyStoreGetDelReturnsValueAndDeletesKey(t *testing.T) {
	t.Parallel()

	store, err := NewValkeyStore(types.ValkeyConfig{URL: "redis://valkey:6379/0"})
	if err != nil {
		t.Skipf("valkey not available: %v", err)
	}
	defer func() { _ = store.Close() }()

	ctx := context.Background()
	if err := store.Ping(ctx); err != nil {
		t.Skipf("valkey unreachable: %v", err)
	}

	key := store.Key("test", "getdel", "ok")
	if err := store.Set(ctx, key, "hello", 5*time.Second); err != nil {
		t.Fatalf("set: %v", err)
	}

	// GetDel で値を取得しつつアトミックに削除する
	val, err := store.GetDel(ctx, key)
	if err != nil {
		t.Fatalf("getdel: %v", err)
	}
	if val != "hello" {
		t.Fatalf("expected hello, got %s", val)
	}

	// キーが削除されていることを確認
	_, err = store.Get(ctx, key)
	if !errors.Is(err, errRESPNil) {
		t.Fatalf("expected errRESPNil after GetDel, got %v", err)
	}
}

func TestValkeyStoreGetDelReturnsNilForMissingKey(t *testing.T) {
	t.Parallel()

	store, err := NewValkeyStore(types.ValkeyConfig{URL: "redis://valkey:6379/0"})
	if err != nil {
		t.Skipf("valkey not available: %v", err)
	}
	defer func() { _ = store.Close() }()

	ctx := context.Background()
	if err := store.Ping(ctx); err != nil {
		t.Skipf("valkey unreachable: %v", err)
	}

	key := store.Key("test", "getdel", "missing")

	_, err = store.GetDel(ctx, key)
	if !errors.Is(err, errRESPNil) {
		t.Fatalf("expected errRESPNil for missing key, got %v", err)
	}
}
