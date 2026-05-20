-- Account.Auth にぶら下がる passkey credential child テーブルを作成する。
-- 旧 table 名は作成せず、管理ビューと Auth 永続化が同じ canonical 名を参照できるようにする。
CREATE TABLE IF NOT EXISTS account_passkey_credentials (
  id                TEXT PRIMARY KEY,
  account_id        TEXT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
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

-- Account 詳細と管理ビューの集計で Account 単位の credential を高速に数える。
CREATE INDEX IF NOT EXISTS idx_account_passkey_credentials_account_id ON account_passkey_credentials(account_id);

-- WebAuthn identifier による検索経路を維持し、認証フローの credential lookup を安定させる。
CREATE INDEX IF NOT EXISTS idx_account_passkey_credentials_identifier ON account_passkey_credentials(identifier);
