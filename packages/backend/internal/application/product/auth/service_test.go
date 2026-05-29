package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	tokenprimitive "www-template/packages/backend/internal/application/shared/tokenprimitive"
	domain "www-template/packages/backend/internal/domain"
)

// [AUTH-BE-S060] Product passkey login は accessToken body と refreshToken Cookie を返す。
func TestAuthBES060LoginWithPasskeyReturnsAccessTokenBodyAndRefreshTokenCookie(t *testing.T) {
	t.Parallel()

	// Step 1: Product AccountAuth projection と空の Product session stores を用意し、passkey login の発行先を Product 境界だけに固定する。
	ctx := context.Background()
	now := time.Unix(1_700_000_000, 0).UTC()
	accountAuth := mustTestAccountAuth(t, testProductAccountID(t, "01ARZ3NDEKTSV4RRFFQ69G5FAW"), "user@example.com", "01ARZ3NDEKTSV4RRFFQ69G5FAX", "credential-a")
	refreshStore := &testRefreshStore{}
	sessionStore := &testSessionStore{sessions: map[string]SessionMetadata{}}
	service := mustTestService(t, accountAuth, refreshStore, sessionStore, now)

	// Step 2: WebAuthn adapter が検証済み credential handle だけを渡した状態を再現し、Product login use case を実行する。
	result, err := service.LoginWithPasskey(ctx, LoginWithPasskeyInput{CredentialHandle: "credential-a", ClientIP: "192.0.2.10", UserAgent: "test-agent"})
	if err != nil {
		t.Fatalf("login with passkey: %v", err)
	}

	// Step 3: body DTO、Cookie command、server-side state、metadata の責務分離を小さい検証 helper で固定する。
	assertProductLoginSessionBody(t, result, accountAuth)
	assertProductLoginBodyHidesRefresh(t, result)
	assertProductLoginRefreshState(t, result, refreshStore, accountAuth, now)
	assertProductLoginMetadata(t, result, sessionStore, accountAuth)
}

func assertProductLoginSessionBody(t *testing.T, result LoginResult, accountAuth domain.AccountAuth) {
	t.Helper()

	// Step 1: body DTO には Product account session と accessToken が入り、refreshToken 平文を必要としないことを確認する。
	if result.Session.AccountID != accountAuth.AccountID() || result.Session.SessionID == "" || result.Session.AccessToken == "" {
		t.Fatalf("expected product authenticated session body, got %+v", result.Session)
	}
}

func assertProductLoginBodyHidesRefresh(t *testing.T, result LoginResult) {
	t.Helper()

	// Step 1: JSON body として marshal される値を作り、transport response body と同じ公開範囲を検証する。
	bodyBytes, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal login result body: %v", err)
	}
	body := string(bodyBytes)

	// Step 2: accessToken は body に公開され、refresh Cookie command は json tag で除外されることを確認する。
	if !strings.Contains(body, result.Session.AccessToken) {
		t.Fatalf("login body must expose accessToken, got %s", body)
	}
	for _, forbidden := range []string{result.RefreshCookie.Value, "RefreshCookie", "refreshToken"} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("login body must not expose %q: %s", forbidden, body)
		}
	}
}

func assertProductLoginRefreshState(t *testing.T, result LoginResult, refreshStore *testRefreshStore, accountAuth domain.AccountAuth, now time.Time) {
	t.Helper()

	// Step 1: refreshToken 平文は Cookie command にだけ存在し、寿命と削除 flag が login 成功用の値になっていることを確認する。
	if result.RefreshCookie.Value != "new-refresh-token" || result.RefreshCookie.Clear || result.RefreshCookie.MaxAge != 30*time.Minute || !result.RefreshCookie.ExpiresAt.Equal(now.Add(30*time.Minute)) {
		t.Fatalf("unexpected login refresh cookie command: %+v", result.RefreshCookie)
	}

	// Step 2: server-side state には refreshToken 平文ではなく hash だけが保存されたことを確認する。
	refreshHash, err := domain.HashOpaqueToken(result.RefreshCookie.Value)
	if err != nil {
		t.Fatalf("hash login refresh token: %v", err)
	}
	if refreshStore.session.TokenHash() != refreshHash || refreshStore.session.AccountID() != accountAuth.AccountID() || refreshStore.session.SessionID().String() != result.Session.SessionID {
		t.Fatalf("refresh store must contain product refresh state hash for the issued session")
	}
}

func assertProductLoginMetadata(t *testing.T, result LoginResult, sessionStore *testSessionStore, accountAuth domain.AccountAuth) {
	t.Helper()

	// Step 1: browser-readable session metadata は accessToken 検証用 selector だけを保持することを確認する。
	metadata, ok := sessionStore.sessions[result.Session.SessionID]
	if !ok || metadata.AccountID != accountAuth.AccountID() || metadata.SessionID != result.Session.SessionID {
		t.Fatalf("expected product session metadata for login result, got %+v", sessionStore.sessions)
	}

	// Step 2: device metadata へ refreshToken 平文が混入していないことを確認する。
	if strings.Contains(metadata.DeviceName, result.RefreshCookie.Value) || strings.Contains(metadata.IPHash, result.RefreshCookie.Value) {
		t.Fatalf("session metadata must not contain plaintext refresh token")
	}
}

func TestRefreshAccountSessionRejectsRevokedMetadataBeforeRotation(t *testing.T) {
	t.Parallel()

	// Step 1: 失効済み metadata を再作成しないことを検証するため、metadata store は空にする。
	ctx := context.Background()
	now := time.Unix(1_700_000_000, 0).UTC()
	accountAuth := mustTestAccountAuth(t, testProductAccountID(t, "01ARZ3NDEKTSV4RRFFQ69G5FAW"), "user@example.com", "01ARZ3NDEKTSV4RRFFQ69G5FAX", "credential-a")
	sessionID := mustTestAccountSessionID(t, "01ARZ3NDEKTSV4RRFFQ69G5FAY")
	refreshToken := "old-refresh-token"
	refreshSession := mustTestRefreshSession(t, accountAuth, sessionID, refreshToken, now)
	refreshStore := &testRefreshStore{session: refreshSession}
	sessionStore := &testSessionStore{sessions: map[string]SessionMetadata{}}
	service := mustTestService(t, accountAuth, refreshStore, sessionStore, now)

	// Step 2: metadata が存在しない session の refresh を実行し、refresh store の Rotate へ到達しないことを確認する。
	_, err := service.RefreshAccountSession(ctx, RefreshAccountSessionInput{RefreshToken: refreshToken, SessionID: sessionID.String(), ClientIP: "192.0.2.10", UserAgent: "test-agent"})
	if !errors.Is(err, ErrProductAuthUnauthorized) {
		t.Fatalf("expected ErrProductAuthUnauthorized, got %v", err)
	}
	if refreshStore.rotateCalled {
		t.Fatalf("refresh rotation must not run after session metadata has been revoked")
	}
}

func TestRefreshAccountSessionRotatesCookieAndOmitsRefreshTokenFromBody(t *testing.T) {
	t.Parallel()

	// Step 1: 旧 refreshToken と既存 metadata を持つ Product session を用意し、rotation 成功経路を再現する。
	ctx := context.Background()
	now := time.Unix(1_700_000_000, 0).UTC()
	accountAuth := mustTestAccountAuth(t, testProductAccountID(t, "01ARZ3NDEKTSV4RRFFQ69G5FAW"), "user@example.com", "01ARZ3NDEKTSV4RRFFQ69G5FAX", "credential-a")
	sessionID := mustTestAccountSessionID(t, "01ARZ3NDEKTSV4RRFFQ69G5FAY")
	oldRefreshToken := "old-refresh-token"
	refreshSession := mustTestRefreshSession(t, accountAuth, sessionID, oldRefreshToken, now)
	refreshStore := &testRefreshStore{session: refreshSession}
	sessionStore := &testSessionStore{sessions: map[string]SessionMetadata{sessionID.String(): {AccountID: accountAuth.AccountID(), SessionID: sessionID.String(), LoginAt: now, LastActiveAt: now}}}
	service := mustTestService(t, accountAuth, refreshStore, sessionStore, now)

	// Step 2: refresh use case を実行し、旧 token が store の Rotate 経由で消費されることを確認できる状態にする。
	result, err := service.RefreshAccountSession(ctx, RefreshAccountSessionInput{RefreshToken: oldRefreshToken, SessionID: sessionID.String(), ClientIP: "192.0.2.10", UserAgent: "test-agent"})
	if err != nil {
		t.Fatalf("refresh account session: %v", err)
	}
	if !refreshStore.rotateCalled {
		t.Fatalf("refresh rotation must consume the old token through the store")
	}

	// Step 3: 新 refreshToken が Cookie command にだけ入り、保存 state も新 hash へ置き換わったことを検証する。
	if result.RefreshCookie.Value != "new-refresh-token" {
		t.Fatalf("expected rotated cookie value, got %q", result.RefreshCookie.Value)
	}
	if result.RefreshCookie.Clear || result.RefreshCookie.MaxAge != 30*time.Minute || !result.RefreshCookie.ExpiresAt.Equal(now.Add(30*time.Minute)) {
		t.Fatalf("unexpected refresh cookie command: %+v", result.RefreshCookie)
	}
	newRefreshHash, err := domain.HashOpaqueToken("new-refresh-token")
	if err != nil {
		t.Fatalf("hash new refresh token: %v", err)
	}
	if refreshStore.session.TokenHash() != newRefreshHash {
		t.Fatalf("refresh store must keep only the rotated token hash")
	}

	// Step 4: response body として marshal される use case result から refreshToken 平文と Cookie command が除外されることを検証する。
	bodyBytes, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal refresh result body: %v", err)
	}
	body := string(bodyBytes)
	for _, forbidden := range []string{oldRefreshToken, "new-refresh-token", "RefreshCookie", "refreshToken"} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("refresh body must not expose %q: %s", forbidden, body)
		}
	}

	// Step 5: 消費済みの旧 token を再提示して拒否され、error に平文 token が混入しないことを確認する。
	_, err = service.RefreshAccountSession(ctx, RefreshAccountSessionInput{RefreshToken: oldRefreshToken, SessionID: sessionID.String(), ClientIP: "192.0.2.10", UserAgent: "test-agent"})
	if !errors.Is(err, ErrProductAuthUnauthorized) {
		t.Fatalf("expected old refresh token to be consumed, got %v", err)
	}
	if strings.Contains(fmt.Sprint(err), oldRefreshToken) {
		t.Fatalf("refresh error must not include the plaintext refresh token")
	}
}

func TestRefreshAccountSessionRotatesOnlyTargetSession(t *testing.T) {
	t.Parallel()

	// Step 1: 同一 Account に 2 つの Product session を用意し、対象 session と非対象 session を同時に保存する。
	ctx := context.Background()
	now := time.Unix(1_700_000_000, 0).UTC()
	accountAuth := mustTestAccountAuth(t, testProductAccountID(t, "01ARZ3NDEKTSV4RRFFQ69G5FAW"), "user@example.com", "01ARZ3NDEKTSV4RRFFQ69G5FAX", "credential-a")
	targetSessionID := mustTestAccountSessionID(t, "01ARZ3NDEKTSV4RRFFQ69G5FAY")
	otherSessionID := mustTestAccountSessionID(t, "01ARZ3NDEKTSV4RRFFQ69G5FB0")
	targetRefreshToken := "target-refresh-token"
	otherRefreshToken := "other-refresh-token"
	targetRefreshSession := mustTestRefreshSession(t, accountAuth, targetSessionID, targetRefreshToken, now)
	otherRefreshSession := mustTestRefreshSession(t, accountAuth, otherSessionID, otherRefreshToken, now)
	otherMetadataBefore := SessionMetadata{
		AccountID:    accountAuth.AccountID(),
		SessionID:    otherSessionID.String(),
		DeviceName:   "other-original-device",
		LoginAt:      now.Add(-2 * time.Hour),
		LastActiveAt: now.Add(-30 * time.Minute),
		IPHash:       "other-original-ip-hash",
	}
	refreshStore := &testRefreshStore{sessionsByTokenHash: map[string]domain.AccountRefreshSession{
		targetRefreshSession.TokenHash().String(): targetRefreshSession,
		otherRefreshSession.TokenHash().String():  otherRefreshSession,
	}}
	sessionStore := &testSessionStore{sessions: map[string]SessionMetadata{
		targetSessionID.String(): {AccountID: accountAuth.AccountID(), SessionID: targetSessionID.String(), LoginAt: now, LastActiveAt: now},
		otherSessionID.String():  otherMetadataBefore,
	}}
	service := mustTestService(t, accountAuth, refreshStore, sessionStore, now)

	// Step 2: 対象 session の Cookie refresh だけを実行し、返却 DTO が対象 session のまま rotation されたことを確認する。
	result, err := service.RefreshAccountSession(ctx, RefreshAccountSessionInput{RefreshToken: targetRefreshToken, SessionID: targetSessionID.String(), ClientIP: "192.0.2.10", UserAgent: "target-agent"})
	if err != nil {
		t.Fatalf("refresh target account session: %v", err)
	}
	if result.Session.SessionID != targetSessionID.String() {
		t.Fatalf("expected target session id %q, got %q", targetSessionID.String(), result.Session.SessionID)
	}

	// Step 3: 対象 session の旧 hash だけが消費され、新 hash として保存されたことを検証する。
	if _, ok := refreshStore.sessionsByTokenHash[targetRefreshSession.TokenHash().String()]; ok {
		t.Fatalf("target old refresh token hash must be consumed")
	}
	targetRotatedHash, err := domain.HashOpaqueToken("new-refresh-token")
	if err != nil {
		t.Fatalf("hash target rotated refresh token: %v", err)
	}
	if rotatedTarget, ok := refreshStore.sessionsByTokenHash[targetRotatedHash.String()]; !ok || rotatedTarget.SessionID() != targetSessionID {
		t.Fatalf("target session must be stored under the rotated token hash")
	}

	// Step 4: 非対象 session の refresh state と metadata は旧 token のまま保持され、対象 session の metadata 更新に巻き込まれないことを確認する。
	storedOther, ok := refreshStore.sessionsByTokenHash[otherRefreshSession.TokenHash().String()]
	if !ok {
		t.Fatalf("other session refresh token hash must remain available")
	}
	if storedOther.SessionID() != otherSessionID || storedOther.TokenHash() != otherRefreshSession.TokenHash() || storedOther.RevokedAt() != nil {
		t.Fatalf("other session refresh state must remain intact: %+v", storedOther)
	}
	otherMetadata := sessionStore.sessions[otherSessionID.String()]
	if otherMetadata != otherMetadataBefore {
		t.Fatalf("other session metadata must not be updated by target refresh: before=%+v after=%+v", otherMetadataBefore, otherMetadata)
	}

	// Step 5: 非対象 session の旧 refreshToken が引き続き使えることを実行で確認し、multi-session の独立性を証明する。
	otherResult, err := service.RefreshAccountSession(ctx, RefreshAccountSessionInput{RefreshToken: otherRefreshToken, SessionID: otherSessionID.String(), ClientIP: "192.0.2.11", UserAgent: "other-agent"})
	if err != nil {
		t.Fatalf("refresh other account session after target rotation: %v", err)
	}
	if otherResult.Session.SessionID != otherSessionID.String() || otherResult.RefreshCookie.Value != "second-refresh-token" {
		t.Fatalf("expected other session to rotate independently, got session=%+v cookie=%+v", otherResult.Session, otherResult.RefreshCookie)
	}
}

func TestValidateAccountBearerRejectsRepositorySubjectMismatch(t *testing.T) {
	t.Parallel()

	// Step 1: token subject と repository が返す AccountAuth を意図的にずらし、repository 不整合を検出できるようにする。
	ctx := context.Background()
	now := time.Unix(1_700_000_000, 0).UTC()
	accountA := testProductAccountID(t, "01ARZ3NDEKTSV4RRFFQ69G5FAW")
	accountB := testProductAccountID(t, "01ARZ3NDEKTSV4RRFFQ69G5FB0")
	authA := mustTestAccountAuth(t, accountA, "a@example.com", "01ARZ3NDEKTSV4RRFFQ69G5FAX", "credential-a")
	authB := mustTestAccountAuth(t, accountB, "b@example.com", "01ARZ3NDEKTSV4RRFFQ69G5FB1", "credential-b")
	sessionID := mustTestAccountSessionID(t, "01ARZ3NDEKTSV4RRFFQ69G5FAY")
	sessionStore := &testSessionStore{sessions: map[string]SessionMetadata{sessionID.String(): {AccountID: accountA, SessionID: sessionID.String()}}}
	service := mustTestService(t, authA, &testRefreshStore{}, sessionStore, now)
	service.accounts = &testAccountRepo{byCredential: map[string]domain.AccountAuth{"credential-a": authA}, byID: map[string]domain.AccountAuth{accountA.String(): authB}}

	// Step 2: subject は accountA のまま署名し、FindByID が accountB を返した場合に unauthorized となることを確認する。
	accessToken := mustTestAccessToken(t, service.signer, accountA, sessionID, "01ARZ3NDEKTSV4RRFFQ69G5FAZ", now, 15*time.Minute)
	_, err := service.ValidateAccountBearer(ctx, ValidateAccountBearerInput{AccessToken: accessToken, SessionID: sessionID.String()})
	if !errors.Is(err, ErrProductAuthUnauthorized) {
		t.Fatalf("expected ErrProductAuthUnauthorized, got %v", err)
	}
}

type testAccountRepo struct {
	byCredential map[string]domain.AccountAuth
	byID         map[string]domain.AccountAuth
}

func (r *testAccountRepo) FindByCredential(_ context.Context, credentialHandle string) (domain.AccountAuth, error) {
	// Step 1: credential handle に対応する Product AccountAuth projection だけを返す。
	accountAuth, ok := r.byCredential[credentialHandle]
	if !ok {
		return domain.AccountAuth{}, domain.ErrAccountAuthNotFound
	}

	// Step 2: 見つかった projection を返す。
	return accountAuth, nil
}

func (r *testAccountRepo) FindByID(_ context.Context, accountID domain.AccountID) (domain.AccountAuth, error) {
	// Step 1: AccountID に対応する Product AccountAuth projection だけを返す。
	accountAuth, ok := r.byID[accountID.String()]
	if !ok {
		return domain.AccountAuth{}, domain.ErrAccountAuthNotFound
	}

	// Step 2: 見つかった projection を返す。
	return accountAuth, nil
}

type testRefreshStore struct {
	session             domain.AccountRefreshSession
	sessionsByTokenHash map[string]domain.AccountRefreshSession
	rotateCalled        bool
}

func (s *testRefreshStore) Save(_ context.Context, session domain.AccountRefreshSession, _ time.Duration) error {
	// Step 1: login 発行された refresh session state を保存する。
	s.session = session
	if s.sessionsByTokenHash != nil {
		// Step 2: multi-session test では token hash を key にして保存し、複数 session の独立性を検証できるようにする。
		s.sessionsByTokenHash[session.TokenHash().String()] = session
	}
	return nil
}

func (s *testRefreshStore) Rotate(_ context.Context, tokenHash domain.OpaqueTokenHash, _ time.Duration, build RefreshRotationBuilder) (domain.AccountRefreshSession, domain.AccountRefreshSession, error) {
	// Step 1: Rotate 呼び出し有無を記録し、metadata 失効時に呼ばれないことを検証可能にする。
	s.rotateCalled = true
	if s.sessionsByTokenHash != nil {
		// Step 2: multi-session test では提示 hash と一致する 1 session だけを取り出し、他 session を触らない。
		stored, ok := s.sessionsByTokenHash[tokenHash.String()]
		if !ok {
			return domain.AccountRefreshSession{}, domain.AccountRefreshSession{}, domain.ErrSessionNotFound
		}

		// Step 3: application callback に対象 session の Product domain validation と次 session 生成を委譲する。
		next, err := build(stored)
		if err != nil {
			return stored, domain.AccountRefreshSession{}, err
		}

		// Step 4: 対象 session の旧 hash だけを消費し、新 hash へ置換することで rotation の局所性を再現する。
		delete(s.sessionsByTokenHash, tokenHash.String())
		s.sessionsByTokenHash[next.TokenHash().String()] = next
		return stored, next, nil
	}

	// Step 5: 単一 session test では提示 hash が保存済み refresh session と一致しない場合は存在しない token として拒否する。
	if s.session.TokenHash() != tokenHash {
		return domain.AccountRefreshSession{}, domain.AccountRefreshSession{}, domain.ErrSessionNotFound
	}

	// Step 6: application callback に Product domain validation と次 session 生成を委譲する。
	next, err := build(s.session)
	if err != nil {
		return s.session, domain.AccountRefreshSession{}, err
	}

	// Step 7: callback が返した次 session を保存し、旧 session と次 session を返す。
	consumed := s.session
	s.session = next
	return consumed, next, nil
}

func (s *testRefreshStore) RevokeSession(_ context.Context, _ domain.AccountID, _ domain.AccountAuthSessionID, _ time.Time) error {
	// Step 1: この test double では revoke 成功だけを表す。
	return nil
}

func (s *testRefreshStore) RevokeAllForAccount(_ context.Context, _ domain.AccountID, _ time.Time) error {
	// Step 1: この test double では全 revoke 成功だけを表す。
	return nil
}

type testSessionStore struct {
	sessions map[string]SessionMetadata
}

func (s *testSessionStore) Save(_ context.Context, metadata SessionMetadata, _ time.Duration) error {
	// Step 1: session selector を key として metadata を保存する。
	s.sessions[metadata.SessionID] = metadata
	return nil
}

func (s *testSessionStore) Get(_ context.Context, sessionID domain.AccountAuthSessionID) (SessionMetadata, error) {
	// Step 1: Product session selector に対応する metadata を取得する。
	metadata, ok := s.sessions[sessionID.String()]
	if !ok {
		return SessionMetadata{}, domain.ErrSessionNotFound
	}

	// Step 2: 見つかった metadata を返す。
	return metadata, nil
}

func (s *testSessionStore) Revoke(_ context.Context, _ domain.AccountID, sessionID domain.AccountAuthSessionID) error {
	// Step 1: session metadata を削除し、bearer validation から見えない状態にする。
	delete(s.sessions, sessionID.String())
	return nil
}

func (s *testSessionStore) RevokeAllForAccount(_ context.Context, accountID domain.AccountID) error {
	// Step 1: 対象 AccountID に紐づく metadata だけを削除する。
	for sessionID, metadata := range s.sessions {
		if metadata.AccountID == accountID {
			delete(s.sessions, sessionID)
		}
	}
	return nil
}

type testIDGenerator struct {
	ids   []string
	index int
}

func (g *testIDGenerator) Next() (string, error) {
	// Step 1: test が用意した ULID を順番に返し、ID 生成を deterministic にする。
	if g.index >= len(g.ids) {
		return "", ErrProductAuthUnavailable
	}
	id := g.ids[g.index]
	g.index++
	return id, nil
}

type testTokenGenerator struct {
	tokens []string
	index  int
}

func (g *testTokenGenerator) NewToken() (string, error) {
	// Step 1: test が用意した opaque token を順番に返し、hash と Cookie command を deterministic にする。
	if g.index >= len(g.tokens) {
		return "", ErrProductAuthUnavailable
	}
	token := g.tokens[g.index]
	g.index++
	return token, nil
}

func mustTestService(t *testing.T, accountAuth domain.AccountAuth, refreshStore *testRefreshStore, sessionStore *testSessionStore, now time.Time) *Service {
	t.Helper()

	// Step 1: test 用 signer と deterministic generator を注入した Product auth service を生成する。
	signer, err := tokenprimitive.NewJWTSignVerifier([]byte("product-auth-test-secret"))
	if err != nil {
		t.Fatalf("create signer: %v", err)
	}
	service, err := NewService(Dependencies{
		Accounts:        &testAccountRepo{byCredential: map[string]domain.AccountAuth{"credential-a": accountAuth}, byID: map[string]domain.AccountAuth{accountAuth.AccountID().String(): accountAuth}},
		RefreshSessions: refreshStore,
		Sessions:        sessionStore,
		Signer:          signer,
		IDGenerator:     &testIDGenerator{ids: []string{"01ARZ3NDEKTSV4RRFFQ69G5FB2", "01ARZ3NDEKTSV4RRFFQ69G5FB3", "01ARZ3NDEKTSV4RRFFQ69G5FB4"}},
		TokenGenerator:  &testTokenGenerator{tokens: []string{"new-refresh-token", "second-refresh-token"}},
		Clock:           func() time.Time { return now },
	}, Config{AccessTokenTTL: 15 * time.Minute, RefreshTokenTTL: time.Hour, RefreshCookieLifetime: 30 * time.Minute})
	if err != nil {
		t.Fatalf("create service: %v", err)
	}

	// Step 2: 生成済み service を返す。
	return service
}

func mustTestAccountAuth(t *testing.T, accountID domain.AccountID, email string, credentialID string, handle string) domain.AccountAuth {
	t.Helper()

	// Step 1: Product AccountAuth projection を domain constructor で作る。
	accountAuth, err := domain.NewAccountAuth(accountID, email, email, credentialID, handle)
	if err != nil {
		t.Fatalf("create account auth: %v", err)
	}

	// Step 2: 検証済み projection を返す。
	return accountAuth
}

func mustTestRefreshSession(t *testing.T, accountAuth domain.AccountAuth, sessionID domain.AccountAuthSessionID, refreshToken string, now time.Time) domain.AccountRefreshSession {
	t.Helper()

	// Step 1: AccountAuth projection を Account root に写像し、refresh session constructor に渡す。
	account, err := accountRootFromAuth(accountAuth)
	if err != nil {
		t.Fatalf("create account root: %v", err)
	}

	// Step 2: 平文 refreshToken を保存用 hash に変換する。
	tokenHash, err := domain.HashOpaqueToken(refreshToken)
	if err != nil {
		t.Fatalf("hash refresh token: %v", err)
	}

	// Step 3: Product refresh session state を domain constructor で作る。
	session, err := domain.NewAccountRefreshSession(account, sessionID, tokenHash, now, now.Add(time.Hour))
	if err != nil {
		t.Fatalf("create refresh session: %v", err)
	}

	// Step 4: 検証済み refresh session を返す。
	return session
}

func mustTestAccessToken(t *testing.T, signer tokenprimitive.JSONSignVerifier, accountID domain.AccountID, sessionID domain.AccountAuthSessionID, jtiValue string, now time.Time, ttl time.Duration) string {
	t.Helper()

	// Step 1: Product accessToken payload DTO を明示的に作り、ValidateAccountBearer の decode path を通す。
	payload := accessTokenPayload{Subject: accountID.String(), SessionID: sessionID.String(), TokenID: jtiValue, Status: domain.AccountStatusActive.String(), IssuedAt: now.Unix(), ExpiresAt: now.Add(ttl).Unix()}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	// Step 2: 中立 signer で payload を署名し、bearer token として返す。
	token, err := signer.SignJSON(payloadBytes)
	if err != nil {
		t.Fatalf("sign payload: %v", err)
	}
	return token
}

func testProductAccountID(t *testing.T, value string) domain.AccountID {
	t.Helper()

	// Step 1: 文字列 ULID を Product AccountID として検証する。
	accountID, err := domain.NewAccountID(value)
	if err != nil {
		t.Fatalf("create account id: %v", err)
	}

	// Step 2: 検証済み AccountID を返す。
	return accountID
}

func mustTestAccountSessionID(t *testing.T, value string) domain.AccountAuthSessionID {
	t.Helper()

	// Step 1: 文字列 ULID を Product AccountAuth session ID として検証する。
	sessionID, err := domain.NewAccountAuthSessionID(value)
	if err != nil {
		t.Fatalf("create account session id: %v", err)
	}

	// Step 2: 検証済み session ID を返す。
	return sessionID
}
