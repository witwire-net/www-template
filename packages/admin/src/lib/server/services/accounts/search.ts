import * as accountModel from '../../models/accounts.js';

import type { AccountSummary } from '../../models/types.js';
import type { PrismaClient as ProductPrismaClient } from '.prisma/product-client';

/**
 * アカウント検索の入力パラメータ。
 */
export interface SearchAccountsParams {
  /** Email 部分一致クエリ */
  query?: string;
  /** ステータスフィルタ */
  status?: string;
  /** ページ番号（1-based） */
  page: number;
  /** 1ページあたり件数 */
  limit: number;
}

/**
 * アカウント検索結果。
 */
export interface SearchAccountsResult {
  accounts: AccountSummary[];
  total: number;
  page: number;
  totalPages: number;
}

/**
 * アカウントを検索し、ページネーション付きで返す。
 * モデル層の `searchAccounts` に委譲する。
 *
 * @param productPrisma Product PrismaClient
 * @param params 検索パラメータ
 * @returns アカウント一覧とページネーション情報
 */
export async function searchAccounts(
  productPrisma: ProductPrismaClient,
  params: SearchAccountsParams
): Promise<{ accounts: AccountSummary[]; total: number; page: number; totalPages: number }> {
  const offset = (params.page - 1) * params.limit;
  const result = await accountModel.searchAccounts(productPrisma, {
    query: params.query,
    status: params.status,
    limit: params.limit,
    offset,
  });

  const totalNumber = Number(result.total);
  const totalPages = totalNumber > 0 ? Math.ceil(totalNumber / params.limit) : 0;

  return {
    accounts: result.items,
    total: totalNumber,
    page: params.page,
    totalPages,
  };
}
