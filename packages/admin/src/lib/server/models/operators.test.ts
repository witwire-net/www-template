import { describe, expect, it, vi } from 'vitest';

import {
  createInitialAdminOperator,
  createOperator,
  listOperators,
  updateOperatorLocale,
  updateLoginTimestamp,
} from './operators.js';

/**
 * Prisma の時刻列を含むオペレーター行を生成する。
 * 入力は必要な列だけを上書きでき、出力はモデル関数が期待する Prisma 行の形に揃える。
 * 副作用はなく、各テストで同じ基準時刻を使うことで検証を決定的にする。
 */
function createOperatorRow(overrides: Partial<OperatorRow> = {}): OperatorRow {
  const now = new Date('2026-05-17T00:00:00.000Z');
  return {
    id: 'operator-1',
    email: 'admin@example.com',
    display_name: 'Admin Operator',
    role: 'admin',
    is_active: true,
    locale: 'ja',
    setup_token_hash: null,
    setup_token_expires_at: null,
    last_login_at: null,
    createdAt: now,
    updatedAt: now,
    ...overrides,
  };
}

/**
 * operators model が参照する最小限の Admin Prisma mock を作る。
 * 入力として delegate の各メソッドを受け取り、出力は PrismaClient 互換の unknown 型にする。
 * DB 接続は行わず、呼び出し引数だけを観測するため副作用はない。
 */
function createAdminPrismaMock(delegate: Partial<AdminOperatorDelegate>): unknown {
  return {
    adminOperator: delegate,
  };
}

interface OperatorRow {
  id: string;
  email: string;
  display_name: string;
  role: string;
  is_active: boolean;
  locale: string;
  setup_token_hash: string | null;
  setup_token_expires_at: Date | null;
  last_login_at: Date | null;
  createdAt: Date;
  updatedAt: Date;
}

interface AdminOperatorDelegate {
  create: ReturnType<typeof vi.fn>;
  findMany: ReturnType<typeof vi.fn>;
  update: ReturnType<typeof vi.fn>;
}

describe('models/operators', () => {
  it('17.7 Admin DB query works: listOperators は adminOperator delegate を呼ぶ', async () => {
    // Admin DB 用 Prisma Client の delegate だけを使い、Product DB に依存せず operator 一覧を取得できることを確認する。
    const findMany = vi.fn().mockResolvedValue([createOperatorRow()]);
    const adminPrisma = createAdminPrismaMock({ findMany });

    const result = await listOperators(adminPrisma as Parameters<typeof listOperators>[0]);

    expect(findMany).toHaveBeenCalledWith({ orderBy: { createdAt: 'desc' } });
    expect(result).toEqual([
      expect.objectContaining({
        id: 'operator-1',
        email: 'admin@example.com',
        role: 'admin',
        locale: 'ja',
      }),
    ]);
  });

  it('13.1 updateLoginTimestamp は last_login_at を現在時刻で更新する', async () => {
    // ログイン成功時の副作用だけを検証するため、Prisma update の引数を捕捉する。
    const update = vi.fn().mockResolvedValue(createOperatorRow());
    const adminPrisma = createAdminPrismaMock({ update });

    await updateLoginTimestamp(
      adminPrisma as Parameters<typeof updateLoginTimestamp>[0],
      'operator-1'
    );

    // 顧客の監査・認証状態に関わるため、対象 ID と Date 型更新を厳密に確認する。
    expect(update).toHaveBeenCalledWith({
      where: { id: 'operator-1' },
      data: { last_login_at: expect.any(Date) as Date },
    });
  });

  it('13.5 createInitialAdminOperator は重複 email の UNIQUE 制約違反を伝播する', async () => {
    // DB 制約をモデル層で握りつぶさないことを確認し、呼び出し元が安全に 409 等へ変換できるようにする。
    const uniqueError = new Error('Unique constraint failed on the fields: (`email`)');
    const create = vi.fn().mockRejectedValue(uniqueError);
    const adminPrisma = createAdminPrismaMock({ create });

    await expect(
      createInitialAdminOperator(adminPrisma as Parameters<typeof createInitialAdminOperator>[0], {
        email: 'admin@example.com',
        displayName: 'Duplicated Admin',
      })
    ).rejects.toThrow('Unique constraint failed');
  });

  it('13.6 createOperator は role CHECK 制約違反を伝播する', async () => {
    // DB の CHECK 制約違反を隠さず返すことで、無効 role の永続化を防ぐ。
    const checkError = new Error('violates check constraint "operators_role_check"');
    const create = vi.fn().mockRejectedValue(checkError);
    const adminPrisma = createAdminPrismaMock({ create });

    await expect(
      createOperator(adminPrisma as Parameters<typeof createOperator>[0], {
        email: 'operator@example.com',
        displayName: 'Invalid Role',
        role: 'owner',
      })
    ).rejects.toThrow('operators_role_check');
  });

  it('updateOperatorLocale は Admin operator の保存済み locale を返す', async () => {
    // Admin operator locale は Product AccountSetting を使わず、Admin DB の operator row だけを更新・復元する。
    const update = vi.fn().mockResolvedValue(createOperatorRow({ locale: 'en' }));
    const adminPrisma = createAdminPrismaMock({ update });

    const operator = await updateOperatorLocale(
      adminPrisma as Parameters<typeof updateOperatorLocale>[0],
      'operator-1',
      'en'
    );

    expect(update).toHaveBeenCalledWith({
      where: { id: 'operator-1' },
      data: { locale: 'en' },
    });
    expect(operator.locale).toBe('en');
  });

  it('未知の永続 operator locale は既定値へ丸めず拒否する', async () => {
    // DB から未知 locale が返った場合に ja へ黙って丸めると破損を隠すため、mapper で fail-closed にする。
    const findMany = vi.fn().mockResolvedValue([createOperatorRow({ locale: 'fr' })]);
    const adminPrisma = createAdminPrismaMock({ findMany });

    await expect(listOperators(adminPrisma as Parameters<typeof listOperators>[0])).rejects.toThrow(
      'unsupported admin operator locale'
    );
  });

  it('13.4 operators 削除時の passkey cascade は Admin DB migration で定義されている', async () => {
    // Prisma mock では FK 動作を再現しないため、migration SQL に ON DELETE CASCADE が固定されていることを検証する。
    const migration = await import('node:fs/promises').then((fs) =>
      fs.readFile(
        'prisma/admin/migrations/000001_create_operators_and_passkeys/migration.sql',
        'utf8'
      )
    );

    expect(migration).toMatch(/REFERENCES admin\.operators\(id\)\s+ON DELETE CASCADE/);
  });
});
