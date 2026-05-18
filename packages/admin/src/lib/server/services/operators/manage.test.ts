import { createRequire as actualCreateRequire } from 'node:module';

import { afterEach, describe, expect, it, vi } from 'vitest';

/**
 * passkeys model が generated Prisma client を runtime require するため、テストでは Prisma.DbNull だけを差し替える。
 * bcryptjs は実装と同じ実ライブラリを使い、setup token hash の性質を実際の bcrypt 形式で検証する。
 */
vi.mock(
  'node:module',
  async (importOriginal: () => Promise<{ createRequire: typeof actualCreateRequire }>) => {
    const actual = await importOriginal();
    return {
      createRequire: (url: string) => {
        const requireFromModule = actual.createRequire(url);
        return (specifier: string) => {
          if (specifier === '.prisma/admin-client')
            return { Prisma: { DbNull: Symbol.for('Prisma.DbNull') } };
          return requireFromModule(specifier);
        };
      },
    };
  }
);

import { ServiceError } from '../errors.js';

import {
  createOperator,
  deactivateOperator,
  rotateSetupToken,
  updateOperatorRole,
} from './manage.js';

import type { Operator } from '../../models/types.js';

const requireFromTest = actualCreateRequire(import.meta.url);
const { compareSync } = requireFromTest('bcryptjs') as {
  compareSync: (value: string, hash: string) => boolean;
};

/**
 * Prisma 行に相当する Operator を決定的な値で生成する。
 * 入力 override だけを差し替え、サービス戻り値と model mapper の両方で同じ形を使う。
 */
function createOperatorDomain(overrides: Partial<Operator> = {}): Operator {
  const now = new Date('2026-05-17T00:00:00.000Z');
  return {
    id: 'operator-1',
    email: 'operator@example.com',
    displayName: 'Operator One',
    role: 'operator',
    isActive: true,
    setupTokenHash: null,
    setupTokenExpiresAt: null,
    lastLoginAt: null,
    createdAt: now,
    updatedAt: now,
    ...overrides,
  };
}

/**
 * models/operators の Prisma row 形式へ変換する。
 * サービスは model 経由で row を Operator に戻すため、snake_case 列名をここで固定する。
 */
function toOperatorRow(operator: Operator): OperatorRow {
  return {
    id: operator.id,
    email: operator.email,
    display_name: operator.displayName,
    role: operator.role,
    is_active: operator.isActive,
    setup_token_hash: operator.setupTokenHash,
    setup_token_expires_at: operator.setupTokenExpiresAt,
    last_login_at: operator.lastLoginAt,
    createdAt: operator.createdAt,
    updatedAt: operator.updatedAt,
  };
}

/**
 * 監査イベント row を返す。
 * auditEvent model の戻り値 mapper が要求する列を満たし、監査作成呼び出しを実 DB なしで完了させる。
 */
function createAuditEventRow(overrides: Partial<AuditEventRow> = {}): AuditEventRow {
  return {
    id: 'audit-1',
    operator_id: 'admin-1',
    action: 'operator.role_changed',
    target_type: 'operator',
    target_id: 'operator-1',
    details: null,
    outcome: 'succeeded',
    error_code: null,
    ip_address: null,
    createdAt: new Date('2026-05-17T00:00:00.000Z'),
    completed_at: new Date('2026-05-17T00:01:00.000Z'),
    ...overrides,
  };
}

/**
 * operators service が必要とする Admin Prisma mock を生成する。
 * 入力 delegate の各関数を実装し、出力 adminPrisma はサービスに渡せる unknown として扱う。
 */
function createAdminPrismaMock(delegate: Partial<AdminDelegate>): unknown {
  return {
    adminAuditEvent: {
      create: delegate.createAuditEvent ?? vi.fn().mockResolvedValue(createAuditEventRow()),
      update: delegate.updateAuditEvent ?? vi.fn().mockResolvedValue(createAuditEventRow()),
    },
    adminOperator: {
      count: delegate.countOperators ?? vi.fn().mockResolvedValue(2),
      create:
        delegate.createOperator ?? vi.fn().mockResolvedValue(toOperatorRow(createOperatorDomain())),
      findUnique:
        delegate.findOperator ?? vi.fn().mockResolvedValue(toOperatorRow(createOperatorDomain())),
      update:
        delegate.updateOperator ?? vi.fn().mockResolvedValue(toOperatorRow(createOperatorDomain())),
    },
    adminOperatorPasskey: { count: delegate.countPasskeys ?? vi.fn().mockResolvedValue(0) },
  };
}

interface OperatorRow {
  id: string;
  email: string;
  display_name: string;
  role: string;
  is_active: boolean;
  setup_token_hash: string | null;
  setup_token_expires_at: Date | null;
  last_login_at: Date | null;
  createdAt: Date;
  updatedAt: Date;
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

interface AdminDelegate {
  countOperators: ReturnType<typeof vi.fn>;
  countPasskeys: ReturnType<typeof vi.fn>;
  createAuditEvent: ReturnType<typeof vi.fn>;
  createOperator: ReturnType<typeof vi.fn>;
  findOperator: ReturnType<typeof vi.fn>;
  updateAuditEvent: ReturnType<typeof vi.fn>;
  updateOperator: ReturnType<typeof vi.fn>;
}

afterEach(() => {
  // process spy と mock 呼び出し履歴をテストごとに戻し、監査や更新の回数検証が独立するようにする。
  vi.restoreAllMocks();
});

describe('services/operators/manage', () => {
  it('14.5 operator role 更新が audit 記録を残す', async () => {
    // 権限変更はセキュリティ上重要な操作のため、旧 role / 新 role を監査 details に残すことを確認する。
    const findOperator = vi
      .fn()
      .mockResolvedValue(toOperatorRow(createOperatorDomain({ role: 'viewer' })));
    const updateOperator = vi
      .fn()
      .mockResolvedValue(toOperatorRow(createOperatorDomain({ role: 'operator' })));
    const createAuditEvent = vi.fn().mockResolvedValue(createAuditEventRow({ id: 'audit-role' }));
    const updateAuditEvent = vi.fn().mockResolvedValue(createAuditEventRow({ id: 'audit-role' }));
    const adminPrisma = createAdminPrismaMock({
      createAuditEvent,
      findOperator,
      updateAuditEvent,
      updateOperator,
    });

    const operator = await updateOperatorRole(
      adminPrisma as Parameters<typeof updateOperatorRole>[0],
      'operator-1',
      'operator',
      'admin-1'
    );

    expect(operator.role).toBe('operator');
    expect(createAuditEvent).toHaveBeenCalledWith({
      data: expect.objectContaining({
        action: 'operator.update_role',
        details: { from_role: 'viewer', to_role: 'operator' },
        operator_id: 'admin-1',
        outcome: 'pending',
        target_id: 'operator-1',
      }),
    });
    expect(updateAuditEvent).toHaveBeenCalledWith({
      data: expect.objectContaining({ outcome: 'succeeded' }),
      where: { id: 'audit-role' },
    });
  });

  it('14.6 operator role 更新は audit intent 作成失敗時に mutation を開始せず 503 を返す', async () => {
    // operator 系 mutation でも未監査変更を防ぐため、pending intent 失敗時は role update を呼ばない。
    const findOperator = vi
      .fn()
      .mockResolvedValue(toOperatorRow(createOperatorDomain({ role: 'viewer' })));
    const createAuditEvent = vi.fn().mockRejectedValue(new Error('audit unavailable'));
    const updateOperator = vi.fn();
    const adminPrisma = createAdminPrismaMock({ createAuditEvent, findOperator, updateOperator });

    await expect(
      updateOperatorRole(
        adminPrisma as Parameters<typeof updateOperatorRole>[0],
        'operator-1',
        'operator',
        'admin-1'
      )
    ).rejects.toMatchObject({ code: 'AUDIT_UNAVAILABLE', statusCode: 503 });

    expect(updateOperator).not.toHaveBeenCalled();
  });

  it('operator role 更新の succeeded outcome 失敗時は structured warning と metric signal を出す', async () => {
    // Admin DB mutation 成功後の outcome 失敗は rollback せず、reconciliation に必要なログと metric signal を残す。
    const warningSpy = vi.spyOn(process, 'emitWarning').mockImplementation(() => undefined);
    const metricSpy = vi.spyOn(process, 'emit').mockImplementation(() => true);
    const findOperator = vi
      .fn()
      .mockResolvedValue(toOperatorRow(createOperatorDomain({ role: 'viewer' })));
    const updateOperator = vi
      .fn()
      .mockResolvedValue(toOperatorRow(createOperatorDomain({ role: 'operator' })));
    const createAuditEvent = vi.fn().mockResolvedValue(createAuditEventRow({ id: 'audit-role' }));
    const updateAuditEvent = vi.fn().mockRejectedValue(new Error('audit outcome unavailable'));
    const adminPrisma = createAdminPrismaMock({
      createAuditEvent,
      findOperator,
      updateAuditEvent,
      updateOperator,
    });

    const operator = await updateOperatorRole(
      adminPrisma as Parameters<typeof updateOperatorRole>[0],
      'operator-1',
      'operator',
      'admin-1'
    );

    expect(operator.role).toBe('operator');
    expect(warningSpy).toHaveBeenCalledWith(
      expect.stringContaining('mark_succeeded'),
      'AuditReconciliationWarning'
    );
    expect(metricSpy).toHaveBeenCalledWith('admin.audit_reconciliation_required', {
      eventId: 'audit-role',
      phase: 'mark_succeeded',
    });
  });

  it('operator role mutation 失敗時は stable fallback error_code で failed outcome にする', async () => {
    // 不安定な DB message を監査 error_code に保存せず、固定 fallback code に正規化する。
    const findOperator = vi
      .fn()
      .mockResolvedValue(toOperatorRow(createOperatorDomain({ role: 'viewer' })));
    const updateOperator = vi
      .fn()
      .mockRejectedValue(new Error('database detail with mutable wording'));
    const createAuditEvent = vi.fn().mockResolvedValue(createAuditEventRow({ id: 'audit-role' }));
    const updateAuditEvent = vi.fn().mockResolvedValue(createAuditEventRow({ id: 'audit-role' }));
    const adminPrisma = createAdminPrismaMock({
      createAuditEvent,
      findOperator,
      updateAuditEvent,
      updateOperator,
    });

    await expect(
      updateOperatorRole(
        adminPrisma as Parameters<typeof updateOperatorRole>[0],
        'operator-1',
        'operator',
        'admin-1'
      )
    ).rejects.toThrow('database detail with mutable wording');

    expect(updateAuditEvent).toHaveBeenCalledWith({
      data: expect.objectContaining({
        error_code: 'OPERATOR_ROLE_UPDATE_FAILED',
        outcome: 'failed',
      }),
      where: { id: 'audit-role' },
    });
  });

  it('14.8 createOperator は one-time setup token を返し hash のみ保存する', async () => {
    // 平文 setup token は一度だけ返し、DB には照合可能な hash のみ保存されることを確認する。
    const createdOperator = createOperatorDomain({ id: 'operator-new', email: 'new@example.com' });
    const createOperatorMock = vi.fn().mockResolvedValue(toOperatorRow(createdOperator));
    const updateOperator = vi.fn().mockResolvedValue(toOperatorRow(createdOperator));
    const createAuditEvent = vi
      .fn()
      .mockResolvedValue(createAuditEventRow({ id: 'audit-create', target_id: 'new@example.com' }));
    const updateAuditEvent = vi.fn().mockResolvedValue(createAuditEventRow({ id: 'audit-create' }));
    const adminPrisma = createAdminPrismaMock({
      createAuditEvent,
      createOperator: createOperatorMock,
      updateAuditEvent,
      updateOperator,
    });

    const result = await createOperator(
      adminPrisma as Parameters<typeof createOperator>[0],
      { displayName: 'New Operator', email: 'new@example.com', role: 'operator' },
      'admin-1'
    );

    const updateCall = updateOperator.mock.calls[0];
    expect(updateCall).toBeDefined();
    if (updateCall === undefined) throw new Error('setup token update was not called');
    const data = updateCall[0].data as { setup_token_expires_at: Date; setup_token_hash: string };
    expect(result.plaintextToken).toHaveLength(64);
    expect(data.setup_token_hash).not.toBe(result.plaintextToken);
    expect(compareSync(result.plaintextToken, data.setup_token_hash)).toBe(true);
    expect(data.setup_token_expires_at).toBeInstanceOf(Date);
    expect(createAuditEvent).toHaveBeenCalledWith({
      data: expect.objectContaining({
        action: 'operator.create',
        outcome: 'pending',
        target_id: 'new@example.com',
      }),
    });
    expect(updateAuditEvent).toHaveBeenCalledWith({
      data: expect.objectContaining({ outcome: 'succeeded' }),
      where: { id: 'audit-create' },
    });
  });

  it('14.9 setup token 再発行が旧 token を無効化して audit 記録を残す', async () => {
    // 旧 token hash を新しい hash で上書きし、再発行操作も監査に残ることを確認する。
    const oldHash = '$2a$10$abcdefghijklmnopqrstuuJSRE7gZVJAmX4y1a48Xf0J0Li4czvPG';
    const target = createOperatorDomain({ id: 'operator-1', setupTokenHash: oldHash });
    const countPasskeys = vi.fn().mockResolvedValue(0);
    const updateOperator = vi.fn().mockResolvedValue(toOperatorRow(target));
    const findOperator = vi.fn().mockResolvedValue(toOperatorRow(target));
    const createAuditEvent = vi
      .fn()
      .mockResolvedValue(
        createAuditEventRow({ action: 'operator.setup_token.rotate', id: 'audit-rotate' })
      );
    const updateAuditEvent = vi.fn().mockResolvedValue(createAuditEventRow({ id: 'audit-rotate' }));
    const adminPrisma = createAdminPrismaMock({
      countPasskeys,
      createAuditEvent,
      findOperator,
      updateAuditEvent,
      updateOperator,
    });

    const result = await rotateSetupToken(
      adminPrisma as Parameters<typeof rotateSetupToken>[0],
      'operator-1',
      'admin-1'
    );

    const updateCall = updateOperator.mock.calls[0];
    expect(updateCall).toBeDefined();
    if (updateCall === undefined) throw new Error('setup token rotation update was not called');
    const data = updateCall[0].data as { setup_token_hash: string };
    expect(data.setup_token_hash).not.toBe(oldHash);
    expect(data.setup_token_hash).not.toBe(result.plaintextToken);
    expect(compareSync(result.plaintextToken, data.setup_token_hash)).toBe(true);
    expect(createAuditEvent).toHaveBeenCalledWith({
      data: expect.objectContaining({
        action: 'operator.setup_token.rotate',
        outcome: 'pending',
        target_id: 'operator-1',
      }),
    });
    expect(updateAuditEvent).toHaveBeenCalledWith({
      data: expect.objectContaining({ outcome: 'succeeded' }),
      where: { id: 'audit-rotate' },
    });
  });

  it('14.10 passkey 登録済みオペレーターの token 再発行を拒否する', async () => {
    // passkey 登録済みなら setup token 再発行で認証強度を下げないよう、DB 更新前に拒否する。
    const countPasskeys = vi.fn().mockResolvedValue(1);
    const updateOperator = vi.fn();
    const createAuditEvent = vi.fn();
    const adminPrisma = createAdminPrismaMock({ countPasskeys, createAuditEvent, updateOperator });

    await expect(
      rotateSetupToken(
        adminPrisma as Parameters<typeof rotateSetupToken>[0],
        'operator-1',
        'admin-1'
      )
    ).rejects.toMatchObject({ code: 'PASSKEY_EXISTS', statusCode: 400 });

    expect(updateOperator).not.toHaveBeenCalled();
    expect(createAuditEvent).not.toHaveBeenCalled();
  });

  it('14.11 最後の admin 無効化を拒否する', async () => {
    // 管理コンソールを復旧不能にしないため、最後の active admin は deactivate 更新前に拒否する。
    const findOperator = vi
      .fn()
      .mockResolvedValue(toOperatorRow(createOperatorDomain({ role: 'admin' })));
    const countOperators = vi.fn().mockResolvedValue(1);
    const updateOperator = vi.fn();
    const adminPrisma = createAdminPrismaMock({ countOperators, findOperator, updateOperator });

    await expect(
      deactivateOperator(
        adminPrisma as Parameters<typeof deactivateOperator>[0],
        'operator-1',
        'admin-2'
      )
    ).rejects.toMatchObject({ code: 'LAST_ADMIN_DEACTIVATION', statusCode: 400 });

    expect(updateOperator).not.toHaveBeenCalled();
  });

  it('14.12 最後の admin 降格を拒否する', async () => {
    // 最後の active admin を viewer/operator に落とす操作を拒否し、管理権限の喪失を防ぐ。
    const findOperator = vi
      .fn()
      .mockResolvedValue(toOperatorRow(createOperatorDomain({ role: 'admin' })));
    const countOperators = vi.fn().mockResolvedValue(1);
    const updateOperator = vi.fn();
    const adminPrisma = createAdminPrismaMock({ countOperators, findOperator, updateOperator });

    await expect(
      updateOperatorRole(
        adminPrisma as Parameters<typeof updateOperatorRole>[0],
        'operator-1',
        'viewer',
        'admin-2'
      )
    ).rejects.toMatchObject({ code: 'LAST_ADMIN_DEMOTION', statusCode: 400 });

    expect(updateOperator).not.toHaveBeenCalled();
  });

  it('ServiceError 型を保ったまま passkey 登録済み token 再発行拒否を返す', async () => {
    // route 層が statusCode/code を安全にレスポンス化できるよう、拒否エラーの型も固定する。
    const countPasskeys = vi.fn().mockResolvedValue(1);
    const adminPrisma = createAdminPrismaMock({ countPasskeys });

    await expect(
      rotateSetupToken(
        adminPrisma as Parameters<typeof rotateSetupToken>[0],
        'operator-1',
        'admin-1'
      )
    ).rejects.toBeInstanceOf(ServiceError);
  });
});
