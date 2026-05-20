-- Account ごとの表示・通知設定を保持する必須 child テーブルを作成する。
-- locale は Product Account の永続設定であり、対応値は ja / en に限定して fail-closed にする。
CREATE TABLE IF NOT EXISTS account_settings (
  account_id TEXT PRIMARY KEY REFERENCES accounts(id) ON DELETE CASCADE,
  locale     TEXT NOT NULL DEFAULT 'ja' CHECK (locale IN ('ja', 'en')),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 既存 Account root がある状態で本 migration が適用されても必須 child が欠落しないように補完する。
INSERT INTO account_settings (account_id)
SELECT id
FROM accounts
ON CONFLICT (account_id) DO NOTHING;

-- Account root 作成と同時に既定 AccountSetting を作成し、application 経路の漏れでも locale snapshot を欠落させない。
CREATE OR REPLACE FUNCTION create_default_account_setting()
RETURNS TRIGGER AS $$
BEGIN
  INSERT INTO account_settings (account_id)
  VALUES (NEW.id)
  ON CONFLICT (account_id) DO NOTHING;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS accounts_create_default_account_setting ON accounts;
CREATE TRIGGER accounts_create_default_account_setting
AFTER INSERT ON accounts
FOR EACH ROW
EXECUTE FUNCTION create_default_account_setting();
