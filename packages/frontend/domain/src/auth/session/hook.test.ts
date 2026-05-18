import { describe, expect, it, vi } from 'vitest';

import { authApi } from '@www-template/api';
import * as apiModule from '@www-template/api';

import { usePasskeyLogin } from '../passkey/hook.svelte';
import { usePasskeyManagement } from '../passkey/management/hook.svelte';

import { useAuthSession } from './hook.svelte';

function buildJwt(claims: { exp: number }): string {
  const header = btoa(JSON.stringify({ alg: 'HS256', typ: 'JWT' }));
  const payload = btoa(
    JSON.stringify({
      sub: '01ARZ3NDEKTSV4RRFFQ69G5FAV',
      sid: '01ARZ3NDEKTSV4RRFFQ69G5FAW',
      iat: claims.exp - 900,
      ...claims,
    })
  );
  return `${header}.${payload}.sig`;
}

function createSession(id: string, token: string, refresh?: string) {
  return {
    requestId: '01ARZ3NDEKTSV4RRFFQ69G5FAV',
    accountId: '01ARZ3NDEKTSV4RRFFQ69G5FAW',
    passkeyCredentialId: '01ARZ3NDEKTSV4RRFFQ69G5FAX',
    sessionId: `01ARZ3NDEKTSV4RRFFQ69G5F${id}`,
    accessToken: token,
    expiresAt: '2026-03-21T00:00:00.000Z',
    refreshToken: refresh,
  };
}

describe('useAuthSession hook', () => {
  beforeEach(() => {
    vi.restoreAllMocks();
    vi.unstubAllGlobals();
    const { actions } = useAuthSession();
    actions.clearInMemorySession();

    vi.spyOn(authApi, 'logout').mockResolvedValue({
      status: 200,
      data: { requestId: '01ARZ3NDEKTSV4RRFFQ69G5FAV', revoked: true },
      headers: new Headers(),
    } as unknown as Awaited<ReturnType<typeof authApi.logout>>);
  });

  const sidA = '01ARZ3NDEKTSV4RRFFQ69G5FA1';
  const sidB = '01ARZ3NDEKTSV4RRFFQ69G5FB2';

  it('[AUTH-FE-S027] acceptSession adds a new session', () => {
    const { data, actions } = useAuthSession();
    actions.acceptSession(createSession('A1', 'token-a'), 'no-store');
    expect(data.state.sessions).toHaveLength(1);
    expect(data.state.activeSessionId).toBe(sidA);
  });

  it('[AUTH-FE-S028] switchSession changes active token', () => {
    const { data, actions } = useAuthSession();
    actions.acceptSession(createSession('A1', 'token-a'), 'no-store');
    actions.acceptSession(createSession('B2', 'token-b'), 'no-store');
    actions.switchSession(sidA);
    expect(data.state.activeSessionId).toBe(sidA);
    expect(data.state.session?.accessToken).toBe('token-a');
  });

  it('[AUTH-FE-S030] logoutCurrentSession removes only active session', async () => {
    const { data, actions } = useAuthSession();
    actions.acceptSession(createSession('A1', 'token-a'), 'no-store');
    actions.acceptSession(createSession('B2', 'token-b'), 'no-store');
    const result = await actions.logoutCurrentSession();
    expect(data.state.sessions).toHaveLength(1);
    expect(data.state.activeSessionId).toBe(sidA);
    expect(result).toBeNull();
  });

  it('[AUTH-FE-S031] removing all sessions results in anonymous state', async () => {
    const { data, actions } = useAuthSession();
    actions.acceptSession(createSession('A1', 'token-a'), 'no-store');
    await actions.logoutCurrentSession();
    expect(data.state.phase).toBe('anonymous');
    expect(data.state.sessions).toHaveLength(0);
  });

  it('[AUTH-FE-S023] refreshActiveSession calls API when token is near expiry', async () => {
    const { data, actions } = useAuthSession();
    const exp = Math.floor(Date.now() / 1000) + 30; // 30 秒残り
    const accessToken = buildJwt({ exp });
    actions.acceptSession(
      {
        ...createSession('A1', accessToken, 'refresh-1'),
        expiresAt: new Date(exp * 1000).toISOString(),
      },
      'no-store'
    );

    const newExp = Math.floor(Date.now() / 1000) + 900;
    vi.spyOn(apiModule, 'refreshToken').mockResolvedValue({
      status: 200,
      data: { accessToken: buildJwt({ exp: newExp }), refreshToken: 'refresh-2' },
      headers: new Headers({ 'cache-control': 'no-store' }),
    } as unknown as Awaited<ReturnType<typeof apiModule.refreshToken>>);

    const result = await actions.refreshActiveSession();
    expect(apiModule.refreshToken).toHaveBeenCalledWith({ refreshToken: 'refresh-1' });
    expect(result).toBeNull();
    expect(data.state.session?.accessToken).not.toBe(accessToken);
    expect(data.state.session?.refreshToken).toBe('refresh-2');
  });

  it('[AUTH-FE-S024] refreshActiveSession is triggered for expired token', async () => {
    const { actions } = useAuthSession();
    const exp = Math.floor(Date.now() / 1000) - 60; // 期限切れ
    const accessToken = buildJwt({ exp });
    actions.acceptSession(
      {
        ...createSession('A1', accessToken, 'refresh-1'),
        expiresAt: new Date(exp * 1000).toISOString(),
      },
      'no-store'
    );

    const newExp = Math.floor(Date.now() / 1000) + 900;
    vi.spyOn(apiModule, 'refreshToken').mockResolvedValue({
      status: 200,
      data: { accessToken: buildJwt({ exp: newExp }), refreshToken: 'refresh-2' },
      headers: new Headers({ 'cache-control': 'no-store' }),
    } as unknown as Awaited<ReturnType<typeof apiModule.refreshToken>>);

    const result = await actions.refreshActiveSession();
    expect(result).toBeNull();
  });

  it('listDevices returns sessions on success', async () => {
    const { actions } = useAuthSession();
    actions.acceptSession(createSession('A1', 'token-a'), 'no-store');

    vi.spyOn(apiModule, 'listSessions').mockResolvedValue({
      status: 200,
      data: {
        requestId: '01ARZ3NDEKTSV4RRFFQ69G5FAV',
        sessions: [
          {
            sessionId: '01ARZ3NDEKTSV4RRFFQ69G5FAV',
            deviceName: 'test-device',
            loginAt: '2026-01-01T00:00:00.000Z',
            lastActiveAt: '2026-01-01T00:00:00.000Z',
            ipHash: 'abc',
            isCurrentSession: true,
          },
        ],
      },
      headers: new Headers(),
    });

    const devices = await actions.listDevices();
    expect(devices).toHaveLength(1);
    expect(devices?.[0]?.deviceName).toBe('test-device');
  });

  it('listDevices triggers refresh when token is near expiry', async () => {
    const { data, actions } = useAuthSession();
    const exp = Math.floor(Date.now() / 1000) + 30; // 30 秒残り
    const accessToken = buildJwt({ exp });
    actions.acceptSession(
      {
        ...createSession('A1', accessToken, 'refresh-1'),
        expiresAt: new Date(exp * 1000).toISOString(),
      },
      'no-store'
    );

    const newExp = Math.floor(Date.now() / 1000) + 900;
    vi.spyOn(apiModule, 'refreshToken').mockResolvedValue({
      status: 200,
      data: { accessToken: buildJwt({ exp: newExp }), refreshToken: 'refresh-2' },
      headers: new Headers({ 'cache-control': 'no-store' }),
    } as unknown as Awaited<ReturnType<typeof apiModule.refreshToken>>);

    vi.spyOn(apiModule, 'listSessions').mockResolvedValue({
      status: 200,
      data: {
        requestId: '01ARZ3NDEKTSV4RRFFQ69G5FAV',
        sessions: [
          {
            sessionId: '01ARZ3NDEKTSV4RRFFQ69G5FAV',
            deviceName: 'refreshed-device',
            loginAt: '2026-01-01T00:00:00.000Z',
            lastActiveAt: '2026-01-01T00:00:00.000Z',
            ipHash: 'abc',
            isCurrentSession: true,
          },
        ],
      },
      headers: new Headers(),
    });

    const devices = await actions.listDevices();
    expect(apiModule.refreshToken).toHaveBeenCalled();
    expect(devices).toHaveLength(1);
    expect(devices?.[0]?.deviceName).toBe('refreshed-device');
    expect(data.state.session?.refreshToken).toBe('refresh-2');
  });

  it('revokeDevice removes the target session locally on success', async () => {
    const { data, actions } = useAuthSession();
    actions.acceptSession(createSession('A1', 'token-a'), 'no-store');
    actions.acceptSession(createSession('B2', 'token-b'), 'no-store');

    vi.spyOn(apiModule, 'revokeSession').mockResolvedValue({
      status: 204,
      data: {},
      headers: new Headers(),
    });

    const result = await actions.revokeDevice(sidA);
    expect(result).toBe(true);
    expect(data.state.sessions).toHaveLength(1);
    expect(data.state.activeSessionId).toBe(sidB);
  });

  it('revokeOtherDevices keeps only active session', async () => {
    const { data, actions } = useAuthSession();
    actions.acceptSession(createSession('A1', 'token-a'), 'no-store');
    actions.acceptSession(createSession('B2', 'token-b'), 'no-store');

    vi.spyOn(apiModule, 'revokeOtherSessions').mockResolvedValue({
      status: 204,
      data: {},
      headers: new Headers(),
    });

    const result = await actions.revokeOtherDevices();
    expect(result).toBe(true);
    expect(data.state.sessions).toHaveLength(1);
    expect(data.state.activeSessionId).toBe(sidB);
  });

  it('concurrent refresh does not overwrite active session after switch', async () => {
    const { data, actions } = useAuthSession();

    // Session A: near expiry
    const expA = Math.floor(Date.now() / 1000) + 30;
    const tokenA = buildJwt({ exp: expA });
    actions.acceptSession(
      {
        ...createSession('A1', tokenA, 'refresh-a'),
        expiresAt: new Date(expA * 1000).toISOString(),
      },
      'no-store'
    );

    // Session B: not near expiry
    const expB = Math.floor(Date.now() / 1000) + 900;
    const tokenB = buildJwt({ exp: expB });
    actions.acceptSession(
      {
        ...createSession('B2', tokenB, 'refresh-b'),
        expiresAt: new Date(expB * 1000).toISOString(),
      },
      'no-store'
    );

    // Switch to A so that listDevices triggers refresh for A
    actions.switchSession(sidA);

    let resolveRefresh: (value: unknown) => void;
    const refreshPromise = new Promise((resolve) => {
      resolveRefresh = resolve;
    });
    vi.spyOn(apiModule, 'refreshToken').mockReturnValue(refreshPromise);

    vi.spyOn(apiModule, 'listSessions').mockResolvedValue({
      status: 200,
      data: {
        requestId: '01ARZ3NDEKTSV4RRFFQ69G5FAV',
        sessions: [
          {
            sessionId: sidB,
            deviceName: 'concurrent-device',
            loginAt: '2026-01-01T00:00:00.000Z',
            lastActiveAt: '2026-01-01T00:00:00.000Z',
            ipHash: 'abc',
            isCurrentSession: true,
          },
        ],
      },
      headers: new Headers(),
    });

    // Start listDevices (triggers refresh for A)
    const listPromise = actions.listDevices();

    // While refresh is in-flight, switch to B
    actions.switchSession(sidB);

    // Resolve the refresh
    const newExp = Math.floor(Date.now() / 1000) + 900;
    resolveRefresh({
      status: 200,
      data: { accessToken: buildJwt({ exp: newExp }), refreshToken: 'refresh-a2' },
      headers: new Headers({ 'cache-control': 'no-store' }),
    });

    await listPromise;

    // B should remain active
    expect(data.state.activeSessionId).toBe(sidB);
    expect(data.state.session?.accessToken).toBe(tokenB);

    // But A's token in the sessions array should be updated
    const sessionA = data.state.sessions?.find((s) => s.sessionId === sidA);
    expect(sessionA?.accessToken).not.toBe(tokenA);
    expect(sessionA?.refreshToken).toBe('refresh-a2');
  });

  it('refresh failure for non-active session removes only that session', async () => {
    const { data, actions } = useAuthSession();

    // Session A: near expiry
    const expA = Math.floor(Date.now() / 1000) + 30;
    const tokenA = buildJwt({ exp: expA });
    actions.acceptSession(
      {
        ...createSession('A1', tokenA, 'refresh-a'),
        expiresAt: new Date(expA * 1000).toISOString(),
      },
      'no-store'
    );
    const sessionA = data.state.session;
    expect(sessionA).not.toBeNull();
    const sidA = sessionA.sessionId;

    // Session B: not near expiry
    const expB = Math.floor(Date.now() / 1000) + 900;
    const tokenB = buildJwt({ exp: expB });
    actions.acceptSession(
      {
        ...createSession('B2', tokenB, 'refresh-b'),
        expiresAt: new Date(expB * 1000).toISOString(),
      },
      'no-store'
    );
    const sessionB = data.state.session;
    expect(sessionB).not.toBeNull();
    const sidB = sessionB.sessionId;

    // Switch to A (triggers refresh need for A)
    actions.switchSession(sidA);
    expect(data.state.activeSessionId).toBe(sidA);

    let rejectRefresh: (reason: unknown) => void;
    const refreshPromise = new Promise((_resolve, reject) => {
      rejectRefresh = reject;
    });
    vi.spyOn(apiModule, 'refreshToken').mockReturnValue(refreshPromise);

    // Start listDevices (triggers refresh for A, which is now active)
    const listPromise = actions.listDevices();

    // While refresh is in-flight, switch to B
    actions.switchSession(sidB);

    // Fail the refresh
    rejectRefresh(new Error('Network Error'));

    await listPromise;

    // B should remain active
    expect(data.state.activeSessionId).toBe(sidB);
    expect(data.state.session?.accessToken).toBe(tokenB);
    expect(data.state.phase).toBe('authenticated');

    // A should be removed from sessions array
    expect(data.state.sessions?.some((s) => s.sessionId === sidA)).toBe(false);
    expect(data.state.sessions?.some((s) => s.sessionId === sidB)).toBe(true);
  });

  it('logout clears session when API fails with network error', async () => {
    const { data, actions } = useAuthSession();
    const exp = Math.floor(Date.now() / 1000) + 30;
    const accessToken = buildJwt({ exp });
    actions.acceptSession(
      {
        ...createSession('A1', accessToken, 'refresh-1'),
        expiresAt: new Date(exp * 1000).toISOString(),
      },
      'no-store'
    );

    vi.spyOn(apiModule, 'refreshToken').mockRejectedValue(new Error('Network Error'));
    vi.spyOn(authApi, 'logout').mockRejectedValue(new Error('Network Error'));

    const result = await actions.logoutCurrentSession();
    // fail-safe: エラー時もアクティブセッションを除去し、in-memory credential を残さない
    expect(data.state.phase).toBe('anonymous');
    expect(data.state.session).toBeNull();
    expect(result).toBe('/login');
  });

  it('[AUTH-FE-S042] refreshActiveSession routes suspended active account to guidance', async () => {
    const { data, actions } = useAuthSession();
    actions.acceptSession(createSession('A1', 'token-a', 'refresh-a'), 'no-store');

    vi.spyOn(apiModule, 'refreshToken').mockResolvedValue({
      status: 403,
      data: { requestId: '01ARZ3NDEKTSV4RRFFQ69G5FAV', error: 'account-suspended' },
      headers: new Headers({ 'cache-control': 'no-store' }),
    } as unknown as Awaited<ReturnType<typeof apiModule.refreshToken>>);

    const result = await actions.refreshActiveSession();

    expect(result).toBe('/account-suspended');
    expect(data.state.phase).toBe('account-suspended');
    expect(data.state.session).toBeNull();
    expect(data.state.sessions).toHaveLength(0);
  });

  it('[AUTH-FE-S044] suspended refresh removes only the target account session', async () => {
    const { data, actions } = useAuthSession();
    const expA = Math.floor(Date.now() / 1000) + 30;
    const expB = Math.floor(Date.now() / 1000) + 900;
    actions.acceptSession(
      {
        ...createSession('A1', buildJwt({ exp: expA }), 'refresh-a'),
        expiresAt: new Date(expA * 1000).toISOString(),
      },
      'no-store'
    );
    actions.acceptSession(
      {
        ...createSession('B2', buildJwt({ exp: expB }), 'refresh-b'),
        expiresAt: new Date(expB * 1000).toISOString(),
      },
      'no-store'
    );
    actions.switchSession(sidA);

    let resolveRefresh: (value: unknown) => void;
    const refreshPromise = new Promise((resolve) => {
      resolveRefresh = resolve;
    });
    vi.spyOn(apiModule, 'refreshToken').mockReturnValue(refreshPromise);
    vi.spyOn(apiModule, 'listSessions').mockResolvedValue({
      status: 200,
      data: { requestId: '01ARZ3NDEKTSV4RRFFQ69G5FAV', sessions: [] },
      headers: new Headers(),
    });

    const listPromise = actions.listDevices();
    actions.switchSession(sidB);
    resolveRefresh({
      status: 403,
      data: { requestId: '01ARZ3NDEKTSV4RRFFQ69G5FAV', error: 'account-suspended' },
      headers: new Headers({ 'cache-control': 'no-store' }),
    });

    await listPromise;

    expect(data.state.phase).toBe('authenticated');
    expect(data.state.activeSessionId).toBe(sidB);
    expect(data.state.sessions?.map((session) => session.sessionId)).toEqual([sidB]);
  });

  it('[AUTH-FE-S042] protected API account-suspended clears current session guidance', async () => {
    const { data, actions } = useAuthSession();
    actions.acceptSession(createSession('A1', 'token-a', 'refresh-a'), 'no-store');

    vi.spyOn(apiModule, 'listSessions').mockResolvedValue({
      status: 403,
      data: { requestId: '01ARZ3NDEKTSV4RRFFQ69G5FAV', error: 'account-suspended' },
      headers: new Headers({ 'cache-control': 'no-store' }),
    });

    const devices = await actions.listDevices();

    expect(devices).toBeNull();
    expect(data.state.phase).toBe('account-suspended');
    expect(data.state.routeIntent).toBe('/account-suspended');
    expect(data.state.session).toBeNull();
  });

  it('[AUTH-FE-S041] suspended passkey login stores no session and returns guidance intent', async () => {
    const { data: sessionData, actions: sessionActions } = useAuthSession();
    sessionActions.clearInMemorySession();
    const passkey = usePasskeyLogin();

    class TestPublicKeyCredential {
      id = 'credential-id';
      rawId = new Uint8Array([1, 2, 3]).buffer;
      type = 'public-key';
      authenticatorAttachment = 'platform';
      response = {
        clientDataJSON: new Uint8Array([4]).buffer,
        authenticatorData: new Uint8Array([5]).buffer,
        signature: new Uint8Array([6]).buffer,
        userHandle: new Uint8Array([7]).buffer,
      };
    }

    vi.stubGlobal('PublicKeyCredential', TestPublicKeyCredential);
    vi.stubGlobal('navigator', {
      credentials: {
        get: vi.fn().mockResolvedValue(new TestPublicKeyCredential()),
      },
    });
    vi.spyOn(authApi, 'startPasskeyAuthentication').mockResolvedValue({
      status: 200,
      data: {
        requestId: '01ARZ3NDEKTSV4RRFFQ69G5FAV',
        challenge: 'YQ',
        rpId: 'localhost',
        timeout: 60000,
        allowCredentials: [],
        userVerification: 'required',
      },
      headers: new Headers({ 'cache-control': 'no-store' }),
    } as unknown as Awaited<ReturnType<typeof authApi.startPasskeyAuthentication>>);
    vi.spyOn(authApi, 'finishPasskeyAuthentication').mockResolvedValue({
      status: 403,
      data: { requestId: '01ARZ3NDEKTSV4RRFFQ69G5FAV', error: 'account-suspended' },
      headers: new Headers({ 'cache-control': 'no-store' }),
    } as unknown as Awaited<ReturnType<typeof authApi.finishPasskeyAuthentication>>);

    const intent = await passkey.actions.signInWithPasskey();

    expect(intent).toBe('/account-suspended');
    expect(passkey.data.state.lastSession).toBeNull();
    expect(sessionData.state.session).toBeNull();
    expect(sessionData.state.phase).toBe('account-suspended');
  });

  it('[AUTH-FE-S042] passkey management protected API account-suspended clears session', async () => {
    const { data: sessionData, actions: sessionActions } = useAuthSession();
    sessionActions.acceptSession(createSession('A1', 'token-a', 'refresh-a'), 'no-store');
    const passkeys = usePasskeyManagement();

    vi.spyOn(authApi, 'listPasskeys').mockResolvedValue({
      status: 403,
      data: { requestId: '01ARZ3NDEKTSV4RRFFQ69G5FAV', error: 'account-suspended' },
      headers: new Headers({ 'cache-control': 'no-store' }),
    } as unknown as Awaited<ReturnType<typeof authApi.listPasskeys>>);

    await passkeys.actions.listPasskeys();

    expect(sessionData.state.phase).toBe('account-suspended');
    expect(sessionData.state.session).toBeNull();
    expect(passkeys.data.error).toBeNull();
  });

  it('[AUTH-FE-S042] passkey device-link 403 account-suspended uses auth failure not form error', async () => {
    const { data: sessionData, actions: sessionActions } = useAuthSession();
    sessionActions.acceptSession(createSession('A1', 'token-a', 'refresh-a'), 'no-store');
    const passkeys = usePasskeyManagement();

    vi.spyOn(authApi, 'sendDeviceLink').mockResolvedValue({
      status: 403,
      data: { requestId: '01ARZ3NDEKTSV4RRFFQ69G5FAV', error: 'account-suspended' },
      headers: new Headers({ 'cache-control': 'no-store' }),
    } as unknown as Awaited<ReturnType<typeof authApi.sendDeviceLink>>);

    const issued = await passkeys.actions.sendDeviceLink('reauth-session');

    expect(issued).toBe(false);
    expect(sessionData.state.phase).toBe('account-suspended');
    expect(passkeys.data.error).toBeNull();
  });
});
