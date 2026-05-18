import * as accountModel from '../../models/accounts.js';
import { ServiceError } from '../errors.js';

import type { PrismaClient as AdminPrismaClient } from '.prisma/admin-client';
import type { PrismaClient as ProductPrismaClient } from '.prisma/product-client';

/**
 * 監査ロガーのインターフェース。
 * インフラ実装をサービス層から分離し、テスト容易性を高めるため抽象化する。
 */
export interface AuditLogger {
  /**
   * pending 状態の監査イベントを作成し、eventId を返す。
   */
  createAuditIntent(input: {
    operatorId: string;
    action: string;
    targetType: string;
    targetId: string;
    details?: unknown;
    ipAddress?: string;
  }): Promise<string>;
  /**
   * 監査イベントを succeeded に更新する。
   */
  markAuditSucceeded(eventId: string): Promise<void>;
  /**
   * 監査イベントを failed に更新する。
   */
  markAuditFailed(eventId: string, errorCode: string): Promise<void>;
}

/**
 * アカウント停止の入力。
 */
export interface SuspendAccountInput {
  adminPrisma: AdminPrismaClient;
  productPrisma: ProductPrismaClient;
  operatorId: string;
  accountId: string;
  reason: string;
  ipAddress: string;
  auditLogger: AuditLogger;
}

function reportAuditReconciliationRequired(eventId: string, phase: string, message: string): void {
  // 構造化された警告ログとして eventId/phase/message を JSON で残し、運用時に対象 audit event を検索できるようにする。
  process.emitWarning(JSON.stringify({ eventId, phase, message }), 'AuditReconciliationWarning');
  // metrics collector が process event を購読できるよう、reconciliation 必要イベントを別 signal として発火する。
  process.emit('admin.audit_reconciliation_required', { eventId, phase });
}

/**
 * アカウントを停止する。
 *
 * 1. 監査イベントを pending で作成（失敗時は 503）
 * 2. Product DB の `admin_op.suspend_account()` を実行
 *    - 失敗時は監査イベントを failed に更新し、domain error を throw
 * 3. 成功時は監査イベントを succeeded に更新
 *    - 失敗時は pending のまま放置し、後続の reconciliation で回収
 * 4. OpenSearch インデックスは auditLogger の実装内で非同期に行われる（warn on fail, never throw）
 *
 * @param input 停止入力パラメータ
 */
export async function suspendAccount(input: SuspendAccountInput): Promise<void> {
  // 1. pending 監査イベントを作成
  let eventId: string;
  try {
    eventId = await input.auditLogger.createAuditIntent({
      operatorId: input.operatorId,
      action: 'account.suspend',
      targetType: 'account',
      targetId: input.accountId,
      details: { reason: input.reason },
      ipAddress: input.ipAddress,
    });
  } catch {
    throw new ServiceError('Audit service unavailable', 503, 'AUDIT_UNAVAILABLE');
  }

  // 2. Product DB でアカウント停止を実行
  try {
    await accountModel.suspendAccountProduct(
      input.productPrisma,
      input.accountId,
      input.operatorId,
      input.reason,
      eventId
    );
  } catch (error) {
    const errorCode = error instanceof Error ? error.message : 'SUSPEND_FAILED';
    try {
      await input.auditLogger.markAuditFailed(eventId, errorCode);
    } catch (markError) {
      // outcome update に失敗した場合は pending のままにする
      const message = markError instanceof Error ? markError.message : String(markError);
      reportAuditReconciliationRequired(eventId, 'mark_failed', message);
    }
    throw new ServiceError('Account suspension failed', 500, 'SUSPEND_FAILED');
  }

  // 3. 成功: 監査イベントを succeeded に更新
  try {
    await input.auditLogger.markAuditSucceeded(eventId);
  } catch (markError) {
    // outcome update に失敗した場合は pending のままにし、reconciliation で回収
    const message = markError instanceof Error ? markError.message : String(markError);
    reportAuditReconciliationRequired(eventId, 'mark_succeeded', message);
  }
}
