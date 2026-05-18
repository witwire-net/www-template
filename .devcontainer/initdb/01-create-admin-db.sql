-- Devcontainer 初期化用 SQL
-- www-template_admin データベースを postgres 起動時に自動作成する
-- Admin Console 用の最小権限 login role も作成し、backend migration で作成される admin_console_write / admin_console_read ロールへのアクセスを準備する
-- このファイルは postgres コンテナの /docker-entrypoint-initdb.d/ にマウントされる

-- admin_console_write / admin_console_read ロール（backend migration と重複しても OK）
DO $$
BEGIN
    CREATE ROLE admin_console_write NOLOGIN;
EXCEPTION WHEN duplicate_object THEN
    NULL;
END $$;

DO $$
BEGIN
    CREATE ROLE admin_console_read NOLOGIN;
EXCEPTION WHEN duplicate_object THEN
    NULL;
END $$;

-- Admin Console 専用 login role（superuser ではなく、base table owner でもない）
DO $$
BEGIN
    CREATE ROLE admin_console WITH LOGIN PASSWORD 'admin_console';
EXCEPTION WHEN duplicate_object THEN
    NULL;
END $$;

-- admin_console に admin_console_write を付与（backend migration で作成されたロールへのアクセス権）
GRANT admin_console_write TO admin_console;

-- admin_view スキーマが存在する場合は読み取り権限を付与
DO $$
BEGIN
    GRANT USAGE ON SCHEMA admin_view TO admin_console_read;
    GRANT SELECT ON ALL TABLES IN SCHEMA admin_view TO admin_console_read;
EXCEPTION WHEN insufficient_privilege OR undefined_schema THEN
    NULL;
END $$;

-- admin_op スキーマが存在する場合は実行権限を付与
DO $$
BEGIN
    GRANT USAGE ON SCHEMA admin_op TO admin_console_write;
    GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA admin_op TO admin_console_write;
EXCEPTION WHEN insufficient_privilege OR undefined_schema THEN
    NULL;
END $$;

-- Admin 専用 DB 作成
CREATE DATABASE www_template_admin;
GRANT ALL PRIVILEGES ON DATABASE www_template_admin TO "www-template";
GRANT ALL PRIVILEGES ON DATABASE www_template_admin TO admin_console;

-- 注意: 本番環境では backend migration（packages/backend/db/migrations/000006_create_admin_functions.up.sql）で
-- admin_console_read / admin_console_write ロールが作成され、release 手順で
-- GRANT admin_console_write TO <環境別 login_role> を実行すること
