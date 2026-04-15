ALTER TABLE passkey_credentials
  DROP COLUMN IF EXISTS public_key,
  DROP COLUMN IF EXISTS sign_count,
  DROP COLUMN IF EXISTS aaguid,
  DROP COLUMN IF EXISTS backup_eligible,
  DROP COLUMN IF EXISTS backup_state,
  DROP COLUMN IF EXISTS transports;
