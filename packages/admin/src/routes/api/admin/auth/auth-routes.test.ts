import { describe, beforeEach, expect, it, vi } from 'vitest';

const routeMocks = vi.hoisted(() => ({
  valkey: { ping: vi.fn(), incr: vi.fn(), expire: vi.fn() },
  getAdminPrisma: vi.fn(),
  findOperatorByEmail: vi.fn(),
  findOperatorById: vi.fn(),
  countOperators: vi.fn(),
  createInitialAdminOperator: vi.fn(),
  consumeOperatorSetupToken: vi.fn(),
  getPasskeyCount: vi.fn(),
  listOperatorPasskeys: vi.fn(),
  findOperatorPasskeyByCredentialHandle: vi.fn(),
  findOperatorPasskeyForOperator: vi.fn(),
  addOperatorPasskey: vi.fn(),
  deleteOperatorPasskey: vi.fn(),
  updateOperatorPasskeySignCount: vi.fn(),
  updateLoginTimestamp: vi.fn(),
  generateChallenge: vi.fn(),
  consumeChallenge: vi.fn(),
  verifyAssertion: vi.fn(),
  generateRegistrationChallenge: vi.fn(),
  verifyAttestation: vi.fn(),
  enforcePreAuthRateLimit: vi.fn(),
  verifyBootstrapSecret: vi.fn(),
  findOperatorBySetupToken: vi.fn(),
  getEnvConfig: vi.fn(),
}));

vi.mock('$lib/server/infrastructure/db/prisma.js', () => ({
  getAdminPrisma: routeMocks.getAdminPrisma,
}));
vi.mock('$lib/server/models/operators.js', () => ({
  findOperatorByEmail: routeMocks.findOperatorByEmail,
  findOperatorById: routeMocks.findOperatorById,
  countOperators: routeMocks.countOperators,
  createInitialAdminOperator: routeMocks.createInitialAdminOperator,
  consumeOperatorSetupToken: routeMocks.consumeOperatorSetupToken,
  updateLoginTimestamp: routeMocks.updateLoginTimestamp,
}));
vi.mock('$lib/server/models/passkeys.js', () => ({
  getPasskeyCount: routeMocks.getPasskeyCount,
  listOperatorPasskeys: routeMocks.listOperatorPasskeys,
  findOperatorPasskeyByCredentialHandle: routeMocks.findOperatorPasskeyByCredentialHandle,
  findOperatorPasskeyForOperator: routeMocks.findOperatorPasskeyForOperator,
  addOperatorPasskey: routeMocks.addOperatorPasskey,
  deleteOperatorPasskey: routeMocks.deleteOperatorPasskey,
  updateOperatorPasskeySignCount: routeMocks.updateOperatorPasskeySignCount,
}));
vi.mock('$lib/server/infrastructure/auth/operator.js', () => ({
  generateChallenge: routeMocks.generateChallenge,
  consumeChallenge: routeMocks.consumeChallenge,
  verifyAssertion: routeMocks.verifyAssertion,
}));
vi.mock('$lib/server/infrastructure/auth/registration.js', () => ({
  generateRegistrationChallenge: routeMocks.generateRegistrationChallenge,
  verifyAttestation: routeMocks.verifyAttestation,
}));
vi.mock('$lib/server/infrastructure/config/env.js', () => ({
  getEnvConfig: routeMocks.getEnvConfig,
}));
vi.mock('$lib/server/infrastructure/config/platform.js', () => ({
  getPlatformConfig: vi.fn(() => ({ adminRpId: 'admin.example.test' })),
}));
vi.mock('$lib/server/services/auth/routes.js', async () => {
  const { json } = await import('@sveltejs/kit');
  return {
    NO_STORE_HEADERS: { 'Cache-Control': 'no-store' },
    loginStartRequestSchema: { parse: (value: unknown) => value },
    challengeFinishRequestSchema: { parse: (value: unknown) => value },
    registrationFinishRequestSchema: { parse: (value: unknown) => value },
    setupStartRequestSchema: { parse: (value: unknown) => value },
    operatorSetupStartRequestSchema: { parse: (value: unknown) => value },
    parseJson: async (event: { request: Request }) => event.request.json(),
    fail: (status: number, message: string) => {
      throw Object.assign(new Error(message), { status });
    },
    requireValkey: async () => {
      await routeMocks.valkey.ping();
      return routeMocks.valkey;
    },
    requireAuthenticatedOperator: (event: { locals: App.Locals }) => {
      if (event.locals.operator === null)
        throw Object.assign(new Error('Unauthorized'), { status: 401 });
      return event.locals.operator;
    },
    serializePasskey: (passkey: { id: string; createdAt: Date }) => ({
      id: passkey.id,
      createdAt: passkey.createdAt.toISOString(),
    }),
    passkeyListResponse: async (operatorId: string) =>
      json(
        {
          passkeys: (await routeMocks.listOperatorPasskeys(null, operatorId)).map(
            (passkey: { id: string }) => ({ id: passkey.id })
          ),
        },
        { headers: { 'Cache-Control': 'no-store' } }
      ),
    sessionRedirectResponse: async () =>
      new Response(null, {
        status: 303,
        headers: {
          Location: '/',
          'Set-Cookie': 'admin_session=jwt-token; Path=/',
          'Cache-Control': 'no-store',
        },
      }),
    enforcePreAuthRateLimit: routeMocks.enforcePreAuthRateLimit,
    sha256: (value: string) => `sha:${value}`,
    verifyBootstrapSecret: routeMocks.verifyBootstrapSecret,
    findOperatorBySetupToken: routeMocks.findOperatorBySetupToken,
  };
});

import { POST as operatorSetupFinish } from './operator-setup/finish/+server.js';
import { POST as operatorSetupStart } from './operator-setup/start/+server.js';
import { POST as loginFinish } from './passkey/finish/+server.js';
import { POST as loginStart } from './passkey/start/+server.js';
import { GET as listPasskeys } from './passkeys/+server.js';
import { DELETE as deletePasskey } from './passkeys/[id]/+server.js';
import { POST as passkeyAddFinish } from './passkeys/finish/+server.js';
import { POST as passkeyAddStart } from './passkeys/start/+server.js';
import { POST as setupFinish } from './setup/finish/+server.js';
import { POST as setupStart } from './setup/start/+server.js';

describe('admin auth route phase 15 coverage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    routeMocks.valkey.ping.mockResolvedValue('PONG');
    routeMocks.getAdminPrisma.mockReturnValue({
      $transaction: async (callback: (tx: unknown) => unknown) => callback({}),
    });
    routeMocks.getEnvConfig.mockReturnValue({
      adminOrigin: 'https://admin.example.test',
      adminBootstrapEnabled: true,
      adminBootstrapExpiresAt: futureDate(),
    });
    routeMocks.findOperatorByEmail.mockResolvedValue(operator());
    routeMocks.findOperatorById.mockResolvedValue(operator());
    routeMocks.countOperators.mockResolvedValue(0);
    routeMocks.getPasskeyCount.mockResolvedValue(2);
    routeMocks.listOperatorPasskeys.mockResolvedValue([passkey('passkey-1'), passkey('passkey-2')]);
    routeMocks.findOperatorPasskeyByCredentialHandle.mockResolvedValue(passkey('passkey-1'));
    routeMocks.findOperatorPasskeyForOperator.mockResolvedValue(passkey('passkey-1'));
    routeMocks.generateChallenge.mockResolvedValue({
      challengeId: 'challenge-1',
      options: { challenge: 'public' },
    });
    routeMocks.generateRegistrationChallenge.mockResolvedValue({
      challengeId: 'challenge-1',
      options: { challenge: 'public' },
    });
    routeMocks.consumeChallenge.mockResolvedValue({
      challenge: 'private',
      operatorId: 'op-1',
      email: 'admin@example.test',
      displayName: 'Admin',
    });
    routeMocks.verifyAssertion.mockResolvedValue({ newSignCount: 9 });
    routeMocks.verifyAttestation.mockResolvedValue(attestation());
    routeMocks.addOperatorPasskey.mockResolvedValue(passkey('passkey-new'));
    routeMocks.createInitialAdminOperator.mockResolvedValue(operator({ role: 'admin' }));
    routeMocks.consumeOperatorSetupToken.mockResolvedValue(true);
    routeMocks.verifyBootstrapSecret.mockReturnValue(true);
    routeMocks.findOperatorBySetupToken.mockResolvedValue(operator());
  });

  it('passkey login happy path は cookie 付き redirect を返す', async () => {
    const response = await loginFinish(
      jsonEvent({ challengeId: 'challenge-1', assertion: { id: 'cred-1' } }) as never
    );

    expect(response.status).toBe(303);
    expect(response.headers.get('set-cookie')).toContain('admin_session=');
    expect(routeMocks.updateOperatorPasskeySignCount).toHaveBeenCalledWith(
      expect.anything(),
      'passkey-1',
      9
    );
  });

  it('invalid assertion / unknown credential / expired challenge / consumed challenge reuse は 401 に集約する', async () => {
    routeMocks.verifyAssertion.mockRejectedValueOnce(new Error('bad assertion'));
    await expect(
      loginFinish(jsonEvent({ challengeId: 'challenge-1', assertion: { id: 'cred-1' } }) as never)
    ).rejects.toMatchObject({ status: 401 });
    routeMocks.findOperatorPasskeyByCredentialHandle.mockResolvedValueOnce(null);
    await expect(
      loginFinish(jsonEvent({ challengeId: 'challenge-1', assertion: { id: 'missing' } }) as never)
    ).rejects.toMatchObject({ status: 401 });
    routeMocks.consumeChallenge.mockRejectedValueOnce(new Error('expired'));
    await expect(
      loginFinish(jsonEvent({ challengeId: 'expired', assertion: { id: 'cred-1' } }) as never)
    ).rejects.toMatchObject({ status: 401 });
    routeMocks.consumeChallenge.mockRejectedValueOnce(new Error('reused'));
    await expect(
      loginFinish(jsonEvent({ challengeId: 'reused', assertion: { id: 'cred-1' } }) as never)
    ).rejects.toMatchObject({ status: 401 });
  });

  it('unknown admin email start は decoy challenge で列挙を防ぎ finish も同じ 401 にする', async () => {
    routeMocks.findOperatorByEmail.mockResolvedValueOnce(null);
    const startResponse = await loginStart(jsonEvent({ email: 'missing@example.test' }) as never);
    const startBody = await startResponse.json();
    expect(startBody).toEqual({ challengeId: 'challenge-1', options: { challenge: 'public' } });
    expect(routeMocks.generateChallenge).toHaveBeenCalledWith(
      expect.objectContaining({ operatorId: 'decoy:sha:missing@example.test' }),
      routeMocks.valkey
    );

    routeMocks.consumeChallenge.mockResolvedValueOnce({
      challenge: 'private',
      operatorId: 'decoy:sha:missing@example.test',
      email: 'missing@example.test',
    });
    await expect(
      loginFinish(jsonEvent({ challengeId: 'challenge-1', assertion: { id: 'cred-1' } }) as never)
    ).rejects.toMatchObject({ status: 401 });
  });

  it('passkey 管理 API は list/add/delete と access control を検証する', async () => {
    const listResponse = await listPasskeys(jsonEvent({}, authed()) as never);
    expect(await listResponse.json()).toMatchObject({
      passkeys: [{ id: 'passkey-1' }, { id: 'passkey-2' }],
    });
    await expect(listPasskeys(jsonEvent({}, null) as never)).rejects.toMatchObject({ status: 401 });
    expect((await passkeyAddStart(jsonEvent({}, authed()) as never)).status).toBe(200);
    expect(
      (
        await passkeyAddFinish(
          jsonEvent({ challengeId: 'challenge-1', attestation: {} }, authed()) as never
        )
      ).status
    ).toBe(201);
    routeMocks.getPasskeyCount.mockResolvedValueOnce(1);
    await expect(
      deletePasskey(jsonEvent({}, authed(), { id: 'passkey-1' }) as never)
    ).rejects.toMatchObject({ status: 400 });
    routeMocks.getPasskeyCount.mockResolvedValueOnce(2);
    expect(
      (await deletePasskey(jsonEvent({}, authed(), { id: 'passkey-1' }) as never)).status
    ).toBe(200);
    routeMocks.findOperatorPasskeyForOperator.mockResolvedValueOnce(null);
    await expect(
      deletePasskey(jsonEvent({}, authed(), { id: 'other-passkey' }) as never)
    ).rejects.toMatchObject({ status: 403 });
  });

  it('initial setup は bootstrap secret / expiry / disable / zero-operator / fail-close を検証する', async () => {
    expect((await setupStart(jsonEvent(setupBody()) as never)).status).toBe(200);
    routeMocks.countOperators.mockResolvedValueOnce(1);
    await expect(setupStart(jsonEvent(setupBody()) as never)).rejects.toMatchObject({
      status: 409,
    });
    routeMocks.getEnvConfig.mockReturnValueOnce({
      adminBootstrapEnabled: false,
      adminBootstrapExpiresAt: futureDate(),
    });
    await expect(setupStart(jsonEvent(setupBody()) as never)).rejects.toMatchObject({
      status: 403,
    });
    routeMocks.getEnvConfig.mockReturnValueOnce({
      adminBootstrapEnabled: true,
      adminBootstrapExpiresAt: new Date(0),
    });
    await expect(setupStart(jsonEvent(setupBody()) as never)).rejects.toMatchObject({
      status: 403,
    });
    routeMocks.verifyBootstrapSecret.mockReturnValueOnce(false);
    await expect(setupStart(jsonEvent(setupBody()) as never)).rejects.toMatchObject({
      status: 403,
    });
    routeMocks.valkey.ping.mockRejectedValueOnce(new Error('down'));
    await expect(setupStart(jsonEvent(setupBody()) as never)).rejects.toThrow('down');
  });

  it('initial setup finish は zero-operator を再確認し role=admin で passkey 登録する', async () => {
    routeMocks.consumeChallenge.mockResolvedValueOnce({
      challenge: 'private',
      operatorId: 'bootstrap:sha:admin@example.test',
      email: 'admin@example.test',
      displayName: 'Admin',
    });
    const response = await setupFinish(
      jsonEvent({ challengeId: 'challenge-1', attestation: {} }) as never
    );
    expect(response.status).toBe(303);
    expect(routeMocks.createInitialAdminOperator).toHaveBeenCalledWith(
      expect.anything(),
      expect.objectContaining({ email: 'admin@example.test' })
    );
    routeMocks.consumeChallenge.mockResolvedValueOnce({
      challenge: 'private',
      operatorId: 'bootstrap:sha:admin@example.test',
      email: 'admin@example.test',
      displayName: 'Admin',
    });
    routeMocks.countOperators.mockResolvedValueOnce(1);
    await expect(
      setupFinish(jsonEvent({ challengeId: 'challenge-1', attestation: {} }) as never)
    ).rejects.toMatchObject({ status: 409 });
  });

  it('operator setup token flow は bad/expired/consumed/registered guard と brute-force 境界を検証する', async () => {
    routeMocks.listOperatorPasskeys.mockResolvedValueOnce([]);
    expect((await operatorSetupStart(jsonEvent({ setupToken: 'token' }) as never)).status).toBe(
      200
    );
    expect(routeMocks.enforcePreAuthRateLimit).toHaveBeenCalledWith(
      expect.anything(),
      'operator-setup',
      'sha:token',
      routeMocks.valkey
    );
    routeMocks.findOperatorBySetupToken.mockResolvedValueOnce(null);
    await expect(
      operatorSetupStart(jsonEvent({ setupToken: 'bad' }) as never)
    ).rejects.toMatchObject({ status: 401 });
    routeMocks.listOperatorPasskeys.mockResolvedValueOnce([passkey('registered')]);
    await expect(
      operatorSetupStart(jsonEvent({ setupToken: 'registered' }) as never)
    ).rejects.toMatchObject({ status: 401 });
    routeMocks.enforcePreAuthRateLimit.mockRejectedValueOnce(
      Object.assign(new Error('locked'), { status: 429 })
    );
    await expect(
      operatorSetupStart(jsonEvent({ setupToken: 'token' }) as never)
    ).rejects.toMatchObject({ status: 429 });
    routeMocks.listOperatorPasskeys.mockResolvedValueOnce([]);
    expect(
      (
        await operatorSetupFinish(
          jsonEvent({ challengeId: 'challenge-1', attestation: {} }) as never
        )
      ).status
    ).toBe(303);
    routeMocks.listOperatorPasskeys.mockResolvedValueOnce([passkey('registered')]);
    await expect(
      operatorSetupFinish(jsonEvent({ challengeId: 'challenge-1', attestation: {} }) as never)
    ).rejects.toMatchObject({ status: 401 });
    routeMocks.listOperatorPasskeys.mockResolvedValueOnce([]);
    routeMocks.consumeOperatorSetupToken.mockResolvedValueOnce(false);
    await expect(
      operatorSetupFinish(jsonEvent({ challengeId: 'challenge-1', attestation: {} }) as never)
    ).rejects.toMatchObject({ status: 401 });
    routeMocks.findOperatorById.mockResolvedValueOnce({ ...operator(), isActive: false });
    await expect(
      operatorSetupFinish(jsonEvent({ challengeId: 'challenge-1', attestation: {} }) as never)
    ).rejects.toMatchObject({ status: 401 });
  });

  it('auth integrity は duplicate credential / throttle / lock / Valkey unavailable / challenge binding mismatch を露出する', async () => {
    routeMocks.addOperatorPasskey.mockRejectedValueOnce(
      Object.assign(new Error('duplicate'), { status: 409 })
    );
    await expect(
      passkeyAddFinish(
        jsonEvent({ challengeId: 'challenge-1', attestation: {} }, authed()) as never
      )
    ).rejects.toMatchObject({ status: 409 });
    routeMocks.enforcePreAuthRateLimit.mockRejectedValueOnce(
      Object.assign(new Error('too many'), { status: 429 })
    );
    await expect(
      loginFinish(jsonEvent({ challengeId: 'challenge-1', assertion: { id: 'cred-1' } }) as never)
    ).rejects.toMatchObject({ status: 429 });
    routeMocks.enforcePreAuthRateLimit.mockRejectedValueOnce(
      Object.assign(new Error('too many'), { status: 429 })
    );
    await expect(setupStart(jsonEvent(setupBody()) as never)).rejects.toMatchObject({
      status: 429,
    });
    routeMocks.valkey.ping.mockRejectedValueOnce(new Error('down'));
    await expect(loginStart(jsonEvent({ email: 'admin@example.test' }) as never)).rejects.toThrow(
      'down'
    );
    routeMocks.consumeChallenge.mockResolvedValueOnce({
      challenge: 'private',
      operatorId: 'op-1',
      email: 'other@example.test',
    });
    await expect(
      loginFinish(jsonEvent({ challengeId: 'challenge-1', assertion: { id: 'cred-1' } }) as never)
    ).rejects.toMatchObject({ status: 401 });
    routeMocks.consumeChallenge.mockResolvedValueOnce({
      challenge: 'private',
      operatorId: 'other-op',
      email: 'other@example.test',
    });
    await expect(
      passkeyAddFinish(
        jsonEvent({ challengeId: 'challenge-1', attestation: {} }, authed()) as never
      )
    ).rejects.toMatchObject({ status: 403 });
  });
});

function jsonEvent(
  body: unknown,
  operator: App.Locals['operator'] = null,
  params: Record<string, string> = {}
) {
  // route handler が参照する JSON Request / locals / params / client address のみを作る。
  return {
    request: new Request('https://admin.example.test/api/admin/auth/test', {
      method: 'POST',
      body: JSON.stringify(body),
      headers: { 'content-type': 'application/json' },
    }),
    locals: { operator },
    params,
    getClientAddress: () => '192.0.2.10',
  };
}

function authed(): NonNullable<App.Locals['operator']> {
  // 認証済み route-level API の本人境界をテストするための locals を返す。
  return {
    id: 'op-1',
    email: 'admin@example.test',
    role: 'admin',
    sessionId: 'sess-1',
    jti: 'jti-1',
  };
}

function operator(
  input: Partial<{
    id: string;
    email: string;
    role: string;
    isActive: boolean;
    displayName: string;
  }> = {}
) {
  // route が参照する Operator model の最小 shape を deterministic に作る。
  return {
    id: 'op-1',
    email: 'admin@example.test',
    role: 'admin',
    isActive: true,
    displayName: 'Admin',
    ...input,
  };
}

function passkey(id: string) {
  // passkey route が JSON 化・検証に使う credential metadata の最小 shape を作る。
  return {
    id,
    operatorId: 'op-1',
    credentialHandle: 'cred-1',
    publicKey: new Uint8Array([1]),
    signCount: 1n,
    transports: [],
    createdAt: new Date('2025-01-01T00:00:00.000Z'),
  };
}

function attestation() {
  // 登録完了 route が保存する検証済み attestation result の最小 shape を作る。
  return {
    credentialHandle: 'cred-new',
    publicKey: new Uint8Array([1]),
    signCount: 0,
    aaguid: new Uint8Array([2]),
    backupEligible: false,
    backupState: false,
    transports: [],
  };
}

function setupBody() {
  // bootstrap start route の正常入力を共通化し、各エラー条件の差分だけをテストで上書きする。
  return { email: 'admin@example.test', displayName: 'Admin', bootstrapSecret: 'secret' };
}

function futureDate(): Date {
  // 期限判定を現在時刻に依存させないため、十分未来の固定日付を返す。
  return new Date('2999-01-01T00:00:00.000Z');
}
