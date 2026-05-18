import { getEnvConfig } from '../config/env.js';

import type { Client as OpenSearchClient } from '@opensearch-project/opensearch';

/**
 * Admin 監査ログ用の月次インデックス名を構築する。
 *
 * @param date 対象日付
 * @returns `${prefix}-YYYY.MM` 形式のインデックス名
 */
export function buildAdminAuditIndexName(date: Date): string {
  const { adminOpensearchAuditIndexPrefix } = getEnvConfig();
  const year = date.getUTCFullYear();
  const month = String(date.getUTCMonth() + 1).padStart(2, '0');
  return `${adminOpensearchAuditIndexPrefix}-${String(year)}.${month}`;
}

/**
 * Admin 監査ログインデックスの wildcard パターンを構築する。
 *
 * @returns `${prefix}-*` 形式のパターン
 */
export function buildAdminAuditIndexPattern(): string {
  const { adminOpensearchAuditIndexPrefix } = getEnvConfig();
  return `${adminOpensearchAuditIndexPrefix}-*`;
}

/**
 * Product ドメインインデックスの wildcard パターンを構築する（将来利用）。
 *
 * @returns `${prefix}-*` 形式のパターン
 */
export function buildProductDomainIndexPattern(): string {
  const { productOpensearchIndexPrefix } = getEnvConfig();
  return `${productOpensearchIndexPrefix}-*`;
}

/**
 * OpenSearch にインデックスする監査イベントドキュメント。
 */
export interface AuditEventDocument {
  id: string;
  operator_id: string;
  operator_email?: string;
  operator_name?: string;
  action: string;
  target_type: string;
  target_id: string;
  details?: unknown;
  details_json?: string;
  ip_address?: string;
  created_at: string;
}

/**
 * 監査イベントを OpenSearch に非同期でインデックスする。
 *
 * @param event 監査イベントドキュメント
 * @param opensearch OpenSearch クライアント
 */
export async function indexAuditEvent(
  event: AuditEventDocument,
  opensearch: OpenSearchClient
): Promise<void> {
  const indexName = buildAdminAuditIndexName(new Date(event.created_at));
  try {
    await opensearch.index({
      index: indexName,
      body: event as never,
    });
  } catch (error) {
    globalThis.console.warn('Admin audit OpenSearch indexing failed', error);
  }
}

/**
 * 監査イベント検索クエリ。
 */
export interface AuditQuery {
  query?: string;
  operatorId?: string;
  action?: string;
  from?: string;
  to?: string;
  limit?: number;
  offset?: number;
}

/**
 * Admin 監査 namespace 内で検索する。
 * raw index 名、_all、カンマ区切り、cross-namespace は拒否する。
 *
 * @param query 検索クエリ
 * @param opensearch OpenSearch クライアント
 * @returns 検索結果
 * @throws Error 不正なクエリが含まれる場合
 */
export async function searchAuditEvents(
  query: AuditQuery,
  opensearch: OpenSearchClient
): Promise<unknown> {
  if (query.query !== undefined && query.query !== '' && isUnsafeSearchQuery(query.query)) {
    throw new Error('Invalid search query: cross-index or _all is not allowed');
  }

  const indexPattern = buildAdminAuditIndexPattern();

  const must: Record<string, unknown>[] = [{ match_all: {} }];
  if (query.query !== undefined && query.query !== '') {
    must.push({ query_string: { query: query.query } });
  }
  if (query.operatorId !== undefined && query.operatorId !== '') {
    must.push({ term: { operator_id: query.operatorId } });
  }
  if (query.action !== undefined && query.action !== '') {
    must.push({ term: { action: query.action } });
  }
  if (query.from !== undefined && query.to !== undefined) {
    must.push({ range: { created_at: { gte: query.from, lte: query.to } } });
  }

  const result = await opensearch.search({
    index: indexPattern,
    body: {
      query: { bool: { must } },
      sort: [{ created_at: { order: 'desc' } }],
      size: query.limit ?? 20,
      from: query.offset ?? 0,
    } as never,
  });
  return result.body;
}

function isUnsafeSearchQuery(query: string): boolean {
  // route/service/model が raw index 指定や cross namespace 検索を query 文字列へ混ぜても、Admin audit namespace から出さない。
  return (
    query === '_all' ||
    query.includes('_all:') ||
    query.includes(',') ||
    query.includes('index:') ||
    query.includes('_index') ||
    query.includes('product-') ||
    query.includes('admin-audit-')
  );
}

/**
 * 監査ログの集計統計を取得する。
 *
 * @param opensearch OpenSearch クライアント
 * @returns 集計結果
 */
export async function getAuditStats(opensearch: OpenSearchClient): Promise<unknown> {
  const indexPattern = buildAdminAuditIndexPattern();
  const result = await opensearch.search({
    index: indexPattern,
    body: {
      size: 0,
      aggs: {
        by_action: { terms: { field: 'action', size: 50 } },
        by_outcome: { terms: { field: 'outcome', size: 10 } },
      },
    } as never,
  });
  return result.body;
}
