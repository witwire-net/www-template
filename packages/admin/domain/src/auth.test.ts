import { beforeEach, describe, expect, it, vi } from 'vitest';

import type {
  AdminOperatorSessionResponse,
  WWWTemplatePasskeyAddStartResponse,
  WWWTemplatePasskeyStartResponse,
  WWWTemplateWebAuthnAttestationCredential,
} from '@www-template/admin-api';

import {
  clearAdminSession,
  finishAdminLogin,
  finishInitialAdminSetup,
  finishOperatorSetup,
  getAdminSession,
  logoutAdminSession,
  refreshAdminSession,
  startAdminLogin,
  startInitialAdminSetup,
  startOperatorSetup,
  verifyProtectedAdminRoute,
} from './auth';
import {
  configureAdminContextIndexStorage,
  getAdminContextIndexStorageKey,
  readAdminContextIndex,
} from './context_index';

const apiMocks = vi.hoisted(() => ({
  requestCurrentAdminOperator: vi.fn(),
  requestFinishAdminLogin: vi.fn(),
  requestFinishInitialAdminSetup: vi.fn(),
  requestFinishOperatorSetup: vi.fn(),
  requestLogoutAdminOperator: vi.fn(),
  requestRefreshAdminSession: vi.fn(),
  requestStartAdminLogin: vi.fn(),
  requestStartInitialAdminSetup: vi.fn(),
  requestStartOperatorSetup: vi.fn(),
}));

vi.mock('@www-template/admin-api', () => ({
  requestCurrentAdminOperator: apiMocks.requestCurrentAdminOperator,
  requestFinishAdminLogin: apiMocks.requestFinishAdminLogin,
  requestFinishInitialAdminSetup: apiMocks.requestFinishInitialAdminSetup,
  requestFinishOperatorSetup: apiMocks.requestFinishOperatorSetup,
  requestLogoutAdminOperator: apiMocks.requestLogoutAdminOperator,
  requestRefreshAdminSession: apiMocks.requestRefreshAdminSession,
  requestStartAdminLogin: apiMocks.requestStartAdminLogin,
  requestStartInitialAdminSetup: apiMocks.requestStartInitialAdminSetup,
  requestStartOperatorSetup: apiMocks.requestStartOperatorSetup,
}));

const sessionResponse: AdminOperatorSessionResponse = {
  requestId: '01JREQUEST0000000000000000',
  credentialMode: 'cookie',
  operator: {
    operatorId: '01JOPERATOR00000000000000',
    email: 'operator@example.com',
    role: 'admin',
    active: true,
  },
  sessionId: '01JSESSION0000000000000000',
  authContextId: '01JSESSION0000000000000000',
  accessToken: 'operator-access-token',
  expiresAt: '2030-01-01T00:00:00.000Z',
  contextIndexUpdateHints: [],
  clearCookieCommands: [],
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

function installLocalStorageMock(): void {
  // Admin domain tests は node environment で走るため、origin-local context index 用の最小 localStorage を差し替える。
  const values = new Map<string, string>();
  const storage = {
    getItem: (key: string) => values.get(key) ?? null,
    setItem: (key: string, value: string) => {
      values.set(key, value);
    },
    removeItem: (key: string) => {
      values.delete(key);
    },
    clear: () => {
      values.clear();
    },
  };
  vi.stubGlobal('localStorage', storage);
  configureAdminContextIndexStorage(storage);
}

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
    installLocalStorageMock();
    clearAdminSession();
    localStorage.removeItem(getAdminContextIndexStorageKey());
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
    await seedSession();

    const session = await refreshAdminSession();

    expect(apiMocks.requestRefreshAdminSession).toHaveBeenCalledWith('01JSESSION0000000000000000');
    expect(session?.accessToken).toBe('refreshed-access-token');
    expect(Object.keys(getAdminSession() ?? {})).not.toContain('refreshToken');
  });

  it('[ADMIN-AUTH-FE-S036] refresh failure clears memory and hides protected state', async () => {
    // refresh 失敗時は protected route に進めず、古い session state を必ず破棄する。
    await seedSession();
    apiMocks.requestRefreshAdminSession.mockResolvedValueOnce({
      status: 401,
      data: { requestId: '01JREFRESHFAIL000000000', error: 'session-expired' },
      headers: new Headers(),
    });

    const result = await refreshAdminSession();

    expect(apiMocks.requestRefreshAdminSession).toHaveBeenCalledWith('01JSESSION0000000000000000');
    expect(result).toBeNull();
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

  it('[ADMIN-AUTH-FE-S041/S042] login writes an Admin-only non-secret context index', async () => {
    // login 成功時は reload bootstrap 用の非 secret hint だけを Admin origin-local key に保存する。
    await seedSession();

    const raw = localStorage.getItem(getAdminContextIndexStorageKey()) ?? '';
    const index = readAdminContextIndex();

    expect(index?.surface).toBe('admin');
    expect(index?.activeAuthContextId).toBe('01JSESSION0000000000000000');
    expect(raw).toContain('operator@example.com');
    expect(raw).toContain('"roleHint":"admin"');
    expect(raw).not.toContain('operator-access-token');
    expect(raw).not.toContain('refreshToken');
    expect(raw).not.toContain('Cookie');
    expect(localStorage.getItem('www-template:product:context-index')).toBeNull();
  });

  it('[ADMIN-AUTH-FE-S041] tampered context index cannot restore an authenticated session', async () => {
    // tamper された index は refresh 対象として採用せず、storage から削除する。
    localStorage.setItem(
      getAdminContextIndexStorageKey(),
      JSON.stringify({ version: 1, surface: 'admin', activeAuthContextId: 'tampered', entries: [] })
    );

    const session = await refreshAdminSession();

    expect(session).toBeNull();
    expect(apiMocks.requestRefreshAdminSession).not.toHaveBeenCalled();
    expect(localStorage.getItem(getAdminContextIndexStorageKey())).toBeNull();
  });

  it('[ADMIN-AUTH-FE-S041] bootstrap fails closed when active context is missing', async () => {
    // activeAuthContextId が無い index は先頭 entry を暗黙採用せず、認証済み state を作らない。
    localStorage.setItem(
      getAdminContextIndexStorageKey(),
      JSON.stringify({
        version: 1,
        surface: 'admin',
        activeAuthContextId: null,
        entries: [
          {
            authContextId: '01JSESSION0000000000000000',
            operatorSessionId: '01JSESSION0000000000000000',
            displayHint: 'operator@example.com',
            roleHint: 'admin',
            lastSeenAt: '2026-03-21T00:00:00.000Z',
            expiresHintAt: '2030-01-01T00:00:00.000Z',
          },
        ],
      })
    );

    const session = await refreshAdminSession();

    expect(session).toBeNull();
    expect(apiMocks.requestRefreshAdminSession).not.toHaveBeenCalled();
  });

  it('[ADMIN-AUTH-FE-S041] bootstrap refresh rehydrates memory only after server success', async () => {
    // memory が消えた後は index entry だけでは authenticated とせず、context refresh 成功後にだけ session を復元する。
    await seedSession();
    clearAdminSession();
    apiMocks.requestRefreshAdminSession.mockResolvedValueOnce({
      status: 200,
      data: { ...sessionResponse, accessToken: 'bootstrapped-access-token' },
      headers: new Headers(),
    });

    const session = await refreshAdminSession();

    expect(apiMocks.requestRefreshAdminSession).toHaveBeenCalledWith('01JSESSION0000000000000000');
    expect(session?.accessToken).toBe('bootstrapped-access-token');
    expect(JSON.stringify(readAdminContextIndex())).not.toContain('bootstrapped-access-token');
  });

  it('[ADMIN-AUTH-FE-S043] refresh failure removes the target context index entry', async () => {
    // refresh failure は memory session と対応する context index entry を同時に cleanup する。
    await seedSession();
    apiMocks.requestRefreshAdminSession.mockResolvedValueOnce({
      status: 401,
      data: { requestId: '01JREFRESHFAIL000000000', error: 'session-expired' },
      headers: new Headers(),
    });

    await expect(refreshAdminSession()).resolves.toBeNull();

    expect(readAdminContextIndex()?.entries).toEqual([]);
  });

  it('[ADMIN-AUTH-FE-S043] logout follows backend cleanup hints for the active context', async () => {
    // logout 成功時は backend が返す contextIndexUpdateHints に従い、古い context を再採用しない。
    await seedSession();
    apiMocks.requestLogoutAdminOperator.mockResolvedValueOnce({
      status: 200,
      data: {
        requestId: '01JLOGOUT00000000000000',
        revoked: true,
        clearCookieCommands: [],
        contextIndexUpdateHints: [
          { action: 'remove', authContextId: '01JSESSION0000000000000000' },
        ],
      },
      headers: new Headers(),
    });

    await expect(logoutAdminSession()).resolves.toBe(true);

    expect(readAdminContextIndex()?.entries).toEqual([]);
    expect(getAdminSession()).toBeNull();
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

  it('[ADMIN-AUTH-FE-S039/S040] operator setup finish stores only the updated session model', async () => {
    // operator setup 完了時は setupToken を API 入力にだけ使い、memory session には operator/accessToken metadata だけを保存する。
    apiMocks.requestFinishOperatorSetup.mockResolvedValueOnce({
      status: 200,
      data: {
        ...sessionResponse,
        refreshToken: 'must-not-be-copied',
        cookieValue: 'must-not-be-copied',
      },
      headers: new Headers(),
    });

    const session = await finishOperatorSetup(
      ' setup-token ',
      '01JOPSETUPREQUEST00000000',
      attestationCredential
    );
    const storedKeys = Object.keys(getAdminSession() ?? {});
    const storedText = JSON.stringify(getAdminSession());

    expect(apiMocks.requestFinishOperatorSetup).toHaveBeenCalledWith({
      setupToken: 'setup-token',
      requestId: '01JOPSETUPREQUEST00000000',
      credentialMode: 'cookie',
      credential: attestationCredential,
    });
    expect(session?.accessToken).toBe('operator-access-token');
    expect(storedKeys).not.toContain('setupToken');
    expect(storedKeys).not.toContain('refreshToken');
    expect(storedKeys).not.toContain('cookieValue');
    expect(storedText).not.toContain('setup-token');
    expect(storedText).not.toContain('must-not-be-copied');
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
