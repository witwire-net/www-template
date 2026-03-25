CREATE TABLE IF NOT EXISTS accounts (
  id TEXT PRIMARY KEY,
  email TEXT NOT NULL UNIQUE
);

CREATE TABLE IF NOT EXISTS passkey_credentials (
  id TEXT PRIMARY KEY,
  account_id TEXT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
  identifier TEXT NOT NULL,
  credential_handle TEXT NOT NULL UNIQUE
);

CREATE INDEX IF NOT EXISTS idx_passkey_credentials_account_id ON passkey_credentials(account_id);
CREATE INDEX IF NOT EXISTS idx_passkey_credentials_identifier ON passkey_credentials(identifier);
