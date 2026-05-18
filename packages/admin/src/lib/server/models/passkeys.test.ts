import { describe, expect, it, vi } from 'vitest';

/**
 * passkeys model の createRequire 経由 Prisma 参照をテスト内で閉じる。
 * 入力の specifier が Prisma client の場合だけ DbNull を返し、その他は通常 require に委譲する。
 * 生成済み client が未配置の環境でも、モデル関数の DB 呼び出し引数を検証できるようにする副作用限定 mock。
 */
vi.mock('node:module', () => {
  return {
    createRequire: () => (specifier: string) => {
      if (specifier === '.prisma/admin-client')
        return { Prisma: { DbNull: Symbol.for('Prisma.DbNull') } };
      throw new Error(`Unexpected require specifier in passkeys test: ${specifier}`);
    },
  };
});

import { addOperatorPasskey, updateOperatorPasskeySignCount } from './passkeys.js';

/**
 * passkey model が必要とする Admin Prisma mock を生成する。
 * 入力は passkey delegate の一部で、出力は PrismaClient 互換として model に渡す。
 * 実 DB へ接続せず、呼び出し内容だけを観測する。
 */
function createAdminPrismaMock(delegate: Partial<PasskeyDelegate>): unknown {
  return {
    adminOperatorPasskey: delegate,
  };
}

/**
 * passkey の Prisma 行を決定的な値で作る。
 * 入力 override で対象列を差し替え、出力は mapper が期待する列名に合わせる。
 * バイナリ列も固定値にして、テスト実行順による差分をなくす。
 */
function createPasskeyRow(overrides: Partial<PasskeyRow> = {}): PasskeyRow {
  return {
    id: 'passkey-1',
    operator_id: 'operator-1',
    credential_handle: 'credential-1',
    public_key: new Uint8Array([1, 2, 3]),
    sign_count: 0n,
    aaguid: new Uint8Array([4, 5, 6]),
    backup_eligible: false,
    backup_state: false,
    transports: ['internal'],
    createdAt: new Date('2026-05-17T00:00:00.000Z'),
    ...overrides,
  };
}

interface PasskeyDelegate {
  create: ReturnType<typeof vi.fn>;
  update: ReturnType<typeof vi.fn>;
}

interface PasskeyRow {
  id: string;
  operator_id: string;
  credential_handle: string;
  public_key: Uint8Array;
  sign_count: bigint;
  aaguid: Uint8Array;
  backup_eligible: boolean;
  backup_state: boolean;
  transports: unknown;
  createdAt: Date;
}

describe('models/passkeys', () => {
  it('13.2 updateOperatorPasskeySignCount は認証後の signCount を BigInt で更新する', async () => {
    // WebAuthn 認証後の replay 防止に直結するため、数値から bigint への変換を確認する。
    const update = vi.fn().mockResolvedValue(createPasskeyRow({ sign_count: 42n }));
    const adminPrisma = createAdminPrismaMock({ update });

    await updateOperatorPasskeySignCount(
      adminPrisma as Parameters<typeof updateOperatorPasskeySignCount>[0],
      'passkey-1',
      42
    );

    expect(update).toHaveBeenCalledWith({
      where: { id: 'passkey-1' },
      data: { sign_count: 42n },
    });
  });

  it('13.3 operator_passkeys.sign_count は migration と Prisma schema の両方で default 0', async () => {
    // DB と Prisma schema の片側だけが default を持つ drift を防ぐため、両ソースを同時に確認する。
    const fs = await import('node:fs/promises');
    const [schema, migration] = await Promise.all([
      fs.readFile('prisma/admin/schema.prisma', 'utf8'),
      fs.readFile(
        'prisma/admin/migrations/000001_create_operators_and_passkeys/migration.sql',
        'utf8'
      ),
    ]);

    expect(schema).toMatch(/sign_count\s+BigInt\s+@default\(0\)/);
    expect(migration.toLowerCase()).toMatch(/sign_count\s+bigint\s+not null\s+default 0/);
  });

  it('addOperatorPasskey は明示 signCount をそのまま保存する', async () => {
    // default 検証とは別に、認証器から受け取った signCount を上書きしないことを確認する。
    const create = vi.fn().mockResolvedValue(createPasskeyRow({ sign_count: 7n }));
    const adminPrisma = createAdminPrismaMock({ create });

    await addOperatorPasskey(adminPrisma as Parameters<typeof addOperatorPasskey>[0], {
      operatorId: 'operator-1',
      credentialHandle: 'credential-1',
      publicKey: new Uint8Array([1]),
      signCount: 7n,
      aaguid: new Uint8Array([2]),
      backupEligible: true,
      backupState: false,
      transports: ['usb'],
    });

    expect(create).toHaveBeenCalledWith({
      data: expect.objectContaining({ sign_count: 7n }),
    });
  });
});
