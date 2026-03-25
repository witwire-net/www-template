import { http, HttpResponse } from 'msw';

interface MockProfile {
  id: number;
  name: string;
  email: string;
  createdAt: string;
}

const initialProfiles: MockProfile[] = [
  {
    id: 1,
    name: 'Sample Profile 1',
    email: 'test1@example.com',
    createdAt: '2024-01-01T00:00:00.000Z',
  },
  {
    id: 2,
    name: 'Sample Profile 2',
    email: 'test2@example.com',
    createdAt: '2024-01-02T00:00:00.000Z',
  },
];

let profiles: MockProfile[] = [...initialProfiles];

/** ULID 形式のテスト用 ID。 */
const TEST_ULID = {
  requestId: '01ARZ3NDEKTSV4RRFFQ69G5FAV',
  accountId: '01ARZ3NDEKTSV4RRFFQ69G5FAW',
  passkeyCredentialId: '01ARZ3NDEKTSV4RRFFQ69G5FAX',
  sessionId: '01ARZ3NDEKTSV4RRFFQ69G5FAY',
  recoveryTokenId: '01ARZ3NDEKTSV4RRFFQ69G5FAZ',
  recoverySessionId: '01ARZ3NDEKTSV4RRFFQ69G5FB0',
} as const;

const NO_STORE_HEADERS = {
  'cache-control': 'private, no-store, max-age=0',
} as const;

const resetMockData = () => {
  profiles = [...initialProfiles];
};

/** MSW handlers for client-side API mocking. */
const handlers = [
  // GET /api/v1/profiles
  http.get('/api/v1/profiles', () => {
    return HttpResponse.json(profiles);
  }),

  // POST /api/v1/profiles
  http.post('/api/v1/profiles', async ({ request }) => {
    // Artificial delay so UI can show a loading state.
    await new Promise((resolve) => setTimeout(resolve, 75));

    const body = (await request.json()) as { name: string; email: string };
    const nextId = Math.max(0, ...profiles.map((u) => u.id)) + 1;
    const newProfile: MockProfile = {
      id: nextId,
      name: body.name,
      email: body.email,
      createdAt: new Date().toISOString(),
    };
    profiles = [...profiles, newProfile];
    return HttpResponse.json(newProfile, { status: 201 });
  }),

  // GET /api/v1/profiles/:id
  http.get('/api/v1/profiles/:id', ({ params }) => {
    const { id } = params;
    const profileId = Number(id);
    const found = profiles.find((u) => u.id === profileId);
    if (found == null) {
      return HttpResponse.json({ error: 'Not Found' }, { status: 404 });
    }
    return HttpResponse.json(found);
  }),

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

  // POST /api/v1/app/auth/logout
  http.post('/api/v1/app/auth/logout', () => {
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
];

export { handlers, NO_STORE_HEADERS, resetMockData, TEST_ULID };
