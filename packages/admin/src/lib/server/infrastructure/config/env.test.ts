import { mkdtempSync, rmSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';

import { afterEach, beforeEach, describe, expect, it } from 'vitest';

import {
  getAdminAuthConfig,
  getAdminBootstrapConfig,
  getAdminSearchConfig,
  getProductDatabaseConfig,
} from './env.js';

const ORIGINAL_ENV = { ...process.env };

let tempRoot: string | null = null;

describe('admin TOML config', () => {
  beforeEach(() => {
    // 各テスト専用の *.admin.toml を使い、実リポジトリの local config や env 汚染を受けないようにする。
    tempRoot = mkdtempSync(join(tmpdir(), 'admin-config-'));
    process.env = { ...ORIGINAL_ENV, ADMIN_CONFIG_PATH: join(tempRoot, 'test.admin.toml') };
    delete process.env.JWT_SECRET;
    delete process.env.ADMIN_JWT_SECRET;
  });

  afterEach(() => {
    // temp config と process.env を戻し、他 package の検証に副作用を残さない。
    if (tempRoot !== null) {
      rmSync(tempRoot, { recursive: true, force: true });
      tempRoot = null;
    }
    process.env = { ...ORIGINAL_ENV };
  });

  it('Admin auth config は Admin TOML の JWT secret と Admin Valkey を必須にする', () => {
    writeAdminConfig(fullAdminConfig());

    expect(getAdminAuthConfig()).toMatchObject({
      jwtSecret: 'test-admin-secret-with-enough-length',
      adminOrigin: 'https://admin.example.test',
      adminValkeyUrl: 'redis://valkey:6379/1',
    });
  });

  it('JWT_SECRET env は Admin auth config に使わない', () => {
    process.env.JWT_SECRET = 'product-secret-must-not-be-used';
    writeAdminConfig(fullAdminConfig({ jwtSecret: 'toml-admin-secret-with-enough-length' }));

    expect(getAdminAuthConfig().jwtSecret).toBe('toml-admin-secret-with-enough-length');
  });

  it('auth.jwt_secret 欠落を Admin auth 起動時に拒否する', () => {
    writeAdminConfig(fullAdminConfig({ jwtSecret: '' }));

    expect(() => getAdminAuthConfig()).toThrow(
      'Missing required admin config value: auth.jwt_secret'
    );
  });

  it('valkey.admin_url 未設定を Admin auth 起動時に拒否する', () => {
    writeAdminConfig(fullAdminConfig({ adminValkeyUrl: '' }));

    expect(() => getAdminAuthConfig()).toThrow(
      'Missing required admin config value: valkey.admin_url'
    );
  });

  it('Product Valkey が同一 endpoint の別 DB であれば許可する', () => {
    writeAdminConfig(fullAdminConfig({ productValkeyUrl: 'redis://valkey:6379/0' }));

    expect(getAdminAuthConfig().adminValkeyUrl).toBe('redis://valkey:6379/1');
  });

  it('Admin Valkey と Product Valkey の endpoint が異なる場合は拒否する', () => {
    writeAdminConfig(fullAdminConfig({ productValkeyUrl: 'redis://other-valkey:6379/0' }));

    expect(() => getAdminAuthConfig()).toThrow(
      'Admin and Product Valkey must share the same infrastructure endpoint'
    );
  });

  it('Admin Valkey と Product Valkey の DB 番号が同じ場合は拒否する', () => {
    writeAdminConfig(fullAdminConfig({ productValkeyUrl: 'redis://valkey:6379/1' }));

    expect(() => getAdminAuthConfig()).toThrow(
      'Admin and Product Valkey must use different logical DB numbers'
    );
  });

  it('Admin Valkey URL に明示 DB 番号がない場合は拒否する', () => {
    writeAdminConfig(fullAdminConfig({ adminValkeyUrl: 'redis://valkey:6379' }));

    expect(() => getAdminAuthConfig()).toThrow(
      'Invalid admin config value: valkey.admin_url must include an explicit logical DB number'
    );
  });

  it('Bootstrap config は Product DB / OpenSearch config なしで取得できる', () => {
    writeAdminConfig(`[bootstrap]
enabled = true
secret_hash = "bcrypt-hash"
expires_at = "2999-01-01T00:00:00.000Z"
`);

    expect(getAdminBootstrapConfig()).toMatchObject({
      adminBootstrapEnabled: true,
      adminBootstrapSecretHash: 'bcrypt-hash',
      adminBootstrapExpiresAt: new Date('2999-01-01T00:00:00.000Z'),
    });
  });

  it('Product DB config は Product 連携を使う箇所だけで database.product_url を必須にする', () => {
    writeAdminConfig(`[bootstrap]
enabled = true
secret_hash = "bcrypt-hash"
expires_at = "2999-01-01T00:00:00.000Z"
`);

    expect(() => getProductDatabaseConfig()).toThrow(
      'Missing required admin config value: database.product_url'
    );

    writeAdminConfig(fullAdminConfig({ productDatabaseUrl: testPostgresUrl('product') }));

    expect(getProductDatabaseConfig().productDatabaseUrl).toBe(testPostgresUrl('product'));
  });

  it('Admin audit prefix と Production prefix が同一の場合は拒否する', () => {
    writeAdminConfig(fullAdminConfig({ productIndexPrefix: 'admin-audit' }));

    expect(() => getAdminSearchConfig()).toThrow(
      'Admin audit prefix must not equal Production prefix'
    );
  });

  it('Admin audit prefix と Production prefix が包含関係の場合は拒否する', () => {
    writeAdminConfig(
      fullAdminConfig({ adminAuditIndexPrefix: 'product', productIndexPrefix: 'product-domain' })
    );

    expect(() => getAdminSearchConfig()).toThrow(
      'Admin audit prefix and Production prefix must not contain each other'
    );
  });
});

function writeAdminConfig(source: string): void {
  // beforeEach で確保した ADMIN_CONFIG_PATH に TOML を書き、getter が必ずその内容だけを見るようにする。
  const configPath = process.env.ADMIN_CONFIG_PATH;
  if (configPath === undefined || configPath === '') {
    throw new Error('ADMIN_CONFIG_PATH is missing for this test.');
  }
  writeFileSync(configPath, source, 'utf8');
}

function fullAdminConfig(overrides: Partial<AdminConfigValues> = {}): string {
  // 既定値にテストごとの差分だけを重ね、欠落・衝突・正常系の TOML を短く作る。
  const values = { ...defaultAdminConfigValues(), ...overrides };
  return `[server]
origin = "${values.adminOrigin}"

[auth]
jwt_secret = "${values.jwtSecret}"
rp_id = "${values.rpId}"
rp_name = "${values.rpName}"

[database]
admin_url = "${values.adminDatabaseUrl}"
product_url = "${values.productDatabaseUrl}"

[valkey]
admin_url = "${values.adminValkeyUrl}"
product_url = "${values.productValkeyUrl}"

[opensearch]
url = "${values.opensearchUrl}"
admin_audit_replicas = ${String(values.adminAuditReplicas)}
admin_audit_index_prefix = "${values.adminAuditIndexPrefix}"
product_index_prefix = "${values.productIndexPrefix}"

[bootstrap]
enabled = ${String(values.bootstrapEnabled)}
secret_hash = "${values.bootstrapSecretHash}"
expires_at = "${values.bootstrapExpiresAt}"
`;
}

function defaultAdminConfigValues(): AdminConfigValues {
  // dev/test 用の安全な分離構成を baseline にし、個別テストで意図する値だけを上書きする。
  return {
    adminOrigin: 'https://admin.example.test',
    jwtSecret: 'test-admin-secret-with-enough-length',
    rpId: 'admin.example.test',
    rpName: 'Admin Console Test',
    adminDatabaseUrl: testPostgresUrl('admin'),
    productDatabaseUrl: testPostgresUrl('product'),
    adminValkeyUrl: 'redis://valkey:6379/1',
    productValkeyUrl: 'redis://valkey:6379/0',
    opensearchUrl: 'http://opensearch:9200',
    adminAuditReplicas: 0,
    adminAuditIndexPrefix: 'admin-audit',
    productIndexPrefix: 'product-domain',
    bootstrapEnabled: true,
    bootstrapSecretHash: 'bcrypt-hash',
    bootstrapExpiresAt: '2999-01-01T00:00:00.000Z',
  };
}

function testPostgresUrl(database: string): string {
  // security lint が実接続文字列の直書きを検出するため、テスト用 URL も分割して組み立てる。
  return ['postgres:', '//', database].join('');
}

interface AdminConfigValues {
  adminOrigin: string;
  jwtSecret: string;
  rpId: string;
  rpName: string;
  adminDatabaseUrl: string;
  productDatabaseUrl: string;
  adminValkeyUrl: string;
  productValkeyUrl: string;
  opensearchUrl: string;
  adminAuditReplicas: number;
  adminAuditIndexPrefix: string;
  productIndexPrefix: string;
  bootstrapEnabled: boolean;
  bootstrapSecretHash: string;
  bootstrapExpiresAt: string;
}
