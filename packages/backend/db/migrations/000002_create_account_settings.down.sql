-- AccountSetting child を削除し、locale 永続化の追加分だけを戻す。
DROP TRIGGER IF EXISTS accounts_create_default_account_setting ON accounts;
DROP FUNCTION IF EXISTS create_default_account_setting();
DROP TABLE IF EXISTS account_settings;
