import { createHash } from 'node:crypto';

import { beforeEach, describe, expect, it, vi } from 'vitest';

const authValkeyMocks = vi.hoisted(() => ({
  connect: vi.fn(),
  ping: vi.fn(),
  incr: vi.fn(),
  expire: vi.fn(),
  getAdminValkey: vi.fn(),
}));

const routeModelMocks = vi.hoisted(() => ({
  findOperatorByEmail: vi.fn(),
  getPasskeyCount: vi.fn(),
  addOperatorPasskey: vi.fn(),
  getAdminPrisma: vi.fn(),
  generateChallenge: vi.fn(),
  consumeChallenge: vi.fn(),
  verifyAttestation: vi.fn(),
}));

vi.mock('$lib/server/infrastructure/auth/valkey.js', () => ({
  getAdminValkey: authValkeyMocks.getAdminValkey,
}));

vi.mock('$lib/server/infrastructure/db/prisma.js', () => ({
  getAdminPrisma: routeModelMocks.getAdminPrisma,
}));

vi.mock('$lib/server/models/operators.js', () => ({
  findOperatorByEmail: routeModelMocks.findOperatorByEmail,
}));

vi.mock('$lib/server/models/passkeys.js', () => ({
  addOperatorPasskey: routeModelMocks.addOperatorPasskey,
  getPasskeyCount: routeModelMocks.getPasskeyCount,
}));

vi.mock('$lib/server/infrastructure/auth/operator.js', () => ({
  consumeChallenge: routeModelMocks.consumeChallenge,
  createOperatorSession: vi.fn(),
  createSessionCookie: vi.fn(),
  generateChallenge: routeModelMocks.generateChallenge,
  signOperatorJwt: vi.fn(),
}));

vi.mock('$lib/server/infrastructure/auth/registration.js', () => ({
  verifyAttestation: routeModelMocks.verifyAttestation,
}));

vi.mock('$lib/server/infrastructure/config/env.js', () => ({
  getAdminBootstrapConfig: vi.fn(() => ({ adminBootstrapSecretHash: 'unused' })),
}));

vi.mock('$lib/server/models/schemas.js', async () => {
  const { z } = await import('zod');
  return {
    createOperatorSchema: z.object({ email: z.string(), displayName: z.string() }),
    loginEmailSchema: z.string(),
  };
});

vi.mock('$lib/server/services/auth/routes.js', () => ({
  fail: (status: number, message: string) => {
    throw Object.assign(new Error(message), { status });
  },
  loginStartRequestSchema: { parse: (value: unknown) => value },
  NO_STORE_HEADERS: { 'Cache-Control': 'no-store' },
  parseJson: async (event: { request: Request }) => event.request.json(),
  registrationFinishRequestSchema: { parse: (value: unknown) => value },
  requireAuthenticatedOperator: (event: { locals: App.Locals }) => event.locals.operator,
  requireValkey: async () => authValkeyMocks.getAdminValkey(),
  serializePasskey: (passkey: unknown) => passkey,
  sha256: (value: string) => sha256ForMock(value),
}));

import { POST as loginStartPost } from '../../../../routes/api/admin/auth/passkey/start/+server.js';
import { POST as passkeyAddFinishPost } from '../../../../routes/api/admin/auth/passkeys/finish/+server.js';
import { enforcePreAuthRateLimit, requireValkey, sha256 } from '../../services/auth/routes.js';

class ExpiringRateLimitValkey {
  private readonly records = new Map<string, { count: number; expiresAt: number | null }>();

  incr = vi.fn(async (key: string) => {
    // fake clock の現在時刻で期限切れ window を破棄し、Valkey TTL 後の retry を deterministic に再現する。
    const now = Date.now();
    const current = this.records.get(key);
    const nextCount =
      current === undefined || (current.expiresAt !== null && current.expiresAt <= now)
        ? 1
        : current.count + 1;
    this.records.set(key, { count: nextCount, expiresAt: current?.expiresAt ?? null });
    return nextCount;
  });

  expire = vi.fn(async (key: string, ttlSeconds: number) => {
    // production helper が初回だけ TTL を設定する挙動を、fake clock と連動する expiresAt として保存する。
    const current = this.records.get(key);
    this.records.set(key, {
      count: current?.count ?? 0,
      expiresAt: Date.now() + ttlSeconds * 1000,
    });
    return 1;
  });
}

function sha256ForMock(value: string): string {
  // route module の alias import を mock した場合も、本番 helper と同じ fingerprint を生成する。
  return createHash('sha256').update(value).digest('hex');
}

describe('auth route security helpers', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    authValkeyMocks.connect.mockResolvedValue(undefined);
    authValkeyMocks.ping.mockResolvedValue('PONG');
    authValkeyMocks.incr.mockResolvedValue(1);
    authValkeyMocks.expire.mockResolvedValue(1);
    authValkeyMocks.getAdminValkey.mockReturnValue({
      connect: authValkeyMocks.connect,
      ping: authValkeyMocks.ping,
      incr: authValkeyMocks.incr,
      expire: authValkeyMocks.expire,
    });
    routeModelMocks.getAdminPrisma.mockReturnValue({});
    routeModelMocks.getPasskeyCount.mockResolvedValue(1);
    routeModelMocks.generateChallenge.mockResolvedValue({
      challengeId: 'challenge-id',
      options: {
        challenge: 'challenge',
        rpId: 'admin.example.test',
        allowCredentials: [],
        userVerification: 'required',
      },
    });
    routeModelMocks.consumeChallenge.mockResolvedValue({
      challenge: 'challenge',
      operatorId: 'op-1',
      email: 'admin@example.test',
    });
    routeModelMocks.verifyAttestation.mockResolvedValue({
      credentialHandle: 'cred-1',
      publicKey: new Uint8Array([1]),
      signCount: 0,
      aaguid: new Uint8Array([2]),
      backupEligible: false,
      backupState: false,
      transports: [],
    });
    routeModelMocks.addOperatorPasskey.mockResolvedValue({ id: 'passkey-1' });
  });

  it('Admin Valkey unavailable は認証境界で 503 fail-close する', async () => {
    authValkeyMocks.ping.mockRejectedValueOnce(new Error('valkey down'));

    await expect(requireValkey()).rejects.toMatchObject({ status: 503 });
  });

  it('Admin Valkey が接続済みの場合は再 connect せず ping で疎通確認する', async () => {
    // ioredis は ready 後の connect() を例外にするため、認証境界では ping だけで可用性を確認する。
    authValkeyMocks.connect.mockRejectedValueOnce(
      new Error('Redis is already connecting/connected')
    );

    await expect(requireValkey()).resolves.toBe(authValkeyMocks.getAdminValkey());
    expect(authValkeyMocks.connect).not.toHaveBeenCalled();
    expect(authValkeyMocks.ping).toHaveBeenCalledTimes(1);
  });

  it('未登録 email と inactive operator の login start は decoy challenge で同一 response shape を返す', async () => {
    routeModelMocks.findOperatorByEmail.mockResolvedValueOnce(null);
    const unknownResponse = await loginStartPost(
      createJsonEvent({ email: 'missing@example.test' }) as never
    );
    const unknownBody = await unknownResponse.json();
    const unknownCall = routeModelMocks.generateChallenge.mock.calls.at(-1)?.[0];

    routeModelMocks.findOperatorByEmail.mockResolvedValueOnce({
      id: 'op-inactive',
      email: 'inactive@example.test',
      isActive: false,
    });
    const inactiveResponse = await loginStartPost(
      createJsonEvent({ email: 'inactive@example.test' }) as never
    );
    const inactiveBody = await inactiveResponse.json();
    const inactiveCall = routeModelMocks.generateChallenge.mock.calls.at(-1)?.[0];

    expect(Object.keys(unknownBody).sort()).toEqual(['challengeId', 'options']);
    expect(Object.keys(inactiveBody).sort()).toEqual(['challengeId', 'options']);
    expect(unknownCall).toMatchObject({
      type: 'login',
      operatorId: `decoy:${sha256('missing@example.test')}`,
    });
    expect(inactiveCall).toMatchObject({
      type: 'login',
      operatorId: `decoy:${sha256('inactive@example.test')}`,
    });
  });

  it('bootstrap secret と operator setup token の pre-auth rate limit は lock し Valkey 失敗時に fail-close する', async () => {
    const event = createJsonEvent({}) as never;

    authValkeyMocks.incr.mockResolvedValueOnce(9);
    await expect(
      enforcePreAuthRateLimit(event, 'bootstrap', 'secret-fp', authValkeyMocks.getAdminValkey())
    ).rejects.toMatchObject({ status: 429 });

    authValkeyMocks.incr.mockResolvedValueOnce(1);
    authValkeyMocks.expire.mockRejectedValueOnce(new Error('valkey down'));
    await expect(
      enforcePreAuthRateLimit(event, 'operator-setup', 'token-fp', authValkeyMocks.getAdminValkey())
    ).rejects.toMatchObject({ status: 503 });
  });

  it('temporary lock TTL expiry allows retry with fake clock', async () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date('2026-01-01T00:00:00.000Z'));
    const valkey = new ExpiringRateLimitValkey();
    const event = createJsonEvent({}) as never;
    try {
      for (let attempt = 0; attempt < 8; attempt += 1) {
        await expect(
          enforcePreAuthRateLimit(event, 'login-finish', 'challenge-fp', valkey as never)
        ).resolves.toBeUndefined();
      }
      await expect(
        enforcePreAuthRateLimit(event, 'login-finish', 'challenge-fp', valkey as never)
      ).rejects.toMatchObject({ status: 429 });
      vi.setSystemTime(new Date('2026-01-01T00:05:01.000Z'));
      await expect(
        enforcePreAuthRateLimit(event, 'login-finish', 'challenge-fp', valkey as never)
      ).resolves.toBeUndefined();
    } finally {
      vi.useRealTimers();
    }
  });

  it('challengeId と Operator binding 不一致を passkey 追加完了時に拒否する', async () => {
    routeModelMocks.consumeChallenge.mockResolvedValueOnce({
      challenge: 'challenge',
      operatorId: 'other-operator',
      email: 'other@example.test',
    });

    await expect(
      passkeyAddFinishPost(
        createJsonEvent(
          { challengeId: 'challenge-id', attestation: {} },
          authenticatedOperator()
        ) as never
      )
    ).rejects.toMatchObject({ status: 403 });
    expect(routeModelMocks.verifyAttestation).not.toHaveBeenCalled();
    expect(routeModelMocks.addOperatorPasskey).not.toHaveBeenCalled();
  });
});

function authenticatedOperator(): NonNullable<App.Locals['operator']> {
  // passkey 追加 route の operator binding 照合に使う認証済み locals を作る。
  return {
    id: 'op-1',
    email: 'admin@example.test',
    role: 'admin',
    locale: 'ja',
    sessionId: 'sess-1',
    jti: 'jti-1',
  };
}

function createJsonEvent(body: unknown, operator: App.Locals['operator'] = null) {
  // route handler が読む JSON body と client IP だけを持つ最小 RequestEvent を作る。
  return {
    request: new Request('https://admin.example.test/api/admin/auth/passkey/start', {
      method: 'POST',
      body: JSON.stringify(body),
      headers: { 'content-type': 'application/json' },
    }),
    getClientAddress: () => '192.0.2.10',
    locals: { operator },
  };
}
