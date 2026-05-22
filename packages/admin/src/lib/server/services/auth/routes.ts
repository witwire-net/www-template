import { createHash, timingSafeEqual } from 'node:crypto';
import { createRequire } from 'node:module';

import { error, json } from '@sveltejs/kit';
import { z } from 'zod';

import {
  createOperatorSession,
  createSessionCookie,
  signOperatorJwt,
} from '$lib/server/infrastructure/auth/operator.js';
import { getAdminValkey } from '$lib/server/infrastructure/auth/valkey.js';
import { getAdminBootstrapConfig } from '$lib/server/infrastructure/config/env.js';
import { getAdminPrisma } from '$lib/server/infrastructure/db/prisma.js';
import * as operatorModel from '$lib/server/models/operators.js';
import * as passkeyModel from '$lib/server/models/passkeys.js';
import { createOperatorSchema, loginEmailSchema } from '$lib/server/models/schemas.js';

import type { Operator, PasskeyInfo } from '$lib/server/models/types.js';
import type { RequestEvent } from '@sveltejs/kit';
import type { Redis } from 'ioredis';

const RATE_LIMIT_TTL_SECONDS = 300;
const RATE_LIMIT_MAX_ATTEMPTS = 8;
const nodeRequire = createRequire(import.meta.url);
const { compareSync } = nodeRequire('bcryptjs') as {
  compareSync: (data: string, encrypted: string) => boolean;
};

/**
 * 認証 route 共通の no-store ヘッダー。
 */
export const NO_STORE_HEADERS = { 'Cache-Control': 'no-store' } as const;

/**
 * ログイン開始リクエストの検証 schema。
 */
export const loginStartRequestSchema = z.object({ email: loginEmailSchema });

/**
 * challengeId を持つ finish リクエストの検証 schema。
 */
export const challengeFinishRequestSchema = z.object({
  challengeId: z.string().min(1).max(128),
  assertion: z.unknown(),
});

/**
 * 登録 challengeId を持つ finish リクエストの検証 schema。
 */
export const registrationFinishRequestSchema = z.object({
  challengeId: z.string().min(1).max(128),
  attestation: z.unknown(),
});

/**
 * 初回 setup start リクエストの検証 schema。
 */
export const setupStartRequestSchema = createOperatorSchema
  .pick({ email: true, displayName: true })
  .extend({ bootstrapSecret: z.string().min(1).max(512) });

/**
 * 追加オペレーター setup start リクエストの検証 schema。
 */
export const operatorSetupStartRequestSchema = z.object({ setupToken: z.string().min(1).max(512) });

/**
 * Request body を JSON として読み、Zod schema で検証する。
 *
 * @param event SvelteKit request event
 * @param schema 入力検証 schema
 * @returns 検証済み入力
 */
export async function parseJson<T>(event: RequestEvent, schema: z.ZodType<T>): Promise<T> {
  // JSON parse と schema validation をまとめ、route 側に未検証入力を残さない。
  const body: unknown = await event.request.json().catch(() => null);
  const parsed = schema.safeParse(body);
  if (!parsed.success) {
    fail(400, 'Invalid request');
  }
  return parsed.data;
}

/**
 * SvelteKit の HTTP error を route/service から一貫して発生させる。
 *
 * @param status HTTP status code
 * @param message 公開してよい短いエラーメッセージ
 */
export function fail(status: number, message: string): never {
  // `throw error(...)` を各所に散らさず、lint と公開エラーメッセージを中央で揃える。
  error(status, message);
}

/**
 * 現在の Admin Valkey 接続を ping して fail-close 条件を確認する。
 *
 * @returns 利用可能な Valkey 接続
 */
export async function requireValkey(): Promise<Redis> {
  const valkey = getAdminValkey();
  // ioredis の lazy client は ping で未接続時の接続開始と接続済み時の疎通確認を両立し、ready 後の再 connect 例外を避ける。
  await valkey.ping().catch(() => {
    fail(503, 'Authentication temporarily unavailable');
  });
  return valkey;
}

/**
 * pre-auth route の IP + fingerprint rate limit を Valkey で実施する。
 *
 * @param event SvelteKit request event
 * @param purpose rate-limit namespace
 * @param fingerprint 秘密値を SHA-256 で短縮した識別子
 * @param valkey Admin Valkey 接続
 */
export async function enforcePreAuthRateLimit(
  event: RequestEvent,
  purpose: string,
  fingerprint: string,
  valkey: Redis
): Promise<void> {
  // IP とトークン fingerprint を組み合わせ、単一 IP と単一 secret の総当たり双方を抑止する。
  const ip = event.getClientAddress();
  const key = `admin:auth:rate:${purpose}:${sha256(ip)}:${fingerprint}`;
  const count = await valkey.incr(key).catch(() => {
    fail(503, 'Authentication temporarily unavailable');
  });
  // 初回試行時だけ TTL を付与し、時間窓が無限に延びないようにする。
  if (count === 1) {
    await valkey.expire(key, RATE_LIMIT_TTL_SECONDS).catch(() => {
      fail(503, 'Authentication temporarily unavailable');
    });
  }
  if (count > RATE_LIMIT_MAX_ATTEMPTS) {
    fail(429, 'Too many attempts');
  }
}

/**
 * 秘密値を response や key に直接出さないため SHA-256 fingerprint に変換する。
 *
 * @param value 入力値
 * @returns hex 形式の SHA-256 digest
 */
export function sha256(value: string): string {
  return createHash('sha256').update(value).digest('hex');
}

/**
 * bootstrap secret を timing-safe に検証する。
 *
 * @param raw 入力 secret
 * @returns 設定 secret と一致する場合 true
 */
export function verifyBootstrapSecret(raw: string): boolean {
  // bcrypt hash はサービス層と同じ環境値を使い、平文 secret を永続化・ログ出力しない。
  const { adminBootstrapSecretHash } = getAdminBootstrapConfig();
  return compareSync(raw, adminBootstrapSecretHash);
}

/**
 * operator setup token に対応するオペレーターを検索する。
 *
 * @param rawToken 入力された one-time setup token
 * @returns 検証済みオペレーター、または null
 */
export async function findOperatorBySetupToken(rawToken: string): Promise<Operator | null> {
  // hash は bcrypt なので DB 側で検索せず、期限内・未登録候補にだけ compare を行う。
  const adminPrisma = getAdminPrisma();
  const now = new Date();
  const operators = await operatorModel.listOperators(adminPrisma);
  for (const operator of operators) {
    if (
      !operator.isActive ||
      operator.setupTokenHash === null ||
      operator.setupTokenExpiresAt === null
    )
      continue;
    if (operator.setupTokenExpiresAt < now) continue;
    if (compareSync(rawToken, operator.setupTokenHash)) return operator;
  }
  return null;
}

/**
 * 認証済みオペレーターを App.Locals から取得する。
 *
 * @param event SvelteKit request event
 * @returns 認証済み operator locals
 */
export function requireAuthenticatedOperator(
  event: RequestEvent
): NonNullable<App.Locals['operator']> {
  // hooks 実装後は locals.operator が唯一の認証済み境界になる。未設定時は route 側で 401 を返す。
  if (event.locals.operator === null) {
    fail(401, 'Unauthorized');
  }
  return event.locals.operator;
}

/**
 * 認証済み session cookie 付き 303 Response を作る。
 *
 * @param operator 認証済みオペレーター
 * @param valkey Admin Valkey 接続
 * @returns Set-Cookie と Location を含む redirect response
 */
export async function sessionRedirectResponse(
  operator: Operator,
  valkey: Redis
): Promise<Response> {
  // session と JWT を一貫して作成し、cookie helper の属性を必ず利用する。
  const session = await createOperatorSession(
    { id: operator.id, email: operator.email, role: operator.role },
    valkey
  );
  const token = await signOperatorJwt(
    { id: operator.id, email: operator.email, role: operator.role },
    session
  );
  return new Response(null, {
    status: 303,
    headers: { ...NO_STORE_HEADERS, Location: '/', 'Set-Cookie': createSessionCookie(token) },
  });
}

/**
 * PasskeyInfo を JSON response 用の安全な形に整える。
 *
 * @param passkey DB 由来の passkey 情報
 * @returns 公開可能な passkey メタデータ
 */
export function serializePasskey(passkey: PasskeyInfo): Record<string, unknown> {
  // public_key や credential handle は認証 material なので一覧 API では返さない。
  return {
    id: passkey.id,
    createdAt: passkey.createdAt.toISOString(),
    backupEligible: passkey.backupEligible,
    backupState: passkey.backupState,
    transports: passkey.transports,
  };
}

/**
 * passkey 一覧を no-store JSON で返す。
 *
 * @param operatorId 対象 operator ID
 */
export async function passkeyListResponse(operatorId: string): Promise<Response> {
  // 一覧はログイン済み本人の passkey metadata だけに限定する。
  const passkeys = await passkeyModel.listOperatorPasskeys(getAdminPrisma(), operatorId);
  return json({ passkeys: passkeys.map(serializePasskey) }, { headers: NO_STORE_HEADERS });
}

/**
 * 文字列の定数時間比較を行う。
 *
 * @param left 比較対象 A
 * @param right 比較対象 B
 * @returns 同一なら true
 */
export function safeEqual(left: string, right: string): boolean {
  const leftBuffer = Buffer.from(left);
  const rightBuffer = Buffer.from(right);
  if (leftBuffer.byteLength !== rightBuffer.byteLength) return false;
  return timingSafeEqual(leftBuffer, rightBuffer);
}
