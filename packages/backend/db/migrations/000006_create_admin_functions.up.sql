-- Admin Console 操作用の最小権限 DB role と SECURITY DEFINER 管理関数を作成する。
-- 環境別 login role は migration では固定名作成せず、release 手順で作成して
-- GRANT admin_console_write TO <product_admin_login_role> を実行する。

-- 読み取り専用 role（PostgreSQL は CREATE ROLE IF NOT EXISTS を持たないため DO ブロックで条件付き作成）
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'admin_console_read') THEN
    CREATE ROLE admin_console_read NOLOGIN;
  END IF;
END
$$;

-- 書き込み role（読み取りを継承）
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'admin_console_write') THEN
    CREATE ROLE admin_console_write NOLOGIN;
  END IF;
END
$$;
GRANT admin_console_read TO admin_console_write;

-- admin_view スキーマの権限設定
GRANT USAGE ON SCHEMA admin_view TO admin_console_read;
GRANT SELECT ON ALL TABLES IN SCHEMA admin_view TO admin_console_read;
-- 将来作成されるビューにも自動適用
ALTER DEFAULT PRIVILEGES IN SCHEMA admin_view GRANT SELECT ON TABLES TO admin_console_read;

-- 管理関数用スキーマ
CREATE SCHEMA IF NOT EXISTS admin_op;

-- suspend_account: active アカウントを停止し、同一 transaction 内で session_revoked_after を更新する。
-- 非 active アカウントで RAISE EXCEPTION する。
-- 競合防止のため UPDATE は WHERE status = 'active' を使用し、RETURNING で変更確認する。
CREATE OR REPLACE FUNCTION admin_op.suspend_account(
  p_account_id       TEXT,
  p_operator_id      TEXT,
  p_reason           TEXT,
  p_audit_event_id   TEXT
) RETURNS TEXT
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = pg_catalog, admin_op
AS $$
DECLARE
  v_now TIMESTAMPTZ := NOW();
  v_result TEXT;
BEGIN
  UPDATE public.accounts
  SET
    status                = 'suspended',
    status_reason         = p_reason,
    status_updated_at     = v_now,
    status_updated_by     = p_operator_id,
    session_revoked_after = v_now
  WHERE id = p_account_id AND status = 'active'
  RETURNING id INTO v_result;

  IF v_result IS NULL THEN
    RAISE EXCEPTION 'account_not_active';
  END IF;

  RETURN v_result;
END;
$$;

-- restore_account: suspended アカウントを復旧する。
-- session_revoked_after は維持し、NULL または過去値には戻さない。
-- 非 suspended アカウントで RAISE EXCEPTION する。
-- 競合防止のため UPDATE は WHERE status = 'suspended' を使用し、RETURNING で変更確認する。
CREATE OR REPLACE FUNCTION admin_op.restore_account(
  p_account_id       TEXT,
  p_operator_id      TEXT,
  p_audit_event_id   TEXT
) RETURNS TEXT
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = pg_catalog, admin_op
AS $$
DECLARE
  v_now TIMESTAMPTZ := NOW();
  v_result TEXT;
BEGIN
  UPDATE public.accounts
  SET
    status            = 'active',
    status_reason     = NULL,
    status_updated_at = v_now,
    status_updated_by = p_operator_id
  WHERE id = p_account_id AND status = 'suspended'
  RETURNING id INTO v_result;

  IF v_result IS NULL THEN
    RAISE EXCEPTION 'account_not_suspended';
  END IF;

  RETURN v_result;
END;
$$;

-- PUBLIC からの実行権限を剥奪し、最小権限 role のみに EXECUTE を許可する。
REVOKE ALL ON FUNCTION admin_op.suspend_account(TEXT, TEXT, TEXT, TEXT) FROM PUBLIC;
REVOKE ALL ON FUNCTION admin_op.restore_account(TEXT, TEXT, TEXT) FROM PUBLIC;

GRANT USAGE ON SCHEMA admin_op TO admin_console_write;
GRANT EXECUTE ON FUNCTION admin_op.suspend_account(TEXT, TEXT, TEXT, TEXT) TO admin_console_write;
GRANT EXECUTE ON FUNCTION admin_op.restore_account(TEXT, TEXT, TEXT) TO admin_console_write;
