-- Admin operator の表示言語を Admin DB の operators table に保存する。
-- Admin Console package-local の永続値として扱い、外部 schema は参照しない。

-- 既存 operator は明示的な設定を持たないため、既定値 ja で埋める。
-- NOT NULL と DEFAULT を同時に付け、以後の operator 作成でも保存済み locale を必ず持たせる。
ALTER TABLE admin.operators
    ADD COLUMN IF NOT EXISTS locale TEXT NOT NULL DEFAULT 'ja';

-- 未対応 locale を DB 層で拒否し、未知値を既定値へ黙って丸めない fail-closed な永続化にする。
ALTER TABLE admin.operators
    DROP CONSTRAINT IF EXISTS operators_locale_check;

ALTER TABLE admin.operators
    ADD CONSTRAINT operators_locale_check CHECK (locale IN ('ja', 'en'));
