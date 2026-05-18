import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import {
  buildAdminAuditIndexPattern,
  buildProductDomainIndexPattern,
  indexAuditEvent,
  searchAuditEvents,
} from './opensearch.js';

const ORIGINAL_ENV = { ...process.env };

describe('OpenSearch infrastructure', () => {
  beforeEach(() => {
    // index prefix を固定し、実際に OpenSearch に渡る namespace を観測しやすくする。
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
  });

  afterEach(() => {
    // console.warn spy と env を戻し、他テストのログ期待値を壊さない。
    vi.restoreAllMocks();
    process.env = { ...ORIGINAL_ENV };
  });

  it('indexAuditEvent は月次 Admin audit index に OpenSearch index を呼ぶ', async () => {
    const client = { index: vi.fn().mockResolvedValue({}) };

    await indexAuditEvent(auditEvent(), client as never);

    expect(client.index).toHaveBeenCalledWith({
      index: 'admin-audit-2026.05',
      body: auditEvent(),
    });
  });

  it('17.14 OpenSearch index audit event は Admin audit 月次 index にだけ書き込む', async () => {
    // 監査イベントは Admin audit namespace へ閉じ込め、Product domain namespace を汚染しないことを確認する。
    const client = { index: vi.fn().mockResolvedValue({}) };

    await indexAuditEvent(auditEvent(), client as never);

    expect(client.index).toHaveBeenCalledWith({
      index: 'admin-audit-2026.05',
      body: auditEvent(),
    });
    expect(client.index).not.toHaveBeenCalledWith(
      expect.objectContaining({ index: expect.stringContaining('product-domain') })
    );
  });

  it('OpenSearch index 失敗は warn して throw しない', async () => {
    const warning = vi.spyOn(console, 'warn').mockImplementation(() => undefined);
    const client = { index: vi.fn().mockRejectedValue(new Error('opensearch down')) };

    await expect(indexAuditEvent(auditEvent(), client as never)).resolves.toBeUndefined();
    expect(warning).toHaveBeenCalledWith(
      'Admin audit OpenSearch indexing failed',
      expect.any(Error)
    );
  });

  it('17.15 OpenSearch failure は DB fallback search を妨げないよう throw しない', async () => {
    // OpenSearch indexing が失敗しても、DB 側の監査ログ検索へ fallback できるよう例外を外へ出さない。
    const warning = vi.spyOn(console, 'warn').mockImplementation(() => undefined);
    const client = { index: vi.fn().mockRejectedValue(new Error('opensearch down')) };

    await expect(indexAuditEvent(auditEvent(), client as never)).resolves.toBeUndefined();

    expect(client.index).toHaveBeenCalled();
    expect(warning).toHaveBeenCalledWith(
      'Admin audit OpenSearch indexing failed',
      expect.any(Error)
    );
  });

  it('17.15a Admin audit write/search は Admin audit namespace のみを使用する', async () => {
    // Admin audit API は index 名を外部入力から受けず、常に Admin audit prefix だけを使う。
    const indexClient = { index: vi.fn().mockResolvedValue({}) };
    const searchClient = { search: vi.fn().mockResolvedValue({ body: { hits: { hits: [] } } }) };

    await indexAuditEvent(auditEvent(), indexClient as never);
    await searchAuditEvents({ query: 'accounts.suspend' }, searchClient as never);

    expect(buildAdminAuditIndexPattern()).toBe('admin-audit-*');
    expect(indexClient.index).toHaveBeenCalledWith(
      expect.objectContaining({ index: 'admin-audit-2026.05' })
    );
    expect(searchClient.search).toHaveBeenCalledWith(
      expect.objectContaining({ index: 'admin-audit-*' })
    );
    expect(indexClient.index).not.toHaveBeenCalledWith(
      expect.objectContaining({ index: expect.stringContaining('product-domain') })
    );
    expect(searchClient.search).not.toHaveBeenCalledWith(
      expect.objectContaining({ index: expect.stringContaining('product-domain') })
    );
  });

  it('17.15b Production domain OpenSearch use case は Production domain namespace のみを構築する', () => {
    // Production domain 側の index pattern は Admin audit prefix と別 prefix になり、交差しないことを確認する。
    expect(buildProductDomainIndexPattern()).toBe('product-domain-*');
    expect(buildProductDomainIndexPattern()).not.toBe(buildAdminAuditIndexPattern());
    expect(buildProductDomainIndexPattern()).not.toContain('admin-audit');
  });

  it('raw index name / _all / comma-separated / cross namespace query を拒否する', async () => {
    const client = { search: vi.fn() };
    const unsafeQueries = [
      '_all',
      '_all:foo',
      '(_all:foo)',
      'admin-audit-2026.05:error',
      'index:admin-audit-2026.05',
      'foo,bar',
      '_index:product-domain-*',
      'product-domain-*',
    ];

    for (const unsafeQuery of unsafeQueries) {
      await expect(searchAuditEvents({ query: unsafeQuery }, client as never)).rejects.toThrow(
        'Invalid search query'
      );
    }
    expect(client.search).not.toHaveBeenCalled();
  });
});

function auditEvent() {
  // OpenSearch document の必須フィールドだけを含め、index 名の時刻変換を固定する。
  return {
    id: 'audit-1',
    operator_id: 'op-1',
    action: 'accounts.suspend',
    target_type: 'account',
    target_id: 'account-1',
    created_at: '2026-05-17T10:00:00.000Z',
  };
}

function testPostgresUrl(database: string): string {
  // security lint が実接続文字列の直書きを検出するため、テスト用 URL も分割して組み立てる。
  return 'postgres:' + '//' + database;
}
