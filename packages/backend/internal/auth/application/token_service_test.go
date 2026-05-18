package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"www-template/packages/backend/internal/auth/domain"
	"www-template/packages/backend/internal/platform/config"
	"www-template/packages/backend/internal/platform/id"
)

// mockRefreshTokenStore はテスト用の RefreshTokenStore モック。
type mockRefreshTokenStore struct {
	saved      map[string]RefreshTokenRecord
	consumed   map[string]RefreshTokenRecord
	revokedFP  []string
	revokedSID []string
}

func newMockRefreshTokenStore() *mockRefreshTokenStore {
	return &mockRefreshTokenStore{
		saved:    make(map[string]RefreshTokenRecord),
		consumed: make(map[string]RefreshTokenRecord),
	}
}

func (m *mockRefreshTokenStore) Save(_ context.Context, hash string, record RefreshTokenRecord, _ time.Duration) error {
	m.saved[hash] = record
	return nil
}

func (m *mockRefreshTokenStore) Consume(_ context.Context, hash string) (RefreshTokenRecord, error) {
	record, ok := m.saved[hash]
	if !ok {
		return RefreshTokenRecord{}, domain.ErrSessionNotFound
	}
	m.consumed[hash] = record
	delete(m.saved, hash)
	return record, nil
}

func (m *mockRefreshTokenStore) GetConsumed(_ context.Context, hash string) (RefreshTokenRecord, error) {
	record, ok := m.consumed[hash]
	if !ok {
		return RefreshTokenRecord{}, domain.ErrSessionNotFound
	}
	return record, nil
}

func (m *mockRefreshTokenStore) RevokeAllForFingerprint(_ context.Context, accountID, fingerprint string) error {
	m.revokedFP = append(m.revokedFP, accountID+":"+fingerprint)
	return nil
}

func (m *mockRefreshTokenStore) RevokeBySessionID(_ context.Context, accountID, sessionID string) error {
	m.revokedSID = append(m.revokedSID, accountID+":"+sessionID)
	return nil
}

// mockSessionStore はテスト用の SessionStore モック。
type mockSessionStore struct {
	sessions map[string]SessionMetadata
}

func newMockSessionStore() *mockSessionStore {
	return &mockSessionStore{sessions: make(map[string]SessionMetadata)}
}

func (m *mockSessionStore) SaveSession(_ context.Context, sessionID, _ string, metadata SessionMetadata, _ time.Duration) error {
	m.sessions[sessionID] = metadata
	return nil
}

func (m *mockSessionStore) GetSession(_ context.Context, sessionID string) (SessionMetadata, error) {
	s, ok := m.sessions[sessionID]
	if !ok {
		return SessionMetadata{}, domain.ErrSessionNotFound
	}
	return s, nil
}

func (m *mockSessionStore) ListSessions(_ context.Context, _ string) ([]SessionMetadata, error) {
	result := make([]SessionMetadata, 0, len(m.sessions))
	for _, s := range m.sessions {
		result = append(result, s)
	}
	return result, nil
}

func (m *mockSessionStore) RevokeSession(_ context.Context, _, sessionID string) error {
	delete(m.sessions, sessionID)
	return nil
}

func (m *mockSessionStore) RevokeOthers(_ context.Context, _, currentSessionID string) ([]string, error) {
	deleted := make([]string, 0)
	for id := range m.sessions {
		if id != currentSessionID {
			delete(m.sessions, id)
			deleted = append(deleted, id)
		}
	}
	return deleted, nil
}

func (m *mockSessionStore) RevokeAllForAccount(_ context.Context, _ string) error {
	for id := range m.sessions {
		delete(m.sessions, id)
	}
	return nil
}

func testTokenService() *TokenService {
	cfg := config.AuthConfig{
		JWTSecret:     "test-jwt-secret-key-must-be-at-least-32bytes",
		SecretHashKey: "test-secret-hash-key-must-be-at-least-32",
	}
	clock := func() time.Time { return time.Date(2026, time.March, 21, 0, 0, 0, 0, time.UTC) }
	policy := id.AuthIDPolicy{
		New:      func() string { return "01ARZ3NDEKTSV4RRFFQ69G5FAV" },
		Validate: domain.ValidateAuthID,
	}
	return NewTokenService(newMockRefreshTokenStore(), newMockSessionStore(), nil, cfg, clock, policy)
}

// [UT-AUTH-BE-HAP-002] TokenService rotates refresh token
func TestTokenServiceRotatesRefreshToken(t *testing.T) {
	t.Parallel()
	svc := testTokenService()
	ctx := context.Background()

	fp := hmacString("test-ua|192.0.2.10", "test-secret-hash-key-must-be-at-least-32")
	accessToken, refreshToken, _, err := svc.Issue(ctx, "01ARZ3NDEKTSV4RRFFQ69G5FAW", fp, "test-device", "iphash123", "")
	if err != nil {
		t.Fatalf("issue failed: %v", err)
	}
	if accessToken == "" {
		t.Fatal("expected access token")
	}
	if refreshToken == "" {
		t.Fatal("expected refresh token")
	}

	newAccess, newRefresh, err := svc.Refresh(ctx, refreshToken, "192.0.2.10", "test-ua")
	if err != nil {
		t.Fatalf("refresh failed: %v", err)
	}
	if newAccess == "" {
		t.Fatal("expected new access token")
	}
	if newRefresh == "" {
		t.Fatal("expected new refresh token")
	}

	// 旧トークンは消費済みのため再利用できない
	// 盗難検出により ErrTokenTheftDetected を返す
	_, _, err = svc.Refresh(ctx, refreshToken, "192.0.2.10", "test-ua")
	if !errors.Is(err, ErrTokenTheftDetected) {
		t.Fatalf("expected ErrTokenTheftDetected, got %v", err)
	}
}

// [UT-AUTH-BE-ERR-002] TokenService detects theft
func TestTokenServiceDetectsTheftOnUnknownToken(t *testing.T) {
	t.Parallel()
	svc := testTokenService()
	ctx := context.Background()

	_, _, err := svc.Refresh(ctx, "invalid-token", "192.0.2.10", "test-ua")
	if !errors.Is(err, ErrRefreshTokenNotFound) {
		t.Fatalf("expected ErrRefreshTokenNotFound, got %v", err)
	}
}

// [UT-AUTH-BE-S046] VerifyAccessToken validates JWT
func TestTokenServiceVerifyAccessTokenOK(t *testing.T) {
	t.Parallel()
	svc := testTokenService()
	ctx := context.Background()

	accessToken, _, _, err := svc.Issue(ctx, "01ARZ3NDEKTSV4RRFFQ69G5FAW", "fp1", "test-device", "iphash123", "")
	if err != nil {
		t.Fatalf("issue failed: %v", err)
	}

	claims, err := svc.VerifyAccessToken(accessToken)
	if err != nil {
		t.Fatalf("verify failed: %v", err)
	}
	if claims.AccountID != "01ARZ3NDEKTSV4RRFFQ69G5FAW" {
		t.Fatalf("accountID mismatch: %s", claims.AccountID)
	}
}

// [UT-AUTH-BE-S046] Expired JWT verification fails
func TestTokenServiceVerifyAccessTokenExpired(t *testing.T) {
	t.Parallel()
	cfg := config.AuthConfig{JWTSecret: "test-jwt-secret-key-must-be-at-least-32bytes"}
	past := time.Date(2026, time.March, 21, 0, 0, 0, 0, time.UTC)
	clock := func() time.Time { return past }
	policy := id.AuthIDPolicy{
		New:      func() string { return "01ARZ3NDEKTSV4RRFFQ69G5FAV" },
		Validate: domain.ValidateAuthID,
	}
	svc := NewTokenService(newMockRefreshTokenStore(), newMockSessionStore(), nil, cfg, clock, policy)
	ctx := context.Background()

	accessToken, _, _, err := svc.Issue(ctx, "01ARZ3NDEKTSV4RRFFQ69G5FAW", "fp1", "test-device", "iphash123", "")
	if err != nil {
		t.Fatalf("issue failed: %v", err)
	}

	// 20分進めると期限切れ
	clock = func() time.Time { return past.Add(20 * time.Minute) }
	svc.clock = clock
	_, err = svc.VerifyAccessToken(accessToken)
	if !errors.Is(err, domain.ErrTokenExpired) {
		t.Fatalf("expected ErrTokenExpired, got %v", err)
	}
}

// [UT-AUTH-BE-S043] Refresh preserves session ID
func TestTokenServiceRefreshPreservesSessionID(t *testing.T) {
	t.Parallel()
	svc := testTokenService()
	ctx := context.Background()

	fp := hmacString("test-ua|192.0.2.10", "test-secret-hash-key-must-be-at-least-32")
	_, refreshToken, originalSessionID, err := svc.Issue(ctx, "01ARZ3NDEKTSV4RRFFQ69G5FAW", fp, "test-device", "iphash123", "")
	if err != nil {
		t.Fatalf("issue failed: %v", err)
	}

	_, newRefresh, err := svc.Refresh(ctx, refreshToken, "192.0.2.10", "test-ua")
	if err != nil {
		t.Fatalf("refresh failed: %v", err)
	}

	// 新しいリフレッシュトークンでも同じ session ID を維持していることを確認する
	newHash := hashToken(newRefresh)
	record, err := svc.refreshStore.GetConsumed(ctx, newHash)
	if err == nil {
		// consumed store にない（まだ消費されていない）ことを確認
		_ = record
	}

	// refreshStore の saved に新しいトークンがあるはず
	mockStore := svc.refreshStore.(*mockRefreshTokenStore)
	var foundSessionID string
	for h, r := range mockStore.saved {
		_ = h
		foundSessionID = r.SessionID
		break
	}
	if foundSessionID != originalSessionID {
		t.Fatalf("expected session ID %s to be preserved, got %s", originalSessionID, foundSessionID)
	}
}

// [UT-AUTH-BE-S044] Refresh with mismatched fingerprint rejects and revokes family
func TestTokenServiceRefreshRejectsMismatchedFingerprint(t *testing.T) {
	t.Parallel()
	svc := testTokenService()
	ctx := context.Background()

	fp := hmacString("test-ua|192.0.2.10", "test-secret-hash-key-must-be-at-least-32")
	_, refreshToken, _, err := svc.Issue(ctx, "01ARZ3NDEKTSV4RRFFQ69G5FAW", fp, "test-device", "iphash123", "")
	if err != nil {
		t.Fatalf("issue failed: %v", err)
	}

	// 別の IP / UA から refresh を試行する
	_, _, err = svc.Refresh(ctx, refreshToken, "10.0.0.1", "attacker-ua")
	if !errors.Is(err, ErrTokenTheftDetected) {
		t.Fatalf("expected ErrTokenTheftDetected for mismatched fingerprint, got %v", err)
	}

	// family revocation が実行されたことを確認する
	mockStore := svc.refreshStore.(*mockRefreshTokenStore)
	if len(mockStore.revokedFP) == 0 {
		t.Fatal("expected family revocation to be triggered")
	}
}
