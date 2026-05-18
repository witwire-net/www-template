import type { AuditEvent } from './types.js';
import type { PrismaClient as AdminPrismaClient } from '.prisma/admin-client';

interface AuditFilters {
  operatorId?: string;
  action?: string;
  from?: Date;
  to?: Date;
  limit: number;
  offset: number;
}

/**
 * 監査イベントを登録する。
 *
 * @param adminPrisma Admin PrismaClient
 * @param data 監査データ
 * @returns 作成された監査イベント
 */
export async function insertAuditEvent(
  adminPrisma: AdminPrismaClient,
  data: {
    operatorId: string;
    action: string;
    targetType: string;
    targetId: string;
    details?: unknown;
    ipAddress?: string;
    outcome?: string;
  }
): Promise<AuditEvent> {
  const row = await adminPrisma.adminAuditEvent.create({
    data: {
      operator_id: data.operatorId,
      action: data.action,
      target_type: data.targetType,
      target_id: data.targetId,
      // generated Prisma value import を build bundle に含めないため、未指定時は列 default/null に委ねる。
      details: data.details ?? undefined,
      ip_address: data.ipAddress ?? null,
      outcome: data.outcome ?? 'pending',
    },
  });
  return toAuditEvent(row);
}

/**
 * 監査イベントをフィルター・ページネーション付きで一覧取得する。
 *
 * @param adminPrisma Admin PrismaClient
 * @param filters フィルター条件
 * @returns 監査イベント一覧と総数
 */
export async function listAuditEvents(
  adminPrisma: AdminPrismaClient,
  filters: AuditFilters
): Promise<{ items: AuditEvent[]; total: number }> {
  const where: Record<string, unknown> = {};
  if (filters.operatorId !== undefined && filters.operatorId !== '') {
    where.operator_id = filters.operatorId;
  }
  if (filters.action !== undefined && filters.action !== '') {
    where.action = filters.action;
  }
  if (filters.from !== undefined || filters.to !== undefined) {
    where.createdAt = {};
    if (filters.from !== undefined) {
      (where.createdAt as Record<string, unknown>).gte = filters.from;
    }
    if (filters.to !== undefined) {
      (where.createdAt as Record<string, unknown>).lte = filters.to;
    }
  }

  const [items, total] = await Promise.all([
    adminPrisma.adminAuditEvent.findMany({
      where,
      orderBy: { createdAt: 'desc' },
      take: filters.limit,
      skip: filters.offset,
    }),
    adminPrisma.adminAuditEvent.count({ where }),
  ]);

  return { items: items.map(toAuditEvent), total };
}

function toAuditEvent(row: {
  id: string;
  operator_id: string;
  action: string;
  target_type: string;
  target_id: string;
  details: unknown;
  outcome: string;
  error_code: string | null;
  ip_address: string | null;
  createdAt: Date;
  completed_at: Date | null;
}): AuditEvent {
  return {
    id: row.id,
    operatorId: row.operator_id,
    action: row.action,
    targetType: row.target_type,
    targetId: row.target_id,
    details: row.details,
    outcome: row.outcome as AuditEvent['outcome'],
    errorCode: row.error_code,
    ipAddress: row.ip_address,
    createdAt: row.createdAt,
    completedAt: row.completed_at,
  };
}
