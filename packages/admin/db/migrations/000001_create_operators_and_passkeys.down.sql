-- Admin DB の初期永続化 schema を削除する。
-- 既知の table を明示的に削除し、未知 object が admin schema に残る場合は schema drop を失敗させる。
DROP TABLE IF EXISTS admin.audit_events;
DROP TABLE IF EXISTS admin.operator_passkeys;
DROP TABLE IF EXISTS admin.operators;
DROP SCHEMA IF EXISTS admin;
