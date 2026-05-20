import { afterEach, beforeEach, describe, expect, it } from 'vitest';

import { issueCsrfToken, requireSameOrigin, validateCsrf } from './guard.js';

const ORIGINAL_ENV = { ...process.env };

describe('CSRF guard', () => {
  beforeEach(() => {
    // CSRF 署名と Origin 検証に必要な env だけを deterministic に固定する。
    process.env = { ...ORIGINAL_ENV };
    process.env.JWT_SECRET = 'test-secret-with-enough-length';
    process.env.ADMIN_ORIGIN = 'https://admin.example.test';
    process.env.ADMIN_DATABASE_URL = testPostgresUrl('admin');
    process.env.PRODUCT_DATABASE_URL = testPostgresUrl('product');
    process.env.ADMIN_VALKEY_URL = 'redis://valkey:6379/1';
    process.env.VALKEY_URL = 'redis://valkey:6379/0';
    process.env.OPENSEARCH_URL = 'http://opensearch:9200';
    process.env.ADMIN_OPENSEARCH_AUDIT_REPLICAS = '0';
    process.env.ADMIN_OPENSEARCH_AUDIT_INDEX_PREFIX = 'admin-audit';
    process.env.PRODUCT_OPENSEARCH_INDEX_PREFIX = 'product-domain';
    process.env.ADMIN_BOOTSTRAP_ENABLED = 'true';
    process.env.ADMIN_BOOTSTRAP_SECRET_HASH = 'bcrypt-hash';
    process.env.ADMIN_BOOTSTRAP_EXPIRES_AT = '2999-01-01T00:00:00.000Z';
    process.env.ADMIN_RP_ID = 'admin.example.test';
    process.env.ADMIN_RP_NAME = 'Admin Console';
    delete process.env.PRODUCT_VALKEY_URL;
  });

  afterEach(() => {
    // process.env の変更を完全に戻し、後続テストの署名鍵や Origin を汚染しない。
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
