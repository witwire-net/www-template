import { getAdminPrisma } from '$lib/server/infrastructure/db/prisma';
import { requirePermission } from '$lib/server/infrastructure/rbac/guard';
import { listAuditEvents } from '$lib/server/services/audit/list';
import { listOperators } from '$lib/server/services/operators/list';

import type { ServerLoad } from '@sveltejs/kit';

const PAGE_SIZE = 25;

/**
 * 文字列フィルターを Date に変換する。
 *
 * @param value 受け取った文字列
 * @returns 有効な Date、または undefined
 */
function parseDate(value: string | null): Date | undefined {
  // 日付フィルターは不正値を無視し、一覧取得を止めない。
  if (value === null || value === '') return undefined;
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? undefined : date;
}

/**
 * 監査ログ一覧ページの読み込み処理。
 *
 * @param locals 認証済みローカル情報
 * @param url 現在のリクエスト URL
 * @returns 監査イベント一覧とフィルター候補
 */
export const load: ServerLoad = async ({ locals, url }) => {
  // 監査ログは管理操作履歴のため audit:read を要求する。
  requirePermission(locals.operator, 'audit:read');
  const page = Math.max(1, Number(url.searchParams.get('page') ?? '1'));
  const operatorId = url.searchParams.get('operatorId') ?? '';
  const action = url.searchParams.get('action') ?? '';
  const dateFrom = url.searchParams.get('dateFrom') ?? '';
  const dateTo = url.searchParams.get('dateTo') ?? '';
  const adminPrisma = getAdminPrisma();
  const [audit, operators] = await Promise.all([
    listAuditEvents(adminPrisma, {
      operatorId: operatorId === '' ? undefined : operatorId,
      action: action === '' ? undefined : action,
      dateFrom: parseDate(dateFrom),
      dateTo: parseDate(dateTo),
      page,
      limit: PAGE_SIZE,
    }),
    listOperators(adminPrisma),
  ]);

  return { ...audit, operators, filters: { operatorId, action, dateFrom, dateTo } };
};
