import { beforeEach, describe, expect, it, vi } from 'vitest';

import type { AdminOperatorSessionResponse } from '@www-template/admin-api';

import { createCustomerAccount, searchAdminAccounts } from './accounts';
import { clearAdminSession, finishAdminLogin } from './auth';

const apiMocks = vi.hoisted(() => ({
  requestAdminAccounts: vi.fn(),
  requestCreateAdminAccount: vi.fn(),
  requestFinishAdminLogin: vi.fn(),
}));

vi.mock('@www-template/admin-api', () => ({
  requestAdminAccounts: apiMocks.requestAdminAccounts,
  requestCreateAdminAccount: apiMocks.requestCreateAdminAccount,
  requestFinishAdminLogin: apiMocks.requestFinishAdminLogin,
}));

const sessionResponse: AdminOperatorSessionResponse = {
  requestId: '01JREQUEST0000000000000000',
  credentialMode: 'cookie',
  operator: {
    operatorId: '01JOPERATOR00000000000000',
    email: 'operator@example.com',
    role: 'operator',
    active: true,
  },
  sessionId: '01JSESSION0000000000000000',
  authContextId: '01JSESSION0000000000000000',
  accessToken: 'operator-access-token',
  expiresAt: '2030-01-01T00:00:00.000Z',
  contextIndexUpdateHints: [],
  clearCookieCommands: [],
};

async function seedSession(): Promise<void> {
  // account domain functions が memory-only Admin session を使えるよう、login 完了結果を事前投入する。
  apiMocks.requestFinishAdminLogin.mockResolvedValueOnce({
    status: 200,
    data: sessionResponse,
    headers: new Headers(),
  });
  await finishAdminLogin('01JLOGINREQUEST0000000000', {
    id: 'credential-id',
    rawId: 'credential-raw-id',
    type: 'public-key',
    response: {
      clientDataJSON: 'client-data',
      authenticatorData: 'auth-data',
      signature: 'signature',
    },
  });
}

describe('Admin account domain orchestration', () => {
  beforeEach(() => {
    // 各 test を独立させるため、module-local session と Admin API mock の履歴を初期化する。
    clearAdminSession();
    vi.clearAllMocks();
  });

  it('[ADMIN-CONSOLE-FE-S040] account data is loaded through the Admin API layer', async () => {
    // Product SDK ではなく Admin API wrapper mock だけが呼ばれることを検索 flow で検証する。
    await seedSession();
    apiMocks.requestAdminAccounts.mockResolvedValueOnce({
      status: 200,
      data: {
        accounts: [
          {
            accountId: '01JACCOUNT000000000000000',
            email: 'customer@example.com',
            status: 'active',
            createdAt: '2030-01-01T00:00:00.000Z',
            passkeyCount: 0,
          },
        ],
        nextCursor: null,
      },
      headers: new Headers(),
    });

    const result = await searchAdminAccounts({ email: ' customer@example.com ', limit: 20 });

    expect(apiMocks.requestAdminAccounts).toHaveBeenCalledWith(
      { email: 'customer@example.com', cursor: undefined, limit: 20 },
      expect.objectContaining({ accessToken: 'operator-access-token' })
    );
    expect(result).toEqual({
      success: true,
      data: {
        accounts: [
          {
            id: '01JACCOUNT000000000000000',
            email: 'customer@example.com',
            status: 'active',
            createdAt: '2030-01-01T00:00:00.000Z',
            passkeyCount: 0,
          },
        ],
        nextCursor: null,
      },
    });
  });

  it('[ADMIN-CONSOLE-FE-S043] operator creates a customer account through Admin API', async () => {
    // 作成成功時は Admin API response の account read model だけを UI 用 data へ写像する。
    await seedSession();
    apiMocks.requestCreateAdminAccount.mockResolvedValueOnce({
      status: 201,
      data: {
        requestId: '01JCREATE000000000000000',
        account: {
          accountId: '01JACCOUNT000000000000000',
          email: 'customer@example.com',
          status: 'active',
          createdAt: '2030-01-01T00:00:00.000Z',
          passkeyCount: 0,
        },
        auditEventId: '01JAUDIT0000000000000000',
      },
      headers: new Headers(),
    });

    const result = await createCustomerAccount({ email: ' customer@example.com ', locale: 'ja' });

    expect(apiMocks.requestCreateAdminAccount).toHaveBeenCalledWith(
      { email: 'customer@example.com', locale: 'ja' },
      expect.objectContaining({ accessToken: 'operator-access-token' })
    );
    expect(result.success).toBe(true);
  });

  it('[ADMIN-CONSOLE-FE-S044] invalid email is rejected before request submission', async () => {
    // 明らかに不正な email は domain validation で止め、Admin backend request を発生させない。
    await seedSession();

    const result = await createCustomerAccount({ email: 'not-an-email', locale: 'ja' });

    expect(result).toEqual({ success: false, error: 'invalid-input' });
    expect(apiMocks.requestCreateAdminAccount).not.toHaveBeenCalled();
  });

  it('[ADMIN-CONSOLE-FE-S045] duplicate email maps to display error without losing input', async () => {
    // 重複時に domain は入力値を書き換えず、UI が同じ form state を保持できる error 分類だけを返す。
    await seedSession();
    const input = { email: 'customer@example.com', locale: 'ja' as const };
    apiMocks.requestCreateAdminAccount.mockResolvedValueOnce({
      status: 409,
      data: { requestId: '01JDUPLICATE000000000000', error: 'duplicate-email' },
      headers: new Headers(),
    });

    const result = await createCustomerAccount(input);

    expect(result).toEqual({ success: false, error: 'duplicate-email' });
    expect(input.email).toBe('customer@example.com');
  });
});
