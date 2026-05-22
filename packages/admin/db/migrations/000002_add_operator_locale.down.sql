-- operator locale の制約と列を削除し、000002 適用前の Admin DB schema に戻す。
ALTER TABLE admin.operators
    DROP CONSTRAINT IF EXISTS operators_locale_check;

ALTER TABLE admin.operators
    DROP COLUMN IF EXISTS locale;
