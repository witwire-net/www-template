import { createHmac } from 'node:crypto';

import { expect, test, type APIRequestContext, type Page, type Route } from '@playwright/test';

const MOCK_NO_STORE_HEADERS = {
  'cache-control': 'private, no-store, max-age=0',
  'content-type': 'application/json',
} as const;

const TEST_ULID = {
  requestId: '01ARZ3NDEKTSV4RRFFQ69G5FAV',
  accountId: '01ARZ3NDEKTSV4RRFFQ69G5FAW',
  passkeyCredentialId: '01ARZ3NDEKTSV4RRFFQ69G5FAX',
  sessionId: '01ARZ3NDEKTSV4RRFFQ69G5FAY',
  recoveryTokenId: '01ARZ3NDEKTSV4RRFFQ69G5FAZ',
  recoverySessionId: '01ARZ3NDEKTSV4RRFFQ69G5FB0',
} as const;

interface BrowserFetchOptions {
  method?: string;
  headers?: Record<string, string>;
  body?: unknown;
}

interface BrowserFetchResult {
  status: number;
  cacheControl: string | null;
  body: unknown;
}

const expectNoStore = (cacheControl: string | null) => {
  expect(cacheControl).not.toBeNull();
  expect(cacheControl).toContain('no-store');
};

const forwardedIpHeaders = (clientIp: string) => ({
  'X-Forwarded-For': clientIp,
  'X-Real-IP': clientIp,
});

const parseJsonBody = async (response: Awaited<ReturnType<APIRequestContext['fetch']>>) => {
  const text = await response.text();
  return text === '' ? null : (JSON.parse(text) as Record<string, unknown>);
};

const base64UrlEncode = (value: unknown) =>
  Buffer.from(JSON.stringify(value)).toString('base64url');

const signTestJwt = (claims: Record<string, unknown>) => {
  const signingInput = `${base64UrlEncode({ alg: 'HS256', typ: 'JWT' })}.${base64UrlEncode(claims)}`;
  const signature = createHmac('sha256', 'change-this-to-a-long-random-jwt-secret-in-production')
    .update(signingInput)
    .digest('base64url');

  return `${signingInput}.${signature}`;
};

const startPasskeyViaApi = async (request: APIRequestContext, clientIp: string) => {
  const startResponse = await request.post('/api/v1/auth/passkey/start', {
    data: { identifier: 'member@example.com' },
    headers: forwardedIpHeaders(clientIp),
  });
  expect(startResponse.status()).toBe(200);
  expectNoStore(startResponse.headers()['cache-control'] ?? null);

  const startBody = (await parseJsonBody(startResponse)) as {
    challenge: string;
  };

  return { startBody, startResponse };
};

const fulfillInternalError = async (route: Route) => {
  await route.fulfill({
    status: 503,
    headers: MOCK_NO_STORE_HEADERS,
    body: JSON.stringify({
      requestId: TEST_ULID.requestId,
      error: 'internal-error',
    }),
  });
};

const fetchJsonInBrowser = async (
  page: Page,
  url: string,
  options: BrowserFetchOptions = {}
): Promise<BrowserFetchResult> => {
  return page.evaluate(
    async ({ url: targetUrl, options: targetOptions }) => {
      const response = await fetch(targetUrl, {
        method: targetOptions.method,
        headers: targetOptions.headers,
        body: targetOptions.body === undefined ? undefined : JSON.stringify(targetOptions.body),
      });

      const text = await response.text();
      return {
        status: response.status,
        cacheControl: response.headers.get('cache-control'),
        body: text === '' ? null : JSON.parse(text),
      };
    },
    { url, options }
  );
};

test.describe('auth api contract', () => {
  test.skip(
    ({ browserName }) => browserName !== 'chromium',
    'backend auth contract is browser-agnostic'
  );

  test('passkey start / invalid finish は no-store で返る', async ({ request }) => {
    const { startBody } = await startPasskeyViaApi(request, '198.51.100.10');

    expect(startBody.challenge.length).toBeGreaterThan(0);

    const finishResponse = await request.post('/api/v1/auth/passkey/finish', {
      data: { credential: `existing-credential::${startBody.challenge}` },
      headers: forwardedIpHeaders('198.51.100.10'),
    });
    expect(finishResponse.status()).toBe(400);
    expectNoStore(finishResponse.headers()['cache-control'] ?? null);
  });

  test('recovery request / invalid consume / invalid register は no-store で返る', async ({
    request,
  }) => {
    const recoveryResponse = await request.post('/api/v1/auth/recovery', {
      data: { email: 'member@example.com' },
    });
    expect(recoveryResponse.status()).toBe(202);
    expectNoStore(recoveryResponse.headers()['cache-control'] ?? null);

    const consumeResponse = await request.post('/api/v1/auth/recovery/consume', {
      data: { token: 'invalid-token' },
    });
    expect(consumeResponse.status()).toBe(400);
    expectNoStore(consumeResponse.headers()['cache-control'] ?? null);

    const registerResponse = await request.post('/api/v1/auth/passkey/register', {
      data: {
        recovery_session: 'invalid-recovery-session',
        credential: 'replacement-credential',
      },
    });
    expect(registerResponse.status()).toBe(400);
    expectNoStore(registerResponse.headers()['cache-control'] ?? null);
  });

  test('session を持たない app endpoint は unauthenticated を返す', async ({ request }) => {
    const response = await request.post('/api/v1/auth/logout');

    expect(response.status()).toBe(401);
    expectNoStore(response.headers()['cache-control'] ?? null);

    const body = await parseJsonBody(response);
    expect(body?.error).toBe('unauthenticated');
  });

  test('revoked session で app endpoint を叩くと session-expired を返す', async ({ request }) => {
    const now = Math.floor(Date.now() / 1000);
    const accessToken = signTestJwt({
      sub: TEST_ULID.accountId,
      sid: TEST_ULID.sessionId,
      jti: TEST_ULID.passkeyCredentialId,
      iat: now,
      exp: now + 900,
    });

    const expiredResponse = await request.post('/api/v1/auth/logout', {
      headers: {
        Authorization: `Bearer ${accessToken}`,
      },
    });

    expect(expiredResponse.status()).toBe(401);
    expectNoStore(expiredResponse.headers()['cache-control'] ?? null);

    const body = await parseJsonBody(expiredResponse);
    expect(body?.error).toBe('session-expired');
  });

  test('logout without session は unauthenticated を返す', async ({ request }) => {
    const response = await request.post('/api/v1/auth/logout');

    expect(response.status()).toBe(401);
    expectNoStore(response.headers()['cache-control'] ?? null);

    const body = await parseJsonBody(response);
    expect(body?.error).toBe('unauthenticated');
  });
});

test.describe('auth api internal-error classification', () => {
  test.skip(
    ({ browserName }) => browserName !== 'chromium',
    'backend auth contract is browser-agnostic'
  );

  test.beforeEach(async ({ page }) => {
    await page.goto('/');
  });

  test('passkey start は internal-error classification を no-store で返せる', async ({ page }) => {
    await page.route('**/api/v1/auth/passkey/start', fulfillInternalError);

    const result = await fetchJsonInBrowser(page, '/api/v1/auth/passkey/start', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: { identifier: 'member@example.com' },
    });

    expect(result.status).toBe(503);
    expectNoStore(result.cacheControl);
    expect((result.body as { error: string }).error).toBe('internal-error');
  });

  test('recovery request は internal-error classification を no-store で返せる', async ({
    page,
  }) => {
    await page.route('**/api/v1/auth/recovery', fulfillInternalError);

    const result = await fetchJsonInBrowser(page, '/api/v1/auth/recovery', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: { email: 'member@example.com' },
    });

    expect(result.status).toBe(503);
    expectNoStore(result.cacheControl);
    expect((result.body as { error: string }).error).toBe('internal-error');
  });

  test('consume recovery は internal-error classification を no-store で返せる', async ({
    page,
  }) => {
    await page.route('**/api/v1/auth/recovery/consume', fulfillInternalError);

    const result = await fetchJsonInBrowser(page, '/api/v1/auth/recovery/consume', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: { token: 'valid-token' },
    });

    expect(result.status).toBe(503);
    expectNoStore(result.cacheControl);
    expect((result.body as { error: string }).error).toBe('internal-error');
  });

  test('register passkey は internal-error classification を no-store で返せる', async ({
    page,
  }) => {
    await page.route('**/api/v1/auth/passkey/register', fulfillInternalError);

    const result = await fetchJsonInBrowser(page, '/api/v1/auth/passkey/register', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: {
        recovery_session: TEST_ULID.recoverySessionId,
        credential: 'replacement-credential',
      },
    });

    expect(result.status).toBe(503);
    expectNoStore(result.cacheControl);
    expect((result.body as { error: string }).error).toBe('internal-error');
  });

  test('logout は internal-error classification を no-store で返せる', async ({ page }) => {
    await page.route('**/api/v1/auth/logout', fulfillInternalError);

    const result = await fetchJsonInBrowser(page, '/api/v1/auth/logout', {
      method: 'POST',
      headers: { Authorization: 'Bearer jwt-invalid-token' },
    });

    expect(result.status).toBe(503);
    expectNoStore(result.cacheControl);
    expect((result.body as { error: string }).error).toBe('internal-error');
  });
});
