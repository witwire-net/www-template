-- Admin Console 用の読み取り専用ビューを提供する admin_view スキーマを作成する。
-- ビューは Product DB のベーステーブルを結合して Admin Console の一覧・詳細表示に必要な
-- 情報を集約する。
CREATE SCHEMA IF NOT EXISTS admin_view;

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
