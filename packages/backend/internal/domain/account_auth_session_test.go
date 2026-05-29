package domain

import (
	"errors"
	"testing"
	"time"
)

// [AUTH-BE-S069] Product AccountAuth accessToken eligibility は停止・revoke 境界・session ID mismatch を拒否する。
func TestAuthBES069AccountAccessTokenEligibilityRejectsInvalidAccountState(t *testing.T) {
	t.Parallel()

	// Step 1: active Account と Product AccountAuth claim fixture を作り、正常な eligible 判定を確認する。
	fixture := newAccountAuthSessionTestFixture(t)
	claims := fixture.newAccessTokenClaims(t)
	if err := claims.EnsureEligible(fixture.account, fixture.sessionID, fixture.now); err != nil {
		t.Fatalf("expected active account token to be eligible: %v", err)
	}

	// Step 2: suspended Account は発行済み accessToken を拒否することを確認する。
	suspended := fixture.suspendedAccount(t)
	if err := claims.EnsureEligible(suspended, fixture.sessionID, fixture.now); !errors.Is(err, ErrAccountAuthTokenIneligible) {
		t.Fatalf("expected suspended account token rejection, got %v", err)
	}

	// Step 3: Restore 後も sessionRevokedAfter 境界以前の accessToken が拒否されることを確認する。
	restored := fixture.restoredAccount(t, suspended)
	if err := claims.EnsureEligible(restored, fixture.sessionID, fixture.now); !errors.Is(err, ErrAccountAuthTokenIneligible) {
		t.Fatalf("expected revoked-boundary token rejection, got %v", err)
	}

	// Step 4: request が選択した session ID と claim の sid が違う場合は拒否する。
	if err := claims.EnsureEligible(fixture.account, fixture.otherSessionID, fixture.now); !errors.Is(err, ErrAccountAuthTokenIneligible) {
		t.Fatalf("expected session id mismatch rejection, got %v", err)
	}
}

// [AUTH-BE-S069] Product AccountAuth refresh session eligibility は停止・revoke 境界・session ID mismatch を拒否する。
func TestAuthBES069AccountRefreshSessionRejectsInvalidAccountState(t *testing.T) {
	t.Parallel()

	// Step 1: active Account と Product refresh session state fixture を作り、正常な rotation eligibility を確認する。
	fixture := newAccountAuthSessionTestFixture(t)
	session := fixture.newRefreshSession(t)
	if err := session.CanRotate(fixture.account, fixture.sessionID, fixture.now); err != nil {
		t.Fatalf("expected active account refresh session to rotate: %v", err)
	}

	// Step 2: suspended Account は refresh session rotation を拒否することを確認する。
	suspended := fixture.suspendedAccount(t)
	if err := session.CanRotate(suspended, fixture.sessionID, fixture.now); !errors.Is(err, ErrAccountAuthTokenIneligible) {
		t.Fatalf("expected suspended account refresh rejection, got %v", err)
	}

	// Step 3: Restore 後も sessionRevokedAfter 境界以前の refresh session が拒否されることを確認する。
	restored := fixture.restoredAccount(t, suspended)
	if err := session.CanRotate(restored, fixture.sessionID, fixture.now); !errors.Is(err, ErrAccountAuthTokenIneligible) {
		t.Fatalf("expected revoked-boundary refresh rejection, got %v", err)
	}

	// Step 4: request selector と保存済み session ID が違う場合は rotation を拒否する。
	if err := session.CanRotate(fixture.account, fixture.otherSessionID, fixture.now); !errors.Is(err, ErrAccountAuthTokenIneligible) {
		t.Fatalf("expected refresh session id mismatch rejection, got %v", err)
	}
}

type accountAuthSessionTestFixture struct {
	account        Account
	sessionID      AccountAuthSessionID
	otherSessionID AccountAuthSessionID
	jti            TokenJTI
	ttl            TokenTTL
	tokenHash      OpaqueTokenHash
	issuedAt       time.Time
	revokedAfter   time.Time
	now            time.Time
}

func newAccountAuthSessionTestFixture(t *testing.T) accountAuthSessionTestFixture {
	t.Helper()

	// Step 1: Product AccountAuth domain が扱う Account root と識別子をすべて constructor 経由で準備する。
	account, err := NewAdminCreatedAccount(mustAccountLifecycleTestAccountID(t), mustAccountLifecycleTestEmail(t, "customer@example.com"))
	if err != nil {
		t.Fatalf("new account: %v", err)
	}
	sessionID := mustAccountAuthSessionTestSessionID(t, "01ARZ3NDEKTSV4RRFFQ69G5FAW")
	otherSessionID := mustAccountAuthSessionTestSessionID(t, "01ARZ3NDEKTSV4RRFFQ69G5FB0")
	jti := mustAccountAuthSessionTestJTI(t, "01ARZ3NDEKTSV4RRFFQ69G5FAX")
	ttl := mustAccountAuthSessionTestTTL(t)
	tokenHash := mustAccountAuthSessionTestHash(t)
	issuedAt := time.Date(2026, 5, 24, 1, 0, 0, 0, time.UTC)

	// Step 2: revoke 境界と now を deterministic に固定し、境界前発行 token/session の拒否を安定検証する。
	return accountAuthSessionTestFixture{
		account:        account,
		sessionID:      sessionID,
		otherSessionID: otherSessionID,
		jti:            jti,
		ttl:            ttl,
		tokenHash:      tokenHash,
		issuedAt:       issuedAt,
		revokedAfter:   issuedAt.Add(time.Minute),
		now:            issuedAt.Add(2 * time.Minute),
	}
}

func (f accountAuthSessionTestFixture) newAccessTokenClaims(t *testing.T) AccountAccessTokenClaims {
	t.Helper()

	// Step 1: active Account から Product AccountAuth accessToken claim を生成する。
	claims, err := NewAccountAccessTokenClaims(f.account, f.sessionID, f.jti, f.issuedAt, f.ttl)
	if err != nil {
		t.Fatalf("new access token claims: %v", err)
	}

	// Step 2: 検証済み claim fixture を返す。
	return claims
}

func (f accountAuthSessionTestFixture) newRefreshSession(t *testing.T) AccountRefreshSession {
	t.Helper()

	// Step 1: active Account から Product AccountAuth refresh session state を生成する。
	session, err := NewAccountRefreshSession(f.account, f.sessionID, f.tokenHash, f.issuedAt, f.issuedAt.Add(30*time.Minute))
	if err != nil {
		t.Fatalf("new refresh session: %v", err)
	}

	// Step 2: 検証済み refresh session fixture を返す。
	return session
}

func (f accountAuthSessionTestFixture) suspendedAccount(t *testing.T) Account {
	t.Helper()

	// Step 1: Account lifecycle の Suspend を使い、Product AccountAuth が参照する停止状態を作る。
	suspended, err := f.account.Suspend(f.revokedAfter)
	if err != nil {
		t.Fatalf("suspend account: %v", err)
	}

	// Step 2: 停止済み Account を返す。
	return suspended
}

func (f accountAuthSessionTestFixture) restoredAccount(t *testing.T, suspended Account) Account {
	t.Helper()

	// Step 1: Account lifecycle の Restore を使い、sessionRevokedAfter が残る active Account を作る。
	restored, err := suspended.Restore()
	if err != nil {
		t.Fatalf("restore account: %v", err)
	}

	// Step 2: 復元済み Account を返す。
	return restored
}

func mustAccountAuthSessionTestSessionID(t *testing.T, value string) AccountAuthSessionID {
	t.Helper()

	// Step 1: session ID fixture を Product AccountAuth 専用 constructor で検証する。
	sessionID, err := NewAccountAuthSessionID(value)
	if err != nil {
		t.Fatalf("invalid session id fixture: %v", err)
	}

	// Step 2: 検証済み session ID を返す。
	return sessionID
}

func mustAccountAuthSessionTestJTI(t *testing.T, value string) TokenJTI {
	t.Helper()

	// Step 1: jti fixture を neutral token primitive で検証する。
	jti, err := NewTokenJTI(value)
	if err != nil {
		t.Fatalf("invalid jti fixture: %v", err)
	}

	// Step 2: 検証済み jti を返す。
	return jti
}

func mustAccountAuthSessionTestTTL(t *testing.T) TokenTTL {
	t.Helper()

	// Step 1: accessToken TTL fixture を neutral token primitive で検証する。
	ttl, err := ValidateTokenTTL(15 * time.Minute)
	if err != nil {
		t.Fatalf("invalid ttl fixture: %v", err)
	}

	// Step 2: 検証済み TTL を返す。
	return ttl
}

func mustAccountAuthSessionTestHash(t *testing.T) OpaqueTokenHash {
	t.Helper()

	// Step 1: refreshToken hash fixture を neutral opaque hash primitive で生成する。
	hash, err := HashOpaqueToken("product-refresh-token-secret")
	if err != nil {
		t.Fatalf("invalid hash fixture: %v", err)
	}

	// Step 2: 検証済み hash を返す。
	return hash
}
