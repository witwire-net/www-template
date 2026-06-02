package postgres

import (
	"os"
	"strings"
	"testing"
)

func TestAccountRepositoryUsesExplicitProductAndAdminSchemas(t *testing.T) {
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

func TestAccountRepositoryBoundaryIsSingleTransaction(t *testing.T) {
	t.Parallel()

	// Step 1: repository source を静的に読み込み、実装が application port と GORM transaction 境界を持つことを検査する。
	source := readAccountRepositorySource(t)

	// Step 2: Product Account root と Admin audit target の両方を同じ repository source 内で扱うことを確認する。
	assertAccountRepositoryContainsAll(t, source,
		"accountsapplication.AccountRepository",
		"CreateAccountWithAuditTarget",
		"Transaction(func(tx *gorm.DB) error",
		"createAccountRoot(ctx, tx, record)",
		"bindAuditTarget(ctx, tx, record)",
		"public.accounts",
		"public.account_settings",
		"admin.audit_events",
	)

	// Step 3: repository が Product persistence adapter へ迂回せず、Admin package 内の port 実装として閉じていることを確認する。
	assertAccountRepositoryNotContainsAny(t, source,
		"internal/adapter/postgres/product",
		"NewGormAccountAuthRepository",
		"NewGormAccountSettingRepository",
	)
}

func TestPostgresRepositoriesUseCapabilityPaths(t *testing.T) {
	t.Parallel()

	// Step 1: [ADMIN-CONSOLE-BE-S097] public account aggregate repository と audit repository が Admin surface package ではなく capability path に分かれていることを固定する。
	accountSource := readAccountRepositorySource(t)
	auditSource, err := os.ReadFile("../audit/repository.go")
	if err != nil {
		t.Fatalf("read audit capability repository: %v", err)
	}

	// Step 2: account repository は accounts application port だけを実装し、audit repository owner は audit capability port だけを実装する。
	assertAccountRepositoryContainsAll(t, accountSource, "accountsapplication.AccountRepository", "accountsapplication.AccountSearchRepository")
	if !strings.Contains(string(auditSource), "auditapplication.Repository") {
		t.Fatalf("[ADMIN-CONSOLE-BE-S097] audit repository must implement audit capability port")
	}
	assertAccountRepositoryNotContainsAny(t, accountSource, "internal/application/admin")
	if strings.Contains(string(auditSource), "internal/application/admin") {
		t.Fatalf("[ADMIN-CONSOLE-BE-S097] audit repository must not import Admin surface application package")
	}
}

func TestAccountRepositoryDoesNotInlineAccountDomainRules(t *testing.T) {
	t.Parallel()

	// Step 1: repository が Account domain constructor の代替実装を持たず、構築済み domain.Account の snapshot だけを保存していることを確認する。
	source := readAccountRepositorySource(t)
	assertAccountRepositoryContainsAll(t, source,
		"record.Account.Email().String()",
		"record.Account.Status().String()",
		"record.Account.Setting().Locale().String()",
		"record.Account.SessionRevokedAfter()",
	)

	// Step 2: email 正規化や lifecycle 初期値の決定に使う domain constructor / enum を repository が直接呼ばないことを確認する。
	assertAccountRepositoryNotContainsAny(t, source,
		"NewAccountEmail(",
		"strings.ToLower(",
		"AccountStatusActive",
		"NewCreatedAccount(",
		"Suspend(",
		"Restore(",
	)
}

// [ADMIN-CONSOLE-BE-S084] Admin account search repository は parameter binding を使い、unsafe raw query を使わない。
func TestAccountSearchRepositoryUsesParameterizedQueries(t *testing.T) {
	t.Parallel()

	// Step 1: repository source を静的に読み込み、search query が GORM parameter binding の形を保つことを検査する。
	source := readAccountRepositorySource(t)
	assertAccountRepositoryContainsAll(t, source,
		"SearchAccounts(ctx context.Context, query accountsapplication.AccountSearchQuery)",
		"Where(\"email ILIKE ?\", accountEmailSearchPattern(query.Email))",
		"Where(\"id < ?\", query.Cursor)",
		"Order(\"id DESC\").Limit(int(query.Limit + 1)).Find(&records)",
	)

	// Step 2: raw query API や SQL fragment の動的生成に戻っていないことを確認し、S084 の repository boundary を固定する。
	assertAccountRepositoryNotContainsAny(t, source,
		".Raw(",
		".Exec(",
		"fmt.Sprintf(",
	)
}

func readAccountRepositorySource(t *testing.T) string {
	t.Helper()

	// Step 1: gosec G304 を避けるため、読み込み対象を package-local の固定 file 名に限定する。
	content, err := os.ReadFile("account_repository.go")
	if err != nil {
		t.Fatalf("read admin account repository: %v", err)
	}

	// Step 2: assertion helper が文字列断片を検査できるよう source 全体を返す。
	return string(content)
}

func assertAccountRepositoryContainsAll(t *testing.T, content string, requiredValues ...string) {
	t.Helper()

	// Step 1: 必須断片を個別に確認し、欠落時に repository boundary のどの証跡が壊れたかを示す。
	for _, required := range requiredValues {
		if !strings.Contains(content, required) {
			t.Fatalf("admin account repository must contain %q", required)
		}
	}
}

func assertAccountRepositoryNotContainsAny(t *testing.T, content string, forbiddenValues ...string) {
	t.Helper()

	// Step 1: 禁止断片を個別に確認し、Product repository 迂回や domain rule 再実装を早期に検出する。
	for _, forbidden := range forbiddenValues {
		if strings.Contains(content, forbidden) {
			t.Fatalf("admin account repository must not contain %q", forbidden)
		}
	}
}
