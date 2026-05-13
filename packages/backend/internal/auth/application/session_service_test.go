package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"www-template/packages/backend/internal/auth/domain"
)

func testSessionService() (*SessionService, *mockSessionStore, *mockRefreshTokenStore) {
	sessStore := newMockSessionStore()
	refStore := newMockRefreshTokenStore()
	return NewSessionService(sessStore, refStore), sessStore, refStore
}

// [UT-AUTH-BE-S047] SessionStore lists sessions with metadata
func TestSessionServiceListReturnsSessions(t *testing.T) {
	t.Parallel()
	svc, store, _ := testSessionService()
	ctx := context.Background()

	_ = store.SaveSession(ctx, "s1", "a1", SessionMetadata{SessionID: "s1", AccountID: "a1", DeviceName: "dev1", LoginAt: time.Now().UTC()}, 0)
	_ = store.SaveSession(ctx, "s2", "a1", SessionMetadata{SessionID: "s2", AccountID: "a1", DeviceName: "dev2", LoginAt: time.Now().UTC()}, 0)

	sessions, err := svc.List(ctx, "a1")
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if len(sessions) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(sessions))
	}
}

// [UT-AUTH-BE-S048] SessionStore revokes specific session
func TestSessionServiceRevokeSpecificSession(t *testing.T) {
	t.Parallel()
	svc, store, refStore := testSessionService()
	ctx := context.Background()

	_ = store.SaveSession(ctx, "s1", "a1", SessionMetadata{SessionID: "s1", AccountID: "a1"}, 0)
	_ = refStore.Save(ctx, "h1", RefreshTokenRecord{AccountID: "a1", SessionID: "s1"}, 0)

	if err := svc.Revoke(ctx, "a1", "s1"); err != nil {
		t.Fatalf("revoke failed: %v", err)
	}

	_, err := store.GetSession(ctx, "s1")
	if !errors.Is(err, domain.ErrSessionNotFound) {
		t.Fatalf("expected session removed, got %v", err)
	}
}

// [UT-AUTH-BE-S048] Revoking another account's session is forbidden
func TestSessionServiceRevokeOtherAccountForbidden(t *testing.T) {
	t.Parallel()
	svc, store, _ := testSessionService()
	ctx := context.Background()

	_ = store.SaveSession(ctx, "s1", "a2", SessionMetadata{SessionID: "s1", AccountID: "a2"}, 0)

	err := svc.Revoke(ctx, "a1", "s1")
	if !errors.Is(err, ErrBadRequest) {
		t.Fatalf("expected ErrBadRequest, got %v", err)
	}
}

// [UT-AUTH-BE-S049] SessionStore revokes others
func TestSessionServiceRevokeOthers(t *testing.T) {
	t.Parallel()
	svc, store, refStore := testSessionService()
	ctx := context.Background()

	_ = store.SaveSession(ctx, "s1", "a1", SessionMetadata{SessionID: "s1", AccountID: "a1"}, 0)
	_ = store.SaveSession(ctx, "s2", "a1", SessionMetadata{SessionID: "s2", AccountID: "a1"}, 0)
	_ = store.SaveSession(ctx, "s3", "a1", SessionMetadata{SessionID: "s3", AccountID: "a1"}, 0)
	_ = refStore.Save(ctx, "h2", RefreshTokenRecord{AccountID: "a1", SessionID: "s2"}, 0)
	_ = refStore.Save(ctx, "h3", RefreshTokenRecord{AccountID: "a1", SessionID: "s3"}, 0)

	if err := svc.RevokeOthers(ctx, "a1", "s2"); err != nil {
		t.Fatalf("revoke others failed: %v", err)
	}

	_, err := store.GetSession(ctx, "s1")
	if !errors.Is(err, domain.ErrSessionNotFound) {
		t.Fatalf("expected s1 removed")
	}
	_, err = store.GetSession(ctx, "s2")
	if err != nil {
		t.Fatalf("expected s2 retained, got %v", err)
	}
	_, err = store.GetSession(ctx, "s3")
	if !errors.Is(err, domain.ErrSessionNotFound) {
		t.Fatalf("expected s3 removed")
	}
}
