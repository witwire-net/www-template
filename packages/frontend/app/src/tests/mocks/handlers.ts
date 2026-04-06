import { http, HttpResponse } from 'msw';

/** ULID 形式のテスト用 ID。 */
const TEST_ULID = {
  requestId: '01ARZ3NDEKTSV4RRFFQ69G5FAV',
  accountId: '01ARZ3NDEKTSV4RRFFQ69G5FAW',
  passkeyCredentialId: '01ARZ3NDEKTSV4RRFFQ69G5FAX',
  passkeyCredentialId2: '01ARZ3NDEKTSV4RRFFQ69G5FB1',
  sessionId: '01ARZ3NDEKTSV4RRFFQ69G5FAY',
  recoveryTokenId: '01ARZ3NDEKTSV4RRFFQ69G5FAZ',
  recoverySessionId: '01ARZ3NDEKTSV4RRFFQ69G5FB0',
} as const;

const NO_STORE_HEADERS = {
  'cache-control': 'private, no-store, max-age=0',
} as const;

const resetMockData = () => undefined;

/** MSW handlers for client-side API mocking. */
const handlers = [
  // --- Auth handlers ---

  // POST /api/v1/auth/passkey/start
  http.post('/api/v1/auth/passkey/start', () => {
    return HttpResponse.json(
      {
        requestId: TEST_ULID.requestId,
        challenge: 'test-challenge-base64',
      },
      {
        status: 200,
        headers: NO_STORE_HEADERS,
      }
    );
  }),

  // POST /api/v1/auth/passkey/finish
  http.post('/api/v1/auth/passkey/finish', () => {
    return HttpResponse.json(
      {
        requestId: TEST_ULID.requestId,
        accountId: TEST_ULID.accountId,
        passkeyCredentialId: TEST_ULID.passkeyCredentialId,
        sessionId: TEST_ULID.sessionId,
        sessionToken: 'opaque-bearer-token',
        expiresAt: '2026-04-04T00:00:00.000Z',
      },
      {
        status: 200,
        headers: NO_STORE_HEADERS,
      }
    );
  }),

  // POST /api/v1/auth/recovery
  http.post('/api/v1/auth/recovery', () => {
    return HttpResponse.json(
      {
        requestId: TEST_ULID.requestId,
      },
      {
        status: 202,
        headers: NO_STORE_HEADERS,
      }
    );
  }),

  // POST /api/v1/auth/recovery/consume
  http.post('/api/v1/auth/recovery/consume', async ({ request }) => {
    const body = (await request.json()) as { token: string };

    if (body.token === 'valid-token') {
      return HttpResponse.json(
        {
          requestId: TEST_ULID.requestId,
          recoveryTokenId: TEST_ULID.recoveryTokenId,
          recoverySessionId: TEST_ULID.recoverySessionId,
          recovery_session: 'recovery-session-opaque',
          expiresAt: '2026-03-21T00:15:00.000Z',
        },
        {
          status: 200,
          headers: NO_STORE_HEADERS,
        }
      );
    }

    return HttpResponse.json(
      {
        error: 'invalid_token',
        message: '復旧リンクが無効または期限切れです。',
      },
      {
        status: 400,
        headers: NO_STORE_HEADERS,
      }
    );
  }),

  // POST /api/v1/auth/passkey/register (recovery branch)
  http.post('/api/v1/auth/passkey/register', () => {
    return HttpResponse.json(
      {
        requestId: TEST_ULID.requestId,
        accountId: TEST_ULID.accountId,
        passkeyCredentialId: TEST_ULID.passkeyCredentialId,
        sessionId: TEST_ULID.sessionId,
        sessionToken: 'opaque-bearer-token-recovery',
        expiresAt: '2026-04-04T00:00:00.000Z',
      },
      {
        status: 200,
        headers: NO_STORE_HEADERS,
      }
    );
  }),

  // POST /api/v1/auth/logout
  http.post('/api/v1/auth/logout', () => {
    return HttpResponse.json(
      {
        message: 'logged out',
      },
      {
        status: 200,
        headers: NO_STORE_HEADERS,
      }
    );
  }),

  // --- Passkey management handlers ---

  // GET /api/v1/passkeys
  http.get('/api/v1/passkeys', () => {
    return HttpResponse.json(
      {
        requestId: TEST_ULID.requestId,
        passkeys: [
          {
            id: TEST_ULID.passkeyCredentialId,
            identifier: 'MacBook Pro',
            createdAt: '2026-01-01T00:00:00.000Z',
          },
          {
            id: TEST_ULID.passkeyCredentialId2,
            identifier: 'iPhone 15',
            createdAt: '2026-02-01T00:00:00.000Z',
          },
        ],
      },
      {
        status: 200,
        headers: NO_STORE_HEADERS,
      }
    );
  }),

  // POST /api/v1/passkeys/start
  http.post('/api/v1/passkeys/start', () => {
    return HttpResponse.json(
      {
        requestId: TEST_ULID.requestId,
        challenge: 'add-challenge-base64',
        rpId: 'localhost',
      },
      {
        status: 200,
        headers: NO_STORE_HEADERS,
      }
    );
  }),

  // POST /api/v1/passkeys/finish
  http.post('/api/v1/passkeys/finish', () => {
    return HttpResponse.json(
      {
        requestId: TEST_ULID.requestId,
        passkeys: [
          {
            id: TEST_ULID.passkeyCredentialId,
            identifier: 'MacBook Pro',
            createdAt: '2026-01-01T00:00:00.000Z',
          },
          {
            id: TEST_ULID.passkeyCredentialId2,
            identifier: 'iPhone 15',
            createdAt: '2026-02-01T00:00:00.000Z',
          },
        ],
      },
      {
        status: 200,
        headers: NO_STORE_HEADERS,
      }
    );
  }),

  // DELETE /api/v1/passkeys/:id
  http.delete('/api/v1/passkeys/:id', ({ params }) => {
    const { id } = params as { id: string };
    if (id === TEST_ULID.passkeyCredentialId) {
      return new HttpResponse(null, {
        status: 204,
        headers: NO_STORE_HEADERS,
      });
    }
    return HttpResponse.json(
      {
        requestId: TEST_ULID.requestId,
        error: 'last_passkey_cannot_be_deleted',
      },
      {
        status: 409,
        headers: NO_STORE_HEADERS,
      }
    );
  }),

  // POST /api/v1/passkeys/otp
  http.post('/api/v1/passkeys/otp', () => {
    return HttpResponse.json(
      {
        requestId: TEST_ULID.requestId,
        otp: '123456',
      },
      {
        status: 200,
        headers: NO_STORE_HEADERS,
      }
    );
  }),

  // POST /api/v1/auth/passkey/add/start
  http.post('/api/v1/auth/passkey/add/start', async ({ request }) => {
    const body = (await request.json()) as { otp: string };
    if (body.otp === 'valid-otp') {
      return HttpResponse.json(
        {
          requestId: TEST_ULID.requestId,
          challenge: 'otp-add-challenge-base64',
          rpId: 'localhost',
        },
        {
          status: 200,
          headers: NO_STORE_HEADERS,
        }
      );
    }
    return HttpResponse.json(
      {
        requestId: TEST_ULID.requestId,
        error: 'invalid_otp',
      },
      {
        status: 400,
        headers: NO_STORE_HEADERS,
      }
    );
  }),

  // POST /api/v1/auth/passkey/add/finish
  http.post('/api/v1/auth/passkey/add/finish', () => {
    return new HttpResponse(null, {
      status: 200,
      headers: NO_STORE_HEADERS,
    });
  }),
];

export { handlers, NO_STORE_HEADERS, resetMockData, TEST_ULID };
