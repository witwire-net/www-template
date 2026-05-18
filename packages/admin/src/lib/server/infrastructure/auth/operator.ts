import { randomBytes } from 'node:crypto';

import {
  generateAuthenticationOptions,
  verifyAuthenticationResponse,
} from '@simplewebauthn/server';
import { SignJWT, jwtVerify } from 'jose';

import { getEnvConfig } from '../config/env.js';
import { getPlatformConfig } from '../config/platform.js';

import type { VerifyAuthenticationResponseOpts } from '@simplewebauthn/server';
import type { Redis } from 'ioredis';

type AuthenticationResponseJSON = VerifyAuthenticationResponseOpts['response'];
type WebAuthnCredential = NonNullable<VerifyAuthenticationResponseOpts['credential']>;
type AuthenticatorTransportFuture = NonNullable<WebAuthnCredential['transports']>[number];

const SESSION_TTL_SECONDS = 86400;
const CHALLENGE_TTL_SECONDS = 300;
const CROFORDS_BASE32 = '0123456789ABCDEFGHJKMNPQRSTVWXYZ';

/**
 * ULID 風の一意識別子を生成する（タイムスタンプ + ランダム）。
 *
 * @returns 26 文字の ULID 形式文字列
 */
export function generateUlid(): string {
  return encodeTime(Date.now(), 10) + encodeRandom(16);
}

function encodeTime(timestamp: number, len: number): string {
  let str = '';
  for (let i = len; i > 0; i--) {
    const mod = timestamp % 32;
    str = CROFORDS_BASE32.charAt(mod) + str;
    timestamp = (timestamp - mod) / 32;
  }
  return str;
}

function encodeRandom(len: number): string {
  let str = '';
  const bytes = randomBytes(len);
  for (const byte of bytes) {
    str += CROFORDS_BASE32.charAt(byte % 32);
  }
  return str;
}

/**
 * Admin WebAuthn challenge の用途種別。
 *
 * login は assertion、setup / operator-setup / passkey-add は attestation の検証で使用する。
 */
export type AdminChallengeType = 'login' | 'setup' | 'operator-setup' | 'passkey-add';

interface ChallengeRecord {
  type: AdminChallengeType;
  operatorId: string;
  email: string;
  displayName?: string;
  challenge: string;
  createdAt: string;
}

interface SessionRecord {
  operatorId: string;
  email: string;
  role: string;
  jti: string;
  createdAt: string;
}

/**
 * WebAuthn challenge を生成し Valkey に保存する。
 *
 * @param input challenge 生成に必要なメタデータ
 * @param valkey Valkey 接続インスタンス
 * @returns challengeId と WebAuthn options
 */
export async function generateChallenge(
  input: { type: AdminChallengeType; operatorId: string; email: string },
  valkey: Redis
): Promise<{
  challengeId: string;
  options: {
    challenge: string;
    rpId: string;
    allowCredentials: { id: string; type: 'public-key' }[];
    userVerification: 'required';
  };
}> {
  const { adminRpId } = getPlatformConfig();
  const challengeId = generateUlid();

  const options = await generateAuthenticationOptions({
    rpID: adminRpId,
    allowCredentials: [],
    userVerification: 'required',
  });

  const record: ChallengeRecord = {
    type: input.type,
    operatorId: input.operatorId,
    email: input.email,
    challenge: options.challenge,
    createdAt: new Date().toISOString(),
  };

  await valkey.setex(
    `admin:webauthn:challenge:${challengeId}`,
    CHALLENGE_TTL_SECONDS,
    JSON.stringify(record)
  );

  return {
    challengeId,
    options: {
      challenge: options.challenge,
      rpId: adminRpId,
      allowCredentials: [],
      userVerification: 'required',
    },
  };
}

/**
 * Valkey から challenge を取得・削除し、型とオペレーター紐付けを検証する。
 *
 * @param challengeId challenge 識別子
 * @param expectedType 期待される challenge タイプ
 * @param valkey Valkey 接続インスタンス
 * @returns challenge とオペレーター情報
 * @throws Error challenge が見つからないか検証に失敗した場合
 */
export async function consumeChallenge(
  challengeId: string,
  expectedType: AdminChallengeType,
  valkey: Redis
): Promise<{ challenge: string; operatorId: string; email: string; displayName?: string }> {
  const raw = await valkey.getdel(`admin:webauthn:challenge:${challengeId}`);
  if (raw === null || raw === '') {
    throw new Error('Challenge not found or expired');
  }

  const record = JSON.parse(raw) as ChallengeRecord;
  if (record.type !== expectedType) {
    throw new Error('Challenge type mismatch');
  }

  return {
    challenge: record.challenge,
    operatorId: record.operatorId,
    email: record.email,
    displayName: record.displayName,
  };
}

/**
 * オペレーター session を Valkey に作成する。
 *
 * @param operator 認証済みオペレーター
 * @param valkey Valkey 接続インスタンス
 * @returns sessionId と jti
 */
export async function createOperatorSession(
  operator: { id: string; email: string; role: string },
  valkey: Redis
): Promise<{ sessionId: string; jti: string }> {
  const sessionId = generateUlid();
  const jti = generateUlid();

  const record: SessionRecord = {
    operatorId: operator.id,
    email: operator.email,
    role: operator.role,
    jti,
    createdAt: new Date().toISOString(),
  };

  await valkey.setex(`admin:session:${sessionId}`, SESSION_TTL_SECONDS, JSON.stringify(record));
  return { sessionId, jti };
}

/**
 * 指定 session を Valkey から削除する。
 *
 * @param sessionId session 識別子
 * @param valkey Valkey 接続インスタンス
 */
export async function revokeOperatorSession(sessionId: string, valkey: Redis): Promise<void> {
  await valkey.del(`admin:session:${sessionId}`);
}

/**
 * JWT cookie と Valkey session の両方を検証する。
 *
 * @param token JWT 文字列
 * @param valkey Valkey 接続インスタンス
 * @returns 検証済みオペレーター情報、または null
 */
export async function verifyOperatorSession(
  token: string,
  valkey: Redis
): Promise<{
  operatorId: string;
  email: string;
  role: string;
  sessionId: string;
  jti: string;
} | null> {
  const { jwtSecret } = getEnvConfig();
  try {
    const { payload } = await jwtVerify(token, new TextEncoder().encode(jwtSecret), {
      clockTolerance: 60,
    });
    const sessionId = payload.sessionId;
    const jti = payload.jti;
    if (
      typeof sessionId !== 'string' ||
      sessionId === '' ||
      typeof jti !== 'string' ||
      jti === ''
    ) {
      return null;
    }

    const raw = await valkey.get(`admin:session:${sessionId}`);
    if (raw === null || raw === '') return null;

    const record = JSON.parse(raw) as SessionRecord;
    if (record.jti !== jti) return null;

    return {
      operatorId: record.operatorId,
      email: record.email,
      role: record.role,
      sessionId,
      jti,
    };
  } catch {
    return null;
  }
}

/**
 * WebAuthn assertion を検証する。
 *
 * @param assertion ブラウザからの認証応答
 * @param expectedChallenge 期待される challenge
 * @param credential DB に保存された credential 情報
 * @param origin 期待される origin
 * @param rpId 期待される RP ID
 * @returns 更新後の signCount
 * @throws Error 検証に失敗した場合
 */
export async function verifyAssertion(
  assertion: AuthenticationResponseJSON,
  expectedChallenge: string,
  credential: {
    credential_handle: string;
    public_key: Buffer;
    sign_count: bigint;
    transports: unknown;
  },
  origin: string,
  rpId: string
): Promise<{ newSignCount: number }> {
  const result = await verifyAuthenticationResponse({
    response: assertion,
    expectedChallenge,
    expectedOrigin: origin,
    expectedRPID: rpId,
    credential: {
      id: credential.credential_handle,
      publicKey: new Uint8Array(credential.public_key),
      counter: Number(credential.sign_count),
      transports: ((credential.transports as string[] | null) ??
        []) as AuthenticatorTransportFuture[],
    },
    requireUserVerification: true,
  });
  if (!result.verified) {
    throw new Error('Authentication verification failed');
  }

  const newSignCount = result.authenticationInfo.newCounter;
  if (newSignCount < Number(credential.sign_count)) {
    throw new Error('Sign count decreased');
  }

  return { newSignCount };
}

/**
 * オペレーター情報から JWT を署名する。
 *
 * @param operator オペレーター情報
 * @param session session メタデータ
 * @returns 署名済み JWT 文字列
 */
export async function signOperatorJwt(
  operator: { id: string; email: string; role: string },
  session: { sessionId: string; jti: string }
): Promise<string> {
  const { jwtSecret } = getEnvConfig();
  const secret = new TextEncoder().encode(jwtSecret);
  return new SignJWT({
    sub: operator.id,
    email: operator.email,
    role: operator.role,
    sessionId: session.sessionId,
    jti: session.jti,
  })
    .setProtectedHeader({ alg: 'HS256' })
    .setIssuedAt()
    .setExpirationTime('24h')
    .sign(secret);
}

/**
 * JWT を検証してペイロードを返す。
 *
 * @param token JWT 文字列
 * @returns デコード済みペイロード、または null
 */
export async function verifyOperatorJwt(token: string): Promise<Record<string, unknown> | null> {
  const { jwtSecret } = getEnvConfig();
  try {
    const { payload } = await jwtVerify(token, new TextEncoder().encode(jwtSecret), {
      clockTolerance: 60,
    });
    return payload as Record<string, unknown>;
  } catch {
    return null;
  }
}

/**
 * セッション cookie 文字列を構築する。
 *
 * @param token JWT 文字列
 * @returns Set-Cookie 用の cookie 文字列
 */
export function createSessionCookie(token: string): string {
  const { isProduction } = getPlatformConfig();
  const parts = [
    `admin_session=${token}`,
    'HttpOnly',
    `SameSite=Lax`,
    'Path=/',
    `Max-Age=${String(SESSION_TTL_SECONDS)}`,
  ];
  if (isProduction) {
    parts.push('Secure');
  }
  return parts.join('; ');
}

/**
 * セッション cookie をクリアする文字列を構築する。
 *
 * @returns クリア用 cookie 文字列
 */
export function clearSessionCookie(): string {
  const { isProduction } = getPlatformConfig();
  const parts = [`admin_session=`, 'HttpOnly', `SameSite=Lax`, 'Path=/', 'Max-Age=0'];
  if (isProduction) {
    parts.push('Secure');
  }
  return parts.join('; ');
}
