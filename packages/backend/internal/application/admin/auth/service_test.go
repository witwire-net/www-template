package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	tokenprimitive "www-template/packages/backend/internal/application/shared/tokenprimitive"
	domain "www-template/packages/backend/internal/domain"
)

// [AUTH-BE-S061] Admin operator login は Admin operator auth domain を使う。
func TestAuthBES061FinishOperatorPasskeyUsesAdminOperatorAuthDomain(t *testing.T) {
	t.Parallel()

	// Step 1: Admin Operator repository と Admin Operator session store だけを用意し、Product account auth state を持たない login 経路に固定する。
	ctx := context.Background()
	now := time.Unix(1_700_000_000, 0).UTC()
	operator := testOperatorSnapshot()
	repo := &testOperatorRepo{byCredential: map[string]OperatorSnapshot{"credential-a": operator}, byID: map[string]OperatorSnapshot{operator.ID: operator}}
	store := &testOperatorSessionStore{records: map[string]OperatorSessionRecord{}}
	service := mustTestAdminAuthService(t, repo, store, now)

	// Step 2: Admin passkey login 完了 use case を実行し、OperatorAuth domain が発行した session/access/CSRF を受け取る。
	result, err := service.FinishOperatorPasskey(ctx, FinishOperatorPasskeyInput{CredentialHandle: "credential-a"})
	if err != nil {
		t.Fatalf("finish operator passkey: %v", err)
	}

	// Step 3: accessToken、refresh state、response body の Admin-only 境界を小さい検証 helper で固定する。
	assertAdminAccessTokenPayload(t, service, result, operator)
	assertAdminOperatorSessionRecord(t, store, result, operator)
	assertAdminLoginCookieAndBody(t, result)
}

func assertAdminAccessTokenPayload(t *testing.T, service *Service, result OperatorSessionResult, operator OperatorSnapshot) {
	t.Helper()

	// Step 1: accessToken payload は Admin operator claim として署名検証できることを確認する。
	payloadBytes, err := service.signer.VerifyJSON(result.AccessToken)
	if err != nil {
		t.Fatalf("verify admin access token: %v", err)
	}
	var payload operatorAccessTokenPayload
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		t.Fatalf("decode admin access token payload: %v", err)
	}
	if payload.OperatorID != operator.ID || payload.SessionID != result.SessionID || payload.Role != operator.Role || !payload.Active {
		t.Fatalf("expected operator access token payload, got %+v", payload)
	}

	// Step 2: Product account claim が Admin token に混入していないことを raw field map で確認する。
	var payloadFields map[string]json.RawMessage
	if err := json.Unmarshal(payloadBytes, &payloadFields); err != nil {
		t.Fatalf("decode admin access token fields: %v", err)
	}
	for _, forbiddenField := range []string{"status", "accountID", "accountId", "accountStatus"} {
		if _, ok := payloadFields[forbiddenField]; ok {
			t.Fatalf("admin access token must not contain product account field %q: %s", forbiddenField, string(payloadBytes))
		}
	}
}

func assertAdminOperatorSessionRecord(t *testing.T, store *testOperatorSessionStore, result OperatorSessionResult, operator OperatorSnapshot) {
	t.Helper()

	// Step 1: Admin refresh state は OperatorSessionRecord として保存され、operator owner/snapshot/hash だけを保持することを確認する。
	record, ok := store.records[result.SessionID]
	if !ok {
		t.Fatalf("expected admin operator session record for %q", result.SessionID)
	}
	if record.OperatorID != operator.ID || record.RoleSnapshot != operator.Role || !record.ActiveSnapshot || record.RefreshTokenHash == "" || record.CSRFTokenHash == "" {
		t.Fatalf("expected operator refresh state, got %+v", record)
	}

	// Step 2: Product account auth state と混在しないことを record field 名の検査で固定する。
	recordType := reflect.TypeOf(record)
	for _, forbiddenField := range []string{"AccountID", "AccountStatus", "AccountSessionID"} {
		if _, ok := recordType.FieldByName(forbiddenField); ok {
			t.Fatalf("operator refresh record must not contain product account field %q", forbiddenField)
		}
	}
}

func assertAdminLoginCookieAndBody(t *testing.T, result OperatorSessionResult) {
	t.Helper()

	// Step 1: refreshToken 平文は HttpOnly Cookie command に閉じ込められ、Cookie 属性が Admin login 用の値であることを確認する。
	if result.RefreshCookie.Name != adminRefreshCookieName || !result.RefreshCookie.HTTPOnly || !result.RefreshCookie.Secure || result.RefreshCookie.SameSite != "Lax" || result.RefreshCookie.Path != "/" {
		t.Fatalf("unexpected admin login refresh cookie command: %+v", result.RefreshCookie)
	}

	// Step 2: login response body は operator accessToken/CSRF だけを公開し、refresh Cookie command を公開しないことを確認する。
	bodyBytes, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal operator login result body: %v", err)
	}
	body := string(bodyBytes)
	if !strings.Contains(body, result.AccessToken) || !strings.Contains(body, result.CSRFToken) {
		t.Fatalf("operator login body must contain accessToken and CSRF token, got %s", body)
	}
	for _, forbidden := range []string{result.RefreshCookie.Value, "RefreshCookie", "refreshToken"} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("operator login body must not expose %q: %s", forbidden, body)
		}
	}
}

func TestRefreshOperatorSessionRotatesCookieAndOmitsRefreshTokenFromBody(t *testing.T) {
	t.Parallel()

	// Step 1: passkey login 済みの Admin Operator session を作り、refresh rotation の旧 Cookie 入力を得る。
	ctx := context.Background()
	now := time.Unix(1_700_000_000, 0).UTC()
	operator := testOperatorSnapshot()
	repo := &testOperatorRepo{byCredential: map[string]OperatorSnapshot{"credential-a": operator}, byID: map[string]OperatorSnapshot{operator.ID: operator}}
	store := &testOperatorSessionStore{records: map[string]OperatorSessionRecord{}}
	service := mustTestAdminAuthService(t, repo, store, now)
	loginResult, err := service.FinishOperatorPasskey(ctx, FinishOperatorPasskeyInput{CredentialHandle: "credential-a"})
	if err != nil {
		t.Fatalf("finish operator passkey: %v", err)
	}
	oldCookieValue := loginResult.RefreshCookie.Value

	// Step 2: Admin refresh use case を実行し、store の atomic rotation port が旧 session/hash を消費したことを検証できる状態にする。
	refreshResult, err := service.RefreshOperatorSession(ctx, RefreshOperatorSessionInput{RefreshCookieValue: oldCookieValue})
	if err != nil {
		t.Fatalf("refresh operator session: %v", err)
	}

	// Step 3: [OpenSpec Task 4.47] SameSite=Lax cookie behavior の追跡点を含め、rotation store、Cookie command、response body、旧 Cookie 再利用拒否を helper ごとに検証する。
	assertAdminRefreshStoreRotation(t, store, loginResult, refreshResult)
	assertAdminRefreshCookieRotated(t, refreshResult, oldCookieValue)
	assertAdminRefreshBodyHidesCookie(t, refreshResult, oldCookieValue)
	assertConsumedAdminCookieRejected(ctx, t, service, oldCookieValue)
}

func assertAdminRefreshStoreRotation(t *testing.T, store *testOperatorSessionStore, loginResult OperatorSessionResult, refreshResult OperatorSessionResult) {
	t.Helper()

	// Step 1: store の atomic rotation port が旧 session/hash を消費し、新 session を保存したことを確認する。
	if !store.rotateCalled || store.rotatedFromSessionID != loginResult.SessionID || store.rotatedCurrentHash == "" {
		t.Fatalf("operator refresh must rotate through the session store: %+v", store)
	}
	if _, ok := store.records[loginResult.SessionID]; ok {
		t.Fatalf("old operator refresh session must be consumed")
	}
	if _, ok := store.records[refreshResult.SessionID]; !ok {
		t.Fatalf("new operator refresh session must be stored")
	}

	// Step 2: rotation store の検証が終わったことを helper 境界で閉じる。
}

func assertAdminRefreshCookieRotated(t *testing.T, refreshResult OperatorSessionResult, oldCookieValue string) {
	t.Helper()

	// Step 1: [OpenSpec Task 4.47] SameSite=Lax cookie behavior として、新 refreshToken が HttpOnly/Secure/SameSite=Lax Cookie command として発行され、旧 Cookie 値と異なることを検証する。
	if refreshResult.RefreshCookie.Value == "" || refreshResult.RefreshCookie.Value == oldCookieValue {
		t.Fatalf("expected rotated admin refresh cookie, got %q", refreshResult.RefreshCookie.Value)
	}
	if refreshResult.RefreshCookie.Name != adminRefreshCookieName || !refreshResult.RefreshCookie.HTTPOnly || !refreshResult.RefreshCookie.Secure || refreshResult.RefreshCookie.SameSite != "Lax" || refreshResult.RefreshCookie.Path != "/" {
		t.Fatalf("unexpected admin refresh cookie command: %+v", refreshResult.RefreshCookie)
	}
}

func assertAdminRefreshBodyHidesCookie(t *testing.T, refreshResult OperatorSessionResult, oldCookieValue string) {
	t.Helper()

	// Step 1: response body として marshal される result から refreshToken 平文と Cookie command が除外されることを検証する。
	bodyBytes, err := json.Marshal(refreshResult)
	if err != nil {
		t.Fatalf("marshal operator refresh result body: %v", err)
	}
	body := string(bodyBytes)
	for _, forbidden := range []string{oldCookieValue, refreshResult.RefreshCookie.Value, "RefreshCookie", "refreshToken"} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("operator refresh body must not expose %q: %s", forbidden, body)
		}
	}
}

func assertConsumedAdminCookieRejected(ctx context.Context, t *testing.T, service *Service, oldCookieValue string) {
	t.Helper()

	// Step 1: 消費済み旧 Cookie の再利用を拒否し、error に平文 Cookie 値が混入しないことを確認する。
	_, err := service.RefreshOperatorSession(ctx, RefreshOperatorSessionInput{RefreshCookieValue: oldCookieValue})
	if !errors.Is(err, ErrAdminAuthUnauthenticated) {
		t.Fatalf("expected consumed admin refresh token to be rejected, got %v", err)
	}
	if strings.Contains(fmt.Sprint(err), oldCookieValue) {
		t.Fatalf("admin refresh error must not include the plaintext refresh token")
	}
}

type testOperatorRepo struct {
	byCredential map[string]OperatorSnapshot
	byID         map[string]OperatorSnapshot
}

func (r *testOperatorRepo) FindOperatorByCredential(_ context.Context, credentialHandle string) (OperatorSnapshot, error) {
	// Step 1: Admin credential handle に対応する Operator snapshot だけを返す。
	operator, ok := r.byCredential[credentialHandle]
	if !ok {
		return OperatorSnapshot{}, domain.ErrSessionNotFound
	}

	// Step 2: 見つかった snapshot を返す。
	return operator, nil
}

func (r *testOperatorRepo) FindOperatorByID(_ context.Context, operatorID string) (OperatorSnapshot, error) {
	// Step 1: OperatorID に対応する現在 snapshot だけを返す。
	operator, ok := r.byID[operatorID]
	if !ok {
		return OperatorSnapshot{}, domain.ErrSessionNotFound
	}

	// Step 2: 見つかった snapshot を返す。
	return operator, nil
}

type testOperatorSessionStore struct {
	records              map[string]OperatorSessionRecord
	rotateCalled         bool
	rotatedFromSessionID string
	rotatedCurrentHash   string
}

func (s *testOperatorSessionStore) SaveOperatorSession(_ context.Context, record OperatorSessionRecord, _ time.Duration) error {
	// Step 1: Admin Operator session ID を key として保存し、refresh Cookie selector で取得できるようにする。
	s.records[record.SessionID] = record
	return nil
}

func (s *testOperatorSessionStore) GetOperatorSession(_ context.Context, sessionID string) (OperatorSessionRecord, error) {
	// Step 1: session selector に対応する Admin refresh session record を取得する。
	record, ok := s.records[sessionID]
	if !ok {
		return OperatorSessionRecord{}, domain.ErrSessionNotFound
	}

	// Step 2: 見つかった record を返す。
	return record, nil
}

func (s *testOperatorSessionStore) RotateOperatorSession(_ context.Context, sessionID string, currentRefreshTokenHash string, replacement OperatorSessionRecord, _ time.Duration) error {
	// Step 1: rotation port が呼ばれた事実と、旧 session/hash selector を記録する。
	s.rotateCalled = true
	s.rotatedFromSessionID = sessionID
	s.rotatedCurrentHash = currentRefreshTokenHash

	// Step 2: 旧 session が存在し、保存済み hash と一致する場合だけ置換を許可する。
	current, ok := s.records[sessionID]
	if !ok || current.RefreshTokenHash != currentRefreshTokenHash {
		return domain.ErrSessionNotFound
	}

	// Step 3: 旧 session を削除してから新 session を保存し、旧 Cookie の再利用を拒否できる状態にする。
	delete(s.records, sessionID)
	s.records[replacement.SessionID] = replacement
	return nil
}

func (s *testOperatorSessionStore) RevokeOperatorSession(_ context.Context, operatorID string, sessionID string) error {
	// Step 1: logout 対象の owner が一致する場合だけ record を削除する。
	if record, ok := s.records[sessionID]; ok && record.OperatorID == operatorID {
		delete(s.records, sessionID)
	}
	return nil
}

type testAdminSecretGenerator struct {
	tokens []string
	index  int
}

func (g *testAdminSecretGenerator) NewOpaqueToken() (string, error) {
	// Step 1: test が用意した refresh/CSRF token を順番に返し、rotation 結果を deterministic にする。
	if g.index >= len(g.tokens) {
		return "", ErrAdminAuthInternal
	}
	token := g.tokens[g.index]
	g.index++
	return token, nil
}

type testAdminIDGenerator struct {
	ids   []string
	index int
}

func (g *testAdminIDGenerator) Next() (string, error) {
	// Step 1: test が用意した Operator session ID / accessToken JTI を順番に返す。
	if g.index >= len(g.ids) {
		return "", ErrAdminAuthInternal
	}
	id := g.ids[g.index]
	g.index++
	return id, nil
}

func mustTestAdminAuthService(t *testing.T, repo OperatorRepository, store OperatorSessionStore, now time.Time) *Service {
	t.Helper()

	// Step 1: Admin auth 専用 signer と deterministic generator を注入した service を生成する。
	signer, err := tokenprimitive.NewJWTSignVerifier([]byte("admin-auth-test-secret"))
	if err != nil {
		t.Fatalf("create admin signer: %v", err)
	}
	service, err := NewService(
		repo,
		store,
		nil,
		signer,
		&testAdminSecretGenerator{tokens: []string{"old-admin-refresh-secret", "old-admin-csrf-secret", "new-admin-refresh-secret", "new-admin-csrf-secret"}},
		&testAdminIDGenerator{ids: []string{"01ARZ3NDEKTSV4RRFFQ69G5FB1", "01ARZ3NDEKTSV4RRFFQ69G5FB2", "01ARZ3NDEKTSV4RRFFQ69G5FB3", "01ARZ3NDEKTSV4RRFFQ69G5FB4"}},
		func() time.Time { return now },
		AdminAuthConfig{AccessTokenTTL: 15 * time.Minute, RefreshSessionTTL: time.Hour, RefreshCookieLifetime: 30 * time.Minute, WebAuthnRPID: "admin.example.com"},
	)
	if err != nil {
		t.Fatalf("create admin auth service: %v", err)
	}

	// Step 2: 生成済み service を返す。
	return service
}

func testOperatorSnapshot() OperatorSnapshot {
	// Step 1: Admin mutation が可能な registered admin operator の snapshot を返す。
	return OperatorSnapshot{
		ID:                       "01ARZ3NDEKTSV4RRFFQ69G5FAW",
		Email:                    "admin@example.com",
		Role:                     string(domain.OperatorRoleAdmin),
		Active:                   true,
		PasskeyRegistrationState: string(domain.OperatorPasskeyRegistrationRegistered),
	}
}
