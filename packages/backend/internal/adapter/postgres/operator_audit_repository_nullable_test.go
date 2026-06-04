package postgres

import (
	"testing"
	"time"

	auditapplication "www-template/packages/backend/internal/application/audit"
)

// TestAuditEventRecordFromIntentSetsNullTargetForPendingIntent は pending audit intent で
// target_account_id が NULL として保存されることを検証する。
//
// pending intent では target account が未確定のため、空文字列ではなく NULL を挿入し、
// FK 制約 (target_account_id REFERENCES public.accounts(id)) を満たす。
func TestAuditEventRecordFromIntentSetsNullTargetForPendingIntent(t *testing.T) {
	t.Parallel()

	// Step 1: target ID が空の intent record を作成し、pending intent で target が未確定な状態を再現する。
	record := auditapplication.IntentRecord{
		OperatorID: "op-1",
		Action:     "accounts:create",
		TargetType: "account",
		TargetID:   "",
		RequestID:  "req-1",
		Outcome:    "pending",
		OccurredAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	// Step 2: auditEventRecordFromIntent で変換し、TargetAccountID が nil であることを確認する。
	result := auditEventRecordFromIntent(record, "audit-1")

	// Step 3: TargetAccountID が nil pointer であり、GORM が SQL NULL として INSERT することを確認する。
	if result.TargetAccountID != nil {
		t.Fatalf("pending intent with empty target must set TargetAccountID to nil, got %q", *result.TargetAccountID)
	}

	// Step 4: TargetAccountEmail も同様に nil であることを確認する。
	if result.TargetAccountEmail != nil {
		t.Fatalf("pending intent must set TargetAccountEmail to nil, got %q", *result.TargetAccountEmail)
	}
}

// TestAuditEventRecordFromIntentSetsPointerTargetForNonblankTarget は target ID が設定されている場合に
// TargetAccountID が pointer value として保存されることを検証する。
func TestAuditEventRecordFromIntentSetsPointerTargetForNonblankTarget(t *testing.T) {
	t.Parallel()

	// Step 1: target ID が設定されている intent record を作成する。
	record := auditapplication.IntentRecord{
		OperatorID: "op-1",
		Action:     "accounts:create",
		TargetType: "account",
		TargetID:   "account-123",
		RequestID:  "req-1",
		Outcome:    "pending",
		OccurredAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	// Step 2: auditEventRecordFromIntent で変換する。
	result := auditEventRecordFromIntent(record, "audit-1")

	// Step 3: TargetAccountID が pointer value として設定されることを確認する。
	if result.TargetAccountID == nil {
		t.Fatal("nonblank target must set TargetAccountID to pointer value")
	}
	if *result.TargetAccountID != "account-123" {
		t.Fatalf("TargetAccountID must be 'account-123', got %q", *result.TargetAccountID)
	}
}

// TestAuditEventRecordToApplicationRecordConvertsNullTargetToEmptyString は DB から読み取った
// NULL target_account_id が application DTO の空文字に変換されることを検証する。
func TestAuditEventRecordToApplicationRecordConvertsNullTargetToEmptyString(t *testing.T) {
	t.Parallel()

	// Step 1: TargetAccountID が nil の DB record を作成する。
	record := auditEventRecord{
		ID:              "audit-1",
		OperatorID:      "op-1",
		TargetAccountID: nil,
		Action:          "accounts:create",
		Outcome:         "pending",
		Metadata:        "{}",
		CreatedAt:       time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	// Step 2: application DTO に変換し、TargetID が空文字列になることを確認する。
	result := record.toApplicationRecord()

	// Step 3: application 層には空文字として渡され、domain reconstitution が既存 rule で解釈できる。
	if result.TargetID != "" {
		t.Fatalf("null TargetAccountID must convert to empty string in application DTO, got %q", result.TargetID)
	}
}

// TestAuditEventRecordToApplicationRecordConvertsPointerTargetToString は DB から読み取った
// non-null target_account_id が application DTO の文字列に変換されることを検証する。
func TestAuditEventRecordToApplicationRecordConvertsPointerTargetToString(t *testing.T) {
	t.Parallel()

	// Step 1: TargetAccountID が設定されている DB record を作成する。
	targetID := "account-123"
	record := auditEventRecord{
		ID:              "audit-1",
		OperatorID:      "op-1",
		TargetAccountID: &targetID,
		Action:          "accounts:create",
		Outcome:         "succeeded",
		Metadata:        "{}",
		CreatedAt:       time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	// Step 2: application DTO に変換し、TargetID が文字列として設定されることを確認する。
	result := record.toApplicationRecord()

	// Step 3: application 層には元の文字列として渡される。
	if result.TargetID != "account-123" {
		t.Fatalf("non-null TargetAccountID must convert to string in application DTO, got %q", result.TargetID)
	}
}

// TestAuditCompletionUpdatesSetsNullStableErrorCodeForSuccess は success completion で
// stable_error_code が NULL として保存されることを検証する。
//
// DB CHECK 制約 (outcome = 'succeeded' AND stable_error_code IS NULL) を満たすため、
// 空文字列ではなく nil pointer を返す必要がある。
func TestAuditCompletionUpdatesSetsNullStableErrorCodeForSuccess(t *testing.T) {
	t.Parallel()

	// Step 1: success completion record を作成する。
	record := auditapplication.CompletionRecord{
		AuditID:         "audit-1",
		Outcome:         "succeeded",
		StableErrorCode: "",
		CompletedAt:     time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	// Step 2: auditCompletionUpdates で変換する。
	updates := auditCompletionUpdates(record)

	// Step 3: stable_error_code が nil であることを確認する。
	// map[string]any に格納された nil *string は interface としては非 nil になるため、
	// 型アサーションで underlying value が nil pointer であることを検証する。
	stableErrorCode, ok := updates["stable_error_code"]
	if !ok {
		t.Fatal("updates must contain stable_error_code key")
	}
	if stableErrorCode == nil {
		// interface 自体が nil の場合は期待する動作である。
		return
	}
	// interface が非 nil の場合、underlying value が nil pointer であることを確認する。
	codePtr, isPtr := stableErrorCode.(*string)
	if !isPtr {
		t.Fatalf("success completion must set stable_error_code to *string, got %T", stableErrorCode)
	}
	if codePtr != nil {
		t.Fatalf("success completion must set stable_error_code to nil pointer, got %q", *codePtr)
	}
}

// TestAuditCompletionUpdatesSetsPointerStableErrorCodeForFailure は failure completion で
// stable_error_code が pointer value として保存されることを検証する。
func TestAuditCompletionUpdatesSetsPointerStableErrorCodeForFailure(t *testing.T) {
	t.Parallel()

	// Step 1: failure completion record を作成する。
	record := auditapplication.CompletionRecord{
		AuditID:         "audit-1",
		Outcome:         "failed",
		StableErrorCode: "duplicate_email",
		CompletedAt:     time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	// Step 2: auditCompletionUpdates で変換する。
	updates := auditCompletionUpdates(record)

	// Step 3: stable_error_code が pointer value として設定されることを確認する。
	stableErrorCode, ok := updates["stable_error_code"]
	if !ok {
		t.Fatal("updates must contain stable_error_code key")
	}
	if stableErrorCode == nil {
		t.Fatal("failure completion must set stable_error_code to pointer value")
	}
	code, ok := stableErrorCode.(*string)
	if !ok {
		t.Fatalf("stable_error_code must be *string, got %T", stableErrorCode)
	}
	if *code != "duplicate_email" {
		t.Fatalf("stable_error_code must be 'duplicate_email', got %q", *code)
	}
}
