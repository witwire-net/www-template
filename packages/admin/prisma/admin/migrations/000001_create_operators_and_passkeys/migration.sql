-- Admin DB Prisma Migrate 用初期マイグレーション
-- operators / operator_passkeys / audit_events テーブルを admin スキーマに作成する
-- 初期オペレーターは作成しない（初回起動セットアップで作成）

-- admin スキーマを作成（存在しない場合）
CREATE SCHEMA IF NOT EXISTS admin;

-- operators テーブル: 管理画面オペレーター
CREATE TABLE IF NOT EXISTS admin.operators (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::TEXT,
    email TEXT NOT NULL UNIQUE,
    display_name TEXT NOT NULL,
    role TEXT NOT NULL DEFAULT 'operator' CHECK (role IN ('admin', 'operator', 'viewer')),
    is_active BOOLEAN NOT NULL DEFAULT true,
    setup_token_hash TEXT,
    setup_token_expires_at TIMESTAMPTZ,
    last_login_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- operator_passkeys テーブル: オペレーターが登録した WebAuthn passkey credential
CREATE TABLE IF NOT EXISTS admin.operator_passkeys (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::TEXT,
    operator_id TEXT NOT NULL,
    credential_handle TEXT NOT NULL UNIQUE,
    public_key BYTEA NOT NULL,
    sign_count BIGINT NOT NULL DEFAULT 0,
    aaguid BYTEA NOT NULL,
    backup_eligible BOOLEAN NOT NULL,
    backup_state BOOLEAN NOT NULL,
    transports JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_operator_passkeys_operator
        FOREIGN KEY (operator_id)
        REFERENCES admin.operators(id)
        ON DELETE CASCADE
);

-- audit_events テーブル: 監査イベント（全操作の追跡・証跡）
CREATE TABLE IF NOT EXISTS admin.audit_events (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::TEXT,
    operator_id TEXT NOT NULL,
    action TEXT NOT NULL,
    target_type TEXT NOT NULL,
    target_id TEXT NOT NULL,
    details JSONB,
    outcome TEXT NOT NULL CHECK (outcome IN ('pending', 'succeeded', 'failed', 'indeterminate')),
    error_code TEXT,
    ip_address TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMPTZ,
    CONSTRAINT fk_audit_events_operator
        FOREIGN KEY (operator_id)
        REFERENCES admin.operators(id)
);

-- インデックス作成
CREATE INDEX IF NOT EXISTS idx_operators_email ON admin.operators(email);
CREATE INDEX IF NOT EXISTS idx_operators_role ON admin.operators(role);
CREATE INDEX IF NOT EXISTS idx_operators_is_active ON admin.operators(is_active);
CREATE INDEX IF NOT EXISTS idx_operator_passkeys_operator_id ON admin.operator_passkeys(operator_id);
CREATE INDEX IF NOT EXISTS idx_operator_passkeys_credential_handle ON admin.operator_passkeys(credential_handle);
CREATE INDEX IF NOT EXISTS idx_audit_events_operator_id ON admin.audit_events(operator_id);
CREATE INDEX IF NOT EXISTS idx_audit_events_action ON admin.audit_events(action);
CREATE INDEX IF NOT EXISTS idx_audit_events_target_type ON admin.audit_events(target_type);
CREATE INDEX IF NOT EXISTS idx_audit_events_created_at ON admin.audit_events(created_at);
CREATE INDEX IF NOT EXISTS idx_audit_events_outcome ON admin.audit_events(outcome);

-- 初期オペレーターは作成しない（Task 1.11 / 1.12 の要求）
-- 初回 admin は /setup の bootstrap フローで作成される
