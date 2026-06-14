import { execFile } from 'node:child_process';
import { mkdir, rm, writeFile } from 'node:fs/promises';
import path from 'node:path';
import { promisify } from 'node:util';

import { describe, expect, it, vi } from 'vitest';

import { authApi } from '@www-template/api';
import * as apiModule from '@www-template/api';

import { usePasskeyLogin } from '../passkey/hook.svelte';
import { usePasskeyManagement } from '../passkey/management/hook.svelte';

import { useAuthSession } from './hook.svelte';

// ESM の import.meta.url が file: 以外のスキーム（data: 等）で読み込まれるテスト環境でも
// 動作するよう、Node.js 20.11+ で導入された import.meta.dirname を使用する。
const sourceDir = import.meta.dirname;
const repoRoot = path.resolve(sourceDir, '..', '..', '..', '..', '..', '..');
const execFileAsync = promisify(execFile);

interface LintMessage {
  ruleId: string | null;
  message: string;
}

async function lintText(filePath: string, source: string): Promise<LintMessage[]> {
  // TypeScript project service が実在ファイルを必要とするため、repo 配下に一時 fixture を作って lint 入力にする。
  const fullPath = path.join(repoRoot, filePath);
  // fixture の親 directory を作成し、テスト終了時に対象ファイルだけを削除できるようにする。
  await mkdir(path.dirname(fullPath), { recursive: true });
  // 期待する import 境界違反だけを含む最小 source を書き込み、他の rule による誤検出を避ける。
  await writeFile(fullPath, source);

  // ESLint CLI を JSON 出力で実行し、ruleId 単位で境界 guardrail の発火を検証する。
  const eslintArgs = ['exec', 'eslint', '--format', 'json', fullPath];
  let stdout = '';
  try {
    const result = await execFileAsync('pnpm', eslintArgs, {
      cwd: repoRoot,
      maxBuffer: 10 * 1024 * 1024,
    });
    stdout = result.stdout;
  } catch (error) {
    // ESLint は違反検出時に non-zero で終了するため、stdout の JSON を成功経路と同じ形で取り出す。
    const lintError = error as { stdout?: string | Buffer };
    stdout =
      typeof lintError.stdout === 'string'
        ? lintError.stdout
        : (lintError.stdout?.toString() ?? '');
    if (stdout === '') {
      throw error;
    }
  } finally {
    // 一時 fixture を必ず削除し、lint contract test が working tree に成果物を残さないようにする。
    await rm(fullPath, { force: true });
  }

  // ESLint JSON の先頭 file result だけを取り出し、呼び出し側で期待 ruleId を検証する。
  const parsed = JSON.parse(stdout) as { filePath: string; messages: LintMessage[] }[];
  return parsed.at(0)?.messages ?? [];
}

function buildJwt(claims: { exp: number; accountId?: string; sessionId?: string }): string {
  const header = btoa(JSON.stringify({ alg: 'HS256', typ: 'JWT' }));
  const payload = btoa(
    JSON.stringify({
      sub: claims.accountId ?? '01ARZ3NDEKTSV4RRFFQ69G5FAW',
      sid: claims.sessionId ?? '01ARZ3NDEKTSV4RRFFQ69G5FA1',
      iat: claims.exp - 900,
      exp: claims.exp,
    })
  );
  return `${header}.${payload}.sig`;
}

function createSession(id: string, token: string, accountId?: string) {
  return {
    requestId: '01ARZ3NDEKTSV4RRFFQ69G5FAV',
    authContextId: `01ARZ3NDEKTSV4RRFFQ69G5F${id}`,
    accountId: accountId ?? '01ARZ3NDEKTSV4RRFFQ69G5FAW',
    passkeyCredentialId: '01ARZ3NDEKTSV4RRFFQ69G5FAX',
    sessionId: `01ARZ3NDEKTSV4RRFFQ69G5F${id}`,
    accessToken: token,
    expiresAt: '2026-03-21T00:00:00.000Z',
  };
}

describe('Frontend SDK ESLint boundary contracts', () => {
  it('[API-CONTRACT-BE-S009] packages/frontend から Admin SDK を import すると lint エラーになる', async () => {
    // Product frontend 配下に一時 fixture を置き、Admin SDK への相対 import を境界違反として検出する。
    // fixture には export-tsdoc/require-export-tsdoc を満たす TSDoc コメントを付与し、
    // sdk-package-boundary/no-cross-sdk-imports 以外の lint 失敗を出さないようにする。
    const messages = await lintText(
      'packages/frontend/domain/src/auth/session/lint-frontend-admin-sdk-import.ts',
      `/**
 * Admin SDK インポート境界違反検出用 fixture。
 */
import type { AdminAccountSummary } from '../../../../../admin/api/src/generated/client';

/**
 * Admin SDK からの型漏洩テスト用 fixture。
 */
export type LeakedAdminSdk = AdminAccountSummary;`
    );
    // SDK package boundary 専用 rule が発火することを確認し、別 rule だけで落ちる偶然の成功を避ける。
    expect(
      messages.find((message) => message.ruleId === 'sdk-package-boundary/no-cross-sdk-imports')
    ).toBeDefined();
  }, 20000);
});

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
  const accountB = '01ARZ3NDEKTSV4RRFFQ69G5FB1';

  it('[AUTH-FE-S027] acceptSession adds a new session', () => {
    const { data, actions } = useAuthSession();
    actions.acceptSession(createSession('A1', 'token-a'), 'no-store');
    expect(data.state.sessions).toHaveLength(1);
    expect(data.state.activeSessionId).toBe(sidA);
  });

  it('[AUTH-FE-S046] refreshToken is not stored in browser-readable session state or storage', () => {
    const { data, actions } = useAuthSession();
    const setItemSpy = vi.spyOn(Storage.prototype, 'setItem');
    const readableRefreshSession = {
      ...createSession('A1', 'token-a'),
      refreshToken: 'browser-readable-refresh-token',
    };

    actions.acceptSession(readableRefreshSession, 'no-store');

    expect('refreshToken' in (data.state.session ?? {})).toBe(false);
    expect(JSON.stringify(data.state.sessions)).not.toContain('browser-readable-refresh-token');

    // context index は localStorage に書き込むが、token/secret は含まない
    const contextIndexCall = setItemSpy.mock.calls.find(
      (call) => call[0] === 'www-template:product:context-index'
    );
    if (contextIndexCall != null) {
      const indexValue = contextIndexCall[1];
      expect(indexValue).not.toContain('browser-readable-refresh-token');
      expect(indexValue).not.toContain('token-a');
    }

    setItemSpy.mockRestore();
  });

  it('[AUTH-FE-S046] secret leakage test: refreshToken and Cookie values do not leak to sessionStorage, console, or URL', () => {
    const { actions } = useAuthSession();
    const sessionStorageSpy = vi.spyOn(Storage.prototype, 'setItem');
    const consoleLogSpy = vi.spyOn(console, 'log').mockImplementation(() => undefined);
    const consoleErrorSpy = vi.spyOn(console, 'error').mockImplementation(() => undefined);
    const consoleWarnSpy = vi.spyOn(console, 'warn').mockImplementation(() => undefined);

    const secretRefreshToken = 'super-secret-refresh-token-123';
    const secretCookieValue = 'super-secret-cookie-value-456';
    const sessionWithSecrets = {
      ...createSession('A1', 'access-token-789'),
      refreshToken: secretRefreshToken,
      cookieValue: secretCookieValue,
    };

    actions.acceptSession(sessionWithSecrets, 'no-store');

    // sessionStorage に secret が書き込まれていないことを確認
    for (const call of sessionStorageSpy.mock.calls) {
      const value = call[1];
      expect(value).not.toContain(secretRefreshToken);
      expect(value).not.toContain(secretCookieValue);
      expect(value).not.toContain('access-token-789');
    }

    // console 出力に secret が含まれていないことを確認
    for (const call of consoleLogSpy.mock.calls) {
      const message = JSON.stringify(call);
      expect(message).not.toContain(secretRefreshToken);
      expect(message).not.toContain(secretCookieValue);
    }
    for (const call of consoleErrorSpy.mock.calls) {
      const message = JSON.stringify(call);
      expect(message).not.toContain(secretRefreshToken);
      expect(message).not.toContain(secretCookieValue);
    }
    for (const call of consoleWarnSpy.mock.calls) {
      const message = JSON.stringify(call);
      expect(message).not.toContain(secretRefreshToken);
      expect(message).not.toContain(secretCookieValue);
    }

    sessionStorageSpy.mockRestore();
    consoleLogSpy.mockRestore();
    consoleErrorSpy.mockRestore();
    consoleWarnSpy.mockRestore();
  });

  it('[AUTH-FE-S048] login adds an accessToken session without a refreshToken field', () => {
    const { data, actions } = useAuthSession();

    actions.acceptSession(createSession('A1', 'login-access-token'), 'no-store');

    expect(data.state.phase).toBe('authenticated');
    expect(data.state.session?.accessToken).toBe('login-access-token');
    expect('refreshToken' in (data.state.session ?? {})).toBe(false);
  });

  it('[AUTH-FE-S055] refresh uses same-origin credentials without manually selecting Cookie headers', async () => {
    const { actions } = useAuthSession();
    const exp = Math.floor(Date.now() / 1000) + 30;
    const accessToken = buildJwt({ exp });
    actions.acceptSession(
      {
        ...createSession('A1', accessToken),
        expiresAt: new Date(exp * 1000).toISOString(),
      },
      'no-store'
    );

    const refreshSpy = vi.spyOn(apiModule, 'refreshToken').mockResolvedValue({
      status: 200,
      data: { accessToken: buildJwt({ exp: exp + 900 }) },
      headers: new Headers({ 'cache-control': 'no-store' }),
    } as unknown as Awaited<ReturnType<typeof apiModule.refreshToken>>);

    await actions.refreshActiveSession();

    // refreshToken は authContextId を path parameter に使い、
    // credentials: 'same-origin' で Cookie を browser に委ねる
    expect(refreshSpy).toHaveBeenCalledWith(
      expect.any(String),
      undefined,
      expect.objectContaining({ credentials: 'same-origin' })
    );
    // 手動で Cookie header を組み立てていないことを確認
    const calls = refreshSpy.mock.calls;
    for (const call of calls) {
      const options = call[2];
      if (options?.headers != null) {
        const headers = new Headers(options.headers);
        expect(headers.has('Cookie')).toBe(false);
      }
    }
  });

  it('[AUTH-FE-S057] protected API uses Authorization Bearer only and no X-Auth-Context-Id', () => {
    const { actions } = useAuthSession();
    actions.acceptSession(createSession('A1', 'token-a'), 'no-store');

    const headers = actions.createAuthorizationHeaders();

    expect(headers).toEqual({ Authorization: 'Bearer token-a' });
    expect(headers).not.toHaveProperty('X-Auth-Context-Id');
  });

  it('[AUTH-FE-S060] logout response clear-cookie command syncs context index and memory state', async () => {
    const { data, actions } = useAuthSession();
    actions.acceptSession(createSession('A1', 'token-a'), 'no-store');
    actions.acceptSession(createSession('B2', 'token-b', accountB), 'no-store');

    // logout response に contextIndexUpdateHints を含める
    vi.spyOn(authApi, 'logout').mockResolvedValue({
      status: 200,
      data: {
        requestId: '01ARZ3NDEKTSV4RRFFQ69G5FAV',
        revoked: true,
        clearCookieCommands: [],
        contextIndexUpdateHints: [
          { action: 'remove', authContextId: createSession('A1', '').authContextId },
        ],
      },
      headers: new Headers(),
    } as unknown as Awaited<ReturnType<typeof authApi.logout>>);

    // localStorage に context index を事前に書き込む
    const preIndex = JSON.stringify({
      version: 1,
      surface: 'product',
      activeAuthContextId: createSession('B2', '', accountB).authContextId,
      entries: [
        {
          authContextId: createSession('A1', '').authContextId,
          sessionId: sidA,
          identityKind: 'account',
          lastSeenAt: '2026-03-21T00:00:00.000Z',
          expiresHintAt: '2026-03-21T01:00:00.000Z',
        },
        {
          authContextId: createSession('B2', '', accountB).authContextId,
          sessionId: sidB,
          identityKind: 'account',
          lastSeenAt: '2026-03-21T00:00:00.000Z',
          expiresHintAt: '2026-03-21T01:00:00.000Z',
        },
      ],
    });
    localStorage.setItem('www-template:product:context-index', preIndex);

    await actions.logoutCurrentSession();

    // memory state: active session (B) が削除され、account A が残る
    expect(data.state.sessions).toHaveLength(1);
    expect(data.state.sessions?.[0]?.sessionId).toBe(sidA);

    // context index: server hint に従い account A の entry が削除される
    const raw = localStorage.getItem('www-template:product:context-index') ?? '';
    const postIndex = JSON.parse(raw);
    expect(postIndex.entries).toHaveLength(1);
    expect(postIndex.entries[0].authContextId).toBe(
      createSession('B2', '', accountB).authContextId
    );
  });

  it('[AUTH-FE-S028] switchSession changes active token', () => {
    const { data, actions } = useAuthSession();
    actions.acceptSession(createSession('A1', 'token-a'), 'no-store');
    actions.acceptSession(createSession('B2', 'token-b', accountB), 'no-store');
    actions.switchSession(sidA);
    expect(data.state.activeSessionId).toBe(sidA);
    expect(data.state.session?.accessToken).toBe('token-a');
  });

  it('[AUTH-FE-S049] account switch changes the bearer accessToken source', () => {
    const { actions } = useAuthSession();
    actions.acceptSession(createSession('A1', 'account-a-access-token'), 'no-store');
    actions.acceptSession(createSession('B2', 'account-b-access-token', accountB), 'no-store');

    actions.switchSession(sidA);

    expect(actions.createAuthorizationHeaders()).toEqual({
      Authorization: 'Bearer account-a-access-token',
    });
  });

  it('[AUTH-FE-S050] logoutCurrentSession asks the server to revoke the target Cookie session', async () => {
    const { data, actions } = useAuthSession();
    actions.acceptSession(createSession('A1', 'token-a'), 'no-store');
    actions.acceptSession(createSession('B2', 'token-b', accountB), 'no-store');
    const result = await actions.logoutCurrentSession();
    expect(authApi.logout).toHaveBeenCalledWith(
      expect.objectContaining({
        credentials: 'same-origin',
        headers: { Authorization: 'Bearer token-b' },
      })
    );
    expect(data.state.sessions).toHaveLength(1);
    expect(data.state.activeSessionId).toBe(sidA);
    expect(result).toBeNull();
  });

  it('logout keeps the existing bearer token when Cookie refresh returns another session token', async () => {
    const { actions } = useAuthSession();
    const exp = Math.floor(Date.now() / 1000) + 30;
    const activeToken = buildJwt({ exp, sessionId: sidB });
    actions.acceptSession(createSession('B2', activeToken, accountB), 'no-store');
    vi.spyOn(apiModule, 'refreshToken').mockResolvedValue({
      status: 200,
      data: { accessToken: buildJwt({ exp: exp + 900, sessionId: sidA }) },
      headers: new Headers({ 'cache-control': 'no-store' }),
    } as unknown as Awaited<ReturnType<typeof apiModule.refreshToken>>);

    await actions.logoutCurrentSession();

    expect(apiModule.refreshToken).toHaveBeenCalledWith(
      expect.any(String),
      undefined,
      expect.objectContaining({ credentials: 'same-origin' })
    );
    expect(authApi.logout).toHaveBeenCalledWith(
      expect.objectContaining({
        credentials: 'same-origin',
        headers: { Authorization: `Bearer ${activeToken}` },
      })
    );
  });

  it('[AUTH-FE-S031] removing all sessions results in anonymous state', async () => {
    const { data, actions } = useAuthSession();
    actions.acceptSession(createSession('A1', 'token-a'), 'no-store');
    await actions.logoutCurrentSession();
    expect(data.state.phase).toBe('anonymous');
    expect(data.state.sessions).toHaveLength(0);
  });

  it('[AUTH-FE-S045] refreshActiveSession uses same-origin Cookie when token is near expiry', async () => {
    const { data, actions } = useAuthSession();
    const exp = Math.floor(Date.now() / 1000) + 30; // 30 秒残り
    const accessToken = buildJwt({ exp });
    actions.acceptSession(
      {
        ...createSession('A1', accessToken),
        expiresAt: new Date(exp * 1000).toISOString(),
      },
      'no-store'
    );

    const newExp = Math.floor(Date.now() / 1000) + 900;
    vi.spyOn(apiModule, 'refreshToken').mockResolvedValue({
      status: 200,
      data: { accessToken: buildJwt({ exp: newExp }) },
      headers: new Headers({ 'cache-control': 'no-store' }),
    } as unknown as Awaited<ReturnType<typeof apiModule.refreshToken>>);

    const result = await actions.refreshActiveSession();
    expect(apiModule.refreshToken).toHaveBeenCalledWith(
      expect.any(String),
      undefined,
      expect.objectContaining({ credentials: 'same-origin' })
    );
    expect(result).toBeNull();
    expect(data.state.session?.accessToken).not.toBe(accessToken);
  });

  it('refreshActiveSession fail-closes when Cookie refresh returns another session token', async () => {
    const { data, actions } = useAuthSession();
    const exp = Math.floor(Date.now() / 1000) + 30;
    actions.acceptSession(
      {
        ...createSession('A1', buildJwt({ exp })),
        expiresAt: new Date(exp * 1000).toISOString(),
      },
      'no-store'
    );
    vi.spyOn(apiModule, 'refreshToken').mockResolvedValue({
      status: 200,
      data: { accessToken: buildJwt({ exp: exp + 900, sessionId: sidB }) },
      headers: new Headers({ 'cache-control': 'no-store' }),
    } as unknown as Awaited<ReturnType<typeof apiModule.refreshToken>>);

    const result = await actions.refreshActiveSession();

    expect(result).toBe('/session-expired');
    expect(data.state.phase).toBe('session-expired');
    expect(data.state.session).toBeNull();
  });

  it('[AUTH-FE-S024] refreshActiveSession is triggered for expired token', async () => {
    const { actions } = useAuthSession();
    const exp = Math.floor(Date.now() / 1000) - 60; // 期限切れ
    const accessToken = buildJwt({ exp });
    actions.acceptSession(
      {
        ...createSession('A1', accessToken),
        expiresAt: new Date(exp * 1000).toISOString(),
      },
      'no-store'
    );

    const newExp = Math.floor(Date.now() / 1000) + 900;
    vi.spyOn(apiModule, 'refreshToken').mockResolvedValue({
      status: 200,
      data: { accessToken: buildJwt({ exp: newExp }) },
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
    const { actions } = useAuthSession();
    const exp = Math.floor(Date.now() / 1000) + 30; // 30 秒残り
    const accessToken = buildJwt({ exp });
    actions.acceptSession(
      {
        ...createSession('A1', accessToken),
        expiresAt: new Date(exp * 1000).toISOString(),
      },
      'no-store'
    );

    const newExp = Math.floor(Date.now() / 1000) + 900;
    vi.spyOn(apiModule, 'refreshToken').mockResolvedValue({
      status: 200,
      data: { accessToken: buildJwt({ exp: newExp }) },
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
    expect(apiModule.refreshToken).toHaveBeenCalledWith(
      expect.any(String),
      undefined,
      expect.objectContaining({ credentials: 'same-origin' })
    );
    expect(devices).toHaveLength(1);
    expect(devices?.[0]?.deviceName).toBe('refreshed-device');
  });

  it('revokeDevice removes the target session locally on success', async () => {
    const { data, actions } = useAuthSession();
    actions.acceptSession(createSession('A1', 'token-a'), 'no-store');
    actions.acceptSession(createSession('B2', 'token-b', accountB), 'no-store');

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
    actions.acceptSession(createSession('B2', 'token-b', accountB), 'no-store');

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
        ...createSession('A1', tokenA),
        expiresAt: new Date(expA * 1000).toISOString(),
      },
      'no-store'
    );

    // Session B: not near expiry
    const expB = Math.floor(Date.now() / 1000) + 900;
    const tokenB = buildJwt({ exp: expB });
    actions.acceptSession(
      {
        ...createSession('B2', tokenB, accountB),
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
      data: { accessToken: buildJwt({ exp: newExp }) },
      headers: new Headers({ 'cache-control': 'no-store' }),
    });

    await listPromise;

    // B should remain active
    expect(data.state.activeSessionId).toBe(sidB);
    expect(data.state.session?.accessToken).toBe(tokenB);

    // But A's token in the sessions array should be updated
    const sessionA = data.state.sessions?.find((s) => s.sessionId === sidA);
    expect(sessionA?.accessToken).not.toBe(tokenA);
  });

  it('[AUTH-FE-S047] refresh failure expires only the targeted session after account switch', async () => {
    const { data, actions } = useAuthSession();

    // Session A: near expiry
    const expA = Math.floor(Date.now() / 1000) + 30;
    const tokenA = buildJwt({ exp: expA });
    actions.acceptSession(
      {
        ...createSession('A1', tokenA),
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
        ...createSession('B2', tokenB, accountB),
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
    vi.spyOn(apiModule, 'listSessions').mockResolvedValue({
      status: 200,
      data: { requestId: '01ARZ3NDEKTSV4RRFFQ69G5FAV', sessions: [] },
      headers: new Headers(),
    });

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
        ...createSession('A1', accessToken),
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
    actions.acceptSession(createSession('A1', 'token-a'), 'no-store');

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
        ...createSession('A1', buildJwt({ exp: expA })),
        expiresAt: new Date(expA * 1000).toISOString(),
      },
      'no-store'
    );
    actions.acceptSession(
      {
        ...createSession('B2', buildJwt({ exp: expB }), accountB),
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
    actions.acceptSession(createSession('A1', 'token-a'), 'no-store');

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
    sessionActions.acceptSession(createSession('A1', 'token-a'), 'no-store');
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
    sessionActions.acceptSession(createSession('A1', 'token-a'), 'no-store');
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
