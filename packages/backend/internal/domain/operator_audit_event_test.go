package domain

import (
	"errors"
	"testing"
	"time"
)

var operatorAuditCompletedAt = time.Date(2026, 5, 24, 10, 30, 0, 0, time.FixedZone("JST", 9*60*60))

// [ADMIN-CONSOLE-BE-S079] OperatorAuditEvent が pending から succeeded/failed へ一度だけ遷移することを検証する。
func TestOperatorAuditEventOutcomeTransitions(t *testing.T) {
	t.Parallel()

	t.Run("pending event can be marked succeeded with completed timestamp", func(t *testing.T) {
		t.Parallel()

		// Step 1: audit intent は必ず pending として開始する。
		event := NewOperatorAuditEvent()
		if event.Outcome() != OperatorAuditOutcomePending {
			t.Fatalf("outcome = %q, want %q", event.Outcome(), OperatorAuditOutcomePending)
		}

		// Step 2: 成功 outcome は completedAt を持ち、stable error code を持たない。
		completed, err := event.MarkSucceeded(operatorAuditCompletedAt)
		if err != nil {
			t.Fatalf("unexpected MarkSucceeded error: %v", err)
		}
		assertCompletedAudit(t, completed, OperatorAuditOutcomeSucceeded, "")
	})

	t.Run("pending event can be marked failed with stable error code", func(t *testing.T) {
		t.Parallel()

		// Step 1: failed outcome は application message ではなく stable code を必須にする。
		code, err := NewStableErrorCode("DUPLICATE_EMAIL")
		if err != nil {
			t.Fatalf("unexpected code error: %v", err)
		}

		// Step 2: failed outcome には stable code と completedAt の両方を保存する。
		completed, err := NewOperatorAuditEvent().MarkFailed(code, operatorAuditCompletedAt)
		if err != nil {
			t.Fatalf("unexpected MarkFailed error: %v", err)
		}
		assertCompletedAudit(t, completed, OperatorAuditOutcomeFailed, "duplicate_email")
	})
}

// [ADMIN-CONSOLE-BE-S079] OperatorAuditEvent が不正な outcome transition を拒否することを検証する。
func TestOperatorAuditEventRejectsInvalidOutcomeTransitions(t *testing.T) {
	t.Parallel()

	t.Run("double completion is rejected", func(t *testing.T) {
		t.Parallel()

		// Step 1: 一度 succeeded へ完了した audit event を作る。
		completed, err := NewOperatorAuditEvent().MarkSucceeded(operatorAuditCompletedAt)
		if err != nil {
			t.Fatalf("unexpected MarkSucceeded error: %v", err)
		}

		// Step 2: 完了済み event に対する再完了は outcome 種別に関係なく拒否する。
		_, err = completed.MarkFailed("duplicate_email", operatorAuditCompletedAt.Add(time.Minute))
		if !errors.Is(err, ErrOperatorAuditAlreadyCompleted) {
			t.Fatalf("expected ErrOperatorAuditAlreadyCompleted, got %v", err)
		}
	})

	t.Run("failed outcome requires stable error code", func(t *testing.T) {
		t.Parallel()

		// Step 1: 空の stable error code は failed outcome の分類に使えないため拒否する。
		_, err := NewOperatorAuditEvent().MarkFailed("", operatorAuditCompletedAt)
		if !errors.Is(err, ErrInvalidOperatorAuditStableErrorCode) {
			t.Fatalf("expected ErrInvalidOperatorAuditStableErrorCode, got %v", err)
		}
	})

	t.Run("succeeded outcome requires completed timestamp", func(t *testing.T) {
		t.Parallel()

		// Step 1: ゼロ時刻で成功 outcome を保存すると時系列監査不能になるため拒否する。
		_, err := NewOperatorAuditEvent().MarkSucceeded(time.Time{})
		if !errors.Is(err, ErrInvalidOperatorAuditCompletedAt) {
			t.Fatalf("expected ErrInvalidOperatorAuditCompletedAt, got %v", err)
		}
	})

	t.Run("failed outcome requires completed timestamp", func(t *testing.T) {
		t.Parallel()

		// Step 1: ゼロ時刻で失敗 outcome を保存すると時系列監査不能になるため拒否する。
		_, err := NewOperatorAuditEvent().MarkFailed("duplicate_email", time.Time{})
		if !errors.Is(err, ErrInvalidOperatorAuditCompletedAt) {
			t.Fatalf("expected ErrInvalidOperatorAuditCompletedAt, got %v", err)
		}
	})
}

func assertCompletedAudit(
	t *testing.T,
	event OperatorAuditEvent,
	wantOutcome OperatorAuditOutcome,
	wantCode StableErrorCode,
) {
	t.Helper()

	// Step 1: outcome が期待どおりの最終状態になったことを検証する。
	if event.Outcome() != wantOutcome {
		t.Fatalf("outcome = %q, want %q", event.Outcome(), wantOutcome)
	}

	// Step 2: stable error code は failed の場合だけ設定されることを検証する。
	if event.StableErrorCode() != wantCode {
		t.Fatalf("stable error code = %q, want %q", event.StableErrorCode(), wantCode)
	}

	// Step 3: completedAt は nil ではなく、UTC に正規化されて保存されることを検証する。
	completedAt := event.CompletedAt()
	if completedAt == nil {
		t.Fatal("completedAt is nil")
	}
	if !completedAt.Equal(operatorAuditCompletedAt.UTC()) {
		t.Fatalf("completedAt = %v, want %v", completedAt, operatorAuditCompletedAt.UTC())
	}
}
