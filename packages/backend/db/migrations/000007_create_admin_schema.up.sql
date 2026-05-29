-- AccountSetting は Admin account 作成時にも必須 child として扱うため、DB 内で存在を保証する。
-- 既存のローカル DB が過去の migration 内容で 000002 適用済みになっていても、ここで現在の Product invariant に揃える。
CREATE TABLE IF NOT EXISTS public.account_settings (
  account_id TEXT PRIMARY KEY REFERENCES public.accounts(id) ON DELETE CASCADE,
  locale     TEXT NOT NULL DEFAULT 'ja' CHECK (locale IN ('ja', 'en')),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 既存 Account root に必須 child が欠けている場合は補完し、Admin create flow が locale 更新できる前提を満たす。
INSERT INTO public.account_settings (account_id)
SELECT id
FROM public.accounts
ON CONFLICT (account_id) DO NOTHING;

-- Account root 作成時の必須 child 作成を DB 側でも保証し、Admin/Product どちらの経路でも不変条件を保つ。
CREATE OR REPLACE FUNCTION public.create_default_account_setting()
RETURNS TRIGGER AS $$
BEGIN
  INSERT INTO public.account_settings (account_id)
  VALUES (NEW.id)
  ON CONFLICT (account_id) DO NOTHING;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- 既存 trigger 定義が古い場合でも、現在の関数へ張り直して Account root と child の同期を固定する。
DROP TRIGGER IF EXISTS accounts_create_default_account_setting ON public.accounts;
CREATE TRIGGER accounts_create_default_account_setting
AFTER INSERT ON public.accounts
FOR EACH ROW
EXECUTE FUNCTION public.create_default_account_setting();

-- Admin backend が所有する永続化領域を admin schema として作成する。
-- Product runtime からは権限を付与せず、Admin 専用 role だけが operator、passkey、audit を扱える境界にする。
CREATE SCHEMA IF NOT EXISTS admin;

-- PUBLIC の暗黙権限を先に剥奪し、schema 作成直後から最小権限 role 以外が参照できない状態にする。
REVOKE ALL ON SCHEMA admin FROM PUBLIC;

-- admin.operators は Admin Operator の認証・認可 snapshot を保持する root table である。
-- role、active、passkey_registration_state は domain.Operator と同じ fail-closed な許可値だけを受け付ける。
CREATE TABLE IF NOT EXISTS admin.operators (
  id                             TEXT PRIMARY KEY,
  email                          TEXT NOT NULL UNIQUE,
  role                           TEXT NOT NULL CHECK (role IN ('admin', 'operator', 'viewer')),
  active                         BOOLEAN NOT NULL DEFAULT TRUE,
  passkey_registration_state     TEXT NOT NULL DEFAULT 'pending' CHECK (passkey_registration_state IN ('pending', 'registered')),
  setup_token_hash               TEXT,
  setup_token_expires_at         TIMESTAMPTZ,
  setup_token_consumed_at        TIMESTAMPTZ,
  created_at                     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at                     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CHECK (
    (setup_token_hash IS NULL AND setup_token_expires_at IS NULL AND setup_token_consumed_at IS NULL)
    OR (setup_token_hash IS NULL AND setup_token_expires_at IS NULL AND setup_token_consumed_at IS NOT NULL)
    OR (setup_token_hash IS NOT NULL AND setup_token_expires_at IS NOT NULL)
  )
);

-- operator login と監査表示の検索経路を安定させるため、email と role/active の lookup を明示する。
CREATE INDEX IF NOT EXISTS idx_admin_operators_email ON admin.operators(email);
CREATE INDEX IF NOT EXISTS idx_admin_operators_role_active ON admin.operators(role, active);

-- admin.operator_passkeys は Admin Operator 専用の WebAuthn credential を保持する child table である。
-- Product account_passkey_credentials と物理的に分け、operator credential が Product auth に混入しないようにする。
CREATE TABLE IF NOT EXISTS admin.operator_passkeys (
  id                TEXT PRIMARY KEY,
  operator_id       TEXT NOT NULL REFERENCES admin.operators(id) ON DELETE CASCADE,
  credential_handle TEXT NOT NULL UNIQUE,
  public_key        BYTEA NOT NULL,
  sign_count        BIGINT NOT NULL DEFAULT 0,
  aaguid            BYTEA,
  backup_eligible   BOOLEAN NOT NULL DEFAULT FALSE,
  backup_state      BOOLEAN NOT NULL DEFAULT FALSE,
  transports        JSONB NOT NULL DEFAULT '[]'::jsonb,
  created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  last_used_at      TIMESTAMPTZ
);

-- operator 単位の passkey 一覧と credential_handle lookup を O(1) 寄りに保つ。
CREATE INDEX IF NOT EXISTS idx_admin_operator_passkeys_operator_id ON admin.operator_passkeys(operator_id);
CREATE INDEX IF NOT EXISTS idx_admin_operator_passkeys_credential_handle ON admin.operator_passkeys(credential_handle);

-- admin.audit_events は Admin mutation の intent と outcome を一度だけ記録する監査 table である。
-- failed outcome では stable_error_code を必須にし、成功・未完了状態へ動的 error message が混ざることを防ぐ。
CREATE TABLE IF NOT EXISTS admin.audit_events (
  id                   TEXT PRIMARY KEY,
  request_id           TEXT NOT NULL,
  operator_id          TEXT REFERENCES admin.operators(id) ON DELETE RESTRICT,
  target_account_id    TEXT REFERENCES public.accounts(id) ON DELETE SET NULL,
  target_account_email TEXT,
  action               TEXT NOT NULL CHECK (action IN ('accounts:create', 'accounts:suspend', 'accounts:restore', 'operators:create', 'operators:setup')),
  outcome              TEXT NOT NULL DEFAULT 'pending' CHECK (outcome IN ('pending', 'succeeded', 'failed')),
  stable_error_code    TEXT,
  metadata             JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  completed_at         TIMESTAMPTZ,
  CHECK (stable_error_code IS NULL OR stable_error_code ~ '^[a-z0-9][a-z0-9_:-]*$'),
  CHECK (
    (outcome = 'pending' AND completed_at IS NULL AND stable_error_code IS NULL)
    OR (outcome = 'succeeded' AND completed_at IS NOT NULL AND stable_error_code IS NULL)
    OR (outcome = 'failed' AND completed_at IS NOT NULL AND stable_error_code IS NOT NULL)
  )
);

-- 監査検索で使う operator、target account、request ID、時系列の access path を明示する。
CREATE INDEX IF NOT EXISTS idx_admin_audit_events_operator_id ON admin.audit_events(operator_id);
CREATE INDEX IF NOT EXISTS idx_admin_audit_events_target_account_id ON admin.audit_events(target_account_id);
CREATE INDEX IF NOT EXISTS idx_admin_audit_events_request_id ON admin.audit_events(request_id);
CREATE INDEX IF NOT EXISTS idx_admin_audit_events_created_at ON admin.audit_events(created_at);

-- table 作成後にも PUBLIC 権限を明示的に剥奪し、migration owner の default に依存しない最小権限状態へ固定する。
REVOKE ALL ON ALL TABLES IN SCHEMA admin FROM PUBLIC;

-- 読み取り role は Admin schema の参照と既存 Admin-owned table の SELECT だけを許可し、mutation 権限を持たせない。
GRANT USAGE ON SCHEMA admin TO admin_console_read;
GRANT SELECT ON admin.operators, admin.operator_passkeys, admin.audit_events TO admin_console_read;

-- 書き込み role は admin_console_read を継承済みの前提で、必要な Admin-owned table mutation だけを明示的に許可する。
-- audit_events には DELETE を付与せず、監査証跡を runtime role が削除できない append-only + outcome update 境界にする。
GRANT USAGE ON SCHEMA admin TO admin_console_write;
GRANT INSERT, UPDATE ON admin.operators TO admin_console_write;
GRANT INSERT, UPDATE, DELETE ON admin.operator_passkeys TO admin_console_write;
GRANT INSERT ON admin.audit_events TO admin_console_write;
GRANT UPDATE (target_account_id, target_account_email, outcome, stable_error_code, metadata, completed_at) ON admin.audit_events TO admin_console_write;

-- Admin account repository が同一 transaction で触る Product Account root だけに public schema の到達を限定する。
-- public.accounts は duplicate 確認の email SELECT と Account root 作成に必要な列 INSERT だけを許可し、UPDATE/DELETE は付与しない。
-- public.account_settings は trigger による child 作成、account_id 条件、locale/updated_at 更新だけを許可し、table 全体の SELECT/DELETE は付与しない。
GRANT USAGE ON SCHEMA public TO admin_console_write;
GRANT SELECT (email) ON public.accounts TO admin_console_write;
GRANT INSERT (id, email, status, session_revoked_after, created_at, updated_at) ON public.accounts TO admin_console_write;
GRANT SELECT (account_id) ON public.account_settings TO admin_console_write;
GRANT INSERT (account_id) ON public.account_settings TO admin_console_write;
GRANT UPDATE (locale, updated_at) ON public.account_settings TO admin_console_write;

-- 将来 Admin schema に table を追加する場合は、その migration で table-specific grant を明示し、自動付与で権限を広げない。
