-- 000007 で付与した table/schema 権限だけを剥奪し、role 自体は 000006 の責務として残す。
REVOKE UPDATE (locale, updated_at) ON public.account_settings FROM admin_console_write;
REVOKE INSERT (account_id) ON public.account_settings FROM admin_console_write;
REVOKE SELECT (account_id) ON public.account_settings FROM admin_console_write;
REVOKE INSERT (id, email, status, session_revoked_after, created_at, updated_at) ON public.accounts FROM admin_console_write;
REVOKE SELECT (email) ON public.accounts FROM admin_console_write;
REVOKE USAGE ON SCHEMA public FROM admin_console_write;
REVOKE UPDATE (target_account_id, target_account_email, outcome, stable_error_code, metadata, completed_at) ON admin.audit_events FROM admin_console_write;
REVOKE INSERT ON admin.audit_events FROM admin_console_write;
REVOKE INSERT, UPDATE, DELETE ON admin.operator_passkeys FROM admin_console_write;
REVOKE INSERT, UPDATE ON admin.operators FROM admin_console_write;
REVOKE SELECT ON admin.operators, admin.operator_passkeys, admin.audit_events FROM admin_console_read;
REVOKE USAGE ON SCHEMA admin FROM admin_console_write;
REVOKE USAGE ON SCHEMA admin FROM admin_console_read;

-- Admin-owned child table から順に削除し、public.accounts など Product table には一切触れない。
DROP TABLE IF EXISTS admin.operator_passkeys;
DROP TABLE IF EXISTS admin.audit_events;
DROP TABLE IF EXISTS admin.operators;

-- Admin-owned schema の追加分を最後に削除し、Account root を持つ public schema は保持する。
DROP SCHEMA IF EXISTS admin;
