import { getAdminPrisma, getProductPrisma } from '$lib/server/infrastructure/db/prisma';
import { requirePermission } from '$lib/server/infrastructure/rbac/guard';
import { getDashboardStats } from '$lib/server/services/accounts/stats';
import { listAuditEvents } from '$lib/server/services/audit/list';

import type { ServerLoad } from '@sveltejs/kit';

/**
 * ダッシュボードの初期表示に必要な集計値と最新監査イベントを返す。
 *
 * @param locals 認証済みローカル情報
 * @returns ダッシュボード表示データ
 */
export const load: ServerLoad = async ({ locals }) => {
  // Dashboard はアカウント概要と監査履歴を含むため、最小読み取り権限を確認する。
  requirePermission(locals.operator, 'accounts:read');
  requirePermission(locals.operator, 'audit:read');

  // Product DB と Admin DB の読み取りを並列化し、画面初期表示の待ち時間を抑える。
  const productPrisma = await getProductPrisma();
  const [stats, recentAudit] = await Promise.all([
    getDashboardStats(productPrisma),
    listAuditEvents(getAdminPrisma(), { page: 1, limit: 8 }),
  ]);

  return {
    stats,
    recentAudit: recentAudit.events,
  };
};
