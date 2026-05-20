import { describe, expect, it } from 'vitest';

/**
 * package.json の scripts だけを検証するための最小型。
 * 入力は JSON.parse 後の unknown 値で、出力は scripts を持つ object として扱う。
 * テスト専用であり、package.json には一切副作用を与えない。
 */
interface PackageJsonWithScripts {
  scripts: Record<string, string>;
}

/**
 * SQL ファイルを UTF-8 文字列として読む。
 * 入力は repository root からの相対 path、出力は SQL 本文。
 * テスト専用の読み取りだけを行い、migration 自体は変更しない。
 */
async function readSql(path: string): Promise<string> {
  const fs = await import('node:fs/promises');
  return fs.readFile(path, 'utf8');
}

/**
 * 空白差分に強い SQL 文字列へ正規化する。
 * 入力 SQL の連続空白を 1 つに畳み、出力は大小文字を保った検証用文字列。
 * ファイル内容は変更せず、assertion の安定性だけを上げる。
 */
function normalizeSql(sql: string): string {
  return sql.replace(/\s+/g, ' ').trim();
}

describe('Prisma and product DB migrations', () => {
  it('17.1 Product DB migration は accounts.status を default active にする', async () => {
    // 既存顧客の状態を安全に保つため、追加列は NOT NULL かつ DEFAULT active に固定されていることを確認する。
    const sql = normalizeSql(
      await readSql('../backend/db/migrations/000004_add_account_status.up.sql')
    );
    expect(sql).toContain("ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'active'");
    expect(sql).toContain("CHECK (status IN ('active', 'suspended'))");
  });

  it('17.2 suspend_account 関数は active アカウントだけを停止し session_revoked_after を更新する', async () => {
    // 停止処理が単一 SQL 関数内で status と session_revoked_after を同時更新することを静的に検証する。
    const sql = normalizeSql(
      await readSql('../backend/db/migrations/000006_create_admin_functions.up.sql')
    );
    expect(sql).toContain('CREATE OR REPLACE FUNCTION admin_op.suspend_account');
    expect(sql).toContain("UPDATE public.accounts SET status = 'suspended'");
    expect(sql).toContain('session_revoked_after = v_now');
    expect(sql).toContain("WHERE id = p_account_id AND status = 'active'");
    expect(sql).toContain("RAISE EXCEPTION 'account_not_active'");
  });

  it('17.3 restore_account 関数は suspended アカウントを active に戻し session_revoked_after を維持する', async () => {
    // 復旧後も停止前セッションを復活させないため、session_revoked_after を更新対象に含めないことを確認する。
    const sql = normalizeSql(
      await readSql('../backend/db/migrations/000006_create_admin_functions.up.sql')
    );
    const restoreBody = sql.slice(
      sql.indexOf('CREATE OR REPLACE FUNCTION admin_op.restore_account')
    );
    expect(restoreBody).toContain("UPDATE public.accounts SET status = 'active'");
    expect(restoreBody).toContain("WHERE id = p_account_id AND status = 'suspended'");
    expect(restoreBody).not.toMatch(/session_revoked_after\s*=/);
  });

  it('17.4 restore_account 関数は非 suspended アカウントで例外を投げる', async () => {
    // active などの非停止アカウントを誤って復旧処理しないため、明示エラーを確認する。
    const sql = await readSql('../backend/db/migrations/000006_create_admin_functions.up.sql');
    expect(sql).toContain("RAISE EXCEPTION 'account_not_suspended';");
  });

  it('17.5 account_summaries view は全 accounts を LEFT JOIN で返す', async () => {
    // passkey 未登録アカウントも管理対象なので、account_passkey_credentials への LEFT JOIN と accounts 起点の view を確認する。
    const sql = normalizeSql(
      await readSql('../backend/db/migrations/000005_create_admin_views.up.sql')
    );
    expect(sql).toContain('CREATE OR REPLACE VIEW admin_view.account_summaries AS SELECT');
    expect(sql).toContain(
      'FROM public.accounts a LEFT JOIN public.account_passkey_credentials p ON p.account_id = a.id'
    );
    expect(sql).toContain('COUNT(p.id)::bigint AS passkey_count');
  });

  it('17.6 account_passkeys view は passkey 詳細情報を返す', async () => {
    // 詳細画面に必要な認証器メタデータが Product base table から schema-qualified に公開されることを確認する。
    const sql = normalizeSql(
      await readSql('../backend/db/migrations/000005_create_admin_views.up.sql')
    );
    expect(sql).toContain('CREATE OR REPLACE VIEW admin_view.account_passkeys AS SELECT');
    expect(sql).toContain('p.credential_handle');
    expect(sql).toContain('p.public_key');
    expect(sql).toContain('p.sign_count');
    expect(sql).toContain('p.aaguid');
    expect(sql).toContain('p.backup_eligible');
    expect(sql).toContain('p.backup_state');
    expect(sql).toContain('p.transports');
  });

  it('17.10 Admin migration deploy script は未適用 migration を適用できる', async () => {
    // Admin DB は Prisma Migrate 管理なので deploy script が admin schema を対象にすることを確認する。
    const packageJson = JSON.parse(await readSql('package.json')) as PackageJsonWithScripts;
    expect(packageJson.scripts['prisma:admin:migrate:deploy']).toBe(
      'prisma migrate deploy --schema prisma/admin/schema.prisma'
    );
  });

  it('17.10a Product DB 拡張 migration は golang-migrate 管理で Prisma Migrate ではない', async () => {
    // Product DB の admin_view/admin_op は backend migration SQL と root の golang-migrate script で管理されることを確認する。
    const rootPackageJson = JSON.parse(
      await readSql('../../package.json')
    ) as PackageJsonWithScripts;
    const productSchema = await readSql('prisma/product/schema.prisma');
    expect(rootPackageJson.scripts['db:migrate:product']).toBe('bash scripts/go/migrate.sh up');
    expect(productSchema).toContain('Prisma Migrate は適用せず');
  });

  it('17.11 Admin migration deploy は適用済み migration を Prisma migration table で skip する運用である', async () => {
    // Prisma Migrate deploy は _prisma_migrations を使うため、schema と script が deploy に限定されることを固定する。
    const packageJson = JSON.parse(await readSql('package.json')) as PackageJsonWithScripts;
    expect(packageJson.scripts['prisma:admin:migrate:deploy']).toContain('migrate deploy');
    expect(packageJson.scripts['prisma:admin:migrate:dev']).toContain('migrate dev');
    expect(
      await readSql('prisma/admin/migrations/000001_create_operators_and_passkeys/migration.sql')
    ).toContain('CREATE SCHEMA IF NOT EXISTS admin');
  });

  it('13.12 Admin Prisma migration は必要な全テーブルを作成する', async () => {
    // Admin 認証と監査の永続化に必要な 3 テーブルが初期 migration に含まれることを確認する。
    const sql = await readSql(
      'prisma/admin/migrations/000001_create_operators_and_passkeys/migration.sql'
    );
    expect(sql).toContain('CREATE TABLE IF NOT EXISTS admin.operators');
    expect(sql).toContain('CREATE TABLE IF NOT EXISTS admin.operator_passkeys');
    expect(sql).toContain('CREATE TABLE IF NOT EXISTS admin.audit_events');
  });

  it('Admin operator locale migration は既定値と制約を持つ', async () => {
    // Admin operator locale は Admin DB 内だけで保持し、未対応値を DB 制約で fail-closed に拒否する。
    const schema = await readSql('prisma/admin/schema.prisma');
    const sql = normalizeSql(
      await readSql('prisma/admin/migrations/000002_add_operator_locale/migration.sql')
    );
    expect(schema).toContain('locale               String   @default("ja") @map("locale")');
    expect(sql).toContain("ADD COLUMN IF NOT EXISTS locale TEXT NOT NULL DEFAULT 'ja'");
    expect(sql).toContain("ADD CONSTRAINT operators_locale_check CHECK (locale IN ('ja', 'en'))");
    expect(`${schema} ${sql}`).not.toMatch(/AccountSetting|AccountLocale|Product AccountSetting/);
  });

  it('13.13 Admin Prisma migration は初期オペレーターを seed しない', async () => {
    // 初期 admin は bootstrap flow だけで作成されるべきなので、migration に INSERT を含めない。
    const sql = await readSql(
      'prisma/admin/migrations/000001_create_operators_and_passkeys/migration.sql'
    );
    expect(sql.toLowerCase()).not.toMatch(/insert\s+into\s+admin\.operators/);
    expect(sql).toContain('初期オペレーターは作成しない');
  });

  it('13.14 SECURITY DEFINER 関数は固定 search_path を設定する', async () => {
    // SECURITY DEFINER の search_path hijack を防ぐため、両関数に固定 search_path があることを確認する。
    const sql = normalizeSql(
      await readSql('../backend/db/migrations/000006_create_admin_functions.up.sql')
    );
    expect(sql).toMatch(/SECURITY DEFINER SET search_path = pg_catalog, admin_op/);
    expect(sql.match(/SECURITY DEFINER SET search_path = pg_catalog, admin_op/g)).toHaveLength(2);
  });

  it('17.16 SECURITY DEFINER search_path は固定され、Product base table 参照は schema-qualified である', async () => {
    // search_path hijack と意図しない同名 table 参照を防ぐため、関数設定と public schema 参照を同時に検証する。
    const sql = normalizeSql(
      await readSql('../backend/db/migrations/000006_create_admin_functions.up.sql')
    );
    expect(sql.match(/SECURITY DEFINER SET search_path = pg_catalog, admin_op/g)).toHaveLength(2);
    expect(sql).toContain('UPDATE public.accounts');
    expect(sql).not.toMatch(/\bUPDATE\s+accounts\b/);
  });

  it('13.15 SECURITY DEFINER 関数は PUBLIC execute を revoke する', async () => {
    // PUBLIC に実行権限が残ると権限境界が破れるため、対象関数ごとに REVOKE を確認する。
    const sql = await readSql('../backend/db/migrations/000006_create_admin_functions.up.sql');
    expect(sql).toContain(
      'REVOKE ALL ON FUNCTION admin_op.suspend_account(TEXT, TEXT, TEXT, TEXT) FROM PUBLIC;'
    );
    expect(sql).toContain(
      'REVOKE ALL ON FUNCTION admin_op.restore_account(TEXT, TEXT, TEXT) FROM PUBLIC;'
    );
  });

  it('17.17 SECURITY DEFINER 関数は PUBLIC execute を revoke する', async () => {
    // PUBLIC execute が残ると Admin role 境界を迂回できるため、対象関数の REVOKE を明示的に確認する。
    const sql = await readSql('../backend/db/migrations/000006_create_admin_functions.up.sql');
    expect(sql).toContain(
      'REVOKE ALL ON FUNCTION admin_op.suspend_account(TEXT, TEXT, TEXT, TEXT) FROM PUBLIC;'
    );
    expect(sql).toContain(
      'REVOKE ALL ON FUNCTION admin_op.restore_account(TEXT, TEXT, TEXT) FROM PUBLIC;'
    );
  });

  it('13.16 admin_console_read には admin_view 読み取り権限だけを付与する', async () => {
    // 読み取り role が view schema の USAGE と SELECT を持つことを確認し、base table 権限付与は避ける。
    const sql = await readSql('../backend/db/migrations/000006_create_admin_functions.up.sql');
    expect(sql).toContain('CREATE ROLE admin_console_read NOLOGIN');
    expect(sql).toContain('GRANT USAGE ON SCHEMA admin_view TO admin_console_read');
    expect(sql).toContain('GRANT SELECT ON ALL TABLES IN SCHEMA admin_view TO admin_console_read');
    expect(sql.toLowerCase()).not.toMatch(
      /grant\s+(?:select|insert|update|delete).*on\s+(?:table\s+)?public\./
    );
  });

  it('17.18 admin_console_read / admin_console_write grants は view 読み取りと関数実行だけに限定する', async () => {
    // 読み取り role は admin_view のみ、書き込み role は admin_op 関数のみという最小権限境界を固定する。
    const sql = await readSql('../backend/db/migrations/000006_create_admin_functions.up.sql');
    expect(sql).toContain('GRANT USAGE ON SCHEMA admin_view TO admin_console_read');
    expect(sql).toContain('GRANT SELECT ON ALL TABLES IN SCHEMA admin_view TO admin_console_read');
    expect(sql).toContain('GRANT admin_console_read TO admin_console_write');
    expect(sql).toContain('GRANT USAGE ON SCHEMA admin_op TO admin_console_write');
    expect(sql).toContain(
      'GRANT EXECUTE ON FUNCTION admin_op.suspend_account(TEXT, TEXT, TEXT, TEXT) TO admin_console_write'
    );
    expect(sql).toContain(
      'GRANT EXECUTE ON FUNCTION admin_op.restore_account(TEXT, TEXT, TEXT) TO admin_console_write'
    );
    expect(sql.toLowerCase()).not.toMatch(
      /grant\s+(?:select|insert|update|delete).*on\s+(?:table\s+)?public\./
    );
  });

  it('17.19 Product DB に Prisma Migrate を適用する script は存在しない', async () => {
    // Product DB は golang-migrate のみで進め、Prisma は client generate だけに限定されることを確認する。
    const rootPackageJson = JSON.parse(
      await readSql('../../package.json')
    ) as PackageJsonWithScripts;
    const adminPackageJson = JSON.parse(await readSql('package.json')) as PackageJsonWithScripts;
    const allScripts = { ...rootPackageJson.scripts, ...adminPackageJson.scripts };
    for (const [scriptName, command] of Object.entries(allScripts)) {
      if (!scriptName.includes('product')) continue;
      expect(command).not.toMatch(/prisma\s+migrate/);
    }
  });

  it('13.17 admin_console_write は read 継承と admin_op 実行権限だけを持つ', async () => {
    // 書き込み role を SECURITY DEFINER 関数の実行に限定し、直接 table write を許可しないことを検証する。
    const sql = await readSql('../backend/db/migrations/000006_create_admin_functions.up.sql');
    expect(sql).toContain('CREATE ROLE admin_console_write NOLOGIN');
    expect(sql).toContain('GRANT admin_console_read TO admin_console_write');
    expect(sql).toContain('GRANT USAGE ON SCHEMA admin_op TO admin_console_write');
    expect(sql).toContain(
      'GRANT EXECUTE ON FUNCTION admin_op.suspend_account(TEXT, TEXT, TEXT, TEXT) TO admin_console_write'
    );
    expect(sql).toContain(
      'GRANT EXECUTE ON FUNCTION admin_op.restore_account(TEXT, TEXT, TEXT) TO admin_console_write'
    );
    expect(sql.toLowerCase()).not.toMatch(
      /grant\s+(?:insert|update|delete).*on\s+(?:table\s+)?public\./
    );
  });
});
