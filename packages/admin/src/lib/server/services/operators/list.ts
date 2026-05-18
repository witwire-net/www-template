import * as operatorModel from '../../models/operators.js';

import type { Operator } from '../../models/types.js';
import type { PrismaClient as AdminPrismaClient } from '.prisma/admin-client';

/**
 * オペレーター一覧を取得する。
 * モデル層の `listOperators` に委譲する。
 *
 * @param adminPrisma Admin PrismaClient
 * @returns オペレーター配列
 */
export async function listOperators(adminPrisma: AdminPrismaClient): Promise<Operator[]> {
  return operatorModel.listOperators(adminPrisma);
}

/**
 * ID でオペレーターを取得する。
 * モデル層の `findOperatorById` に委譲する。
 *
 * @param adminPrisma Admin PrismaClient
 * @param id オペレーター ID
 * @returns オペレーター、または null
 */
export async function getOperatorById(
  adminPrisma: AdminPrismaClient,
  id: string
): Promise<Operator | null> {
  return operatorModel.findOperatorById(adminPrisma, id);
}
