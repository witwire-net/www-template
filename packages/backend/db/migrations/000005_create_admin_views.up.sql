-- Admin Console 用の読み取り専用ビューを提供する admin_view スキーマを作成する。
-- ビューは Product DB のベーステーブルを結合して Admin Console の一覧・詳細表示に必要な
-- 情報を集約する。
CREATE SCHEMA IF NOT EXISTS admin_view;

-- 既存 development DB に旧 accounts schema が残っている場合、Admin view と現在の Product schema が要求する時刻列を補う。
-- clean DB では 000001 が作成済みのため no-op になる。
ALTER TABLE public.accounts
  ADD COLUMN IF NOT EXISTS created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

-- Admin view が参照する Account.Auth の canonical child table を保証する。
-- clean DB では 000003 が作成済みのため no-op になり、旧 table 名は参照しない。
CREATE TABLE IF NOT EXISTS public.account_passkey_credentials (
  id                TEXT PRIMARY KEY,
  account_id        TEXT NOT NULL REFERENCES public.accounts(id) ON DELETE CASCADE,
  identifier        TEXT NOT NULL,
  credential_handle TEXT NOT NULL UNIQUE,
  public_key        BYTEA,
  sign_count        BIGINT NOT NULL DEFAULT 0,
  aaguid            BYTEA,
  backup_eligible   BOOLEAN NOT NULL DEFAULT FALSE,
  backup_state      BOOLEAN NOT NULL DEFAULT FALSE,
  transports        JSONB NOT NULL DEFAULT '[]'::jsonb,
  created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_account_passkey_credentials_account_id ON public.account_passkey_credentials(account_id);
CREATE INDEX IF NOT EXISTS idx_account_passkey_credentials_identifier ON public.account_passkey_credentials(identifier);

-- account_summaries: アカウント基本情報と紐づく Account.Auth passkey 件数を集約したビュー。
-- Admin Console のアカウント一覧・ダッシュボード集計で使用する。
CREATE OR REPLACE VIEW admin_view.account_summaries AS
SELECT
  a.id,
  a.email,
  a.status,
  a.status_reason,
  a.status_updated_at,
  a.status_updated_by,
  a.session_revoked_after,
  a.created_at,
  COUNT(p.id)::bigint AS passkey_count
FROM public.accounts a
LEFT JOIN public.account_passkey_credentials p ON p.account_id = a.id
GROUP BY a.id, a.email, a.status, a.status_reason, a.status_updated_at, a.status_updated_by, a.session_revoked_after, a.created_at;

-- account_passkeys: アカウントと Account.Auth passkey 詳細を結合したビュー。
-- Admin Console のアカウント詳細画面で passkey 一覧を表示する際に使用する。
CREATE OR REPLACE VIEW admin_view.account_passkeys AS
SELECT
  a.id          AS account_id,
  a.email       AS account_email,
  p.id          AS passkey_id,
  p.identifier  AS passkey_identifier,
  p.credential_handle,
  p.created_at  AS passkey_created_at,
  p.public_key,
  p.sign_count,
  p.aaguid,
  p.backup_eligible,
  p.backup_state,
  p.transports
FROM public.accounts a
JOIN public.account_passkey_credentials p ON p.account_id = a.id;
