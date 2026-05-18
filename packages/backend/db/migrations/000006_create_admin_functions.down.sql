-- 管理関数、スキーマ、role の削除（down migration）。
-- 依存関係を考慮し、関数 → default privileges → スキーマ → role の順で削除する。

DROP FUNCTION IF EXISTS admin_op.suspend_account(TEXT, TEXT, TEXT, TEXT);
DROP FUNCTION IF EXISTS admin_op.restore_account(TEXT, TEXT, TEXT);
DROP SCHEMA IF EXISTS admin_op;

-- default privileges を revoke してから role を削除する
ALTER DEFAULT PRIVILEGES IN SCHEMA admin_view REVOKE SELECT ON TABLES FROM admin_console_read;
REVOKE ALL ON ALL TABLES IN SCHEMA admin_view FROM admin_console_read;
REVOKE USAGE ON SCHEMA admin_view FROM admin_console_read;

-- admin_console_write は admin_console_read を継承しているため先に削除
DROP ROLE IF EXISTS admin_console_write;
DROP ROLE IF EXISTS admin_console_read;
