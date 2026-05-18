import { describe, it, expect, vi, beforeEach } from 'vitest';

/**
 * PrismaClient のモック。
 * vi.mock は hoisting されるため、vi.hoisted でモック関数を先行定義する。
 */
const mocks = vi.hoisted(() => ({
  adminDisconnect: vi.fn().mockResolvedValue(undefined),
  productDisconnect: vi.fn().mockResolvedValue(undefined),
  productQueryRaw: vi
    .fn()
    .mockResolvedValue([{ has_role: true, is_superuser: false, is_owner: false }]),
}));

vi.mock('.prisma/admin-client', () => ({
  PrismaClient: vi.fn().mockImplementation(() => ({
    $disconnect: mocks.adminDisconnect,
  })),
}));

vi.mock('.prisma/product-client', () => ({
  PrismaClient: vi.fn().mockImplementation(() => ({
    $disconnect: mocks.productDisconnect,
    $queryRaw: mocks.productQueryRaw,
  })),
}));

import {
  disconnectPrisma,
  getProductPrisma,
  setPrismaClientFactoriesForTest,
  validateProductDbRuntimeRole,
} from './prisma.js';

type ProductPrismaClient = Awaited<ReturnType<typeof getProductPrisma>>;

describe('getProductPrisma', () => {
  beforeEach(async () => {
    setPrismaClientFactoriesForTest({
      admin: vi.fn().mockImplementation(() => ({
        $disconnect: mocks.adminDisconnect,
      })),
      product: vi.fn().mockImplementation(() => ({
        $disconnect: mocks.productDisconnect,
        $queryRaw: mocks.productQueryRaw,
      })),
    });
    await disconnectPrisma();
    vi.clearAllMocks();
  });

  it('初回呼び出し時に validateProductDbRuntimeRole を実行する', async () => {
    await getProductPrisma();
    expect(mocks.productQueryRaw).toHaveBeenCalledTimes(1);
  });

  it('検証成功後の再呼び出しでは validateProductDbRuntimeRole をスキップする', async () => {
    await getProductPrisma();
    await getProductPrisma();
    expect(mocks.productQueryRaw).toHaveBeenCalledTimes(1);
  });

  it('検証失敗時は validated フラグを立てず、次回呼び出しで再検証する', async () => {
    mocks.productQueryRaw.mockRejectedValueOnce(new Error('role validation failed'));
    await expect(getProductPrisma()).rejects.toThrow('role validation failed');

    mocks.productQueryRaw.mockResolvedValueOnce([
      { has_role: true, is_superuser: false, is_owner: false },
    ]);
    await getProductPrisma();
    expect(mocks.productQueryRaw).toHaveBeenCalledTimes(2);
  });

  it('17.9 DB connection failure throws: Product DB role validation failure は呼び出し元へ伝播する', async () => {
    // 接続・検証失敗時に Product DB を使い続けないよう、初回取得が fail-close することを確認する。
    mocks.productQueryRaw.mockRejectedValueOnce(new Error('database connection failed'));

    await expect(getProductPrisma()).rejects.toThrow('database connection failed');
    expect(mocks.productQueryRaw).toHaveBeenCalledTimes(1);
  });

  it('disconnectPrisma 後の再取得で再度 validateProductDbRuntimeRole が実行される', async () => {
    await getProductPrisma();
    await disconnectPrisma();

    mocks.productQueryRaw.mockClear();
    await getProductPrisma();
    expect(mocks.productQueryRaw).toHaveBeenCalledTimes(1);
  });
});

describe('validateProductDbRuntimeRole', () => {
  function createMockClient(result: unknown): ProductPrismaClient {
    return {
      $queryRaw: vi.fn().mockResolvedValue(result),
    } as unknown as ProductPrismaClient;
  }

  it('admin_console_write メンバーかつ superuser でなく owner でない場合は成功する', async () => {
    const client = createMockClient([{ has_role: true, is_superuser: false, is_owner: false }]);
    await expect(validateProductDbRuntimeRole(client)).resolves.toBeUndefined();
  });

  it('17.18a getProductPrisma startup validation は role membership を確認し superuser/base table owner を拒否する', async () => {
    // 起動時に admin_console_write 所属、非 superuser、非 owner の 3 条件をすべて検証することを確認する。
    const allowedClient = createMockClient([
      { has_role: true, is_superuser: false, is_owner: false },
    ]);
    const nonMemberClient = createMockClient([
      { has_role: false, is_superuser: false, is_owner: false },
    ]);
    const superuserClient = createMockClient([
      { has_role: true, is_superuser: true, is_owner: false },
    ]);
    const ownerClient = createMockClient([{ has_role: true, is_superuser: false, is_owner: true }]);

    await expect(validateProductDbRuntimeRole(allowedClient)).resolves.toBeUndefined();
    await expect(validateProductDbRuntimeRole(nonMemberClient)).rejects.toThrow('not a member');
    await expect(validateProductDbRuntimeRole(superuserClient)).rejects.toThrow('superuser');
    await expect(validateProductDbRuntimeRole(ownerClient)).rejects.toThrow('base table owner');
  });

  it('admin_console_write メンバーでない場合はエラーを throw する', async () => {
    const client = createMockClient([{ has_role: false, is_superuser: false, is_owner: false }]);
    await expect(validateProductDbRuntimeRole(client)).rejects.toThrow(
      'Product DB login role is not a member of admin_console_write'
    );
  });

  it('superuser の場合はエラーを throw する', async () => {
    const client = createMockClient([{ has_role: true, is_superuser: true, is_owner: false }]);
    await expect(validateProductDbRuntimeRole(client)).rejects.toThrow(
      'Product DB login role is superuser'
    );
  });

  it('base table owner の場合はエラーを throw する', async () => {
    const client = createMockClient([{ has_role: true, is_superuser: false, is_owner: true }]);
    await expect(validateProductDbRuntimeRole(client)).rejects.toThrow(
      'Product DB login role is base table owner'
    );
  });

  it('クエリ結果が空の場合はエラーを throw する', async () => {
    const client = createMockClient([]);
    await expect(validateProductDbRuntimeRole(client)).rejects.toThrow(
      'Product DB runtime role validation returned no rows'
    );
  });
});
