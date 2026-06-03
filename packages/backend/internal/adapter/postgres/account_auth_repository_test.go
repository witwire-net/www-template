package postgres

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProductDBMigrationsUseAccountRootSchema(t *testing.T) {
	t.Parallel()

	// migration のファイル名は backend 永続化境界の完了条件そのものなので、最初に厳密なペアを確認する。
	requiredFiles := []string{
		"000001_create_accounts.up.sql",
		"000001_create_accounts.down.sql",
		"000002_create_account_settings.up.sql",
		"000002_create_account_settings.down.sql",
		"000003_create_account_passkey_credentials.up.sql",
		"000003_create_account_passkey_credentials.down.sql",
		"000004_add_account_status.up.sql",
		"000004_add_account_status.down.sql",
		"000005_create_admin_views.up.sql",
		"000005_create_admin_views.down.sql",
		"000006_create_admin_functions.up.sql",
		"000006_create_admin_functions.down.sql",
		"000007_create_admin_schema.up.sql",
		"000007_create_admin_schema.down.sql",
	}

	// migration directory を読み込み、必要な pair が欠けていないことを確認する。
	migrations := readMigrationFiles(t)
	for _, fileName := range requiredFiles {
		if _, ok := migrations[fileName]; !ok {
			t.Fatalf("required migration file %q is missing", fileName)
		}
	}

	// 旧 migration 名が残ると旧 schema 併存 path になるため、ファイル名でも禁止する。
	assertMissingMigrationFiles(t, migrations,
		"000001_create_legacy_accounts_and_passkey_credentials.up.sql",
		"000001_create_legacy_accounts_and_passkey_credentials.down.sql",
		"000002_add_passkey_credentials_created_at.up.sql",
		"000002_add_passkey_credentials_created_at.down.sql",
		"000003_add_webauthn_credential_data.up.sql",
		"000003_add_webauthn_credential_data.down.sql",
	)

	// Account root と child table を別 migration に分け、locale が accounts へ残らないことを確認する。
	accountsSQL := migrations["000001_create_accounts.up.sql"]
	assertContainsAll(t, accountsSQL, "CREATE TABLE IF NOT EXISTS accounts", "email      TEXT NOT NULL UNIQUE", "created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()")
	assertNotContainsAny(t, accountsSQL, "  locale", "ADD COLUMN IF NOT EXISTS locale")

	// AccountSetting は Account child として locale を所有し、対応 locale 以外を DB 制約で拒否する。
	settingsSQL := migrations["000002_create_account_settings.up.sql"]
	assertContainsAll(t, settingsSQL, "CREATE TABLE IF NOT EXISTS account_settings", "account_id TEXT PRIMARY KEY REFERENCES accounts(id) ON DELETE CASCADE", "locale     TEXT NOT NULL DEFAULT 'ja' CHECK (locale IN ('ja', 'en'))")
	t.Run("[LOCALIZATION-BE-S005] account settings defaults to ja", func(t *testing.T) {
		t.Parallel()
		assertContainsAll(t, settingsSQL, "locale     TEXT NOT NULL DEFAULT 'ja'", "INSERT INTO account_settings (account_id)", "CREATE TRIGGER accounts_create_default_account_setting", "AFTER INSERT ON accounts", "EXECUTE FUNCTION create_default_account_setting()")
	})
	t.Run("[LOCALIZATION-BE-S008] account settings rejects unsupported locale", func(t *testing.T) {
		t.Parallel()
		assertContainsAll(t, settingsSQL, "CHECK (locale IN ('ja', 'en'))")
	})

	// Account.Auth の credential table は canonical 名だけを作成し、WebAuthn 復元に必要な列を初期 schema に含める。
	credentialsSQL := migrations["000003_create_account_passkey_credentials.up.sql"]
	assertContainsAll(t, credentialsSQL, "CREATE TABLE IF NOT EXISTS account_passkey_credentials", "account_id        TEXT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE", "created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()", "public_key        BYTEA", "transports        JSONB NOT NULL DEFAULT '[]'::jsonb")

	// migration surface では旧 table 名と Account 直下 locale を作らず、Account-root schema だけを正とする。
	for fileName, sql := range migrations {
		if strings.HasSuffix(fileName, ".up.sql") {
			legacyPasskeyTable := "passkey" + "_credentials"
			assertNotContainsAny(t, sql, "CREATE TABLE IF NOT EXISTS "+legacyPasskeyTable, "ALTER TABLE "+legacyPasskeyTable, "CREATE TABLE IF NOT EXISTS legacy_accounts", "ALTER TABLE legacy_accounts", "accounts"+"."+"locale")
		}
	}
}

func TestAdminViewsAndFunctionsMatchAccountRootSpec(t *testing.T) {
	t.Parallel()

	// 管理 view / function の SQL を読み込み、admin-console-be の Account root 要件を静的に検証する。
	migrations := readMigrationFiles(t)
	viewsSQL := migrations["000005_create_admin_views.up.sql"]
	functionsSQL := migrations["000006_create_admin_functions.up.sql"]

	// 各 subtest 名に spec scenario ID を含め、task 1.4 が要求する admin-console-be シナリオ対応を明示する。
	t.Run("[ADMIN-CONSOLE-BE-S007] accounts status defaults to active", func(t *testing.T) {
		t.Parallel()
		statusSQL := migrations["000004_add_account_status.up.sql"]
		assertContainsAll(t, statusSQL, "status                TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'suspended'))", "session_revoked_after TIMESTAMPTZ")
	})

	t.Run("[ADMIN-CONSOLE-BE-S008] suspend_account suspends active account", func(t *testing.T) {
		t.Parallel()
		assertContainsAll(t, functionsSQL, "CREATE OR REPLACE FUNCTION admin_op.suspend_account", "status                = 'suspended'", "session_revoked_after = v_now", "WHERE id = p_account_id AND status = 'active'", "RETURNING id INTO v_result")
	})

	t.Run("[ADMIN-CONSOLE-BE-S009] suspend_account rejects non active account", func(t *testing.T) {
		t.Parallel()
		assertContainsAll(t, functionsSQL, "RAISE EXCEPTION 'account_not_active'")
	})

	t.Run("[ADMIN-CONSOLE-BE-S010] restore_account restores suspended account", func(t *testing.T) {
		t.Parallel()
		assertContainsAll(t, functionsSQL, "CREATE OR REPLACE FUNCTION admin_op.restore_account", "status            = 'active'", "status_reason     = NULL", "WHERE id = p_account_id AND status = 'suspended'", "RETURNING id INTO v_result")
		assertNotContainsAny(t, functionsSQL, "session_revoked_after = NULL")
	})

	t.Run("[ADMIN-CONSOLE-BE-S011] restore_account rejects non suspended account", func(t *testing.T) {
		t.Parallel()
		assertContainsAll(t, functionsSQL, "RAISE EXCEPTION 'account_not_suspended'")
	})

	t.Run("[ADMIN-CONSOLE-BE-S012] account_summaries uses account passkey count", func(t *testing.T) {
		t.Parallel()
		assertContainsAll(t, viewsSQL, "CREATE OR REPLACE VIEW admin_view.account_summaries AS", "COUNT(p.id)::bigint AS passkey_count", "FROM public.accounts a", "LEFT JOIN public.account_passkey_credentials p ON p.account_id = a.id")
	})

	t.Run("[ADMIN-CONSOLE-BE-S013] account_passkeys exposes account auth passkeys", func(t *testing.T) {
		t.Parallel()
		assertContainsAll(t, viewsSQL, "CREATE OR REPLACE VIEW admin_view.account_passkeys AS", "a.email       AS account_email", "JOIN public.account_passkey_credentials p ON p.account_id = a.id")
		assertNotContainsAny(t, viewsSQL, "public."+"passkey"+"_credentials")
	})

	t.Run("[ADMIN-CONSOLE-BE-S037] security definer functions pin search path", func(t *testing.T) {
		t.Parallel()
		assertContainsAll(t, functionsSQL, "SECURITY DEFINER", "SET search_path = pg_catalog, admin_op", "UPDATE public.accounts")
	})

	t.Run("[ADMIN-CONSOLE-BE-S038] public execute privilege is revoked", func(t *testing.T) {
		t.Parallel()
		assertContainsAll(t, functionsSQL, "REVOKE ALL ON FUNCTION admin_op.suspend_account(TEXT, TEXT, TEXT, TEXT) FROM PUBLIC", "REVOKE ALL ON FUNCTION admin_op.restore_account(TEXT, TEXT, TEXT) FROM PUBLIC")
	})

	t.Run("[ADMIN-CONSOLE-BE-S042] read role only selects admin views", func(t *testing.T) {
		t.Parallel()
		assertContainsAll(t, functionsSQL, "CREATE ROLE admin_console_read NOLOGIN", "GRANT USAGE ON SCHEMA admin_view TO admin_console_read", "GRANT SELECT ON ALL TABLES IN SCHEMA admin_view TO admin_console_read")
		assertNotContainsAny(t, functionsSQL, "GRANT EXECUTE ON FUNCTION admin_op.suspend_account(TEXT, TEXT, TEXT, TEXT) TO admin_console_read")
	})

	t.Run("[ADMIN-CONSOLE-BE-S043] write role executes admin functions", func(t *testing.T) {
		t.Parallel()
		assertContainsAll(t, functionsSQL, "CREATE ROLE admin_console_write NOLOGIN", "GRANT admin_console_read TO admin_console_write", "GRANT USAGE ON SCHEMA admin_op TO admin_console_write", "GRANT EXECUTE ON FUNCTION admin_op.suspend_account(TEXT, TEXT, TEXT, TEXT) TO admin_console_write", "GRANT EXECUTE ON FUNCTION admin_op.restore_account(TEXT, TEXT, TEXT) TO admin_console_write")
	})

	t.Run("[ADMIN-CONSOLE-BE-S044] login role is release managed least privilege", func(t *testing.T) {
		t.Parallel()
		assertContainsAll(t, functionsSQL, "GRANT admin_console_write TO <product_admin_login_role>")
		assertNotContainsAny(t, functionsSQL, "CREATE ROLE product_admin", "SUPERUSER")
	})
}

func TestAdminSchemaMigrationMatchesAccountCreationSpec(t *testing.T) {
	t.Parallel()

	// Admin-owned schema を作る 000007 migration pair を静的に読み込む。
	migrations := readMigrationFiles(t)
	upSQL := migrations["000007_create_admin_schema.up.sql"]
	downSQL := migrations["000007_create_admin_schema.down.sql"]

	// Admin schema と Operator root は Product Account root と物理分離され、domain.Operator と同じ role/state を保持する。
	assertContainsAll(t, upSQL,
		"CREATE SCHEMA IF NOT EXISTS admin",
		"REVOKE ALL ON SCHEMA admin FROM PUBLIC",
		"CREATE TABLE IF NOT EXISTS admin.operators",
		"role                           TEXT NOT NULL CHECK (role IN ('admin', 'operator', 'viewer'))",
		"passkey_registration_state     TEXT NOT NULL DEFAULT 'pending' CHECK (passkey_registration_state IN ('pending', 'registered'))",
	)

	// Operator passkey は Product credential table と別 schema に置き、WebAuthn 検証に必要な credential state を保持する。
	assertContainsAll(t, upSQL,
		"CREATE TABLE IF NOT EXISTS admin.operator_passkeys",
		"operator_id       TEXT NOT NULL REFERENCES admin.operators(id) ON DELETE CASCADE",
		"credential_handle TEXT NOT NULL UNIQUE",
		"public_key        BYTEA NOT NULL",
		"sign_count        BIGINT NOT NULL DEFAULT 0",
	)

	// Audit event は mutation intent/outcome と request/account/operator 相関を保存し、failed outcome の stable code を必須にする。
	assertContainsAll(t, upSQL,
		"CREATE TABLE IF NOT EXISTS admin.audit_events",
		"operator_id          TEXT REFERENCES admin.operators(id) ON DELETE RESTRICT",
		"target_account_id    TEXT REFERENCES public.accounts(id) ON DELETE SET NULL",
		"outcome              TEXT NOT NULL DEFAULT 'pending' CHECK (outcome IN ('pending', 'succeeded', 'failed'))",
		"(outcome = 'failed' AND completed_at IS NOT NULL AND stable_error_code IS NOT NULL)",
	)

	// Least-privilege grants は read/write role の責務を分け、PUBLIC と Product runtime への Admin schema 権限漏れを避ける。
	assertContainsAll(t, upSQL,
		"REVOKE ALL ON ALL TABLES IN SCHEMA admin FROM PUBLIC",
		"GRANT USAGE ON SCHEMA admin TO admin_console_read",
		"GRANT SELECT ON admin.operators, admin.operator_passkeys, admin.audit_events TO admin_console_read",
		"GRANT USAGE ON SCHEMA admin TO admin_console_write",
		"GRANT INSERT, UPDATE ON admin.operators TO admin_console_write",
		"GRANT INSERT, UPDATE, DELETE ON admin.operator_passkeys TO admin_console_write",
		"GRANT INSERT ON admin.audit_events TO admin_console_write",
		"GRANT UPDATE (target_account_id, target_account_email, outcome, stable_error_code, metadata, completed_at) ON admin.audit_events TO admin_console_write",
	)

	// Admin account repository が同一 transaction で必要とする Product Account root の列だけを admin_console_write に許可する。
	assertContainsAll(t, upSQL,
		"GRANT USAGE ON SCHEMA public TO admin_console_write",
		"GRANT SELECT (email) ON public.accounts TO admin_console_write",
		"GRANT INSERT (id, email, status, session_revoked_after, created_at, updated_at) ON public.accounts TO admin_console_write",
		"GRANT SELECT (account_id) ON public.account_settings TO admin_console_write",
		"GRANT INSERT (account_id) ON public.account_settings TO admin_console_write",
		"GRANT UPDATE (locale, updated_at) ON public.account_settings TO admin_console_write",
	)
	assertNotContainsAny(t, upSQL,
		"GRANT SELECT ON ALL TABLES IN SCHEMA admin TO PUBLIC",
		"GRANT INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA admin TO admin_console_write",
		"GRANT DELETE ON admin.audit_events",
		"GRANT INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA admin TO admin_console_read",
		"GRANT UPDATE ON public.accounts TO admin_console_write",
		"GRANT DELETE ON public.accounts TO admin_console_write",
		"GRANT SELECT ON public.accounts TO admin_console_read",
		"GRANT SELECT ON public.account_settings TO admin_console_write",
		"GRANT DELETE ON public.account_settings TO admin_console_write",
	)

	// Down migration は 000007 の schema/tables/grants だけを戻し、Product public.accounts を削除しない。
	assertContainsAll(t, downSQL,
		"REVOKE UPDATE (locale, updated_at) ON public.account_settings FROM admin_console_write",
		"REVOKE INSERT (account_id) ON public.account_settings FROM admin_console_write",
		"REVOKE SELECT (account_id) ON public.account_settings FROM admin_console_write",
		"REVOKE INSERT (id, email, status, session_revoked_after, created_at, updated_at) ON public.accounts FROM admin_console_write",
		"REVOKE SELECT (email) ON public.accounts FROM admin_console_write",
		"REVOKE USAGE ON SCHEMA public FROM admin_console_write",
		"REVOKE UPDATE (target_account_id, target_account_email, outcome, stable_error_code, metadata, completed_at) ON admin.audit_events FROM admin_console_write",
		"REVOKE INSERT ON admin.audit_events FROM admin_console_write",
		"REVOKE INSERT, UPDATE, DELETE ON admin.operator_passkeys FROM admin_console_write",
		"REVOKE INSERT, UPDATE ON admin.operators FROM admin_console_write",
		"REVOKE SELECT ON admin.operators, admin.operator_passkeys, admin.audit_events FROM admin_console_read",
		"DROP TABLE IF EXISTS admin.operator_passkeys",
		"DROP TABLE IF EXISTS admin.audit_events",
		"DROP TABLE IF EXISTS admin.operators",
		"DROP SCHEMA IF EXISTS admin",
	)
	assertNotContainsAny(t, downSQL, "DROP TABLE IF EXISTS accounts", "DROP TABLE IF EXISTS public.accounts", "DROP SCHEMA IF EXISTS public")
}

func TestAdminSchemaMigrationScenarioEvidence(t *testing.T) {
	t.Parallel()

	// Admin schema / grant / migration version を scenario ID ごとに確認するため、同じ migration file group を読み込む。
	migrations := readMigrationFiles(t)
	upSQL := migrations["000007_create_admin_schema.up.sql"]
	downSQL := migrations["000007_create_admin_schema.down.sql"]

	// [ADMIN-CONSOLE-BE-S059] backend migration が Admin-owned schema と主要 table を作ることを明示する。
	t.Run("[ADMIN-CONSOLE-BE-S059] Admin schema exists in backend migration", func(t *testing.T) {
		t.Parallel()
		assertContainsAll(t, upSQL, "CREATE SCHEMA IF NOT EXISTS admin", "CREATE TABLE IF NOT EXISTS admin.operators", "CREATE TABLE IF NOT EXISTS admin.operator_passkeys", "CREATE TABLE IF NOT EXISTS admin.audit_events")
	})

	// [ADMIN-CONSOLE-BE-S060] PUBLIC 権限を剥奪し、Admin 専用 role だけへ schema/table 権限を付与することで Product runtime role の参照経路を残さない。
	t.Run("[ADMIN-CONSOLE-BE-S060] Product runtime role has no Admin schema grant", func(t *testing.T) {
		t.Parallel()
		assertContainsAll(t, upSQL, "REVOKE ALL ON SCHEMA admin FROM PUBLIC", "REVOKE ALL ON ALL TABLES IN SCHEMA admin FROM PUBLIC", "GRANT USAGE ON SCHEMA admin TO admin_console_read", "GRANT SELECT ON admin.operators, admin.operator_passkeys, admin.audit_events TO admin_console_read", "GRANT USAGE ON SCHEMA admin TO admin_console_write")
		assertNotContainsAny(t, upSQL, "GRANT USAGE ON SCHEMA admin TO PUBLIC", "GRANT SELECT ON admin.operators, admin.operator_passkeys, admin.audit_events TO PUBLIC", "GRANT USAGE ON SCHEMA admin TO product", "GRANT SELECT ON admin.operators, admin.operator_passkeys, admin.audit_events TO product")
	})

	// [ADMIN-CONSOLE-BE-S071] 既存最大 000006 の次として 000007 の up/down pair だけを使い、zero-only version を許さない。
	t.Run("[ADMIN-CONSOLE-BE-S071] Admin schema migration uses next monotonic version pair", func(t *testing.T) {
		t.Parallel()
		assertMigrationFilesExist(t, migrations, "000006_create_admin_functions.up.sql", "000006_create_admin_functions.down.sql", "000007_create_admin_schema.up.sql", "000007_create_admin_schema.down.sql")
		assertNoMigrationVersionPrefix(t, migrations, "000000_")
	})

	// [ADMIN-CONSOLE-BE-S081] Admin schema migration は backend migration directory の 000007 pair に集約し、Admin package-local Prisma migration を使わない。
	t.Run("[ADMIN-CONSOLE-BE-S081] Admin schema migration runs only through backend migration system", func(t *testing.T) {
		t.Parallel()
		assertMigrationFilesExist(t, migrations, "000007_create_admin_schema.up.sql", "000007_create_admin_schema.down.sql")
		assertNoAdminPrismaMigrationSystem(t)
	})

	// [ADMIN-CONSOLE-BE-S082] 000007 rollback は Admin schema / grant だけを戻し、Product public.accounts を削除しない。
	t.Run("[ADMIN-CONSOLE-BE-S082] Admin schema migration rollback satisfies pair policy", func(t *testing.T) {
		t.Parallel()
		assertContainsAll(t, downSQL,
			"REVOKE UPDATE (locale, updated_at) ON public.account_settings FROM admin_console_write",
			"REVOKE INSERT (account_id) ON public.account_settings FROM admin_console_write",
			"REVOKE SELECT (account_id) ON public.account_settings FROM admin_console_write",
			"REVOKE INSERT (id, email, status, session_revoked_after, created_at, updated_at) ON public.accounts FROM admin_console_write",
			"REVOKE SELECT (email) ON public.accounts FROM admin_console_write",
			"REVOKE USAGE ON SCHEMA public FROM admin_console_write",
			"REVOKE UPDATE (target_account_id, target_account_email, outcome, stable_error_code, metadata, completed_at) ON admin.audit_events FROM admin_console_write",
			"REVOKE INSERT ON admin.audit_events FROM admin_console_write",
			"REVOKE INSERT, UPDATE, DELETE ON admin.operator_passkeys FROM admin_console_write",
			"REVOKE INSERT, UPDATE ON admin.operators FROM admin_console_write",
			"REVOKE SELECT ON admin.operators, admin.operator_passkeys, admin.audit_events FROM admin_console_read",
			"REVOKE USAGE ON SCHEMA admin FROM admin_console_write",
			"REVOKE USAGE ON SCHEMA admin FROM admin_console_read",
			"DROP TABLE IF EXISTS admin.operator_passkeys",
			"DROP TABLE IF EXISTS admin.audit_events",
			"DROP TABLE IF EXISTS admin.operators",
			"DROP SCHEMA IF EXISTS admin",
		)
		assertNotContainsAny(t, downSQL, "DROP TABLE IF EXISTS accounts", "DROP TABLE IF EXISTS public.", "DROP TABLE public.", "DELETE FROM public.", "TRUNCATE public.", "ALTER TABLE public.accounts DROP", "ALTER TABLE public.account_settings DROP", "DROP SCHEMA IF EXISTS public")
	})
}

func TestAccountAuthRepositoryBoundary(t *testing.T) {
	t.Parallel()

	// [LOCALIZATION-BE-S014] ARCH-BE-ACCOUNT-AUTH-SUBORDINATION / ARCH-BE-AUTH-NO-ACCOUNT-SETTING は Auth repository が AccountSetting と旧 passkey table を読まないことを検証する。
	content, err := os.ReadFile("account_auth_repository.go")
	if err != nil {
		t.Fatalf("read account auth repository: %v", err)
	}
	source := string(content)
	assertNotContainsAny(t, source, "account_settings", "\""+"passkey"+"_credentials\"", "auth"+"_accounts", "Auth"+"Account", "Auth"+"Subject", "Auth"+"AccountRepository", "AccountClient"+"Settings")
	assertContainsAll(t, source, "account_passkey_credentials", "AccountAuth")
}

func TestAccountAuthRepositoryUsesExplicitPublicSchema(t *testing.T) {
	t.Parallel()

	// 単一 postgres package 化後、AccountAuthRepository の TableName が public schema を明示していることを確認する。
	if got := (gormAccountRecord{}).TableName(); got != "public.accounts" {
		t.Fatalf("account table must be public.accounts, got %q", got)
	}
	if got := (gormPasskeyCredentialRecord{}).TableName(); got != "public.account_passkey_credentials" {
		t.Fatalf("passkey credential table must be public.account_passkey_credentials, got %q", got)
	}
}

func readMigrationFiles(t *testing.T) map[string]string {
	t.Helper()

	// テストを package directory から実行しても migration directory を安定して解決できるよう相対 path を固定する。
	// 単一 postgres package は internal/adapter/postgres/ にあるため、backend root へは 3 段上がる。
	migrationDir := filepath.Join("..", "..", "..", "db", "migrations")
	entries, err := os.ReadDir(migrationDir)
	if err != nil {
		t.Fatalf("read migration directory: %v", err)
	}

	// SQL ファイルだけを読み込み、ファイル名から内容へ引ける map として返す。
	migrations := make(map[string]string, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}

		content := readMigrationFile(t, entry.Name())
		migrations[entry.Name()] = content
	}

	return migrations
}

func readMigrationFile(t *testing.T, fileName string) string {
	t.Helper()

	// gosec G304 を避けるため、読み込み対象を backend-owned の固定 migration 名だけに限定する。
	switch fileName {
	case "000001_create_accounts.up.sql":
		content, err := os.ReadFile(filepath.Join("..", "..", "..", "db", "migrations", "000001_create_accounts.up.sql"))
		return migrationContentOrFail(t, fileName, content, err)
	case "000001_create_accounts.down.sql":
		content, err := os.ReadFile(filepath.Join("..", "..", "..", "db", "migrations", "000001_create_accounts.down.sql"))
		return migrationContentOrFail(t, fileName, content, err)
	case "000002_create_account_settings.up.sql":
		content, err := os.ReadFile(filepath.Join("..", "..", "..", "db", "migrations", "000002_create_account_settings.up.sql"))
		return migrationContentOrFail(t, fileName, content, err)
	case "000002_create_account_settings.down.sql":
		content, err := os.ReadFile(filepath.Join("..", "..", "..", "db", "migrations", "000002_create_account_settings.down.sql"))
		return migrationContentOrFail(t, fileName, content, err)
	case "000003_create_account_passkey_credentials.up.sql":
		content, err := os.ReadFile(filepath.Join("..", "..", "..", "db", "migrations", "000003_create_account_passkey_credentials.up.sql"))
		return migrationContentOrFail(t, fileName, content, err)
	case "000003_create_account_passkey_credentials.down.sql":
		content, err := os.ReadFile(filepath.Join("..", "..", "..", "db", "migrations", "000003_create_account_passkey_credentials.down.sql"))
		return migrationContentOrFail(t, fileName, content, err)
	case "000004_add_account_status.up.sql":
		content, err := os.ReadFile(filepath.Join("..", "..", "..", "db", "migrations", "000004_add_account_status.up.sql"))
		return migrationContentOrFail(t, fileName, content, err)
	case "000004_add_account_status.down.sql":
		content, err := os.ReadFile(filepath.Join("..", "..", "..", "db", "migrations", "000004_add_account_status.down.sql"))
		return migrationContentOrFail(t, fileName, content, err)
	default:
		return readAdminMigrationFile(t, fileName)
	}
}

func readAdminMigrationFile(t *testing.T, fileName string) string {
	t.Helper()

	// Admin 系 migration は Product root migration と分けて列挙し、readMigrationFile の複雑度を抑えつつ固定 path だけを許可する。
	switch fileName {
	case "000005_create_admin_views.up.sql":
		content, err := os.ReadFile(filepath.Join("..", "..", "..", "db", "migrations", "000005_create_admin_views.up.sql"))
		return migrationContentOrFail(t, fileName, content, err)
	case "000005_create_admin_views.down.sql":
		content, err := os.ReadFile(filepath.Join("..", "..", "..", "db", "migrations", "000005_create_admin_views.down.sql"))
		return migrationContentOrFail(t, fileName, content, err)
	case "000006_create_admin_functions.up.sql":
		content, err := os.ReadFile(filepath.Join("..", "..", "..", "db", "migrations", "000006_create_admin_functions.up.sql"))
		return migrationContentOrFail(t, fileName, content, err)
	case "000006_create_admin_functions.down.sql":
		content, err := os.ReadFile(filepath.Join("..", "..", "..", "db", "migrations", "000006_create_admin_functions.down.sql"))
		return migrationContentOrFail(t, fileName, content, err)
	case "000007_create_admin_schema.up.sql":
		content, err := os.ReadFile(filepath.Join("..", "..", "..", "db", "migrations", "000007_create_admin_schema.up.sql"))
		return migrationContentOrFail(t, fileName, content, err)
	case "000007_create_admin_schema.down.sql":
		content, err := os.ReadFile(filepath.Join("..", "..", "..", "db", "migrations", "000007_create_admin_schema.down.sql"))
		return migrationContentOrFail(t, fileName, content, err)
	default:
		t.Fatalf("unexpected migration file %q", fileName)
	}

	return ""
}

func migrationContentOrFail(t *testing.T, fileName string, content []byte, readErr error) string {
	t.Helper()

	// 固定 path の読み込み結果を検査し、上位テストが scenario ごとに内容を検証できるよう文字列化する。
	if readErr != nil {
		t.Fatalf("read migration %s: %v", fileName, readErr)
	}

	return string(content)
}

func assertContainsAll(t *testing.T, content string, requiredValues ...string) {
	t.Helper()

	// 必須 SQL 断片を個別に確認し、欠落時にどの要件が壊れたかを明示する。
	for _, required := range requiredValues {
		if !strings.Contains(content, required) {
			t.Fatalf("migration must contain %q", required)
		}
	}
}

func assertMissingMigrationFiles(t *testing.T, migrations map[string]string, forbiddenFileNames ...string) {
	t.Helper()

	// 旧 migration ファイル名を個別に確認し、schema 作成経路の二重化を防ぐ。
	for _, fileName := range forbiddenFileNames {
		if _, ok := migrations[fileName]; ok {
			t.Fatalf("migration file %q must not remain", fileName)
		}
	}
}

func assertMigrationFilesExist(t *testing.T, migrations map[string]string, requiredFileNames ...string) {
	t.Helper()

	// 必須 migration pair を個別に確認し、version の欠落を scenario ID 付きテストから直接説明できるようにする。
	for _, fileName := range requiredFileNames {
		if _, ok := migrations[fileName]; !ok {
			t.Fatalf("required migration file %q is missing", fileName)
		}
	}
}

func assertNoMigrationVersionPrefix(t *testing.T, migrations map[string]string, forbiddenPrefix string) {
	t.Helper()

	// zero-only など禁止 version prefix を全 migration 名に対して確認し、無効な連番が追加された時点で失敗させる。
	for fileName := range migrations {
		if strings.HasPrefix(fileName, forbiddenPrefix) {
			t.Fatalf("migration version prefix %q must not exist: %s", forbiddenPrefix, fileName)
		}
	}
}

func assertNoAdminPrismaMigrationSystem(t *testing.T) {
	t.Helper()

	// Admin package 群は schema.prisma をまだ持ち得るが、DB 変更の実行境界は backend migration system に限定する。
	for _, packagePath := range []string{
		filepath.Join("packages", "admin", "api", "package.json"),
		filepath.Join("packages", "admin", "app", "package.json"),
		filepath.Join("packages", "admin", "domain", "package.json"),
	} {
		adminPackageJSON := readRepositoryFile(t, packagePath)
		assertNotContainsAny(t, adminPackageJSON, "prisma migrate", "migrate:deploy", "migrate:dev")
	}

	// package-local ORM migration の出力 directory が残ると Admin schema 管理経路が二重化するため、固定 path を直接検査する。
	adminPrismaMigrationsDir := repositoryPath("packages", "admin", "prisma", "migrations")
	if _, err := os.Stat(adminPrismaMigrationsDir); err == nil {
		t.Fatalf("packages/admin/prisma/migrations must not exist for Admin schema migrations")
	} else if !os.IsNotExist(err) {
		t.Fatalf("stat packages/admin/prisma/migrations: %v", err)
	}
}

func readRepositoryFile(t *testing.T, pathParts ...string) string {
	t.Helper()

	// repository root からの固定 path だけを読み、Admin package の command policy を静的を静的証跡として検査する。
	content, err := os.ReadFile(repositoryPath(pathParts...))
	if err != nil {
		t.Fatalf("read repository file %s: %v", filepath.Join(pathParts...), err)
	}

	return string(content)
}

func repositoryPath(pathParts ...string) string {
	// package test の作業 directory から repository root へ戻し、backend と admin の境界証跡を同じ test で参照する。
	// 単一 postgres package は internal/adapter/postgres/ にあるため、backend root へは 3 段上がる。
	parts := append([]string{"..", "..", "..", "..", ".."}, pathParts...)
	return filepath.Join(parts...)
}

func assertNotContainsAny(t *testing.T, content string, forbiddenValues ...string) {
	t.Helper()

	// 禁止 SQL 断片を個別に確認し、旧 schema 併存や権限逸脱を早期に検出する。
	for _, forbidden := range forbiddenValues {
		if strings.Contains(content, forbidden) {
			t.Fatalf("migration must not contain %q", forbidden)
		}
	}
}
