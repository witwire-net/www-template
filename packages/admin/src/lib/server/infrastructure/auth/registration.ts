import { createHash } from 'node:crypto';

import { generateRegistrationOptions, verifyRegistrationResponse } from '@simplewebauthn/server';

import { getEnvConfig } from '../config/env.js';
import { getPlatformConfig } from '../config/platform.js';

import { generateUlid, type AdminChallengeType } from './operator.js';

import type {
  GenerateRegistrationOptionsOpts,
  VerifyRegistrationResponseOpts,
} from '@simplewebauthn/server';
import type { Redis } from 'ioredis';

type RegistrationResponseJSON = VerifyRegistrationResponseOpts['response'];

const CHALLENGE_TTL_SECONDS = 300;
const WEBAUTHN_USER_HANDLE_MAX_BYTES = 64;

/**
 * WebAuthn 登録 challenge を作成し、認証用 challenge と同じ Valkey namespace に保存する。
 *
 * @param input 登録対象のオペレーターと challenge 種別
 * @param valkey 共有 Valkey infrastructure の Admin 用 logical DB 接続
 * @returns challengeId とブラウザへ返す登録 options
 */
export async function generateRegistrationChallenge(
  input: {
    type: Extract<AdminChallengeType, 'setup' | 'operator-setup' | 'passkey-add'>;
    operatorId: string;
    email: string;
    displayName: string;
    excludeCredentialIds: string[];
  },
  valkey: Redis
): Promise<{ challengeId: string; options: PublicKeyCredentialCreationOptionsJSON }> {
  // WebAuthn RP 設定は実行環境から取得し、Host ヘッダー由来の値を混ぜない。
  const { adminRpId, adminRpName } = getPlatformConfig();
  // challengeId はログや URL に露出しても秘密にならない一時 ID として ULID 形式で生成する。
  const challengeId = generateUlid();
  // SimpleWebAuthn に登録 options を生成させ、userVerification を必須化する。
  const options = await generateRegistrationOptions({
    rpID: adminRpId,
    rpName: adminRpName,
    userID: deriveUserHandle(input.operatorId),
    userName: input.email,
    userDisplayName: input.displayName,
    excludeCredentials: input.excludeCredentialIds.map((id) => ({ id, type: 'public-key' })),
    authenticatorSelection: {
      residentKey: 'required',
      requireResidentKey: true,
      userVerification: 'required',
    },
  } satisfies GenerateRegistrationOptionsOpts);
  // finish 側で challenge / operator / type binding を検証するため、必要最小限のメタデータだけ保存する。
  await valkey.setex(
    `admin:webauthn:challenge:${challengeId}`,
    CHALLENGE_TTL_SECONDS,
    JSON.stringify({
      type: input.type,
      operatorId: input.operatorId,
      email: input.email,
      displayName: input.displayName,
      challenge: options.challenge,
      createdAt: new Date().toISOString(),
    })
  );
  return { challengeId, options };
}

function deriveUserHandle(operatorId: string): Uint8Array {
  // WebAuthn user handle は最大 64 bytes なので、bootstrap 用 decoy ID 等が長い場合は SHA-256 digest に短縮する。
  const encoded = new TextEncoder().encode(operatorId);
  if (encoded.byteLength <= WEBAUTHN_USER_HANDLE_MAX_BYTES) return encoded;
  return createHash('sha256').update(operatorId).digest();
}

/**
 * WebAuthn attestation 応答を検証し、DB 保存用 credential 情報へ正規化する。
 *
 * @param response ブラウザから受け取った登録応答
 * @param expectedChallenge Valkey challenge record の challenge
 * @returns passkey 登録モデルへ渡す credential 情報
 */
export async function verifyAttestation(
  response: RegistrationResponseJSON,
  expectedChallenge: string
): Promise<{
  credentialHandle: string;
  publicKey: Uint8Array;
  signCount: number;
  aaguid: Uint8Array;
  backupEligible: boolean;
  backupState: boolean;
  transports: unknown;
}> {
  // Origin / RP ID は設定値だけを信頼し、リクエスト由来の Host からは組み立てない。
  const { adminOrigin } = getEnvConfig();
  const { adminRpId } = getPlatformConfig();
  // SimpleWebAuthn の検証で challenge・origin・RP ID・User Verification をまとめて確認する。
  const result = await verifyRegistrationResponse({
    response,
    expectedChallenge,
    expectedOrigin: adminOrigin,
    expectedRPID: adminRpId,
    requireUserVerification: true,
  });
  if (!result.verified || result.registrationInfo === undefined) {
    throw new Error('Registration verification failed');
  }
  // DB には検証済み credential のみを保存し、クライアント入力をそのまま永続化しない。
  const { credential, aaguid, credentialBackedUp, credentialDeviceType } = result.registrationInfo;
  return {
    credentialHandle: credential.id,
    publicKey: credential.publicKey,
    signCount: credential.counter,
    aaguid: new TextEncoder().encode(aaguid),
    backupEligible: credentialDeviceType === 'multiDevice',
    backupState: credentialBackedUp,
    transports: credential.transports ?? [],
  };
}
