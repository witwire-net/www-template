import { afterEach, describe, expect, it, vi } from 'vitest';

import { ServiceError } from '../errors.js';

import { restoreAccount } from './restore.js';
import { suspendAccount } from './suspend.js';

import type { AuditLogger } from './suspend.js';

/**
 * サービス層テスト用の最小 Product DB 状態。
 * 実 DB ではなく admin_op 関数の主要な副作用だけを表現し、停止・復旧の状態遷移を決定的に検証する。
 */
interface ProductAccountState {
  status: 'active' | 'suspended';
  statusReason: string | null;
  statusUpdatedBy: string | null;
  sessionRevokedAfter: Date | null;
}

/**
 * 監査イベントの in-memory 表現。
 * pending 作成、成功/失敗 outcome 更新、completedAt 記録を同一配列上で検証できるようにする。
 */
interface AuditEventState {
  id: string;
  operatorId: string;
  action: string;
  targetType: string;
  targetId: string;
  details?: unknown;
  ipAddress?: string;
  outcome: 'pending' | 'succeeded' | 'failed';
  errorCode: string | null;
  completedAt: Date | null;
}

/**
 * Product Prisma の `$executeRaw` mock と状態オブジェクトを返す。
 * 入力の initialState を mutation で更新し、出力の productPrisma はサービスに渡せる unknown として扱う。
 * 副作用は渡された状態オブジェクト内に閉じるため、各テストで独立して検証できる。
 */
function createProductPrismaMock(initialState: ProductAccountState): {
  productPrisma: unknown;
  executeRaw: ReturnType<typeof vi.fn>;
  state: ProductAccountState;
} {
  const state = initialState;
  const executeRaw = vi.fn(
    (strings: TemplateStringsArray, _accountId: string, operatorId: string, third: string) => {
      // SQL tag の固定文字列から suspend / restore のどちらの DB 関数呼び出しかを判定する。
      const sqlText = Array.from(strings).join('');
      if (sqlText.includes('admin_op.suspend_account')) {
        // Product DB 関数と同じく、非 active の二重停止は状態を書き換える前に拒否する。
        if (state.status !== 'active') throw new Error('account_not_active');
        state.status = 'suspended';
        state.statusReason = third;
        state.statusUpdatedBy = operatorId;
        state.sessionRevokedAfter = new Date('2026-05-17T12:00:00.000Z');
        return Promise.resolve(1);
      }

      if (sqlText.includes('admin_op.restore_account')) {
        // Product DB 関数と同じく、active への復旧だけを許可し、session revoke 境界は維持する。
        if (state.status !== 'suspended') throw new Error('account_not_suspended');
        state.status = 'active';
        state.statusReason = null;
        state.statusUpdatedBy = operatorId;
        return Promise.resolve(1);
      }

      throw new Error(`Unexpected Product DB call: ${sqlText}`);
    }
  );

  return { productPrisma: { $executeRaw: executeRaw }, executeRaw, state };
}

/**
 * 監査ロガー mock とイベント配列を作成する。
 * 入力の failStep で特定段階だけ失敗させ、サービスが Product DB mutation 前後の境界を守るか検証する。
 */
function createAuditLoggerMock(failStep?: 'create' | 'succeeded' | 'failed'): {
  auditLogger: AuditLogger;
  events: AuditEventState[];
} {
  const events: AuditEventState[] = [];
  const auditLogger: AuditLogger = {
    createAuditIntent: vi.fn(async (input) => {
      if (failStep === 'create') throw new Error('audit unavailable');
      const event: AuditEventState = {
        id: `audit-${String(events.length + 1)}`,
        operatorId: input.operatorId,
        action: input.action,
        targetType: input.targetType,
        targetId: input.targetId,
        details: input.details,
        ipAddress: input.ipAddress,
        outcome: 'pending',
        errorCode: null,
        completedAt: null,
      };
      events.push(event);
      return event.id;
    }),
    markAuditSucceeded: vi.fn(async (eventId) => {
      if (failStep === 'succeeded') throw new Error('outcome write failed');
      const event = events.find((item) => item.id === eventId);
      if (event === undefined) throw new Error('missing audit event');
      event.outcome = 'succeeded';
      event.completedAt = new Date('2026-05-17T12:01:00.000Z');
    }),
    markAuditFailed: vi.fn(async (eventId, errorCode) => {
      if (failStep === 'failed') throw new Error('failed outcome write failed');
      const event = events.find((item) => item.id === eventId);
      if (event === undefined) throw new Error('missing audit event');
      event.outcome = 'failed';
      event.errorCode = errorCode;
      event.completedAt = new Date('2026-05-17T12:02:00.000Z');
    }),
  };

  return { auditLogger, events };
}

/**
 * サービス入力で使うアクティブな Product アカウント状態を生成する。
 * 各テストが状態を共有しないよう、新しい Date/null フィールドを毎回返す。
 */
function createActiveAccountState(): ProductAccountState {
  return {
    status: 'active',
    statusReason: null,
    statusUpdatedBy: null,
    sessionRevokedAfter: null,
  };
}

afterEach(() => {
  // process.emitWarning の spy をテストごとに戻し、後続テストへ警告捕捉の副作用を漏らさない。
  vi.restoreAllMocks();
});

describe('services/accounts suspend/restore', () => {
  it('14.1 suspend が pending 作成と succeeded 更新で audit 記録を残す', async () => {
    // 停止操作の監査証跡が顧客サポート・不正調査で追跡できるよう、入力値と outcome を検証する。
    const { productPrisma } = createProductPrismaMock(createActiveAccountState());
    const { auditLogger, events } = createAuditLoggerMock();

    await suspendAccount({
      adminPrisma: {} as Parameters<typeof suspendAccount>[0]['adminPrisma'],
      productPrisma: productPrisma as Parameters<typeof suspendAccount>[0]['productPrisma'],
      operatorId: 'operator-1',
      accountId: 'account-1',
      reason: 'policy violation',
      ipAddress: '203.0.113.10',
      auditLogger,
    });

    expect(events).toEqual([
      expect.objectContaining({
        action: 'account.suspend',
        details: { reason: 'policy violation' },
        outcome: 'succeeded',
        targetId: 'account-1',
      }),
    ]);
  });

  it('14.2 restore が pending 作成と succeeded 更新で audit 記録を残す', async () => {
    // 復旧操作も停止と同じ監査境界を持つため、action と reason が失われないことを確認する。
    const suspendedState = createActiveAccountState();
    suspendedState.status = 'suspended';
    suspendedState.sessionRevokedAfter = new Date('2026-05-17T11:00:00.000Z');
    const { productPrisma } = createProductPrismaMock(suspendedState);
    const { auditLogger, events } = createAuditLoggerMock();

    await restoreAccount({
      adminPrisma: {} as Parameters<typeof restoreAccount>[0]['adminPrisma'],
      productPrisma: productPrisma as Parameters<typeof restoreAccount>[0]['productPrisma'],
      operatorId: 'operator-1',
      accountId: 'account-1',
      reason: 'appeal accepted',
      ipAddress: '203.0.113.10',
      auditLogger,
    });

    expect(events).toEqual([
      expect.objectContaining({
        action: 'account.restore',
        details: { reason: 'appeal accepted' },
        outcome: 'succeeded',
        targetId: 'account-1',
      }),
    ]);
  });

  it('14.3 二重 suspend はエラーになり Product 状態を変更しない', async () => {
    // 二重停止で理由や session revoke 境界を上書きしないことを、Product DB mock の状態差分で確認する。
    const alreadySuspendedState: ProductAccountState = {
      status: 'suspended',
      statusReason: 'original reason',
      statusUpdatedBy: 'operator-old',
      sessionRevokedAfter: new Date('2026-05-17T10:00:00.000Z'),
    };
    const { productPrisma, state } = createProductPrismaMock(alreadySuspendedState);
    const { auditLogger, events } = createAuditLoggerMock();

    await expect(
      suspendAccount({
        adminPrisma: {} as Parameters<typeof suspendAccount>[0]['adminPrisma'],
        productPrisma: productPrisma as Parameters<typeof suspendAccount>[0]['productPrisma'],
        operatorId: 'operator-2',
        accountId: 'account-1',
        reason: 'second reason',
        ipAddress: '203.0.113.20',
        auditLogger,
      })
    ).rejects.toMatchObject({ code: 'SUSPEND_FAILED', statusCode: 500 });

    expect(state).toEqual({
      status: 'suspended',
      statusReason: 'original reason',
      statusUpdatedBy: 'operator-old',
      sessionRevokedAfter: new Date('2026-05-17T10:00:00.000Z'),
    });
    expect(events[0]).toEqual(
      expect.objectContaining({ errorCode: 'account_not_active', outcome: 'failed' })
    );
  });

  it('14.4 suspend→restore の正常サイクルで active に戻り session_revoked_after を維持する', async () => {
    // 復旧後に古いセッションが復活しないよう、restore が session_revoked_after を null に戻さないことを確認する。
    const { productPrisma, state } = createProductPrismaMock(createActiveAccountState());
    const { auditLogger } = createAuditLoggerMock();

    await suspendAccount({
      adminPrisma: {} as Parameters<typeof suspendAccount>[0]['adminPrisma'],
      productPrisma: productPrisma as Parameters<typeof suspendAccount>[0]['productPrisma'],
      operatorId: 'operator-1',
      accountId: 'account-1',
      reason: 'risk',
      ipAddress: '203.0.113.10',
      auditLogger,
    });
    const revokedAfter = state.sessionRevokedAfter;

    await restoreAccount({
      adminPrisma: {} as Parameters<typeof restoreAccount>[0]['adminPrisma'],
      productPrisma: productPrisma as Parameters<typeof restoreAccount>[0]['productPrisma'],
      operatorId: 'operator-1',
      accountId: 'account-1',
      reason: 'resolved',
      ipAddress: '203.0.113.10',
      auditLogger,
    });

    expect(state).toEqual({
      status: 'active',
      statusReason: null,
      statusUpdatedBy: 'operator-1',
      sessionRevokedAfter: revokedAfter,
    });
  });

  it('14.6 pending audit intent 作成失敗時は Product DB mutation を開始せず 503 を返す', async () => {
    // 監査不能な状態で Product DB を変更しない fail-close 境界を、executeRaw 未呼び出しで検証する。
    const { productPrisma, executeRaw } = createProductPrismaMock(createActiveAccountState());
    const { auditLogger } = createAuditLoggerMock('create');

    await expect(
      suspendAccount({
        adminPrisma: {} as Parameters<typeof suspendAccount>[0]['adminPrisma'],
        productPrisma: productPrisma as Parameters<typeof suspendAccount>[0]['productPrisma'],
        operatorId: 'operator-1',
        accountId: 'account-1',
        reason: 'risk',
        ipAddress: '203.0.113.10',
        auditLogger,
      })
    ).rejects.toMatchObject({ code: 'AUDIT_UNAVAILABLE', statusCode: 503 });

    expect(executeRaw).not.toHaveBeenCalled();
  });

  it('14.7 suspend は Product DB mutation で session_revoked_after を書く', async () => {
    // 認証セッション失効境界を停止操作と同一 mutation で進めることを、状態更新と DB 呼び出しで検証する。
    const { productPrisma, executeRaw, state } = createProductPrismaMock(
      createActiveAccountState()
    );
    const { auditLogger } = createAuditLoggerMock();

    await suspendAccount({
      adminPrisma: {} as Parameters<typeof suspendAccount>[0]['adminPrisma'],
      productPrisma: productPrisma as Parameters<typeof suspendAccount>[0]['productPrisma'],
      operatorId: 'operator-1',
      accountId: 'account-1',
      reason: 'risk',
      ipAddress: '203.0.113.10',
      auditLogger,
    });

    expect(executeRaw).toHaveBeenCalledTimes(1);
    expect(state.sessionRevokedAfter).toEqual(new Date('2026-05-17T12:00:00.000Z'));
  });

  it('14.7a outcome 更新失敗時は pending audit event を残して warning を出す', async () => {
    // Product DB mutation 成功後の outcome 書き込みだけが失敗した場合、reconciliation 対象を残す必要がある。
    const warningSpy = vi.spyOn(process, 'emitWarning').mockImplementation(() => undefined);
    const metricSpy = vi.spyOn(process, 'emit').mockImplementation(() => true);
    const { productPrisma } = createProductPrismaMock(createActiveAccountState());
    const { auditLogger, events } = createAuditLoggerMock('succeeded');

    await suspendAccount({
      adminPrisma: {} as Parameters<typeof suspendAccount>[0]['adminPrisma'],
      productPrisma: productPrisma as Parameters<typeof suspendAccount>[0]['productPrisma'],
      operatorId: 'operator-1',
      accountId: 'account-1',
      reason: 'risk',
      ipAddress: '203.0.113.10',
      auditLogger,
    });

    expect(events[0]).toEqual(expect.objectContaining({ completedAt: null, outcome: 'pending' }));
    expect(warningSpy).toHaveBeenCalledWith(
      expect.stringContaining('mark_succeeded'),
      'AuditReconciliationWarning'
    );
    expect(metricSpy).toHaveBeenCalledWith('admin.audit_reconciliation_required', {
      eventId: 'audit-1',
      phase: 'mark_succeeded',
    });
  });

  it('14.7b Product DB mutation 失敗時は failed outcome/error_code/completed_at を記録する', async () => {
    // mutation 失敗を監査に確定記録し、障害分析で stable error_code を参照できることを確認する。
    const alreadySuspendedState: ProductAccountState = {
      status: 'suspended',
      statusReason: 'original reason',
      statusUpdatedBy: 'operator-old',
      sessionRevokedAfter: new Date('2026-05-17T10:00:00.000Z'),
    };
    const { productPrisma } = createProductPrismaMock(alreadySuspendedState);
    const { auditLogger, events } = createAuditLoggerMock();

    await expect(
      suspendAccount({
        adminPrisma: {} as Parameters<typeof suspendAccount>[0]['adminPrisma'],
        productPrisma: productPrisma as Parameters<typeof suspendAccount>[0]['productPrisma'],
        operatorId: 'operator-1',
        accountId: 'account-1',
        reason: 'risk',
        ipAddress: '203.0.113.10',
        auditLogger,
      })
    ).rejects.toBeInstanceOf(ServiceError);

    expect(events[0]).toEqual(
      expect.objectContaining({
        completedAt: new Date('2026-05-17T12:02:00.000Z'),
        errorCode: 'account_not_active',
        outcome: 'failed',
      })
    );
  });

  it('14.7b failed outcome 更新も失敗した場合は pending のまま warning を出す', async () => {
    // mutation と failed outcome 更新の両方が失敗しても、pending event を消さず reconciliation に残す。
    const warningSpy = vi.spyOn(process, 'emitWarning').mockImplementation(() => undefined);
    const metricSpy = vi.spyOn(process, 'emit').mockImplementation(() => true);
    const alreadySuspendedState: ProductAccountState = {
      status: 'suspended',
      statusReason: 'original reason',
      statusUpdatedBy: 'operator-old',
      sessionRevokedAfter: new Date('2026-05-17T10:00:00.000Z'),
    };
    const { productPrisma } = createProductPrismaMock(alreadySuspendedState);
    const { auditLogger, events } = createAuditLoggerMock('failed');

    await expect(
      suspendAccount({
        adminPrisma: {} as Parameters<typeof suspendAccount>[0]['adminPrisma'],
        productPrisma: productPrisma as Parameters<typeof suspendAccount>[0]['productPrisma'],
        operatorId: 'operator-1',
        accountId: 'account-1',
        reason: 'risk',
        ipAddress: '203.0.113.10',
        auditLogger,
      })
    ).rejects.toMatchObject({ code: 'SUSPEND_FAILED' });

    expect(events[0]).toEqual(
      expect.objectContaining({ completedAt: null, errorCode: null, outcome: 'pending' })
    );
    expect(warningSpy).toHaveBeenCalledWith(
      expect.stringContaining('mark_failed'),
      'AuditReconciliationWarning'
    );
    expect(metricSpy).toHaveBeenCalledWith('admin.audit_reconciliation_required', {
      eventId: 'audit-1',
      phase: 'mark_failed',
    });
  });
});
