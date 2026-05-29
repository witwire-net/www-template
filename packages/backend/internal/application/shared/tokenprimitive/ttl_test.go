package application

import (
	"errors"
	"testing"
	"time"

	domain "www-template/packages/backend/internal/domain"
)

// [AUTH-BE-S064] shared tokenprimitive が TTL と Cookie lifetime の大小関係だけを検証することを確認する。
func TestValidateDurations(t *testing.T) {
	t.Parallel()

	// token TTL と Cookie lifetime をまとめて検証し、後続処理で使う TTL wrapper を受け取る。
	ttl, err := ValidateDurations(15*time.Minute, 10*time.Minute)
	if err != nil {
		t.Fatalf("validate durations: %v", err)
	}

	// 検証済み TTL は元の duration を保持する。
	if ttl.Duration() != 15*time.Minute {
		t.Fatalf("unexpected ttl: got %s, want %s", ttl.Duration(), 15*time.Minute)
	}

	// 失効時刻は注入した発行時刻からだけ計算される。
	issuedAt := time.Date(2026, 5, 24, 12, 0, 0, 0, time.FixedZone("JST", 9*60*60))
	expiresAt := ttl.ExpiresAt(issuedAt)
	if !expiresAt.Equal(issuedAt.UTC().Add(15 * time.Minute)) {
		t.Fatalf("unexpected expiresAt: got %s", expiresAt)
	}
}

// [AUTH-BE-S064] shared tokenprimitive が不正な TTL と Cookie lifetime を domain error のまま返すことを確認する。
func TestValidateDurationsRejectsInvalidValues(t *testing.T) {
	t.Parallel()

	// 0 以下の token TTL は domain primitive の TTL error として拒否される。
	_, err := ValidateDurations(0, 10*time.Minute)
	if !errors.Is(err, domain.ErrInvalidTokenTTL) {
		t.Fatalf("expected ErrInvalidTokenTTL, got %v", err)
	}

	// token TTL より長い Cookie lifetime は Cookie lifetime error として拒否される。
	_, err = ValidateDurations(15*time.Minute, 20*time.Minute)
	if !errors.Is(err, domain.ErrInvalidTokenCookieLifetime) {
		t.Fatalf("expected ErrInvalidTokenCookieLifetime, got %v", err)
	}

	// ゼロ値 TTL を直接渡した場合も未検証 TTL として拒否される。
	err = ValidateCookieLifetime(1*time.Minute, TTL{})
	if !errors.Is(err, domain.ErrInvalidTokenTTL) {
		t.Fatalf("expected ErrInvalidTokenTTL for zero TTL, got %v", err)
	}
}
