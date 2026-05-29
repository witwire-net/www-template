import { afterEach, describe, expect, it, vi } from 'vitest';

import {
  assertAdminApiPath,
  createAdminRequestInit,
  requestRefreshAdminSession,
  requestStartAdminLogin,
  requestStartInitialAdminSetup,
} from './client';

type FetchCall = [RequestInfo | URL, RequestInit | undefined];

function firstFetchCall(fetchMock: { mock: { calls: unknown[] } }): FetchCall {
  // Vitest mock の可変長 call tuple を、generated SDK が使う fetch(input, init) の形に絞って検証する。
  const call = fetchMock.mock.calls[0];
  expect(call).toBeDefined();
  return call as FetchCall;
}

describe('Admin API wrapper boundary', () => {
  afterEach(() => {
    // fetch の差し替えを各 test 後に戻し、別 test の request 監視へ影響を残さない。
    vi.unstubAllGlobals();
  });

  it('[ADMIN-CONSOLE-FE-S041] same-origin /api/v1 path and credentials are enforced', async () => {
    // generated SDK の fetch 呼び出し先を記録し、wrapper が absolute Product origin を作らないことを検証する。
    const fetchMock = vi.fn(
      async () =>
        new Response(
          JSON.stringify({
            requestId: '01JADMINLOGINSTART0000000000',
            challenge: 'challenge',
            rpId: 'admin.example.com',
            userVerification: 'required',
          }),
          { status: 200 }
        )
    );
    vi.stubGlobal('fetch', fetchMock);

    const init = createAdminRequestInit({ accessToken: 'access-token', csrfToken: 'csrf-token' });
    const headers = init.headers as Record<string, string>;

    expect(assertAdminApiPath('/api/v1/accounts')).toBe('/api/v1/accounts');
    expect(init.credentials).toBe('same-origin');
    expect(headers.Authorization).toBe('Bearer access-token');
    expect(headers['X-CSRF-Token']).toBeUndefined();

    await requestStartAdminLogin({ identifier: 'operator@example.com' });
    const [requestUrl, requestInit] = firstFetchCall(fetchMock);

    expect(requestUrl).toBe('/api/v1/auth/passkey/start');
    expect(requestInit?.credentials).toBe('same-origin');
  });

  it('[ADMIN-CONSOLE-FE-S042] Product domains and BFF escape paths are rejected', () => {
    // Product origin・旧 BFF・scope 外 path をすべて送信前に拒否し、request 自体を発生させない。
    expect(() => assertAdminApiPath('https://product.example.com/api/v1/accounts')).toThrow(
      'admin-api-absolute-url-forbidden'
    );
    expect(() => assertAdminApiPath('//product.example.com/api/v1/accounts')).toThrow(
      'admin-api-absolute-url-forbidden'
    );
    expect(() => assertAdminApiPath(['', 'api', 'admin', 'accounts'].join('/'))).toThrow(
      'admin-api-bff-path-forbidden'
    );
    expect(() => assertAdminApiPath('/sessions')).toThrow('admin-api-path-out-of-scope');
  });

  it('[ADMIN-AUTH-FE-S035] refresh uses same-origin Cookie credentials without a readable token', async () => {
    // refreshToken は Cookie 専用なので、body や Authorization header ではなく credentials だけを確認する。
    const fetchMock = vi.fn(
      async () =>
        new Response(
          JSON.stringify({
            requestId: '01JADMINREFRESH000000000000',
            operator: {
              operatorId: '01JOPERATOR00000000000000',
              email: 'operator@example.com',
              role: 'admin',
              active: true,
            },
            sessionId: '01JSESSION0000000000000000',
            accessToken: 'new-access-token',
            expiresAt: '2030-01-01T00:00:00.000Z',
            csrfToken: 'csrf-token',
          }),
          { status: 200 }
        )
    );
    vi.stubGlobal('fetch', fetchMock);

    await requestRefreshAdminSession();
    const [requestUrl, requestInit] = firstFetchCall(fetchMock);
    const body = requestInit?.body;

    expect(requestUrl).toBe('/api/v1/auth/operator/refresh');
    expect(requestInit?.credentials).toBe('same-origin');
    expect(body).toBeUndefined();
  });

  it('[ADMIN-AUTH-FE-S038] initial setup calls same-origin /api/v1/auth/setup start', async () => {
    // 初回 setup secret を package-local BFF に送らず、Admin backend の setup start path だけへ送る。
    const fetchMock = vi.fn(
      async () =>
        new Response(
          JSON.stringify({
            requestId: '01JADMINSETUPSTART000000000',
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
          }),
          { status: 200 }
        )
    );
    vi.stubGlobal('fetch', fetchMock);

    await requestStartInitialAdminSetup({
      email: 'admin@example.com',
      displayName: 'Admin Operator',
      bootstrapSecret: 'bootstrap-secret',
    });
    const [requestUrl, requestInit] = firstFetchCall(fetchMock);

    expect(requestUrl).toBe('/api/v1/auth/setup/start');
    expect(requestInit?.credentials).toBe('same-origin');
  });
});
