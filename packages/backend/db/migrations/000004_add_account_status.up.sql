-- accounts テーブルに Account ライフサイクル管理用カラムを追加する。
-- status は active / suspended のみ許可し、管理関数が fail-closed に状態遷移できるようにする。
-- session_revoked_after は停止時に Account.Auth の既存セッションを一括失効させる境界時刻として扱う。
-- status_reason / status_updated_* は Admin Console 操作の監査情報を保持する。
ALTER TABLE accounts
  ADD COLUMN IF NOT EXISTS status                TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'suspended')),
  ADD COLUMN IF NOT EXISTS status_reason         TEXT,
  ADD COLUMN IF NOT EXISTS status_updated_at     TIMESTAMPTZ,
  ADD COLUMN IF NOT EXISTS status_updated_by     TEXT,
  ADD COLUMN IF NOT EXISTS session_revoked_after TIMESTAMPTZ;

-- 既存レコードは default により active へそろい、一覧・絞り込みのため status index を作成する。
CREATE INDEX IF NOT EXISTS idx_accounts_status ON accounts(status);
