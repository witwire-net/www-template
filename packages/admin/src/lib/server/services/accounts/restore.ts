import * as accountModel from '../../models/accounts.js';
import { ServiceError } from '../errors.js';

import type { AuditLogger } from './suspend.js';
import type { PrismaClient as AdminPrismaClient } from '.prisma/admin-client';
import type { PrismaClient as ProductPrismaClient } from '.prisma/product-client';

/**
 * アカウント復旧の入力。
 */
export interface RestoreAccountInput {
  adminPrisma: AdminPrismaClient;
  productPrisma: ProductPrismaClient;
  operatorId: string;
  accountId: string;
  reason: string;
  ipAddress: string;
  auditLogger: AuditLogger;
}

function reportAuditReconciliationRequired(eventId: string, phase: string, message: string): void {
  // 構造化された警告ログとして eventId/phase/message を JSON で残し、復旧操作の未完了 outcome を追跡できるようにする。
  process.emitWarning(JSON.stringify({ eventId, phase, message }), 'AuditReconciliationWarning');
  // metrics collector が process event を購読できるよう、reconciliation 必要イベントを別 signal として発火する。
  process.emit('admin.audit_reconciliation_required', { eventId, phase });
}

/**
 * アカウントを復旧する。
 *
 * 1. 監査イベントを pending で作成（失敗時は 503）
 * 2. Product DB の `admin_op.restore_account()` を実行
 *    - 失敗時は監査イベントを failed に更新し、domain error を throw
 * 3. 成功時は監査イベントを succeeded に更新
 *    - 失敗時は pending のまま放置し、reconciliation で回収
 * 4. session_revoked_after は DB 関数内で保持される（過去のセッションは復活しない）
 *
 * @param input 復旧入力パラメータ
 */
export async function restoreAccount(input: RestoreAccountInput): Promise<void> {
  // 1. pending 監査イベントを作成
  let eventId: string;
  try {
    eventId = await input.auditLogger.createAuditIntent({
      operatorId: input.operatorId,
      action: 'account.restore',
      targetType: 'account',
      targetId: input.accountId,
      details: { reason: input.reason },
      ipAddress: input.ipAddress,
    });
  } catch {
    throw new ServiceError('Audit service unavailable', 503, 'AUDIT_UNAVAILABLE');
  }

  // 2. Product DB でアカウント復旧を実行
  try {
    await accountModel.restoreAccountProduct(
      input.productPrisma,
      input.accountId,
      input.operatorId,
      eventId
    );
  } catch (error) {
    const errorCode = error instanceof Error ? error.message : 'RESTORE_FAILED';
    try {
      await input.auditLogger.markAuditFailed(eventId, errorCode);
    } catch (markError) {
      const message = markError instanceof Error ? markError.message : String(markError);
      reportAuditReconciliationRequired(eventId, 'mark_failed', message);
    }
    throw new ServiceError('Account restoration failed', 500, 'RESTORE_FAILED');
  }

  // 3. 成功: 監査イベントを succeeded に更新
  try {
    await input.auditLogger.markAuditSucceeded(eventId);
  } catch (markError) {
    const message = markError instanceof Error ? markError.message : String(markError);
    reportAuditReconciliationRequired(eventId, 'mark_succeeded', message);
  }
}
