import { getAdminPrisma } from '$lib/server/infrastructure/db/prisma';
import { requirePermission } from '$lib/server/infrastructure/rbac/guard';
import { listOperators } from '$lib/server/services/operators/list';

import type { ServerLoad } from '@sveltejs/kit';

/**
 * 設定ページの読み込み処理。
 *
 * @param locals 認証済みローカル情報
 * @returns オペレーター数の集計値
 */
export const load: ServerLoad = async ({ locals }) => {
  // Settings は admin 専用の運用領域なので operators:read で入口を保護する。
  requirePermission(locals.operator, 'operators:read');
  const operators = await listOperators(getAdminPrisma());
  return {
    operatorCount: operators.length,
    activeOperatorCount: operators.filter((operator) => operator.isActive).length,
  };
};
