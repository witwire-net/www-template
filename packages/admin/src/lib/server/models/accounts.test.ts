import { describe, expect, it, vi } from 'vitest';
import { ZodError } from 'zod';

import { getAccountPasskeys, searchAccounts, suspendAccountProduct } from './accounts.js';
import { searchParamsSchema } from './schemas.js';

/**
 * Product Prisma の raw query / execute mock を生成する。
 * 入力は任意の mock 関数で、出力は model 関数が要求する PrismaClient 互換の unknown。
 * 実 DB に触れず、SQL tag が parameterized call として使われたことを検証する。
 */
function createProductPrismaMock(delegate: ProductPrismaDelegate): unknown {
  return delegate;
}

/**
 * admin_view.account_summaries の raw 行を生成する。
 * 入力で必要な列を上書きでき、出力は mapper の期待列と同じ snake_case にする。
 * 日付と件数を固定して、pagination の検証を決定的にする。
 */
function createAccountRow(overrides: Partial<AccountSummaryRaw> = {}): AccountSummaryRaw {
  return {
    id: 'account-1',
    email: 'customer@example.com',
    status: 'active',
    status_reason: null,
    status_updated_at: null,
    status_updated_by: null,
    session_revoked_after: null,
    created_at: new Date('2026-05-17T00:00:00.000Z'),
    passkey_count: 2n,
    ...overrides,
  };
}

interface ProductPrismaDelegate {
  $queryRaw?: ReturnType<typeof vi.fn>;
  $executeRaw?: ReturnType<typeof vi.fn>;
}

interface AccountSummaryRaw {
  id: string;
  email: string;
  status: string;
  status_reason: string | null;
  status_updated_at: Date | null;
  status_updated_by: string | null;
  session_revoked_after: Date | null;
  created_at: Date;
  passkey_count: bigint;
}

describe('models/accounts', () => {
  it('17.8 Product DB query works: searchAccounts は admin_view.account_summaries を query する', async () => {
    // Product DB への読み取り経路が Prisma の parameterized raw query を使い、admin_view に限定されることを確認する。
    const queryRaw = vi
      .fn()
      .mockResolvedValueOnce([{ count: 1n }])
      .mockResolvedValueOnce([createAccountRow()]);
    const productPrisma = createProductPrismaMock({ $queryRaw: queryRaw });

    const result = await searchAccounts(productPrisma as Parameters<typeof searchAccounts>[0], {
      query: 'customer',
      status: 'active',
      limit: 20,
      offset: 0,
    });

    const countCall = queryRaw.mock.calls[0];
    const itemsCall = queryRaw.mock.calls[1];
    expect(countCall).toBeDefined();
    expect(itemsCall).toBeDefined();
    if (countCall === undefined || itemsCall === undefined)
      throw new Error('Product DB query was not executed');
    expect(Array.from(countCall[0] as TemplateStringsArray).join('')).toContain(
      'admin_view.account_summaries'
    );
    expect(Array.from(itemsCall[0] as TemplateStringsArray).join('')).toContain(
      'admin_view.account_summaries'
    );
    expect(result.items).toHaveLength(1);
    expect(result.total).toBe(1n);
  });

  it('13.7 suspendAccountProduct は非 active エラーを伝播する', async () => {
    // SECURITY DEFINER 関数の business error を握りつぶさず、サービス層が監査失敗へ変換できる状態にする。
    const executeRaw = vi.fn().mockRejectedValue(new Error('account_not_active'));
    const productPrisma = createProductPrismaMock({ $executeRaw: executeRaw });

    await expect(
      suspendAccountProduct(
        productPrisma as Parameters<typeof suspendAccountProduct>[0],
        'account-1',
        'operator-1',
        'policy violation',
        'audit-1'
      )
    ).rejects.toThrow('account_not_active');
  });

  it('13.8 search params の limit が 1 未満なら ZodError', () => {
    // 過大・無効な DB 読み取りを入口で止めるため、limit 下限を schema で検証する。
    expect(() => searchParamsSchema.parse({ limit: 0, offset: 0 })).toThrow(ZodError);
  });

  it('17.12 invalid limit は route 層で 400 に変換可能な ZodError として拒否する', () => {
    // 不正な limit を DB に到達させず、呼び出し元が 400 応答へ変換できる検証エラーにする。
    expect(() => searchParamsSchema.parse({ limit: 0, offset: 0 })).toThrow(ZodError);
  });

  it('13.9 search params の offset が負数なら ZodError', () => {
    // 負の offset を DB に渡さないことで、ページングの意味を固定する。
    expect(() => searchParamsSchema.parse({ limit: 20, offset: -1 })).toThrow(ZodError);
  });

  it('17.13 negative offset は route 層で 400 に変換可能な ZodError として拒否する', () => {
    // 負数 offset を Product DB に渡さないため、検索 schema が境界値を fail-close することを確認する。
    expect(() => searchParamsSchema.parse({ limit: 20, offset: -1 })).toThrow(ZodError);
  });

  it('17.6 Product DB query works: getAccountPasskeys は passkey info view を返す', async () => {
    // アカウント詳細に必要な passkey 情報が admin_view.account_passkeys から mapping されることを確認する。
    const createdAt = new Date('2026-05-17T00:00:00.000Z');
    const queryRaw = vi.fn().mockResolvedValue([
      {
        id: 'passkey-1',
        operatorId: 'account-1',
        credentialHandle: 'credential-1',
        publicKey: new Uint8Array([1, 2, 3]),
        signCount: 4n,
        aaguid: new Uint8Array([4, 5, 6]),
        backupEligible: true,
        backupState: false,
        transports: ['internal'],
        createdAt,
      },
    ]);
    const productPrisma = createProductPrismaMock({ $queryRaw: queryRaw });

    const result = await getAccountPasskeys(
      productPrisma as Parameters<typeof getAccountPasskeys>[0],
      'account-1'
    );

    const call = queryRaw.mock.calls[0];
    expect(call).toBeDefined();
    if (call === undefined) throw new Error('Passkey query was not executed');
    expect(Array.from(call[0] as TemplateStringsArray).join('')).toContain(
      'admin_view.account_passkeys'
    );
    expect(call).toContain('account-1');
    expect(result).toEqual([
      expect.objectContaining({
        id: 'passkey-1',
        credentialHandle: 'credential-1',
        signCount: 4n,
        createdAt,
      }),
    ]);
  });

  it('13.10 searchAccounts は検索語を SQL 文字列へ連結せず parameterized query にする', async () => {
    // SQL injection 文字列を入力し、tagged template の値配列として分離されることを確認する。
    const queryRaw = vi
      .fn()
      .mockResolvedValueOnce([{ count: 0n }])
      .mockResolvedValueOnce([]);
    const productPrisma = createProductPrismaMock({ $queryRaw: queryRaw });
    const maliciousQuery = "%' OR '1'='1";

    await searchAccounts(productPrisma as Parameters<typeof searchAccounts>[0], {
      query: maliciousQuery,
      status: 'active',
      limit: 10,
      offset: 0,
    });

    const firstCall = queryRaw.mock.calls[0];
    // Vitest の型上は undefined の可能性があるため、呼び出し存在を先に検証してから SQL と bind 値を見る。
    expect(firstCall).toBeDefined();
    if (firstCall === undefined) throw new Error('searchAccounts did not execute count query');
    const sqlText = Array.from(firstCall[0] as TemplateStringsArray).join('');
    const boundValues = firstCall.slice(1);
    expect(sqlText).not.toContain(maliciousQuery);
    expect(boundValues).toContain(`%${maliciousQuery}%`);
  });

  it('13.11 searchAccounts は limit / offset 付きで検索し total と items を返す', async () => {
    // 一覧 UI のページングが壊れないよう、count と rows の両 query 結果を統合して返すことを確認する。
    const account = createAccountRow({ id: 'account-2', email: 'second@example.com' });
    const queryRaw = vi
      .fn()
      .mockResolvedValueOnce([{ count: 2n }])
      .mockResolvedValueOnce([account]);
    const productPrisma = createProductPrismaMock({ $queryRaw: queryRaw });

    const result = await searchAccounts(productPrisma as Parameters<typeof searchAccounts>[0], {
      query: 'second',
      status: 'active',
      limit: 1,
      offset: 1,
    });

    const itemsCall = queryRaw.mock.calls[1];
    // items query が実行されたことを先に固定し、その後に pagination bind 値を検証する。
    expect(itemsCall).toBeDefined();
    if (itemsCall === undefined) throw new Error('searchAccounts did not execute items query');
    expect(itemsCall).toContain(1);
    expect(result).toEqual({
      total: 2n,
      items: [
        expect.objectContaining({
          id: 'account-2',
          email: 'second@example.com',
          passkeyCount: 2n,
        }),
      ],
    });
  });
});
