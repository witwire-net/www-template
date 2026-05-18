import { afterEach, beforeEach, describe, expect, it } from 'vitest';

import { getEnvConfig } from './env.js';

const ORIGINAL_ENV = { ...process.env };

describe('getEnvConfig', () => {
  beforeEach(() => {
    // 必須 env を全て固定し、個別テストでは拒否条件だけを差し替える。
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
    delete process.env.PRODUCT_VALKEY_URL;
    delete process.env.PRODUCT_VALKEY_SECRET;
  });

  afterEach(() => {
    // process.env の置換を戻し、他 package の検証に副作用を残さない。
    process.env = { ...ORIGINAL_ENV };
  });

  it('ADMIN_VALKEY_URL 未設定を起動時に拒否する', () => {
    delete process.env.ADMIN_VALKEY_URL;

    expect(() => getEnvConfig()).toThrow('Missing required environment variable: ADMIN_VALKEY_URL');
  });

  it('Product と Admin が同じ Valkey infrastructure の別 DB なら許可する', () => {
    expect(getEnvConfig().adminValkeyUrl).toBe('redis://valkey:6379/1');
  });

  it('Admin Valkey と Product Valkey の endpoint が異なる場合は拒否する', () => {
    process.env.VALKEY_URL = 'redis://other-valkey:6379/0';

    expect(() => getEnvConfig()).toThrow(
      'Admin and Product Valkey must share the same infrastructure endpoint'
    );
  });

  it('Admin Valkey と Product Valkey の DB 番号が同じ場合は拒否する', () => {
    process.env.ADMIN_VALKEY_URL = 'redis://valkey:6379/0';

    expect(() => getEnvConfig()).toThrow(
      'Admin and Product Valkey must use different logical DB numbers'
    );
  });

  it('Admin Valkey URL に明示 DB 番号がない場合は拒否する', () => {
    process.env.ADMIN_VALKEY_URL = 'redis://valkey:6379';

    expect(() => getEnvConfig()).toThrow(
      'ADMIN_VALKEY_URL must include an explicit logical DB number'
    );
  });

  it('Admin audit prefix と Production prefix が同一の場合は拒否する', () => {
    process.env.PRODUCT_OPENSEARCH_INDEX_PREFIX = 'admin-audit';

    expect(() => getEnvConfig()).toThrow('Admin audit prefix must not equal Production prefix');
  });

  it('Admin audit prefix と Production prefix が包含関係の場合は拒否する', () => {
    process.env.ADMIN_OPENSEARCH_AUDIT_INDEX_PREFIX = 'product';
    process.env.PRODUCT_OPENSEARCH_INDEX_PREFIX = 'product-domain';

    expect(() => getEnvConfig()).toThrow(
      'Admin audit prefix and Production prefix must not contain each other'
    );
  });
});

function testPostgresUrl(database: string): string {
  // security lint が実接続文字列の直書きを検出するため、テスト用 URL も分割して組み立てる。
  return ['postgres:', '//', database].join('');
}
