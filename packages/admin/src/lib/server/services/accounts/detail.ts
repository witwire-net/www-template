import * as accountModel from '../../models/accounts.js';

import type { AccountSummary, PasskeyInfo } from '../../models/types.js';
import type { PrismaClient as AdminPrismaClient } from '.prisma/admin-client';
import type { PrismaClient as ProductPrismaClient } from '.prisma/product-client';

/**
 * アカウント詳細取得結果。
 */
export interface AccountDetailResult {
  account: AccountSummary;
  passkeys: PasskeyInfo[];
}

/**
 * アカウント詳細情報を取得する。
 * アカウント概要に加え、紐づく passkey 一覧も Product DB から取得する。
 *
 * @param productPrisma Product PrismaClient
 * @param adminPrisma Admin PrismaClient（将来の拡張用に受け取るが、現状は Product DB のみ使用）
 * @param id アカウント ID
 * @returns アカウント詳細、または null（存在しない場合）
 */
export async function getAccountDetail(
  productPrisma: ProductPrismaClient,
  adminPrisma: AdminPrismaClient,
  id: string
): Promise<AccountDetailResult | null> {
  // adminPrisma は将来的な拡張（Admin DB 側のメモ・タグ等）のため予約
  void adminPrisma;

  const account = await accountModel.getAccountById(productPrisma, id);
  if (account === null) {
    return null;
  }

  const passkeys = await accountModel.getAccountPasskeys(productPrisma, id);

  return {
    account,
    passkeys,
  };
}
