-- accounts テーブルにアカウント状態管理用カラムを追加する。
-- status: active / suspended のみ許可する CHECK 制約付き。
-- session_revoked_after: 停止時に全セッションを失効させるためのタイムスタンプ。
-- その他、停止理由・更新時刻・更新者を監査用に保持する。
ALTER TABLE accounts
  ADD COLUMN IF NOT EXISTS status                TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'suspended')),
  ADD COLUMN IF NOT EXISTS status_reason         TEXT,
  ADD COLUMN IF NOT EXISTS status_updated_at     TIMESTAMPTZ,
  ADD COLUMN IF NOT EXISTS status_updated_by     TEXT,
  ADD COLUMN IF NOT EXISTS session_revoked_after TIMESTAMPTZ;

-- 既存レコードはデフォルトで active となるため、インデックスは任意。
-- ただし suspended 判定のパフォーマンスを考慮し、status インデックスを作成する。
CREATE INDEX IF NOT EXISTS idx_accounts_status ON accounts(status);
