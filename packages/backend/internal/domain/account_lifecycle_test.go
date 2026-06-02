package domain

import (
	"errors"
	"testing"
	"time"
)

// [ADMIN-CONSOLE-BE-S077] AccountEmail は空白除去と lowercase 正規化を行う。
func TestAdminConsoleBES077AccountEmailNormalizesCanonicalValue(t *testing.T) {
	t.Parallel()

	// Step 1: 管理画面入力に近い前後空白と大文字を含む email を domain constructor に渡す。
	email, err := NewAccountEmail("  Customer+Tag@Example.COM  ")
	if err != nil {
		t.Fatalf("expected normalized email, got error: %v", err)
	}

	// Step 2: canonical string が trim と lowercase を反映していることを検証する。
	if got, want := email.String(), "customer+tag@example.com"; got != want {
		t.Fatalf("unexpected canonical email: got %q want %q", got, want)
	}
}

// [ADMIN-CONSOLE-BE-S077] AccountEmail は不正形式を domain error として拒否する。
func TestAdminConsoleBES077AccountEmailRejectsInvalidFormat(t *testing.T) {
	t.Parallel()

	// Step 1: format が不正な入力を並べ、validation が application に漏れないことを確認する。
	invalidEmails := []string{
		"",
		"customer.example.com",
		"customer@@example.com",
		"customer@localhost",
		"customer @example.com",
		"customer@example..com",
		"customer@-example.com",
	}

	// Step 2: すべての不正入力が同じ domain error に畳み込まれることを検証する。
	for _, raw := range invalidEmails {
		_, err := NewAccountEmail(raw)
		if !errors.Is(err, ErrInvalidAccountEmail) {
			t.Fatalf("expected ErrInvalidAccountEmail for %q, got %v", raw, err)
		}
	}
}

// [ADMIN-CONSOLE-BE-S077] Admin 作成 Account は active 初期状態と既定設定を保持する。
func TestAdminConsoleBES077NewCreatedAccountInitializesActiveRoot(t *testing.T) {
	t.Parallel()

	// Step 1: AccountID と AccountEmail を canonical value object として準備する。
	accountID := mustAccountLifecycleTestAccountID(t)
	email := mustAccountLifecycleTestEmail(t, "customer@example.com")

	// Step 2: Admin 作成 constructor を通し、初期 lifecycle と child setting を同時生成する。
	account, err := NewCreatedAccount(accountID, email)
	if err != nil {
		t.Fatalf("expected admin-created account, got error: %v", err)
	}

	// Step 3: root が AccountEmail、active status、DefaultAccountSetting、revoke 境界なしを保持することを検証する。
	assertInitialCreatedAccount(t, account, accountID)
}

// [ADMIN-CONSOLE-BE-S077] Admin 作成 Account constructor は不正な email 値オブジェクトを再検証する。
func TestAdminConsoleBES077NewCreatedAccountRejectsInvalidEmail(t *testing.T) {
	t.Parallel()

	// Step 1: 手組みの不正 AccountEmail を渡し、constructor が再検証することを確認する。
	_, err := NewCreatedAccount(mustAccountLifecycleTestAccountID(t), AccountEmail("not-email"))
	if !errors.Is(err, ErrInvalidAccountEmail) {
		t.Fatalf("expected ErrInvalidAccountEmail, got %v", err)
	}
}

// [ADMIN-CONSOLE-BE-S077] Suspend/Restore は停止状態と session revoke 境界を検証する。
func TestAdminConsoleBES077SuspendRestoreAndSessionBoundary(t *testing.T) {
	t.Parallel()

	// Step 1: active Account を作成し、停止操作に使う deterministic な境界時刻を準備する。
	account, err := NewCreatedAccount(mustAccountLifecycleTestAccountID(t), mustAccountLifecycleTestEmail(t, "customer@example.com"))
	if err != nil {
		t.Fatalf("expected admin-created account, got error: %v", err)
	}
	suspendAt := time.Date(2026, 5, 24, 2, 0, 0, 0, time.FixedZone("JST", 9*60*60))

	// Step 2: Suspend が suspended status と UTC revoke 境界を設定することを検証する。
	suspended := assertSuspendedAccount(t, account, suspendAt)

	// Step 3: Restore 後も境界以前の token は拒否し、境界後の token だけを許すことを検証する。
	assertRestoredAccountRejectsOldTokens(t, suspended, suspendAt)
}

// [ADMIN-CONSOLE-BE-S077] Suspend は zero time の session revoke 境界を拒否する。
func TestAdminConsoleBES077SuspendRejectsZeroBoundary(t *testing.T) {
	t.Parallel()

	// Step 1: zero time を停止境界に使えないことを domain error として確認する。
	account, err := NewCreatedAccount(mustAccountLifecycleTestAccountID(t), mustAccountLifecycleTestEmail(t, "customer@example.com"))
	if err != nil {
		t.Fatalf("expected admin-created account, got error: %v", err)
	}
	_, err = account.Suspend(time.Time{})
	if !errors.Is(err, ErrInvalidSessionRevocationBoundary) {
		t.Fatalf("expected ErrInvalidSessionRevocationBoundary, got %v", err)
	}
}

func assertInitialCreatedAccount(t *testing.T, account Account, accountID AccountID) {
	t.Helper()

	// Step 1: Account root が指定した AccountID と canonical email を保持することを検証する。
	if account.ID() != accountID {
		t.Fatalf("unexpected account id: got %q want %q", account.ID(), accountID)
	}
	if got, want := account.Email().String(), "customer@example.com"; got != want {
		t.Fatalf("unexpected account email: got %q want %q", got, want)
	}

	// Step 2: 作成直後の lifecycle が active で、既存 session revoke 境界を持たないことを検証する。
	if account.Status() != AccountStatusActive {
		t.Fatalf("expected active status, got %q", account.Status())
	}
	if account.SessionRevokedAfter() != nil {
		t.Fatalf("new admin-created account must not have session revoke boundary")
	}
	if account.RejectsTokenIssuedAt(time.Date(2026, 5, 24, 1, 0, 0, 0, time.UTC)) {
		t.Fatalf("active account without revoke boundary must not reject token by lifecycle")
	}

	// Step 3: Account root が自身に属する DefaultAccountSetting を同時に保持することを検証する。
	if account.Setting().AccountID() != accountID {
		t.Fatalf("setting must belong to account %q", accountID)
	}
	if account.Setting().Locale() != DefaultAccountLocale() {
		t.Fatalf("unexpected default locale: got %q", account.Setting().Locale())
	}
}

func assertSuspendedAccount(t *testing.T, account Account, suspendAt time.Time) Account {
	t.Helper()

	// Step 1: Account を停止し、domain method がエラーなく新しい Account 値を返すことを確認する。
	suspended, err := account.Suspend(suspendAt)
	if err != nil {
		t.Fatalf("expected suspended account, got error: %v", err)
	}

	// Step 2: suspended status は発行時刻に関係なく token を拒否することを検証する。
	if suspended.Status() != AccountStatusSuspended {
		t.Fatalf("expected suspended status, got %q", suspended.Status())
	}
	if !suspended.RejectsTokenIssuedAt(suspendAt.Add(time.Hour)) {
		t.Fatalf("suspended account must reject all issued tokens")
	}

	// Step 3: SessionRevokedAfter が UTC の defensive copy として返ることを検証する。
	revokedAfter := suspended.SessionRevokedAfter()
	if revokedAfter == nil {
		t.Fatalf("suspended account must expose session revoke boundary")
	}
	if !revokedAfter.Equal(suspendAt.UTC()) {
		t.Fatalf("unexpected revoke boundary: got %v want %v", revokedAfter, suspendAt.UTC())
	}

	// Step 4: 後続の restore 検証に使う suspended Account を返す。
	return suspended
}

func assertRestoredAccountRejectsOldTokens(t *testing.T, suspended Account, suspendAt time.Time) {
	t.Helper()

	// Step 1: 停止済み Account を復元し、status が active に戻ることを確認する。
	restored, err := suspended.Restore()
	if err != nil {
		t.Fatalf("expected restored account, got error: %v", err)
	}
	if restored.Status() != AccountStatusActive {
		t.Fatalf("expected restored active status, got %q", restored.Status())
	}

	// Step 2: 停止境界以前または同時刻に発行された token が復元後も拒否されることを検証する。
	if !restored.RejectsTokenIssuedAt(suspendAt.Add(-time.Second)) {
		t.Fatalf("restored account must reject tokens issued before revoke boundary")
	}
	if !restored.RejectsTokenIssuedAt(suspendAt) {
		t.Fatalf("restored account must reject tokens issued exactly at revoke boundary")
	}

	// Step 3: 停止境界後に発行された token は lifecycle 境界では拒否されないことを検証する。
	if restored.RejectsTokenIssuedAt(suspendAt.Add(time.Second)) {
		t.Fatalf("restored account must allow tokens issued after revoke boundary")
	}
}

func mustAccountLifecycleTestAccountID(t *testing.T) AccountID {
	t.Helper()

	// Step 1: テスト用の固定 ULID を domain constructor で作り、fixture 自体の正当性を保証する。
	accountID, err := NewAccountID("01ARZ3NDEKTSV4RRFFQ69G5FAV")
	if err != nil {
		t.Fatalf("invalid test account id: %v", err)
	}

	// Step 2: 検証済み AccountID を各 test に渡す。
	return accountID
}

func mustAccountLifecycleTestEmail(t *testing.T, raw string) AccountEmail {
	t.Helper()

	// Step 1: テスト入力 email を domain constructor で作り、fixture の typo を早期検出する。
	email, err := NewAccountEmail(raw)
	if err != nil {
		t.Fatalf("invalid test email: %v", err)
	}

	// Step 2: 検証済み AccountEmail を Account constructor に渡す。
	return email
}
