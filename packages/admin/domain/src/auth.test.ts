import { beforeEach, describe, expect, it, vi } from 'vitest';

import type {
  AdminAuthSessionResponse,
  WWWTemplatePasskeyAddStartResponse,
  WWWTemplatePasskeyStartResponse,
  WWWTemplateWebAuthnAttestationCredential,
} from '@www-template/admin-api';

import {
  clearAdminSession,
  finishAdminLogin,
  finishInitialAdminSetup,
  getAdminSession,
  refreshAdminSession,
  startAdminLogin,
  startInitialAdminSetup,
  startOperatorSetup,
  verifyProtectedAdminRoute,
} from './auth';

const apiMocks = vi.hoisted(() => ({
  requestCurrentAdminOperator: vi.fn(),
  requestFinishAdminLogin: vi.fn(),
  requestFinishInitialAdminSetup: vi.fn(),
  requestRefreshAdminSession: vi.fn(),
  requestStartAdminLogin: vi.fn(),
  requestStartInitialAdminSetup: vi.fn(),
  requestStartOperatorSetup: vi.fn(),
}));

vi.mock('@www-template/admin-api', () => ({
  requestCurrentAdminOperator: apiMocks.requestCurrentAdminOperator,
  requestFinishAdminLogin: apiMocks.requestFinishAdminLogin,
  requestFinishInitialAdminSetup: apiMocks.requestFinishInitialAdminSetup,
  requestRefreshAdminSession: apiMocks.requestRefreshAdminSession,
  requestStartAdminLogin: apiMocks.requestStartAdminLogin,
  requestStartInitialAdminSetup: apiMocks.requestStartInitialAdminSetup,
  requestStartOperatorSetup: apiMocks.requestStartOperatorSetup,
}));

const sessionResponse: AdminAuthSessionResponse = {
  requestId: '01JREQUEST0000000000000000',
  operator: {
    operatorId: '01JOPERATOR00000000000000',
    email: 'operator@example.com',
    role: 'admin',
    active: true,
  },
  sessionId: '01JSESSION0000000000000000',
  accessToken: 'operator-access-token',
  expiresAt: '2030-01-01T00:00:00.000Z',
  csrfToken: 'csrf-token',
};

const loginOptions: WWWTemplatePasskeyStartResponse = {
  requestId: '01JLOGINSTART000000000000',
  challenge: 'challenge',
  rpId: 'admin.example.com',
  userVerification: 'required',
};

const setupOptions: WWWTemplatePasskeyAddStartResponse = {
  requestId: '01JSETUPSTART000000000000',
  challenge: 'challenge',
  rpId: 'admin.example.com',
  rpName: 'Admin Console',
  user: {
    id: 'operator',
    name: 'admin@example.com',
    displayName: 'Admin Operator',
  },
  pubKeyCredParams: [{ type: 'public-key', alg: -7 }],
  residentKey: 'required',
  requireResidentKey: true,
  userVerification: 'required',
};

const attestationCredential: WWWTemplateWebAuthnAttestationCredential = {
  id: 'credential-id',
  rawId: 'credential-raw-id',
  type: 'public-key',
  response: { clientDataJSON: 'client-data', attestationObject: 'attestation-object' },
};

async function seedSession(): Promise<void> {
  // login finish の成功結果で memory-only accessToken state を作り、protected route tests の前提にする。
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

describe('Admin auth domain orchestration', () => {
  beforeEach(() => {
    // module-local session と mock call を毎回初期化し、token state の残留を検出可能にする。
    clearAdminSession();
    vi.clearAllMocks();
  });

  it('[ADMIN-AUTH-FE-S027] login start calls Admin backend auth API wrapper', async () => {
    // email の空白を取り除いたうえで、Admin API wrapper mock にだけ認証開始を委譲する。
    apiMocks.requestStartAdminLogin.mockResolvedValueOnce({
      status: 200,
      data: loginOptions,
      headers: new Headers(),
    });

    const result = await startAdminLogin(' operator@example.com ');

    expect(apiMocks.requestStartAdminLogin).toHaveBeenCalledWith({
      identifier: 'operator@example.com',
    });
    expect(result).toEqual({ requestId: loginOptions.requestId, options: loginOptions });
  });

  it('[ADMIN-AUTH-FE-S029] setup token failures are reduced to the same null result', async () => {
    // invalid / expired / consumed の詳細は domain result に出さず、UI が同一文言へ写像できる null に畳む。
    for (const status of [400, 403, 503]) {
      apiMocks.requestStartOperatorSetup.mockResolvedValueOnce({
        status,
        data: { requestId: '01JSETUPERROR0000000000', error: 'setup-unavailable' },
        headers: new Headers(),
      });

      const result = await startOperatorSetup('setup-token');

      expect(result).toBeNull();
    }
  });

  it('[ADMIN-AUTH-FE-S033] login stores accessToken but never refreshToken in browser-readable state', async () => {
    // response に余分な refreshToken 風の値が混ざっても、toSessionState が許可 field だけを取り出すことを確認する。
    apiMocks.requestFinishAdminLogin.mockResolvedValueOnce({
      status: 200,
      data: { ...sessionResponse, refreshToken: 'must-not-be-copied' },
      headers: new Headers(),
    });

    const session = await finishAdminLogin('01JLOGINREQUEST0000000000', {
      id: 'credential-id',
      rawId: 'credential-raw-id',
      type: 'public-key',
      response: {
        clientDataJSON: 'client-data',
        authenticatorData: 'auth-data',
        signature: 'signature',
      },
    });

    expect(session?.accessToken).toBe('operator-access-token');
    expect(Object.keys(getAdminSession() ?? {})).not.toContain('refreshToken');
  });

  it('[ADMIN-AUTH-FE-S034] protected route verifies the operator accessToken', async () => {
    // protected content 表示前に、memory accessToken を Admin current operator API に渡す。
    await seedSession();
    apiMocks.requestCurrentAdminOperator.mockResolvedValueOnce({
      status: 200,
      data: { requestId: '01JCURRENT00000000000000', operator: sessionResponse.operator },
      headers: new Headers(),
    });

    const result = await verifyProtectedAdminRoute();

    expect(apiMocks.requestCurrentAdminOperator).toHaveBeenCalledWith(
      expect.objectContaining({ accessToken: 'operator-access-token' })
    );
    expect(result.status).toBe('authenticated');
  });

  it('[ADMIN-AUTH-FE-S035] refresh uses HttpOnly Cookie orchestration and updates accessToken', async () => {
    // refresh domain function は request body token を受け取らず、Admin API wrapper の Cookie refresh 結果だけを反映する。
    apiMocks.requestRefreshAdminSession.mockResolvedValueOnce({
      status: 200,
      data: { ...sessionResponse, accessToken: 'refreshed-access-token' },
      headers: new Headers(),
    });

    const session = await refreshAdminSession();

    expect(apiMocks.requestRefreshAdminSession).toHaveBeenCalledWith();
    expect(session?.accessToken).toBe('refreshed-access-token');
    expect(Object.keys(getAdminSession() ?? {})).not.toContain('refreshToken');
  });

  it('[ADMIN-AUTH-FE-S036] refresh failure clears memory and hides protected state', async () => {
    // refresh 失敗時は protected route に進めず、古い session state を必ず破棄する。
    apiMocks.requestRefreshAdminSession.mockResolvedValueOnce({
      status: 401,
      data: { requestId: '01JREFRESHFAIL000000000', error: 'session-expired' },
      headers: new Headers(),
    });

    const result = await verifyProtectedAdminRoute();

    expect(result).toEqual({ status: 'unauthenticated' });
    expect(getAdminSession()).toBeNull();
  });

  it('[ADMIN-AUTH-FE-S037] session expiry reasons are not exposed by route state', async () => {
    // current operator の失敗理由が違っても、UI へは generic unauthenticated/forbidden state だけを返す。
    for (const status of [401, 403]) {
      await seedSession();
      apiMocks.requestCurrentAdminOperator.mockResolvedValueOnce({
        status,
        data: { requestId: '01JCURRENTFAIL000000000', error: 'expired' },
        headers: new Headers(),
      });

      const result = await verifyProtectedAdminRoute();

      expect(result).toEqual({ status: status === 403 ? 'forbidden' : 'unauthenticated' });
      clearAdminSession();
    }
  });

  it('[ADMIN-AUTH-FE-S038] initial setup stores accessToken memory state without refreshToken', async () => {
    // 初回 setup は `/auth/setup/*` wrapper を経由し、finish 成功後も refreshToken 平文を state に含めない。
    apiMocks.requestStartInitialAdminSetup.mockResolvedValueOnce({
      status: 200,
      data: setupOptions,
      headers: new Headers(),
    });
    apiMocks.requestFinishInitialAdminSetup.mockResolvedValueOnce({
      status: 200,
      data: sessionResponse,
      headers: new Headers(),
    });

    const start = await startInitialAdminSetup({
      email: ' admin@example.com ',
      displayName: ' Admin Operator ',
      bootstrapSecret: ' bootstrap-secret ',
    });
    expect(start.status).toBe('started');
    if (start.status !== 'started') throw new Error('initial setup did not start');

    const session = await finishInitialAdminSetup(
      {
        email: 'admin@example.com',
        displayName: 'Admin Operator',
        bootstrapSecret: 'bootstrap-secret',
      },
      start.requestId,
      attestationCredential
    );

    expect(apiMocks.requestStartInitialAdminSetup).toHaveBeenCalledWith({
      email: 'admin@example.com',
      displayName: 'Admin Operator',
      bootstrapSecret: 'bootstrap-secret',
    });
    expect(session?.accessToken).toBe('operator-access-token');
    expect(Object.keys(getAdminSession() ?? {})).not.toContain('refreshToken');
  });

  it('[ADMIN-AUTH-FE-S039/S040] initial setup start classifies unavailable form states', async () => {
    // operator 既存と bootstrap gate 無効は、setup form を閉じるための state に変換する。
    apiMocks.requestStartInitialAdminSetup
      .mockResolvedValueOnce({
        status: 409,
        data: { requestId: '01JSETUPEXISTS000000000', error: 'operator-exists' },
        headers: new Headers(),
      })
      .mockResolvedValueOnce({
        status: 403,
        data: { requestId: '01JSETUPGATEDISABLED00', error: 'bootstrap-disabled' },
        headers: new Headers(),
      });

    await expect(
      startInitialAdminSetup({
        email: 'admin@example.com',
        displayName: 'Admin Operator',
        bootstrapSecret: 'bootstrap-secret',
      })
    ).resolves.toEqual({ status: 'operator-exists' });
    await expect(
      startInitialAdminSetup({
        email: 'admin@example.com',
        displayName: 'Admin Operator',
        bootstrapSecret: 'bootstrap-secret',
      })
    ).resolves.toEqual({ status: 'bootstrap-disabled' });
  });
});
