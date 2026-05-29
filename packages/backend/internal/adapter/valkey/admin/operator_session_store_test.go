package admin

import (
	"context"
	"errors"
	"testing"
	"time"

	adminauth "www-template/packages/backend/internal/application/admin/auth"
	domain "www-template/packages/backend/internal/domain"
	"www-template/packages/backend/internal/platform/config"
)

func TestOperatorSessionStoreRotateConsumesOldRefreshSession(t *testing.T) {
	// Step 1: Valkey integration test はローカル service が無い環境でも server suite を止めないよう、接続不能時は skip する。
	store, err := NewStore(config.ValkeyConfig{URL: "redis://valkey:6379/1"})
	if err != nil {
		t.Skipf("admin valkey not available: %v", err)
	}
	defer func() { _ = store.Close() }()

	ctx := context.Background()
	if err := store.Ping(ctx); err != nil {
		t.Skipf("admin valkey unreachable: %v", err)
	}

	// Step 2: 旧 session と置換後 session を固定値で用意し、同じ旧 refresh hash の再利用可否だけを検証する。
	sessions := NewOperatorSessionStore(store)
	oldRecord := testOperatorSessionRecord("01ARZ3NDEKTSV4RRFFQ69G5FAV", "old-refresh-hash")
	nextRecord := testOperatorSessionRecord("01ARZ3NDEKTSV4RRFFQ69G5FAW", "next-refresh-hash")
	replayReplacement := testOperatorSessionRecord("01ARZ3NDEKTSV4RRFFQ69G5FAX", "replay-refresh-hash")
	cleanupOperatorSessionKeys(ctx, sessions, oldRecord, nextRecord, replayReplacement)
	defer cleanupOperatorSessionKeys(ctx, sessions, oldRecord, nextRecord, replayReplacement)
	if err := sessions.SaveOperatorSession(ctx, oldRecord, time.Minute); err != nil {
		t.Fatalf("SaveOperatorSession: %v", err)
	}

	// Step 3: 1 回目の rotation は成功し、旧 session key から次 session key へ置換される。
	if err := sessions.RotateOperatorSession(ctx, oldRecord.SessionID, oldRecord.RefreshTokenHash, nextRecord, time.Minute); err != nil {
		t.Fatalf("RotateOperatorSession first use: %v", err)
	}
	if _, err := sessions.GetOperatorSession(ctx, oldRecord.SessionID); !errors.Is(err, domain.ErrSessionNotFound) {
		t.Fatalf("old session must be consumed, got %v", err)
	}
	if _, err := sessions.GetOperatorSession(ctx, nextRecord.SessionID); err != nil {
		t.Fatalf("replacement session must exist: %v", err)
	}

	// Step 4: 同じ旧 refresh hash を再利用した 2 回目の rotation は拒否され、replay 防止境界を確認できる。
	err = sessions.RotateOperatorSession(ctx, oldRecord.SessionID, oldRecord.RefreshTokenHash, replayReplacement, time.Minute)
	if !errors.Is(err, domain.ErrSessionNotFound) {
		t.Fatalf("old refresh session replay must fail with ErrSessionNotFound, got %v", err)
	}
}

func cleanupOperatorSessionKeys(ctx context.Context, sessions *OperatorSessionStore, records ...adminauth.OperatorSessionRecord) {
	// Step 1: 固定 ID の integration test が Admin logical DB に残した session と index を削除し、次回実行へ影響させない。
	for _, record := range records {
		_ = sessions.store.client.Del(ctx, sessions.sessionKey(record.SessionID), sessions.operatorIndexKey(record.OperatorID)).Err()
	}
}

func testOperatorSessionRecord(sessionID string, refreshHash string) adminauth.OperatorSessionRecord {
	// Step 1: Admin Operator session store が必要とする application DTO を最小の固定値で構築する。
	now := time.Date(2026, 5, 26, 0, 0, 0, 0, time.UTC)
	return adminauth.OperatorSessionRecord{
		SessionID:        sessionID,
		OperatorID:       "01BRZ3NDEKTSV4RRFFQ69G5FAV",
		RefreshTokenHash: refreshHash,
		CSRFTokenHash:    "csrf-hash",
		RoleSnapshot:     "admin",
		ActiveSnapshot:   true,
		IssuedAt:         now,
		ExpiresAt:        now.Add(time.Hour),
		Revoked:          false,
	}
}
