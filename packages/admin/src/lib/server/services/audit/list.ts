import * as auditEventModel from '../../models/audit-events.js';

import type { AuditEvent } from '../../models/types.js';
import type { PrismaClient as AdminPrismaClient } from '.prisma/admin-client';

/**
 * 監査イベント一覧フィルター。
 */
export interface AuditListFilters {
  operatorId?: string;
  action?: string;
  dateFrom?: Date;
  dateTo?: Date;
  page: number;
  limit: number;
}

/**
 * 監査イベント一覧結果。
 */
export interface AuditListResult {
  events: AuditEvent[];
  total: number;
  page: number;
  totalPages: number;
}

/**
 * 監査イベントをフィルター・ページネーション付きで一覧取得する。
 * モデル層の `listAuditEvents` に委譲し、created_at desc でソートする。
 *
 * @param adminPrisma Admin PrismaClient
 * @param filters フィルター条件
 * @returns 監査イベント一覧とページネーション情報
 */
export async function listAuditEvents(
  adminPrisma: AdminPrismaClient,
  filters: AuditListFilters
): Promise<AuditListResult> {
  const offset = (filters.page - 1) * filters.limit;
  const result = await auditEventModel.listAuditEvents(adminPrisma, {
    operatorId: filters.operatorId,
    action: filters.action,
    from: filters.dateFrom,
    to: filters.dateTo,
    limit: filters.limit,
    offset,
  });

  const totalPages = result.total > 0 ? Math.ceil(result.total / filters.limit) : 0;

  return {
    events: result.items,
    total: result.total,
    page: filters.page,
    totalPages,
  };
}
