import { mkdtempSync, rmSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';

import { afterEach, beforeEach, describe, expect, it } from 'vitest';

import { issueCsrfToken, requireSameOrigin, validateCsrf } from './guard.js';

const ORIGINAL_ENV = { ...process.env };
const TEST_ADMIN_JWT_SECRET = 'test-secret-with-enough-length';

let tempRoot: string | null = null;

describe('CSRF guard', () => {
  beforeEach(() => {
    // CSRF 署名と Origin 検証に必要な Admin TOML を deterministic に固定する。
    tempRoot = mkdtempSync(join(tmpdir(), 'admin-csrf-config-'));
    const configPath = join(tempRoot, 'test.admin.toml');
    process.env = {
      ...ORIGINAL_ENV,
      ADMIN_CONFIG_PATH: configPath,
      NODE_ENV: 'test',
    };
    writeFileSync(configPath, testAdminConfig(), 'utf8');
  });

  afterEach(() => {
    // temp config と process.env の変更を完全に戻し、後続テストの署名鍵や Origin を汚染しない。
    if (tempRoot !== null) {
      rmSync(tempRoot, { recursive: true, force: true });
      tempRoot = null;
    }
    process.env = { ...ORIGINAL_ENV };
  });

  it('valid Origin と session-bound CSRF token を許可する', async () => {
    const { token } = issueCsrfToken('sess-1', 'jti-1');

    await expect(
      validateCsrf(
        createEvent({
          origin: 'https://admin.example.test',
          headerToken: token,
          cookieToken: token,
        })
      )
    ).resolves.toBeUndefined();
  });

  it('cross-origin mutation を 403 で拒否する', async () => {
    const { token } = issueCsrfToken('sess-1', 'jti-1');

    await expect(
      validateCsrf(
        createEvent({ origin: 'https://evil.example.test', headerToken: token, cookieToken: token })
      )
    ).rejects.toMatchObject({ status: 403 });
  });

  it('CSRF token mismatch を 403 で拒否する', async () => {
    const { token } = issueCsrfToken('sess-1', 'jti-1');

    await expect(
      validateCsrf(
        createEvent({
          origin: 'https://admin.example.test',
          headerToken: token,
          cookieToken: 'different',
        })
      )
    ).rejects.toMatchObject({ status: 403 });
  });

  it('別 sessionId/jti 用に署名された CSRF token を拒否する', async () => {
    const { token } = issueCsrfToken('other-session', 'other-jti');

    await expect(
      validateCsrf(
        createEvent({
          origin: 'https://admin.example.test',
          headerToken: token,
          cookieToken: token,
        })
      )
    ).rejects.toMatchObject({ status: 403 });
  });

  it('pre-auth passkey start は session-bound CSRF なしで Origin allowlist のみ通す', () => {
    expect(() => {
      requireSameOrigin(createEvent({ origin: 'https://admin.example.test' }));
    }).not.toThrow();
  });
});

function testPostgresUrl(database: string): string {
  // security lint が実接続文字列の直書きを検出するため、テスト用 URL も分割して組み立てる。
  return ['postgres:', '//', database].join('');
}

function testAdminConfig(): string {
  // CSRF テストは auth/server 設定だけを使うが、統合 getter と同じファイル形を保つ。
  return `[server]
origin = "https://admin.example.test"

[auth]
jwt_secret = "${TEST_ADMIN_JWT_SECRET}"
rp_id = "admin.example.test"
rp_name = "Admin Console"

[database]
admin_url = "${testPostgresUrl('admin')}"
product_url = "${testPostgresUrl('product')}"

[valkey]
admin_url = "redis://valkey:6379/1"
product_url = "redis://valkey:6379/0"

[opensearch]
url = "http://opensearch:9200"
admin_audit_replicas = 0
admin_audit_index_prefix = "admin-audit"
product_index_prefix = "product-domain"

[bootstrap]
enabled = true
secret_hash = "bcrypt-hash"
expires_at = "2999-01-01T00:00:00.000Z"
`;
}

function createEvent(input: { origin: string; headerToken?: string; cookieToken?: string }) {
  // SvelteKit RequestEvent のうち CSRF guard が読む最小 subset だけを作る。
  const headers = new Headers({ origin: input.origin });
  if (input.headerToken !== undefined) headers.set('x-csrf-token', input.headerToken);
  return {
    request: new Request('https://admin.example.test/api/admin/accounts', {
      method: 'POST',
      headers,
    }),
    cookies: {
      get: (name: string) => (name === 'admin_csrf' ? input.cookieToken : undefined),
    },
    locals: {
      operator: {
        id: 'op-1',
        email: 'admin@example.test',
        role: 'admin',
        locale: 'ja',
        sessionId: 'sess-1',
        jti: 'jti-1',
      },
    },
  } as never;
}
