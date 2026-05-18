import { randomBytes } from 'node:crypto';
import { createRequire } from 'node:module';

import * as auditEventModel from '../../models/audit-events.js';
import * as operatorModel from '../../models/operators.js';
import * as passkeyModel from '../../models/passkeys.js';
import { ServiceError } from '../errors.js';

import type { Operator } from '../../models/types.js';
import type { PrismaClient as AdminPrismaClient } from '.prisma/admin-client';

/**
 * セットアップトークンの有効期限（ミリ秒）。
 */
const SETUP_TOKEN_TTL_MS = 24 * 60 * 60 * 1000; // 24時間
const nodeRequire = createRequire(import.meta.url);
const { hashSync } = nodeRequire('bcryptjs') as {
  hashSync: (value: string, salt: number) => string;
};

interface OperatorAuditIntentInput {
  operatorId: string;
  action: string;
  targetType: string;
  targetId: string;
  details?: unknown;
}

/**
 * 暗号学的に安全なランダムセットアップトークンを生成する。
 *
 * @returns 平文トークン（hex 64文字）
 */
function generateSetupToken(): string {
  return randomBytes(32).toString('hex');
}

async function createOperatorAuditIntent(
  adminPrisma: AdminPrismaClient,
  input: OperatorAuditIntentInput
): Promise<string> {
  try {
    // mutation 開始前に pending intent を永続化し、未監査の権限変更・オペレーター変更を防ぐ。
    const event = await auditEventModel.insertAuditEvent(adminPrisma, {
      operatorId: input.operatorId,
      action: input.action,
      targetType: input.targetType,
      targetId: input.targetId,
      details: input.details,
      outcome: 'pending',
    });
    return event.id;
  } catch {
    // 監査 intent が作れない場合は Admin DB mutation を開始せず、呼び出し側で 503 に変換できる形に固定する。
    throw new ServiceError('Audit service unavailable', 503, 'AUDIT_UNAVAILABLE');
  }
}

async function markOperatorAuditSucceeded(
  adminPrisma: AdminPrismaClient,
  eventId: string
): Promise<void> {
  try {
    // 成功 outcome と completed_at を同時に記録し、pending intent を完了済み監査証跡へ遷移させる。
    await adminPrisma.adminAuditEvent.update({
      where: { id: eventId },
      data: { outcome: 'succeeded', error_code: null, completed_at: new Date() },
    });
  } catch (markError) {
    // outcome 更新失敗時は mutation を戻さず、reconciliation で回収できるよう warning と metric signal を残す。
    const message = markError instanceof Error ? markError.message : String(markError);
    reportOperatorAuditReconciliationRequired(eventId, 'mark_succeeded', message);
  }
}

async function markOperatorAuditFailed(
  adminPrisma: AdminPrismaClient,
  eventId: string,
  errorCode: string
): Promise<void> {
  try {
    // mutation 失敗を failed outcome と stable error code に変換し、調査時に pending と区別できるようにする。
    await adminPrisma.adminAuditEvent.update({
      where: { id: eventId },
      data: { outcome: 'failed', error_code: errorCode, completed_at: new Date() },
    });
  } catch (markError) {
    // failed outcome 更新すら失敗した場合も pending intent は残るため、reconciliation 対象として警告と metric signal を出す。
    const message = markError instanceof Error ? markError.message : String(markError);
    reportOperatorAuditReconciliationRequired(eventId, 'mark_failed', message);
  }
}

function reportOperatorAuditReconciliationRequired(
  eventId: string,
  phase: string,
  message: string
): void {
  // 構造化された警告ログとして eventId/phase/message を JSON 化し、運用者が対象 audit event を直接追跡できるようにする。
  process.emitWarning(JSON.stringify({ eventId, phase, message }), 'AuditReconciliationWarning');
  // metrics collector が process event を購読できるよう、operator audit の reconciliation 必要 signal を発火する。
  process.emit('admin.audit_reconciliation_required', { eventId, phase });
}

function hasStableCode(error: unknown): error is { code: string } {
  // ServiceError や Prisma known request error の code だけを stable code として採用するため、unknown から安全に絞り込む。
  return (
    typeof error === 'object' &&
    error !== null &&
    'code' in error &&
    typeof (error as { code?: unknown }).code === 'string'
  );
}

function toErrorCode(error: unknown, fallback: string): string {
  // 人間向け message は監査の stable code に使わず、ServiceError/Prisma の code か固定 fallback へ正規化する。
  if (hasStableCode(error)) return error.code;
  return fallback;
}

/**
 * 新規オペレーターを作成する。
 * セットアップトークンを生成し bcrypt ハッシュして保存する。
 * 平文トークンは一度きりの返却のみ可能。
 *
 * @param adminPrisma Admin PrismaClient
 * @param input 作成データ
 * @param actingOperatorId 操作を実行したオペレーター ID（監査ログ用）
 * @returns 作成されたオペレーターと平文セットアップトークン
 */
export async function createOperator(
  adminPrisma: AdminPrismaClient,
  input: { email: string; displayName: string; role: string },
  actingOperatorId: string
): Promise<{ operator: Operator; plaintextToken: string }> {
  const eventId = await createOperatorAuditIntent(adminPrisma, {
    operatorId: actingOperatorId,
    action: 'operator.create',
    targetType: 'operator',
    // 作成前は Operator ID が未採番のため、未監査 mutation 防止を優先して email を暫定 target として記録する。
    targetId: input.email,
    details: { email: input.email, role: input.role },
  });

  const plaintextToken = generateSetupToken();
  const hash = hashSync(plaintextToken, 10);
  const expiresAt = new Date(Date.now() + SETUP_TOKEN_TTL_MS);

  try {
    const operator = await operatorModel.createOperator(adminPrisma, {
      email: input.email,
      displayName: input.displayName,
      role: input.role,
    });

    await operatorModel.updateOperatorSetupToken(adminPrisma, operator.id, hash, expiresAt);
    await markOperatorAuditSucceeded(adminPrisma, eventId);

    return { operator, plaintextToken };
  } catch (error) {
    await markOperatorAuditFailed(
      adminPrisma,
      eventId,
      toErrorCode(error, 'OPERATOR_CREATE_FAILED')
    );
    throw error;
  }
}

/**
 * オペレーターのセットアップトークンを再発行する。
 * passkey が既に登録されているオペレーターには再発行不可。
 *
 * @param adminPrisma Admin PrismaClient
 * @param operatorId 対象オペレーター ID
 * @param actingOperatorId 操作を実行したオペレーター ID（監査ログ用）
 * @returns 新しい平文セットアップトークン
 */
export async function rotateSetupToken(
  adminPrisma: AdminPrismaClient,
  operatorId: string,
  actingOperatorId: string
): Promise<{ operator: Operator; plaintextToken: string }> {
  const passkeyCount = await passkeyModel.getPasskeyCount(adminPrisma, operatorId);
  if (passkeyCount > 0) {
    throw new ServiceError(
      'Cannot rotate token for operator with registered passkeys',
      400,
      'PASSKEY_EXISTS'
    );
  }

  const eventId = await createOperatorAuditIntent(adminPrisma, {
    operatorId: actingOperatorId,
    action: 'operator.setup_token.rotate',
    targetType: 'operator',
    targetId: operatorId,
  });

  const plaintextToken = generateSetupToken();
  const hash = hashSync(plaintextToken, 10);
  const expiresAt = new Date(Date.now() + SETUP_TOKEN_TTL_MS);

  try {
    await operatorModel.updateOperatorSetupToken(adminPrisma, operatorId, hash, expiresAt);

    const operator = await operatorModel.findOperatorById(adminPrisma, operatorId);
    if (operator === null) {
      throw new ServiceError('Operator not found', 404, 'OPERATOR_NOT_FOUND');
    }

    await markOperatorAuditSucceeded(adminPrisma, eventId);
    return { operator, plaintextToken };
  } catch (error) {
    await markOperatorAuditFailed(
      adminPrisma,
      eventId,
      toErrorCode(error, 'SETUP_TOKEN_ROTATE_FAILED')
    );
    throw error;
  }
}

/**
 * オペレーターのロールを変更する。
 * 最後のアクティブ admin が降格される操作は拒否する。
 *
 * @param adminPrisma Admin PrismaClient
 * @param operatorId 対象オペレーター ID
 * @param newRole 新しいロール
 * @param actingOperatorId 操作を実行したオペレーター ID（監査ログ用）
 * @returns 更新されたオペレーター
 */
export async function updateOperatorRole(
  adminPrisma: AdminPrismaClient,
  operatorId: string,
  newRole: string,
  actingOperatorId: string
): Promise<Operator> {
  const target = await operatorModel.findOperatorById(adminPrisma, operatorId);
  if (target === null) {
    throw new ServiceError('Operator not found', 404, 'OPERATOR_NOT_FOUND');
  }

  // 最後の admin 降格を拒否
  if (target.role === 'admin' && newRole !== 'admin') {
    const activeAdminCount = await operatorModel.countActiveAdmins(adminPrisma);
    if (activeAdminCount <= 1 && target.isActive) {
      throw new ServiceError('Cannot demote the last active admin', 400, 'LAST_ADMIN_DEMOTION');
    }
  }

  const eventId = await createOperatorAuditIntent(adminPrisma, {
    operatorId: actingOperatorId,
    action: 'operator.update_role',
    targetType: 'operator',
    targetId: operatorId,
    details: { from_role: target.role, to_role: newRole },
  });

  try {
    const operator = await operatorModel.updateOperatorRole(adminPrisma, operatorId, newRole);
    await markOperatorAuditSucceeded(adminPrisma, eventId);
    return operator;
  } catch (error) {
    await markOperatorAuditFailed(
      adminPrisma,
      eventId,
      toErrorCode(error, 'OPERATOR_ROLE_UPDATE_FAILED')
    );
    throw error;
  }
}

/**
 * オペレーターを無効化する。
 * 自己無効化と、最後の admin 無効化を拒否する。
 *
 * @param adminPrisma Admin PrismaClient
 * @param operatorId 対象オペレーター ID
 * @param actingOperatorId 操作を実行したオペレーター ID（監査ログ用）
 * @returns 更新されたオペレーター
 */
export async function deactivateOperator(
  adminPrisma: AdminPrismaClient,
  operatorId: string,
  actingOperatorId: string
): Promise<Operator> {
  if (operatorId === actingOperatorId) {
    throw new ServiceError('Self-deactivation is not allowed', 409, 'SELF_DEACTIVATION');
  }

  const target = await operatorModel.findOperatorById(adminPrisma, operatorId);
  if (target === null) {
    throw new ServiceError('Operator not found', 404, 'OPERATOR_NOT_FOUND');
  }

  // 最後の admin 無効化を拒否
  if (target.role === 'admin' && target.isActive) {
    const activeAdminCount = await operatorModel.countActiveAdmins(adminPrisma);
    if (activeAdminCount <= 1) {
      throw new ServiceError(
        'Cannot deactivate the last active admin',
        400,
        'LAST_ADMIN_DEACTIVATION'
      );
    }
  }

  const eventId = await createOperatorAuditIntent(adminPrisma, {
    operatorId: actingOperatorId,
    action: 'operator.deactivate',
    targetType: 'operator',
    targetId: operatorId,
  });

  try {
    const operator = await operatorModel.deactivateOperator(adminPrisma, operatorId);
    await markOperatorAuditSucceeded(adminPrisma, eventId);
    return operator;
  } catch (error) {
    await markOperatorAuditFailed(
      adminPrisma,
      eventId,
      toErrorCode(error, 'OPERATOR_DEACTIVATE_FAILED')
    );
    throw error;
  }
}
