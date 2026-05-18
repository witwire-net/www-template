import { describe, expect, it, vi } from 'vitest';

import { listAuditEvents } from './audit-events.js';

/**
 * 監査イベントの Prisma 行を生成する。
 * 入力は差分だけを上書きでき、出力は model mapper が要求する列名に揃える。
 * 実 DB には接続せず、DB fallback 検索の mapping と pagination 呼び出しを決定的に検証する。
 */
function createAuditEventRow(overrides: Partial<AuditEventRow> = {}): AuditEventRow {
  const now = new Date('2026-05-17T10:00:00.000Z');
  return {
    id: 'audit-1',
    operator_id: 'operator-1',
    action: 'accounts.suspend',
    target_type: 'account',
    target_id: 'account-1',
    details: null,
    outcome: 'succeeded',
    error_code: null,
    ip_address: null,
    createdAt: now,
    completed_at: now,
    ...overrides,
  };
}

interface AuditEventRow {
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
}

describe('models/audit-events', () => {
  it('17.15 DB fallback search は Admin DB の監査ログから結果を返す', async () => {
    // OpenSearch が使えない場合でも一覧画面が DB を正として監査ログを検索できることを確認する。
    const findMany = vi.fn().mockResolvedValue([createAuditEventRow()]);
    const count = vi.fn().mockResolvedValue(1);
    const adminPrisma = { adminAuditEvent: { findMany, count } };

    const result = await listAuditEvents(
      adminPrisma as unknown as Parameters<typeof listAuditEvents>[0],
      {
        operatorId: 'operator-1',
        action: 'accounts.suspend',
        limit: 20,
        offset: 0,
      }
    );

    expect(findMany).toHaveBeenCalledWith({
      where: { operator_id: 'operator-1', action: 'accounts.suspend' },
      orderBy: { createdAt: 'desc' },
      take: 20,
      skip: 0,
    });
    expect(count).toHaveBeenCalledWith({
      where: { operator_id: 'operator-1', action: 'accounts.suspend' },
    });
    expect(result).toEqual({
      items: [
        expect.objectContaining({
          id: 'audit-1',
          operatorId: 'operator-1',
          action: 'accounts.suspend',
          outcome: 'succeeded',
        }),
      ],
      total: 1,
    });
  });
});
