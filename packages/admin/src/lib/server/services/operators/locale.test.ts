import { describe, expect, it, vi } from 'vitest';

import { updateOwnOperatorLocale } from './locale.js';

function createOperatorRow(overrides: Partial<OperatorRow> = {}): OperatorRow {
  const now = new Date('2026-05-17T00:00:00.000Z');
  return {
    id: 'operator-1',
    email: 'operator@example.com',
    display_name: 'Operator One',
    role: 'operator',
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

function createAdminPrismaMock(input: {
  update: ReturnType<typeof vi.fn>;
  findUnique?: ReturnType<typeof vi.fn>;
}): unknown {
  // service が更新前に operator の存在と active 状態を確認するため、findUnique と update の両方を差し替える。
  return {
    adminOperator: {
      findUnique: input.findUnique ?? vi.fn().mockResolvedValue(createOperatorRow()),
      update: input.update,
    },
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

describe('services/operators/locale', () => {
  it('LOCALIZATION-BE-S010 オペレーターは自分の locale を更新できる', async () => {
    // form 由来の locale を認証済み本人 ID にだけ適用し、他 operator ID を入力として受け取らないことを確認する。
    const update = vi.fn().mockResolvedValue(createOperatorRow({ locale: 'en' }));
    const findUnique = vi.fn().mockResolvedValue(createOperatorRow());
    const adminPrisma = createAdminPrismaMock({ findUnique, update });

    const operator = await updateOwnOperatorLocale(
      adminPrisma as Parameters<typeof updateOwnOperatorLocale>[0],
      'operator-1',
      'en'
    );

    expect(findUnique).toHaveBeenCalledWith({ where: { id: 'operator-1' } });
    expect(update).toHaveBeenCalledWith({
      where: { id: 'operator-1' },
      data: { locale: 'en' },
    });
    expect(operator.locale).toBe('en');
  });

  it('LOCALIZATION-BE-S011 Admin の未対応 locale 更新は保存値を変更せず拒否される', async () => {
    // 未対応 locale は Prisma update を呼ぶ前に拒否し、DB の保存済み locale を変えない。
    const update = vi.fn();
    const adminPrisma = createAdminPrismaMock({ update });

    await expect(
      updateOwnOperatorLocale(
        adminPrisma as Parameters<typeof updateOwnOperatorLocale>[0],
        'operator-1',
        'fr'
      )
    ).rejects.toMatchObject({ code: 'UNSUPPORTED_OPERATOR_LOCALE', statusCode: 400 });

    expect(update).not.toHaveBeenCalled();
  });
});
