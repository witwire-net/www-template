DROP INDEX IF EXISTS idx_accounts_status;

ALTER TABLE accounts
  DROP COLUMN IF EXISTS status,
  DROP COLUMN IF EXISTS status_reason,
  DROP COLUMN IF EXISTS status_updated_at,
  DROP COLUMN IF EXISTS status_updated_by,
  DROP COLUMN IF EXISTS session_revoked_after;
