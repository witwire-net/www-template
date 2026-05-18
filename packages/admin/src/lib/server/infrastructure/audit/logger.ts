import { indexAuditEvent } from '../search/opensearch.js';

import type { PrismaClient as AdminPrismaClient } from '.prisma/admin-client';
import type { Client as OpenSearchClient } from '@opensearch-project/opensearch';

interface AuditIntentInput {
  operatorId: string;
  action: string;
  targetType: string;
  targetId: string;
  details?: unknown;
  ipAddress?: string;
}

/**
 * 監査イベントを pending 状態で DB に登録し、eventId を返す。
 *
 * @param adminPrisma Admin PrismaClient
 * @param input 監査イベント入力
 * @returns 作成された監査イベントの ID
 */
export async function createAuditIntent(
  adminPrisma: AdminPrismaClient,
  input: AuditIntentInput
): Promise<string> {
  const event = await adminPrisma.adminAuditEvent.create({
    data: {
      operator_id: input.operatorId,
      action: input.action,
      target_type: input.targetType,
      target_id: input.targetId,
      // Prisma namespace の runtime import を避け、未指定時は DB 側の null/default に委ねる。
      details: input.details ?? undefined,
      ip_address: input.ipAddress ?? null,
      outcome: 'pending',
    },
  });
  return event.id;
}

/**
 * 監査イベントを succeeded に更新し、非同期で OpenSearch にインデックスする。
 *
 * @param adminPrisma Admin PrismaClient
 * @param eventId 監査イベント ID
 * @param opensearch オプションの OpenSearch クライアント
 */
export async function markAuditSucceeded(
  adminPrisma: AdminPrismaClient,
  eventId: string,
  opensearch?: OpenSearchClient
): Promise<void> {
  await adminPrisma.adminAuditEvent.update({
    where: { id: eventId },
    data: { outcome: 'succeeded', completed_at: new Date() },
  });

  if (opensearch !== undefined) {
    const event = await adminPrisma.adminAuditEvent.findUnique({
      where: { id: eventId },
      include: { operator: true },
    });
    if (event !== null) {
      indexAuditEvent(
        {
          id: event.id,
          operator_id: event.operator_id,
          operator_email: event.operator.email,
          operator_name: event.operator.display_name,
          action: event.action,
          target_type: event.target_type,
          target_id: event.target_id,
          details: event.details,
          details_json: event.details !== null ? JSON.stringify(event.details) : undefined,
          ip_address: event.ip_address ?? undefined,
          created_at: event.createdAt.toISOString(),
        },
        opensearch
      ).catch((error: unknown) => {
        const message = error instanceof Error ? error.message : String(error);
        process.emitWarning(
          `OpenSearch audit indexing failed for event ${eventId}: ${message}`,
          'AuditOpenSearchWarning'
        );
      });
    }
  }
}

/**
 * 監査イベントを failed に更新する。
 *
 * @param adminPrisma Admin PrismaClient
 * @param eventId 監査イベント ID
 * @param errorCode エラーコード
 */
export async function markAuditFailed(
  adminPrisma: AdminPrismaClient,
  eventId: string,
  errorCode: string
): Promise<void> {
  await adminPrisma.adminAuditEvent.update({
    where: { id: eventId },
    data: { outcome: 'failed', error_code: errorCode, completed_at: new Date() },
  });
}
