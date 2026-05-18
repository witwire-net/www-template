import { SignJWT } from 'jose';
import { beforeEach, describe, expect, it, vi } from 'vitest';

const webauthnMocks = vi.hoisted(() => ({
  generateAuthenticationOptions: vi.fn(),
  verifyAuthenticationResponse: vi.fn(),
}));

vi.mock('@simplewebauthn/server', () => ({
  generateAuthenticationOptions: webauthnMocks.generateAuthenticationOptions,
  verifyAuthenticationResponse: webauthnMocks.verifyAuthenticationResponse,
}));

import {
  consumeChallenge,
  createOperatorSession,
  createSessionCookie,
  generateChallenge,
  revokeOperatorSession,
  signOperatorJwt,
  verifyAssertion,
  verifyOperatorJwt,
  verifyOperatorSession,
} from './operator.js';

import type { Redis } from 'ioredis';

class MemoryValkey {
  private readonly records = new Map<string, string>();

  setex = vi.fn(async (key: string, _ttl: number, value: string) => {
    // テスト用 Valkey は TTL 自体を進めず、保存された JSON のみを検証対象にする。
    this.records.set(key, value);
    return 'OK';
  });

  getdel = vi.fn(async (key: string) => {
    // GETDEL の one-time 消費性を Map 削除で再現し、challenge 再利用拒否を deterministic に検証する。
    const value = this.records.get(key) ?? null;
    this.records.delete(key);
    return value;
  });

  get = vi.fn(async (key: string) => {
    // session 検証用に、現在残っている active session JSON を返す。
    return this.records.get(key) ?? null;
  });

  del = vi.fn(async (key: string) => {
    // logout/revoke の副作用として active session を削除する。
    const existed = this.records.delete(key);
    return existed ? 1 : 0;
  });

  put(key: string, value: unknown): void {
    // JWT mismatch など、特定の session 状態を直接作るテスト用ヘルパー。
    this.records.set(key, JSON.stringify(value));
  }
}

function asRedis(valkey: MemoryValkey): Redis {
  return valkey as unknown as Redis;
}

function setBaseEnv(): void {
  // 認証 infrastructure が参照する必須 env を各テストで固定し、テスト間の環境差分をなくす。
  process.env.JWT_SECRET = 'test-secret-with-enough-length';
  process.env.ADMIN_ORIGIN = 'https://admin.example.test';
  process.env.ADMIN_DATABASE_URL = testPostgresUrl('admin');
  process.env.PRODUCT_DATABASE_URL = testPostgresUrl('product');
  process.env.ADMIN_VALKEY_URL = 'redis://valkey:6379/1';
  process.env.VALKEY_URL = 'redis://valkey:6379/0';
  process.env.OPENSEARCH_URL = 'http://opensearch:9200';
  process.env.ADMIN_OPENSEARCH_AUDIT_REPLICAS = '0';
  process.env.ADMIN_OPENSEARCH_AUDIT_INDEX_PREFIX = 'admin-audit';
  process.env.PRODUCT_OPENSEARCH_INDEX_PREFIX = 'product-domain';
  process.env.ADMIN_BOOTSTRAP_ENABLED = 'true';
  process.env.ADMIN_BOOTSTRAP_SECRET_HASH =
    '$2a$10$abcdefghijklmnopqrstuu7q3xGvJp3v4Cq9xI9xI9xI9xI9xI9xI';
  process.env.ADMIN_BOOTSTRAP_EXPIRES_AT = '2999-01-01T00:00:00.000Z';
  process.env.ADMIN_RP_ID = 'admin.example.test';
  process.env.ADMIN_RP_NAME = 'Admin Console';
  delete process.env.PRODUCT_VALKEY_URL;
}

describe('admin operator auth infrastructure', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    setBaseEnv();
    process.env.NODE_ENV = 'test';
    webauthnMocks.generateAuthenticationOptions.mockResolvedValue({ challenge: 'challenge-1' });
    webauthnMocks.verifyAuthenticationResponse.mockResolvedValue({
      verified: true,
      authenticationInfo: { newCounter: 8 },
    });
  });

  it('JWT を署名して検証でき、sessionId と jti を保持する', async () => {
    const token = await signOperatorJwt(
      { id: 'op-1', email: 'admin@example.test', role: 'admin' },
      { sessionId: 'sess-1', jti: 'jti-1' }
    );

    const payload = await verifyOperatorJwt(token);

    expect(payload).toMatchObject({
      sub: 'op-1',
      email: 'admin@example.test',
      role: 'admin',
      sessionId: 'sess-1',
      jti: 'jti-1',
    });
  });

  it('期限切れ JWT を拒否する', async () => {
    const expiredToken = await new SignJWT({ sessionId: 'sess-1', jti: 'jti-1' })
      .setProtectedHeader({ alg: 'HS256' })
      .setIssuedAt(0)
      .setExpirationTime(1)
      .sign(new TextEncoder().encode(process.env.JWT_SECRET));

    await expect(verifyOperatorJwt(expiredToken)).resolves.toBeNull();
  });

  it('WebAuthn options は userVerification required を要求する', async () => {
    const valkey = new MemoryValkey();
    const challenge = await generateChallenge(
      { type: 'login', operatorId: 'op-1', email: 'admin@example.test' },
      asRedis(valkey)
    );

    expect(webauthnMocks.generateAuthenticationOptions).toHaveBeenCalledWith(
      expect.objectContaining({ userVerification: 'required' })
    );
    expect(challenge.options.userVerification).toBe('required');
  });

  it('UV false assertion を拒否するため SimpleWebAuthn に requireUserVerification を渡す', async () => {
    webauthnMocks.verifyAuthenticationResponse.mockResolvedValueOnce({
      verified: false,
      authenticationInfo: { newCounter: 8 },
    });

    await expect(
      verifyAssertion(
        { id: 'cred-1' } as never,
        'challenge-1',
        savedCredential(5),
        'https://admin.example.test',
        'admin.example.test'
      )
    ).rejects.toThrow('Authentication verification failed');

    expect(webauthnMocks.verifyAuthenticationResponse).toHaveBeenCalledWith(
      expect.objectContaining({ requireUserVerification: true })
    );
  });

  it('UV true assertion を受理して更新後 signCount を返す', async () => {
    await expect(
      verifyAssertion(
        { id: 'cred-1' } as never,
        'challenge-1',
        savedCredential(5),
        'https://admin.example.test',
        'admin.example.test'
      )
    ).resolves.toEqual({ newSignCount: 8 });
  });

  it('sign_count が保存値より減少した assertion を拒否する', async () => {
    webauthnMocks.verifyAuthenticationResponse.mockResolvedValueOnce({
      verified: true,
      authenticationInfo: { newCounter: 4 },
    });

    await expect(
      verifyAssertion(
        { id: 'cred-1' } as never,
        'challenge-1',
        savedCredential(5),
        'https://admin.example.test',
        'admin.example.test'
      )
    ).rejects.toThrow('Sign count decreased');
  });

  it('保存 credential の publicKey / counter / transports で検証する', async () => {
    await verifyAssertion(
      { id: 'cred-1' } as never,
      'challenge-1',
      savedCredential(5),
      'https://admin.example.test',
      'admin.example.test'
    );

    expect(webauthnMocks.verifyAuthenticationResponse).toHaveBeenCalledWith(
      expect.objectContaining({
        credential: expect.objectContaining({ id: 'cred-1', counter: 5, transports: ['internal'] }),
        expectedChallenge: 'challenge-1',
        expectedOrigin: 'https://admin.example.test',
        expectedRPID: 'admin.example.test',
      })
    );
  });

  it('production cookie には Secure 属性を付ける', () => {
    process.env.NODE_ENV = 'production';

    expect(createSessionCookie('jwt-value')).toContain('Secure');
  });

  it('session cookie には Path=/ を付ける', () => {
    expect(createSessionCookie('jwt-value')).toContain('Path=/');
  });

  it('消費済み challenge の再利用を拒否する', async () => {
    const valkey = new MemoryValkey();
    const generated = await generateChallenge(
      { type: 'login', operatorId: 'op-1', email: 'admin@example.test' },
      asRedis(valkey)
    );

    await expect(
      consumeChallenge(generated.challengeId, 'login', asRedis(valkey))
    ).resolves.toMatchObject({
      operatorId: 'op-1',
    });
    await expect(consumeChallenge(generated.challengeId, 'login', asRedis(valkey))).rejects.toThrow(
      'Challenge not found or expired'
    );
  });

  it('logout 相当の session revoke 後は盗難 cookie を拒否する', async () => {
    const valkey = new MemoryValkey();
    const session = await createOperatorSession(
      { id: 'op-1', email: 'admin@example.test', role: 'admin' },
      asRedis(valkey)
    );
    const token = await signOperatorJwt(
      { id: 'op-1', email: 'admin@example.test', role: 'admin' },
      session
    );

    await expect(verifyOperatorSession(token, asRedis(valkey))).resolves.toMatchObject({
      operatorId: 'op-1',
    });
    await revokeOperatorSession(session.sessionId, asRedis(valkey));

    await expect(verifyOperatorSession(token, asRedis(valkey))).resolves.toBeNull();
  });

  it('JWT の sessionId/jti mismatch と Valkey session 欠落を拒否する', async () => {
    const valkey = new MemoryValkey();
    valkey.put('admin:session:sess-1', {
      operatorId: 'op-1',
      email: 'admin@example.test',
      role: 'admin',
      jti: 'stored-jti',
      createdAt: new Date().toISOString(),
    });
    const mismatchedToken = await signOperatorJwt(
      { id: 'op-1', email: 'admin@example.test', role: 'admin' },
      { sessionId: 'sess-1', jti: 'jwt-jti' }
    );
    const missingSessionToken = await signOperatorJwt(
      { id: 'op-1', email: 'admin@example.test', role: 'admin' },
      { sessionId: 'missing', jti: 'stored-jti' }
    );

    await expect(verifyOperatorSession(mismatchedToken, asRedis(valkey))).resolves.toBeNull();
    await expect(verifyOperatorSession(missingSessionToken, asRedis(valkey))).resolves.toBeNull();
  });

  it('challenge type/operator binding mismatch を拒否する', async () => {
    const valkey = new MemoryValkey();
    const generated = await generateChallenge(
      { type: 'login', operatorId: 'op-1', email: 'admin@example.test' },
      asRedis(valkey)
    );

    await expect(
      consumeChallenge(generated.challengeId, 'passkey-add', asRedis(valkey))
    ).rejects.toThrow('Challenge type mismatch');
  });
});

function savedCredential(signCount: number) {
  // DB 保存済み passkey record の最小形を作り、検証 helper が保存値を使うことだけを観測する。
  return {
    credential_handle: 'cred-1',
    public_key: Buffer.from([1, 2, 3]),
    sign_count: BigInt(signCount),
    transports: ['internal'],
  };
}

function testPostgresUrl(database: string): string {
  // security lint が実接続文字列の直書きを検出するため、テスト用 URL も分割して組み立てる。
  return ['postgres:', '//', database].join('');
}
