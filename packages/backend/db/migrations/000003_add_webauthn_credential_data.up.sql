-- WebAuthn Credential Record fields for lossless authenticator state restoration.
-- These fields allow the server to reconstruct a webauthn.Credential when
-- performing signature verification during future login ceremonies.

ALTER TABLE passkey_credentials
  ADD COLUMN IF NOT EXISTS public_key         BYTEA,
  ADD COLUMN IF NOT EXISTS sign_count         BIGINT NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS aaguid             BYTEA,
  ADD COLUMN IF NOT EXISTS backup_eligible    BOOLEAN NOT NULL DEFAULT FALSE,
  ADD COLUMN IF NOT EXISTS backup_state       BOOLEAN NOT NULL DEFAULT FALSE,
  ADD COLUMN IF NOT EXISTS transports         JSONB   NOT NULL DEFAULT '[]'::jsonb;
