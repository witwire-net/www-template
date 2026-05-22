import { beforeEach, describe, expect, it, vi } from 'vitest';

const hookMocks = vi.hoisted(() => ({
  redisConstructor: vi.fn(),
  verifyOperatorSession: vi.fn(),
  getAdminAuthConfig: vi.fn(),
  validateCsrf: vi.fn(),
  requireSameOrigin: vi.fn(),
  issueCsrfToken: vi.fn(),
  getAdminPrisma: vi.fn(),
  findOperatorById: vi.fn(),
}));

vi.mock('ioredis', () => ({
  default: hookMocks.redisConstructor,
}));

vi.mock('$lib/server/infrastructure/auth/operator', () => ({
  verifyOperatorSession: hookMocks.verifyOperatorSession,
}));

vi.mock('$lib/server/infrastructure/config/env', () => ({
  getAdminAuthConfig: hookMocks.getAdminAuthConfig,
}));

vi.mock('$lib/server/infrastructure/csrf/guard', () => ({
  validateCsrf: hookMocks.validateCsrf,
  requireSameOrigin: hookMocks.requireSameOrigin,
  issueCsrfToken: hookMocks.issueCsrfToken,
}));

vi.mock('$lib/server/infrastructure/db/prisma', () => ({
  getAdminPrisma: hookMocks.getAdminPrisma,
}));

vi.mock('$lib/server/models/operators', () => ({
  findOperatorById: hookMocks.findOperatorById,
}));

import { handle } from '../../../../hooks.server.js';

interface HookTestEvent {
  url: URL;
  request: Request;
  cookies: { get: (name: string) => string | undefined; delete: ReturnType<typeof vi.fn> };
  locals: App.Locals;
}

describe('Admin hooks infrastructure contract', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    hookMocks.redisConstructor.mockImplementation(() => ({ ping: vi.fn() }));
    hookMocks.getAdminAuthConfig.mockReturnValue({ adminValkeyUrl: 'redis://valkey:6379/1' });
    hookMocks.getAdminPrisma.mockReturnValue({});
    hookMocks.issueCsrfToken.mockReturnValue({
      token: 'csrf-token',
      cookieValue: 'admin_csrf=csrf-token; Path=/',
    });
  });

  it('JWT role claim ではなく DB current role を locals に設定する', async () => {
    hookMocks.verifyOperatorSession.mockResolvedValue({
      operatorId: 'op-1',
      email: 'admin@example.test',
      role: 'admin',
      sessionId: 'sess-1',
      jti: 'jti-1',
    });
    hookMocks.findOperatorById.mockResolvedValue({
      id: 'op-1',
      email: 'admin@example.test',
      role: 'viewer',
      isActive: true,
      locale: 'en',
    });
    const event = createHookEvent();

    const response = await handle({
      event,
      resolve: async (resolvedEvent: HookTestEvent) => {
        // route/load が参照する locals は DB 現在 role で上書きされている必要がある。
        expect(resolvedEvent.locals.operator).toMatchObject({
          role: 'viewer',
          locale: 'en',
          sessionId: 'sess-1',
          jti: 'jti-1',
        });
        return new Response('ok');
      },
    } as never);

    expect(response.status).toBe(200);
    expect(event.locals.operator).toMatchObject({ role: 'viewer', locale: 'en' });
  });

  it('LOCALIZATION-BE-S009 Admin 認証 context は DB 保存済み operator locale を読み込む', async () => {
    mockValidSession({ locale: 'en' });
    const event = createHookEvent();

    await handle({ event, resolve: async () => new Response('ok') } as never);

    expect(event.locals.operator).toMatchObject({
      id: 'op-1',
      role: 'admin',
      sessionId: 'sess-1',
      jti: 'jti-1',
      locale: 'en',
    });
  });

  it('valid cookie は operator locals と no-store / CSRF cookie を付与する', async () => {
    mockValidSession();
    const event = createHookEvent();

    const response = await handle({ event, resolve: async () => new Response('ok') } as never);

    expect(response.status).toBe(200);
    expect(response.headers.get('Cache-Control')).toBe('no-store');
    expect(response.headers.get('Set-Cookie')).toContain('admin_csrf=csrf-token');
    expect(event.locals.operator).toMatchObject({ id: 'op-1', role: 'admin' });
  });

  it('expired cookie / tampered JWT は login redirect と cookie clear に集約する', async () => {
    hookMocks.verifyOperatorSession.mockResolvedValue(null);
    const expiredEvent = createHookEvent({ pathname: '/accounts' });
    const expiredResponse = await handle({
      event: expiredEvent,
      resolve: async () => new Response('unreachable'),
    } as never);
    expect(expiredResponse.status).toBe(303);
    expect(expiredResponse.headers.get('Location')).toBe('/login?redirectTo=%2Faccounts');
    expect(expiredEvent.cookies.delete).toHaveBeenCalledWith('admin_session', { path: '/' });

    hookMocks.verifyOperatorSession.mockResolvedValue(null);
    const tamperedEvent = createHookEvent({ pathname: '/settings' });
    const tamperedResponse = await handle({
      event: tamperedEvent,
      resolve: async () => new Response('unreachable'),
    } as never);
    expect(tamperedResponse.status).toBe(303);
    expect(tamperedResponse.headers.get('Location')).toBe('/login?redirectTo=%2Fsettings');
    expect(tamperedEvent.cookies.delete).toHaveBeenCalledWith('admin_session', { path: '/' });
  });

  it('inactive operator は cookie を破棄して login redirect する', async () => {
    hookMocks.verifyOperatorSession.mockResolvedValue({
      operatorId: 'op-1',
      email: 'admin@example.test',
      role: 'admin',
      sessionId: 'sess-1',
      jti: 'jti-1',
    });
    hookMocks.findOperatorById.mockResolvedValue({
      id: 'op-1',
      email: 'admin@example.test',
      role: 'admin',
      isActive: false,
      locale: 'ja',
    });
    const event = createHookEvent({ pathname: '/accounts' });

    const response = await handle({
      event,
      resolve: async () => new Response('unreachable'),
    } as never);

    expect(response.status).toBe(303);
    expect(event.cookies.delete).toHaveBeenCalledWith('admin_session', { path: '/' });
  });

  it('pre-auth bypass / authed login redirect / redirectTo preserved / BFF no-store を検証する', async () => {
    hookMocks.verifyOperatorSession.mockResolvedValue(null);
    const preAuth = await handle({
      event: createHookEvent({ pathname: '/operator-setup', cookie: undefined }),
      resolve: async () => new Response('setup'),
    } as never);
    expect(preAuth.status).toBe(200);

    mockValidSession();
    const loginResponse = await handle({
      event: createHookEvent({ pathname: '/login' }),
      resolve: async () => new Response('unreachable'),
    } as never);
    expect(loginResponse.status).toBe(303);
    expect(loginResponse.headers.get('Location')).toBe('/');

    hookMocks.verifyOperatorSession.mockResolvedValue(null);
    const protectedResponse = await handle({
      event: createHookEvent({ pathname: '/accounts', search: '?page=2', cookie: undefined }),
      resolve: async () => new Response('unreachable'),
    } as never);
    expect(protectedResponse.headers.get('Location')).toBe(
      '/login?redirectTo=%2Faccounts%3Fpage%3D2'
    );

    const bffResponse = await handle({
      event: createHookEvent({ pathname: '/api/admin/accounts', cookie: undefined }),
      resolve: async () => new Response('unreachable'),
    } as never);
    expect(bffResponse.status).toBe(401);
    expect(bffResponse.headers.get('Cache-Control')).toBe('no-store');
  });

  it('CSRF valid / cross-origin / mismatch と pre-auth Origin bypass を hook 境界で検証する', async () => {
    mockValidSession();
    hookMocks.validateCsrf.mockResolvedValueOnce(undefined);
    const validMutation = await handle({
      event: createHookEvent({ method: 'POST', pathname: '/accounts' }),
      resolve: async () => new Response('ok'),
    } as never);
    expect(validMutation.status).toBe(200);

    mockValidSession();
    hookMocks.validateCsrf.mockRejectedValueOnce(new Error('csrf'));
    const mismatch = await handle({
      event: createHookEvent({ method: 'POST', pathname: '/accounts' }),
      resolve: async () => new Response('unreachable'),
    } as never);
    expect(mismatch.status).toBe(403);

    hookMocks.verifyOperatorSession.mockResolvedValue(null);
    hookMocks.requireSameOrigin.mockImplementationOnce(() => {
      throw new Error('origin');
    });
    const crossOrigin = await handle({
      event: createHookEvent({
        method: 'POST',
        pathname: '/api/admin/auth/passkey/start',
        cookie: undefined,
      }),
      resolve: async () => new Response('unreachable'),
    } as never);
    expect(crossOrigin.status).toBe(403);

    hookMocks.requireSameOrigin.mockReturnValueOnce(undefined);
    const preAuthStart = await handle({
      event: createHookEvent({
        method: 'POST',
        pathname: '/api/admin/auth/passkey/start',
        cookie: undefined,
      }),
      resolve: async () => new Response('ok'),
    } as never);
    expect(preAuthStart.status).toBe(200);
  });
});

function mockValidSession(input: { locale?: 'ja' | 'en' } = {}): void {
  // 有効 cookie の検証済み session と active operator をセットにし、hook の正常 path を簡潔に再利用する。
  hookMocks.verifyOperatorSession.mockResolvedValue({
    operatorId: 'op-1',
    email: 'admin@example.test',
    role: 'admin',
    sessionId: 'sess-1',
    jti: 'jti-1',
  });
  hookMocks.findOperatorById.mockResolvedValue({
    id: 'op-1',
    email: 'admin@example.test',
    role: 'admin',
    isActive: true,
    locale: input.locale ?? 'ja',
  });
}

function createHookEvent(
  input: { pathname?: string; search?: string; method?: string; cookie?: string } = {}
): HookTestEvent {
  // hooks が読む cookie/url/request/locals の subset だけを持つ RequestEvent を作る。
  const pathname = input.pathname ?? '/';
  const url = new URL(`https://admin.example.test${pathname}${input.search ?? ''}`);
  const cookie =
    input.cookie === undefined && 'cookie' in input ? undefined : (input.cookie ?? 'jwt-token');
  return {
    url,
    request: new Request(url, { method: input.method ?? 'GET' }),
    cookies: {
      get: (name: string) => (name === 'admin_session' ? cookie : undefined),
      delete: vi.fn(),
    },
    locals: { operator: null },
  };
}
