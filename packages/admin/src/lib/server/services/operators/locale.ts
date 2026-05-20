import { parseOperatorLocale } from '../../models/operator_locale.js';
import * as operatorModel from '../../models/operators.js';
import { ServiceError } from '../errors.js';

import type { Operator } from '../../models/types.js';
import type { PrismaClient as AdminPrismaClient } from '.prisma/admin-client';

/**
 * 認証済みオペレーター本人の Admin Console 表示 locale を更新します。
 *
 * @param adminPrisma Admin PrismaClient
 * @param operatorId 認証済み本人の operator ID
 * @param localeValue form から受け取った locale 候補値
 * @returns 更新後のオペレーター
 * @throws ServiceError 未対応 locale、対象 operator が存在しない場合、または対象 operator が無効化済みの場合
 */
export async function updateOwnOperatorLocale(
  adminPrisma: AdminPrismaClient,
  operatorId: string,
  localeValue: string
): Promise<Operator> {
  // 未対応 locale は永続化層に到達させず、保存済み値を変えない 400 として扱う。
  let locale;
  try {
    locale = parseOperatorLocale(localeValue);
  } catch {
    throw new ServiceError('Unsupported operator locale', 400, 'UNSUPPORTED_OPERATOR_LOCALE');
  }

  // 更新前に本人 operator の存在と有効状態を確認し、存在しない ID や無効化済み operator の locale を変更しない。
  const existingOperator = await operatorModel.findOperatorById(adminPrisma, operatorId);
  if (existingOperator === null) {
    throw new ServiceError('Operator not found', 404, 'OPERATOR_NOT_FOUND');
  }
  if (!existingOperator.isActive) {
    throw new ServiceError('Operator is inactive', 403, 'OPERATOR_INACTIVE');
  }

  // 更新対象は hooks.server.ts が認証した本人 ID に固定し、form から別 operator ID を受け取らない。
  const operator = await operatorModel.updateOperatorLocale(adminPrisma, operatorId, locale);
  return operator;
}
