package application

import (
	"os"
	"strings"
	"testing"
)

func TestAdminAccountRepositoryPortExposesOnlyApplicationAndDomainTypes(t *testing.T) {
	t.Parallel()

	// Step 1: port source を固定 file から読み込み、application 境界に adapter/generated 型が入っていないことを静的に検査する。
	content, err := os.ReadFile("account_repository.go")
	if err != nil {
		t.Fatalf("read admin account repository port: %v", err)
	}
	source := string(content)

	// Step 2: application port と concrete domain Account が境界になっていることを確認する。
	assertAdminAccountPortContainsAll(t, source,
		"type AdminAccountRepository interface",
		"CreateAccountWithAuditTarget(ctx context.Context, record AdminAccountCreationRecord) (AdminAccountRecord, error)",
		"Account         domain.Account",
		"AuditCompletion AdminAuditCompletionRecord",
	)

	// Step 3: GORM/Gin/generated/adapter 型を public port に混ぜないことを確認し、Clean Architecture 境界を保つ。
	assertAdminAccountPortNotContainsAny(t, source,
		"gorm.io/",
		"gin.Context",
		"internal/generated",
		"internal/adapter",
		"*gorm.DB",
	)
}

func assertAdminAccountPortContainsAll(t *testing.T, content string, requiredValues ...string) {
	t.Helper()

	// Step 1: 必須断片を個別に確認し、port と domain 境界の証跡が失われていないかを示す。
	for _, required := range requiredValues {
		if !strings.Contains(content, required) {
			t.Fatalf("admin account repository port must contain %q", required)
		}
	}
}

func assertAdminAccountPortNotContainsAny(t *testing.T, content string, forbiddenValues ...string) {
	t.Helper()

	// Step 1: 禁止断片を個別に確認し、application port purity violation を早期に検出する。
	for _, forbidden := range forbiddenValues {
		if strings.Contains(content, forbidden) {
			t.Fatalf("admin account repository port must not contain %q", forbidden)
		}
	}
}
