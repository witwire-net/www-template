import type { PrismaClient as ProductPrismaClient } from '.prisma/product-client';

/**
 * ダッシュボード統計結果。
 */
export interface DashboardStats {
  totalAccounts: number;
  activeAccounts: number;
  suspendedAccounts: number;
  recentAccounts: {
    id: string;
    email: string;
    status: string;
    createdAt: Date;
  }[];
}

/**
 * ダッシュボード用の統計情報を Product DB から取得する。
 *
 * - totalAccounts: アカウント総数
 * - activeAccounts: ステータス 'active' のアカウント数
 * - suspendedAccounts: ステータス 'suspended' のアカウント数
 * - recentAccounts: 直近 5 件の新規アカウント（作成日時降順）
 *
 * @param productPrisma Product PrismaClient
 * @returns ダッシュボード統計
 */
export async function getDashboardStats(
  productPrisma: ProductPrismaClient
): Promise<DashboardStats> {
  const [totalResult, activeResult, suspendedResult, recentResult] = await Promise.all([
    productPrisma.$queryRaw<{ count: bigint }[]>`
			SELECT COUNT(*) as count FROM admin_view.account_summaries
		`,
    productPrisma.$queryRaw<{ count: bigint }[]>`
			SELECT COUNT(*) as count FROM admin_view.account_summaries WHERE status = 'active'
		`,
    productPrisma.$queryRaw<{ count: bigint }[]>`
			SELECT COUNT(*) as count FROM admin_view.account_summaries WHERE status = 'suspended'
		`,
    productPrisma.$queryRaw<
      {
        id: string;
        email: string;
        status: string;
        created_at: Date;
      }[]
    >`
			SELECT id, email, status, created_at
			FROM admin_view.account_summaries
			ORDER BY created_at DESC
			LIMIT 5
		`,
  ]);

  const totalAccounts = Number(totalResult[0]?.count ?? 0n);
  const activeAccounts = Number(activeResult[0]?.count ?? 0n);
  const suspendedAccounts = Number(suspendedResult[0]?.count ?? 0n);

  const recentAccounts = recentResult.map((row) => ({
    id: row.id,
    email: row.email,
    status: row.status,
    createdAt: row.created_at,
  }));

  return {
    totalAccounts,
    activeAccounts,
    suspendedAccounts,
    recentAccounts,
  };
}
