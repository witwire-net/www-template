package postgres

import (
	"context"
	"crypto/rand"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
	"time"

	"gorm.io/gorm"

	accountsapplication "www-template/packages/backend/internal/application/accounts"
	auditapplication "www-template/packages/backend/internal/application/audit"
	domain "www-template/packages/backend/internal/domain"
	"www-template/packages/backend/internal/platform/id"
)

// testPostgresOwnerURL はテスト用 DB の作成・削除に使う owner/maintenance DSN である。
// 環境変数 BACKEND_POSTGRES_TEST_OWNER_URL が未設定の場合、統合テストは skip される。
// CI では postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable を設定する。
const testPostgresOwnerURLEnv = "BACKEND_POSTGRES_TEST_OWNER_URL"

// safeIdentifierPattern はテスト生成の DB 名・role 名として安全な文字列だけを許可する。
// 英小文字・数字・underscore のみを許可し、SQL injection や意図しない resource 削除を防ぐ。
var safeIdentifierPattern = regexp.MustCompile(`^[a-z][a-z0-9_]{0,50}$`)

// testDBSuffix はテスト実行ごとにユニークな DB 名・role 名を生成するための接尾辞である。
// crypto/rand.Reader を entropy に使い、collision を最小化する。
func testDBSuffix(t *testing.T) string {
	t.Helper()

	// Step 1: ULID を使ってユニークな接尾辞を生成し、テスト実行間の collision を防ぐ。
	// crypto/rand.Reader を entropy として明示的に渡し、nil entropy による非決定的動作を防ぐ。
	ulid, err := id.NewULID(time.Now(), rand.Reader)
	if err != nil {
		t.Fatalf("generate test DB suffix ULID: %v", err)
	}

	// Step 2: ULID の末尾 12 文字だけを使い、DB 名の長さ制限 (63 bytes) を満たす。
	return strings.ToLower(ulid[len(ulid)-12:])
}

// validateIdentifier はテスト生成の識別子が safe pattern に従うことを検証する。
// 不正な文字列が DB/role 名として使われることを防ぐ。
func validateIdentifier(t *testing.T, name string) {
	t.Helper()

	if !safeIdentifierPattern.MatchString(name) {
		t.Fatalf("unsafe identifier %q: must match %s", name, safeIdentifierPattern)
	}
}

// testInfra は統合テストで使う DB 接続と cleanup 情報を保持する構造体である。
type testInfra struct {
	// ownerDB はテスト用 DB の作成・削除に使う owner 接続である。
	ownerDB *sql.DB

	// testOwnerDB はテスト用 DB への owner 接続である。operator seed など owner 権限が必要な操作に使う。
	testOwnerDB *sql.DB

	// runtimeDB は Admin runtime role として接続した GORM handle である。
	runtimeDB *gorm.DB

	// testDBName はテスト生成の DB 名である。cleanup で DROP DATABASE に使う。
	testDBName string

	// testRoleName はテスト生成の Admin runtime login role 名である。cleanup で DROP ROLE に使う。
	testRoleName string

	// ownerURL は owner 接続の DSN である。cleanup 後の再接続に使う。
	ownerURL string
}

// auditQueryResult は GORM の Raw query で複数列をスキャンするための構造体である。
// GORM の .Scan() は単一の struct pointer を受け取るため、複数列の場合は struct で受ける。
type auditQueryResult struct {
	TargetAccountID    sql.NullString `gorm:"column:target_account_id"`
	TargetAccountEmail sql.NullString `gorm:"column:target_account_email"`
	Outcome            string         `gorm:"column:outcome"`
	StableErrorCode    sql.NullString `gorm:"column:stable_error_code"`
	CompletedAt        sql.NullTime   `gorm:"column:completed_at"`
}

// setupTestPostgres は統合テスト用の isolated DB 環境を構築する。
//
// 処理内容:
//   - BACKEND_POSTGRES_TEST_OWNER_URL 環境変数から owner DSN を読み、未設定なら skip する。
//   - ユニークなテスト DB を作成し、backend の全 migration を適用する。
//   - ユニークな Admin runtime login role を作成し、password を設定し、admin_console_write を付与する。
//   - runtime DB に Admin role として接続した GORM handle を返す。
//   - t.Cleanup で DB と role を安全に削除する。
func setupTestPostgres(t *testing.T) *testInfra {
	t.Helper()

	// Step 1: owner DSN を環境変数から読み、未設定なら統合テストを skip する。
	ownerURL := os.Getenv(testPostgresOwnerURLEnv)
	if ownerURL == "" {
		t.Skipf("skipping integration test: %s is not set", testPostgresOwnerURLEnv)
	}

	// Step 2: owner DB に接続し、テスト用 DB の作成・削除に使う。
	// GORM の OpenDatabase を使い、GORM 内部で登録された postgres driver を経由する。
	ownerGORM, err := OpenDatabase(ownerURL)
	if err != nil {
		t.Fatalf("open owner database: %v", err)
	}
	ownerDB, err := ownerGORM.DB()
	if err != nil {
		t.Fatalf("get owner sql.DB: %v", err)
	}

	// Step 3: owner 接続の疎通確認を行い、接続失敗を早期に検出する。
	if err := ownerDB.PingContext(context.Background()); err != nil {
		_ = ownerDB.Close()
		t.Fatalf("ping owner database: %v", err)
	}

	// Step 4: ユニークな DB 名と role 名を生成し、テスト実行間の collision を防ぐ。
	suffix := testDBSuffix(t)
	testDBName := "test_be_" + suffix
	testRoleName := "test_admin_" + suffix
	testRolePassword := "test_pass_" + suffix
	validateIdentifier(t, testDBName)
	validateIdentifier(t, testRoleName)

	// Step 5: cleanup を登録し、テスト失敗時にも DB と role を安全に削除する。
	t.Cleanup(func() {
		cleanupTestPostgres(t, ownerURL, testDBName, testRoleName)
	})

	// Step 6: テスト用 DB を作成する。
	ctx := context.Background()
	if _, err := ownerDB.ExecContext(ctx, fmt.Sprintf("CREATE DATABASE %s", safeQuoteIdentifier(testDBName))); err != nil {
		_ = ownerDB.Close()
		t.Fatalf("create test database %s: %v", testDBName, err)
	}

	// Step 7: テスト用 DB に migration を適用する。
	testDBURL := replaceDatabaseInURL(ownerURL, testDBName)
	applyMigrations(t, testDBURL)

	// Step 8: Admin runtime login role を作成し、password を設定し、admin_console_write を付与する。
	createTestAdminRole(t, ownerDB, testRoleName, testRolePassword)

	// Step 9: テスト用 DB への owner 接続を開く。operator seed など owner 権限が必要な操作に使う。
	testOwnerGORM, err := OpenDatabase(testDBURL)
	if err != nil {
		t.Fatalf("open test owner database: %v", err)
	}
	testOwnerDB, err := testOwnerGORM.DB()
	if err != nil {
		t.Fatalf("get test owner sql.DB: %v", err)
	}

	// Step 10: runtime GORM handle を Admin role として接続する。
	runtimeDB := openTestRuntimeDB(t, testDBURL, testRoleName, testRolePassword)

	return &testInfra{
		ownerDB:      ownerDB,
		testOwnerDB:  testOwnerDB,
		runtimeDB:    runtimeDB,
		testDBName:   testDBName,
		testRoleName: testRoleName,
		ownerURL:     ownerURL,
	}
}

// cleanupTestPostgres はテスト生成の DB と role を安全に削除する。
//
// 安全条件:
//   - 削除対象はテスト生成の識別子だけであり、safeIdentifierPattern で検証済み。
//   - FORCE で接続を切断してから DROP DATABASE し、テスト中の接続残留を防ぐ。
//   - role は DB 削除後に DROP し、依存関係の問題を避ける。
func cleanupTestPostgres(t *testing.T, ownerURL, testDBName, testRoleName string) {
	t.Helper()

	// Step 1: cleanup 用の owner 接続を開く。テスト中の接続状態に依存しない。
	cleanupGORM, err := OpenDatabase(ownerURL)
	if err != nil {
		t.Logf("cleanup: open owner database: %v", err)
		return
	}
	cleanupDB, err := cleanupGORM.DB()
	if err != nil {
		t.Logf("cleanup: get owner sql.DB: %v", err)
		return
	}
	defer func() { _ = cleanupDB.Close() }()

	ctx := context.Background()

	// Step 2: テスト用 DB への全接続を強制切断し、DROP DATABASE が失敗しないようにする。
	_, _ = cleanupDB.ExecContext(ctx,
		fmt.Sprintf("SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = '%s'", testDBName))

	// Step 3: テスト用 DB を削除する。
	if _, err := cleanupDB.ExecContext(ctx, fmt.Sprintf("DROP DATABASE IF EXISTS %s", safeQuoteIdentifier(testDBName))); err != nil {
		t.Logf("cleanup: drop database %s: %v", testDBName, err)
	}

	// Step 4: テスト用 role を削除する。DB 削除後のため依存関係の問題を避ける。
	if _, err := cleanupDB.ExecContext(ctx, fmt.Sprintf("DROP ROLE IF EXISTS %s", safeQuoteIdentifier(testRoleName))); err != nil {
		t.Logf("cleanup: drop role %s: %v", testRoleName, err)
	}
}

// applyMigrations はテスト用 DB に backend の全 migration を適用する。
//
// migration ファイルはファイル名の辞書順で適用され、DO block/function を含むため
// ファイル全体を 1 つの SQL として実行する。セミコロンで分割しない。
func applyMigrations(t *testing.T, dbURL string) {
	t.Helper()

	// Step 1: migration ディレクトリの全 .up.sql ファイルを読み込む。
	migrationDir := filepath.Join("..", "..", "..", "db", "migrations")
	entries, err := os.ReadDir(migrationDir)
	if err != nil {
		t.Fatalf("read migration directory: %v", err)
	}

	// Step 2: ファイル名でソートし、辞書順で適用する。
	var upFiles []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".up.sql") {
			upFiles = append(upFiles, entry.Name())
		}
	}
	sort.Strings(upFiles)

	// Step 3: migration 用の DB 接続を開く。
	migrationGORM, err := OpenDatabase(dbURL)
	if err != nil {
		t.Fatalf("open migration database: %v", err)
	}
	db, err := migrationGORM.DB()
	if err != nil {
		t.Fatalf("get migration sql.DB: %v", err)
	}
	defer func() { _ = db.Close() }()

	ctx := context.Background()

	// Step 4: 各 migration ファイルを全体として実行し、DO block/function の途中分割を防ぐ。
	// gosec G304: migration ディレクトリは backend-owned の固定 path であり、テスト生成の識別子ではない。
	for _, fileName := range upFiles { // #nosec G304
		content, err := os.ReadFile(filepath.Join(migrationDir, fileName))
		if err != nil {
			t.Fatalf("read migration %s: %v", fileName, err)
		}

		if _, err := db.ExecContext(ctx, string(content)); err != nil {
			t.Fatalf("apply migration %s: %v", fileName, err)
		}
	}
}

// createTestAdminRole はテスト用の Admin runtime login role を作成し、password を設定し、admin_console_write を付与する。
//
// 安全条件:
//   - role 名は safeIdentifierPattern で検証済み。
//   - LOGIN 権限と password を持ち、テスト中の DB 接続に使う。
//   - admin_console_write を継承し、Admin schema への INSERT/UPDATE が可能になる。
func createTestAdminRole(t *testing.T, ownerDB *sql.DB, roleName, password string) {
	t.Helper()

	ctx := context.Background()

	// Step 1: LOGIN 権限と password を持つ role を作成する。
	if _, err := ownerDB.ExecContext(ctx, fmt.Sprintf("CREATE ROLE %s LOGIN PASSWORD %s",
		safeQuoteIdentifier(roleName), safeQuoteSQLString(password))); err != nil {
		t.Fatalf("create test admin role %s: %v", roleName, err)
	}

	// Step 2: admin_console_write を付与し、Admin schema への INSERT/UPDATE を許可する。
	if _, err := ownerDB.ExecContext(ctx, fmt.Sprintf("GRANT admin_console_write TO %s", safeQuoteIdentifier(roleName))); err != nil {
		t.Fatalf("grant admin_console_write to %s: %v", roleName, err)
	}
}

// openTestRuntimeDB は Admin runtime role としてテスト用 DB に接続した GORM handle を返す。
func openTestRuntimeDB(t *testing.T, dbURL, roleName, password string) *gorm.DB {
	t.Helper()

	// Step 1: role 名と password を DSN に含め、Admin runtime role として接続する。
	runtimeURL := replaceRoleAndPasswordInURL(dbURL, roleName, password)
	db, err := OpenDatabase(runtimeURL)
	if err != nil {
		t.Fatalf("open runtime database with role %s: %v", roleName, err)
	}

	// Step 2: 接続の疎通確認を行い、role 権限の問題を早期に検出する。
	if err := PingDatabase(context.Background(), db); err != nil {
		t.Fatalf("ping runtime database with role %s: %v", roleName, err)
	}

	return db
}

// safeQuoteIdentifier は SQL 識別子を安全にクォートする。
// PostgreSQL の標準的なダブルクォートエスケープを使い、SQL injection を防ぐ。
func safeQuoteIdentifier(name string) string {
	// Step 1: ダブルクォート内のダブルクォートをエスケープし、SQL injection を防ぐ。
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

// safeQuoteSQLString は SQL 文字列リテラルを安全にクォートする。
// PostgreSQL の標準的なシングルクォートエスケープを使い、SQL injection を防ぐ。
func safeQuoteSQLString(s string) string {
	return `'` + strings.ReplaceAll(s, `'`, `''`) + `'`
}

// replaceDatabaseInURL は DSN のデータベース名を指定の値に置き換える。
// postgres://user:pass@host:port/oldDB?params 形式の URL を扱う。
func replaceDatabaseInURL(baseURL, newDB string) string {
	// Step 1: URL のパス部分 (/oldDB) を新しい DB 名に置き換える。
	idx := strings.LastIndex(baseURL, "/")
	if idx < 0 {
		return baseURL
	}
	queryIdx := strings.Index(baseURL[idx:], "?")
	if queryIdx < 0 {
		return baseURL[:idx+1] + newDB
	}
	return baseURL[:idx+1] + newDB + baseURL[idx+queryIdx:]
}

// replaceRoleAndPasswordInURL は DSN の user:pass 部分を指定の role 名と password に置き換える。
// postgres://oldUser:oldPass@host:port/db?params 形式の URL を扱う。
func replaceRoleAndPasswordInURL(baseURL, newRole, newPassword string) string {
	// Step 1: "://" の後の user:pass 部分を新しい role 名と password に置き換える。
	schemeEnd := strings.Index(baseURL, "://")
	if schemeEnd < 0 {
		return baseURL
	}
	afterScheme := baseURL[schemeEnd+3:]
	atIdx := strings.Index(afterScheme, "@")
	if atIdx < 0 {
		return baseURL
	}
	// Step 2: user:pass を新しい role:pass に置き換える。
	return baseURL[:schemeEnd+3] + newRole + ":" + newPassword + afterScheme[atIdx:]
}

// TestAdminAuditPersistenceSatisfiesPostgresConstraints は PostgreSQL の実際の制約に対して
// audit intent と completion が正しく永続化されることを検証する統合テストである。
//
// 検証内容:
//   - pending audit intent で target_account_id が NULL として保存され、FK 制約を満たす。
//   - success completion で stable_error_code が NULL として保存され、CHECK 制約を満たす。
//   - failure completion で stable_error_code が文字列として保存され、CHECK 制約を満たす。
//   - Admin runtime role として実際の DB 権限で INSERT/UPDATE が成功する。
func TestAdminAuditPersistenceSatisfiesPostgresConstraints(t *testing.T) {
	// Step 1: 統合テスト用の isolated DB 環境を構築する。
	infra := setupTestPostgres(t)
	defer func() {
		if infra.ownerDB != nil {
			_ = infra.ownerDB.Close()
		}
		if infra.testOwnerDB != nil {
			_ = infra.testOwnerDB.Close()
		}
	}()

	ctx := context.Background()

	// Step 2: OperatorAuditRepository を Admin runtime role の GORM handle で構築する。
	auditRepo := NewOperatorAuditRepository(infra.runtimeDB)

	// Step 3: テスト用 DB の admin.operators に operator を seed する。
	// audit_events の operator_id は FK 制約 (operator_id REFERENCES admin.operators(id)) を持つため、
	// audit intent を insert する前に operator が存在していなければならない。
	createTestOperator(t, infra.testOwnerDB, "test-operator-1")

	// Step 4: pending audit intent を作成し、target_account_id が NULL として保存されることを確認する。
	intentTime := time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)
	intentRecord := auditapplication.IntentRecord{
		OperatorID:  "test-operator-1",
		Action:      "accounts:create",
		TargetType:  "account",
		TargetID:    "",
		RequestID:   "test-request-1",
		DetailsJSON: `{"requested_email":"test@example.com"}`,
		Outcome:     "pending",
		OccurredAt:  intentTime,
	}

	// Step 5: RecordAuditIntent を実行し、FK 制約違反が発生しないことを確認する。
	storedIntent, err := auditRepo.RecordAuditIntent(ctx, intentRecord)
	if err != nil {
		t.Fatalf("RecordAuditIntent must succeed with NULL target_account_id: %v", err)
	}

	// Step 6: 保存された intent を DB から直接確認し、target_account_id が NULL であることを検証する。
	var pendingTargetID sql.NullString
	if err := infra.runtimeDB.WithContext(ctx).Raw(
		"SELECT target_account_id FROM admin.audit_events WHERE id = ?", storedIntent.AuditID,
	).Scan(&pendingTargetID).Error; err != nil {
		t.Fatalf("query stored intent: %v", err)
	}
	if pendingTargetID.Valid {
		t.Fatalf("pending intent must have NULL target_account_id, got %q", pendingTargetID.String)
	}

	// Step 7: success completion を実行し、CHECK 制約 (stable_error_code IS NULL) が満たされることを確認する。
	completionTime := time.Date(2025, 6, 1, 12, 0, 1, 0, time.UTC)
	completionRecord := auditapplication.CompletionRecord{
		AuditID:         storedIntent.AuditID,
		Outcome:         "succeeded",
		StableErrorCode: "",
		CompletedAt:     completionTime,
	}

	completedRecord, err := auditRepo.CompleteAudit(ctx, completionRecord)
	if err != nil {
		t.Fatalf("CompleteAudit must succeed with NULL stable_error_code: %v", err)
	}

	// Step 8: 保存された completion を DB から直接確認し、CHECK 制約が満たされていることを検証する。
	var result auditQueryResult
	if err := infra.runtimeDB.WithContext(ctx).Raw(
		"SELECT outcome, stable_error_code, completed_at FROM admin.audit_events WHERE id = ?", storedIntent.AuditID,
	).Scan(&result).Error; err != nil {
		t.Fatalf("query completed audit: %v", err)
	}

	// Step 9: outcome が succeeded であることを確認する。
	if result.Outcome != "succeeded" {
		t.Fatalf("outcome must be 'succeeded', got %q", result.Outcome)
	}

	// Step 10: stable_error_code が NULL であることを確認する。
	if result.StableErrorCode.Valid {
		t.Fatalf("succeeded outcome must have NULL stable_error_code, got %q", result.StableErrorCode.String)
	}

	// Step 11: completed_at が設定されていることを確認する。
	if !result.CompletedAt.Valid {
		t.Fatal("succeeded outcome must have non-NULL completed_at")
	}

	// Step 12: application DTO の変換も正しいことを確認する。
	if completedRecord.StableErrorCode != "" {
		t.Fatalf("succeeded completion must return empty StableErrorCode in application DTO, got %q", completedRecord.StableErrorCode)
	}
	if completedRecord.Outcome != "succeeded" {
		t.Fatalf("completion must return 'succeeded' outcome, got %q", completedRecord.Outcome)
	}
}

// TestAdminAuditFailureCompletionSatisfiesPostgresConstraints は failure completion で
// stable_error_code が文字列として保存され、CHECK 制約を満たすことを検証する。
func TestAdminAuditFailureCompletionSatisfiesPostgresConstraints(t *testing.T) {
	// Step 1: 統合テスト用の isolated DB 環境を構築する。
	infra := setupTestPostgres(t)
	defer func() {
		if infra.ownerDB != nil {
			_ = infra.ownerDB.Close()
		}
		if infra.testOwnerDB != nil {
			_ = infra.testOwnerDB.Close()
		}
	}()

	ctx := context.Background()

	// Step 2: OperatorAuditRepository を Admin runtime role の GORM handle で構築する。
	auditRepo := NewOperatorAuditRepository(infra.runtimeDB)

	// Step 3: テスト用 DB の admin.operators に operator を seed する。
	createTestOperator(t, infra.testOwnerDB, "test-operator-1")

	// Step 4: pending audit intent を作成する。
	intentTime := time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)
	intentRecord := auditapplication.IntentRecord{
		OperatorID:  "test-operator-1",
		Action:      "accounts:create",
		TargetType:  "account",
		TargetID:    "",
		RequestID:   "test-request-1",
		DetailsJSON: `{"requested_email":"test@example.com"}`,
		Outcome:     "pending",
		OccurredAt:  intentTime,
	}

	storedIntent, err := auditRepo.RecordAuditIntent(ctx, intentRecord)
	if err != nil {
		t.Fatalf("RecordAuditIntent must succeed: %v", err)
	}

	// Step 5: failure completion を実行し、CHECK 制約 (stable_error_code IS NOT NULL) が満たされることを確認する。
	completionTime := time.Date(2025, 6, 1, 12, 0, 1, 0, time.UTC)
	completionRecord := auditapplication.CompletionRecord{
		AuditID:         storedIntent.AuditID,
		Outcome:         "failed",
		StableErrorCode: "duplicate_email",
		CompletedAt:     completionTime,
	}

	completedRecord, err := auditRepo.CompleteAudit(ctx, completionRecord)
	if err != nil {
		t.Fatalf("CompleteAudit must succeed with non-NULL stable_error_code: %v", err)
	}

	// Step 6: 保存された failure completion を DB から直接確認する。
	var result auditQueryResult
	if err := infra.runtimeDB.WithContext(ctx).Raw(
		"SELECT outcome, stable_error_code FROM admin.audit_events WHERE id = ?", storedIntent.AuditID,
	).Scan(&result).Error; err != nil {
		t.Fatalf("query completed audit: %v", err)
	}

	// Step 7: outcome が failed であることを確認する。
	if result.Outcome != "failed" {
		t.Fatalf("outcome must be 'failed', got %q", result.Outcome)
	}

	// Step 8: stable_error_code が設定されていることを確認する。
	if !result.StableErrorCode.Valid {
		t.Fatal("failed outcome must have non-NULL stable_error_code")
	}
	if result.StableErrorCode.String != "duplicate_email" {
		t.Fatalf("stable_error_code must be 'duplicate_email', got %q", result.StableErrorCode.String)
	}

	// Step 9: application DTO の変換も正しいことを確認する。
	if completedRecord.StableErrorCode != "duplicate_email" {
		t.Fatalf("failed completion must return 'duplicate_email' in application DTO, got %q", completedRecord.StableErrorCode)
	}
}

// TestAdminAccountCreationViaRepositorySatisfiesPostgresConstraints は
// AccountManagementRepository.CreateAccountWithAuditTarget を通じた account creation が
// 実際の PostgreSQL 制約を満たすことを検証する統合テストである。
//
// 検証内容:
//   - pending audit intent が NULL target_account_id で保存される。
//   - CreateAccountWithAuditTarget が account root と audit target を同一 transaction で保存する。
//   - success completion が NULL stable_error_code で保存される。
//   - FK 制約 (target_account_id REFERENCES public.accounts(id)) が満たされる。
func TestAdminAccountCreationViaRepositorySatisfiesPostgresConstraints(t *testing.T) {
	// Step 1: 統合テスト用の isolated DB 環境を構築する。
	infra := setupTestPostgres(t)
	defer func() {
		if infra.ownerDB != nil {
			_ = infra.ownerDB.Close()
		}
		if infra.testOwnerDB != nil {
			_ = infra.testOwnerDB.Close()
		}
	}()

	ctx := context.Background()

	// Step 2: テスト用 DB の admin.operators に operator を seed する。
	createTestOperator(t, infra.testOwnerDB, "test-operator-1")

	// Step 3: pending audit intent を作成し、NULL target_account_id で保存されることを確認する。
	auditRepo := NewOperatorAuditRepository(infra.runtimeDB)
	storedIntent := createPendingAuditIntent(ctx, t, auditRepo)
	verifyPendingIntentHasNullTarget(ctx, t, infra.runtimeDB, storedIntent.AuditID)

	// Step 4: CreateAccountWithAuditTarget を実行し、account root と audit target を同一 transaction で保存する。
	accountRepo := NewAccountManagementRepository(infra.runtimeDB)
	created := executeAccountCreationWithAuditTarget(ctx, t, accountRepo, storedIntent.AuditID)

	// Step 5: 保存された結果を検証する。owner 接続で public.accounts の存在を確認する。
	verifyAccountCreationAuditResult(ctx, t, infra.runtimeDB, infra.testOwnerDB, storedIntent.AuditID, created.AccountID)
}

// createPendingAuditIntent は pending audit intent を作成し、保存済みの intent を返す。
func createPendingAuditIntent(ctx context.Context, t *testing.T, auditRepo *OperatorAuditRepository) auditapplication.Record {
	t.Helper()

	intentTime := time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)
	intentRecord := auditapplication.IntentRecord{
		OperatorID:  "test-operator-1",
		Action:      "accounts:create",
		TargetType:  "account",
		TargetID:    "",
		RequestID:   "test-request-1",
		DetailsJSON: `{"requested_email":"newuser@example.com"}`,
		Outcome:     "pending",
		OccurredAt:  intentTime,
	}

	storedIntent, err := auditRepo.RecordAuditIntent(ctx, intentRecord)
	if err != nil {
		t.Fatalf("RecordAuditIntent must succeed: %v", err)
	}
	return storedIntent
}

// executeAccountCreationWithAuditTarget は CreateAccountWithAuditTarget を実行し、作成された account を返す。
func executeAccountCreationWithAuditTarget(ctx context.Context, t *testing.T, accountRepo *AccountManagementRepository, auditID string) accountsapplication.AccountRecord {
	t.Helper()

	// Step 1: domain.Account を構築する。AccountID は canonical ULID 形式 (26文字 Crockford Base32) が必要である。
	rawULID, err := id.NewULID(time.Now(), rand.Reader)
	if err != nil {
		t.Fatalf("generate account ULID: %v", err)
	}
	accountID, err := domain.NewAccountID(rawULID)
	if err != nil {
		t.Fatalf("create account ID: %v", err)
	}
	email, err := domain.NewAccountEmail("newuser@example.com")
	if err != nil {
		t.Fatalf("create account email: %v", err)
	}
	account, err := domain.NewCreatedAccount(accountID, email)
	if err != nil {
		t.Fatalf("create domain account: %v", err)
	}

	// Step 2: success completion を構築する。
	completionTime := time.Date(2025, 6, 1, 12, 0, 1, 0, time.UTC)
	completionRecord := auditapplication.CompletionRecord{
		AuditID:         auditID,
		Outcome:         "succeeded",
		StableErrorCode: "",
		CompletedAt:     completionTime,
	}

	// Step 3: CreateAccountWithAuditTarget を実行する。
	created, err := accountRepo.CreateAccountWithAuditTarget(ctx, accountsapplication.AccountCreationRecord{
		Account:         account,
		AuditID:         auditID,
		AuditCompletion: completionRecord,
	})
	if err != nil {
		t.Fatalf("CreateAccountWithAuditTarget must succeed: %v", err)
	}

	return created
}

// verifyAccountCreationAuditResult は account creation の audit 結果を検証する。
// ownerDB は public.accounts の存在確認に使う。runtimeDB は admin.audit_events の確認に使う。
func verifyAccountCreationAuditResult(ctx context.Context, t *testing.T, runtimeDB *gorm.DB, ownerDB *sql.DB, auditID, expectedAccountID string) {
	t.Helper()

	// Step 1: audit target が更新されていることを runtimeDB から直接確認する。
	var result auditQueryResult
	if err := runtimeDB.WithContext(ctx).Raw(
		`SELECT target_account_id, target_account_email, outcome, stable_error_code, completed_at
		 FROM admin.audit_events WHERE id = ?`, auditID,
	).Scan(&result).Error; err != nil {
		t.Fatalf("query completed audit: %v", err)
	}

	// Step 2: target_account_id が作成された account ID であることを確認する。
	if !result.TargetAccountID.Valid {
		t.Fatal("completed audit must have non-NULL target_account_id")
	}
	if result.TargetAccountID.String != expectedAccountID {
		t.Fatalf("target_account_id must be %q, got %q", expectedAccountID, result.TargetAccountID.String)
	}

	// Step 3: target_account_email が設定されていることを確認する。
	if !result.TargetAccountEmail.Valid {
		t.Fatal("completed audit must have non-NULL target_account_email")
	}
	if result.TargetAccountEmail.String != "newuser@example.com" {
		t.Fatalf("target_account_email must be 'newuser@example.com', got %q", result.TargetAccountEmail.String)
	}

	// Step 4: outcome が succeeded であることを確認する。
	if result.Outcome != "succeeded" {
		t.Fatalf("outcome must be 'succeeded', got %q", result.Outcome)
	}

	// Step 5: stable_error_code が NULL であることを確認する。
	if result.StableErrorCode.Valid {
		t.Fatalf("succeeded outcome must have NULL stable_error_code, got %q", result.StableErrorCode.String)
	}

	// Step 6: completed_at が設定されていることを確認する。
	if !result.CompletedAt.Valid {
		t.Fatal("succeeded outcome must have non-NULL completed_at")
	}

	// Step 7: ownerDB を使って public.accounts に account が存在することを確認する。
	// runtimeDB (admin_console_write role) は public.accounts の id 列への SELECT 権限を持たないため、
	// owner 接続で検証する。
	// PostgreSQL の database/sql ドライバは positional parameter として $1 を使う。? は使えない。
	var accountExists bool
	if err := ownerDB.QueryRowContext(ctx,
		"SELECT EXISTS(SELECT 1 FROM public.accounts WHERE id = $1)", expectedAccountID,
	).Scan(&accountExists); err != nil {
		t.Fatalf("check account existence: %v", err)
	}
	if !accountExists {
		t.Fatalf("account %s must exist in public.accounts", expectedAccountID)
	}
}

// verifyPendingIntentHasNullTarget は pending intent が NULL target_account_id で保存されていることを確認する。
func verifyPendingIntentHasNullTarget(ctx context.Context, t *testing.T, db *gorm.DB, auditID string) {
	t.Helper()

	var pendingTargetID sql.NullString
	if err := db.WithContext(ctx).Raw(
		"SELECT target_account_id FROM admin.audit_events WHERE id = ?", auditID,
	).Scan(&pendingTargetID).Error; err != nil {
		t.Fatalf("query pending intent: %v", err)
	}
	if pendingTargetID.Valid {
		t.Fatalf("pending intent must have NULL target_account_id, got %q", pendingTargetID.String)
	}
}

// createTestOperator はテスト用の operator をテスト用 DB の admin.operators に直接作成する。
// Account creation flow は operator_id が admin.operators に存在することを前提にするため、
// FK 制約 (operator_id REFERENCES admin.operators(id)) を満たす必要がある。
func createTestOperator(t *testing.T, testOwnerDB *sql.DB, operatorID string) {
	t.Helper()

	ctx := context.Background()

	// Step 1: テスト用 DB の admin.operators に operator を挿入する。
	// testOwnerDB はテスト用 DB への owner 接続であり、admin schema への INSERT 権限を持つ。
	_, err := testOwnerDB.ExecContext(ctx,
		`INSERT INTO admin.operators (id, email, role, active, passkey_registration_state)
		 VALUES ($1, $2, 'admin', TRUE, 'registered')
		 ON CONFLICT (id) DO NOTHING`,
		operatorID, operatorID+"@test.example.com",
	)
	if err != nil {
		t.Fatalf("create test operator %s: %v", operatorID, err)
	}
}
