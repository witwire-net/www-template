package valkey

import (
	"context"
	"sync"
	"testing"
	"time"

	"www-template/packages/backend/internal/auth/domain"
	"www-template/packages/backend/internal/platform/config"
)

// TestConsumeRecoveryTokenAtomicConcurrent は ConsumeRecoveryTokenAtomic が
// 並行リクエストに対して二重消費を許さず、ちょうど 1 件のみ成功することを検証する。
// Lua スクリプトによるアトミックな GET → 検証 → DEL により TOCTOU race condition を防止する。
func TestConsumeRecoveryTokenAtomicConcurrent(t *testing.T) {
	t.Parallel()

	store, err := NewStore(config.ValkeyConfig{URL: "redis://valkey:6379/0"})
	if err != nil {
		t.Skipf("valkey not available: %v", err)
	}
	defer func() { _ = store.Close() }()

	ctx := context.Background()
	if err := store.Ping(ctx); err != nil {
		t.Skipf("valkey unreachable: %v", err)
	}

	// SecretHashKey にはテスト用の固定値を用いる。本番運用では強力なランダム値を用いる。
	const testHashKey = "test-hash-key-32bytes-random-value"
	repo, err := NewAuthStateRepository(store, testHashKey)
	if err != nil {
		t.Fatalf("NewAuthStateRepository: %v", err)
	}

	// テスト用の recovery token を発行する。
	tokenID := "01ARZ3NDEKTSV4RRFFQ69G5FAV"                  // #nosec G101 -- test ULID, not a secret
	secret := "test-random-plain-secret-for-concurrent-test" // #nosec G101 -- test value, not a secret

	// 有効な token を Valkey にセット。
	token, err := domain.NewRecoveryToken(tokenID, "01ARZ3NDEKTSV4RRFFQ69G5FAW", secret, domain.TokenKindRecovery, time.Now().UTC().Add(30*time.Minute))
	if err != nil {
		t.Fatalf("domain.NewRecoveryToken: %v", err)
	}
	if err := repo.IssueRecoveryToken(ctx, token, 30*time.Minute); err != nil {
		t.Fatalf("IssueRecoveryToken: %v", err)
	}

	// 並行 goroutine 数を設定。
	const concurrency = 20
	var wg sync.WaitGroup
	successes := make(chan struct{}, concurrency)

	// 並行して ConsumeRecoveryTokenAtomic を呼び出す。
	for range concurrency {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := repo.ConsumeRecoveryTokenAtomic(ctx, tokenID, secret)
			if err == nil {
				successes <- struct{}{}
			}
		}()
	}
	wg.Wait()
	close(successes)

	// 成功数がちょうど 1 であることを検証する。
	successCount := len(successes)
	if successCount != 1 {
		t.Fatalf("expected exactly 1 successful atomic consume, got %d (possible TOCTOU race or double-consume)", successCount)
	}
}
