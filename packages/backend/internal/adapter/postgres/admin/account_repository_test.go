package admin

import (
	"os"
	"strings"
	"testing"
)

func TestAdminAccountRepositoryUsesExplicitProductAndAdminSchemas(t *testing.T) {
	t.Parallel()

	// Step 1: GORM record の TableName を直接検査し、search_path 依存ではなく schema 名を明示していることを確認する。
	if got := (accountRecord{}).TableName(); got != "public.accounts" {
		t.Fatalf("account root table must be public.accounts, got %q", got)
	}
	if got := (accountSettingRecord{}).TableName(); got != "public.account_settings" {
		t.Fatalf("account setting table must be public.account_settings, got %q", got)
	}
	if got := (accountSummaryRecord{}).TableName(); got != "admin_view.account_summaries" {
		t.Fatalf("account summary view must be admin_view.account_summaries, got %q", got)
	}
	if got := (auditTargetRecord{}).TableName(); got != "admin.audit_events" {
		t.Fatalf("audit target table must be admin.audit_events, got %q", got)
	}
}

func TestAdminAccountRepositoryBoundaryIsSingleTransaction(t *testing.T) {
	t.Parallel()

	// Step 1: repository source を静的に読み込み、実装が application port と GORM transaction 境界を持つことを検査する。
	source := readAdminAccountRepositorySource(t)

	// Step 2: Product Account root と Admin audit target の両方を同じ repository source 内で扱うことを確認する。
	assertAdminAccountRepositoryContainsAll(t, source,
		"adminapplication.AdminAccountRepository",
		"CreateAccountWithAuditTarget",
		"Transaction(func(tx *gorm.DB) error",
		"createAccountRoot(ctx, tx, record)",
		"bindAuditTarget(ctx, tx, record)",
		"public.accounts",
		"public.account_settings",
		"admin.audit_events",
	)

	// Step 3: repository が Product persistence adapter へ迂回せず、Admin package 内の port 実装として閉じていることを確認する。
	assertAdminAccountRepositoryNotContainsAny(t, source,
		"internal/adapter/postgres/product",
		"NewGormAccountAuthRepository",
		"NewGormAccountSettingRepository",
	)
}

func TestAdminAccountRepositoryDoesNotInlineAccountDomainRules(t *testing.T) {
	t.Parallel()

	// Step 1: repository が Account domain constructor の代替実装を持たず、構築済み domain.Account の snapshot だけを保存していることを確認する。
	source := readAdminAccountRepositorySource(t)
	assertAdminAccountRepositoryContainsAll(t, source,
		"record.Account.Email().String()",
		"record.Account.Status().String()",
		"record.Account.Setting().Locale().String()",
		"record.Account.SessionRevokedAfter()",
	)

	// Step 2: email 正規化や lifecycle 初期値の決定に使う domain constructor / enum を repository が直接呼ばないことを確認する。
	assertAdminAccountRepositoryNotContainsAny(t, source,
		"NewAccountEmail(",
		"strings.ToLower(",
		"AccountStatusActive",
		"NewAdminCreatedAccount(",
		"Suspend(",
		"Restore(",
	)
}

// [ADMIN-CONSOLE-BE-S084] Admin account search repository は parameter binding を使い、unsafe raw query を使わない。
func TestAdminAccountSearchRepositoryUsesParameterizedQueries(t *testing.T) {
	t.Parallel()

	// Step 1: repository source を静的に読み込み、search query が GORM parameter binding の形を保つことを検査する。
	source := readAdminAccountRepositorySource(t)
	assertAdminAccountRepositoryContainsAll(t, source,
		"SearchAccounts(ctx context.Context, query adminapplication.AdminAccountSearchQuery)",
		"Where(\"email ILIKE ?\", accountEmailSearchPattern(query.Email))",
		"Where(\"id < ?\", query.Cursor)",
		"Order(\"id DESC\").Limit(int(query.Limit + 1)).Find(&records)",
	)

	// Step 2: raw query API や SQL fragment の動的生成に戻っていないことを確認し、S084 の repository boundary を固定する。
	assertAdminAccountRepositoryNotContainsAny(t, source,
		".Raw(",
		".Exec(",
		"fmt.Sprintf(",
	)
}

func readAdminAccountRepositorySource(t *testing.T) string {
	t.Helper()

	// Step 1: gosec G304 を避けるため、読み込み対象を package-local の固定 file 名に限定する。
	content, err := os.ReadFile("account_repository.go")
	if err != nil {
		t.Fatalf("read admin account repository: %v", err)
	}

	// Step 2: assertion helper が文字列断片を検査できるよう source 全体を返す。
	return string(content)
}

func assertAdminAccountRepositoryContainsAll(t *testing.T, content string, requiredValues ...string) {
	t.Helper()

	// Step 1: 必須断片を個別に確認し、欠落時に repository boundary のどの証跡が壊れたかを示す。
	for _, required := range requiredValues {
		if !strings.Contains(content, required) {
			t.Fatalf("admin account repository must contain %q", required)
		}
	}
}

func assertAdminAccountRepositoryNotContainsAny(t *testing.T, content string, forbiddenValues ...string) {
	t.Helper()

	// Step 1: 禁止断片を個別に確認し、Product repository 迂回や domain rule 再実装を早期に検出する。
	for _, forbidden := range forbiddenValues {
		if strings.Contains(content, forbidden) {
			t.Fatalf("admin account repository must not contain %q", forbidden)
		}
	}
}
