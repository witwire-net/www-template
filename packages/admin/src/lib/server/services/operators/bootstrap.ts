import { createRequire } from 'node:module';

import {
  createOperatorSession,
  createSessionCookie,
  signOperatorJwt,
} from '../../infrastructure/auth/operator.js';
import { getAdminBootstrapConfig } from '../../infrastructure/config/env.js';
import * as operatorModel from '../../models/operators.js';
import * as passkeyModel from '../../models/passkeys.js';
import { ServiceError } from '../errors.js';

import type { Operator } from '../../models/types.js';
import type { PrismaClient as AdminPrismaClient } from '.prisma/admin-client';
import type { Redis } from 'ioredis';

const nodeRequire = createRequire(import.meta.url);
const { compareSync } = nodeRequire('bcryptjs') as {
  compareSync: (value: string, hash: string) => boolean;
};

/**
 * WebAuthn 登録応答から抽出した検証済み credential 情報。
 */
export interface VerifiedWebAuthnCredential {
  credentialHandle: string;
  publicKey: Uint8Array;
  signCount: number;
  aaguid: Uint8Array;
  backupEligible: boolean;
  backupState: boolean;
  transports: unknown;
}

/**
 * ブートストラップ入力。
 */
export interface BootstrapAdminInput {
  adminPrisma: AdminPrismaClient;
  valkey: Redis;
  bootstrapSecret: string;
  email: string;
  displayName: string;
  webAuthnCredential: VerifiedWebAuthnCredential;
}

/**
 * ブートストラップ結果。
 */
export interface BootstrapAdminResult {
  operator: Operator;
  cookie: string;
}

/**
 * 初回 admin オペレーターをブートストラップで作成する。
 *
 * 事前条件の検証:
 * 1. 既存オペレーター数が 0 であること
 * 2. Admin TOML の bootstrap.enabled が true であること
 * 3. bootstrapSecret が Admin TOML の bootstrap.secret_hash と一致すること
 * 4. Admin TOML の bootstrap.expires_at が期限切れでないこと
 *
 * 検証成功後、同一トランザクション内で:
 * - admin ロールのオペレーターを作成
 * - 検証済み WebAuthn passkey を登録
 *
 * その後:
 * - Valkey にセッションを作成
 * - JWT を署名
 * - session cookie を構築して返却
 *
 * @param input ブートストラップ入力
 * @returns 作成されたオペレーターと session cookie
 */
export async function bootstrapAdmin(input: BootstrapAdminInput): Promise<BootstrapAdminResult> {
  // 1. 既存オペレーター数チェック
  const operatorCount = await operatorModel.countOperators(input.adminPrisma);
  if (operatorCount !== 0) {
    throw new ServiceError(
      'Bootstrap requires no existing operators',
      409,
      'BOOTSTRAP_NOT_ALLOWED'
    );
  }

  // 2-4. Admin TOML 設定によるブートストラップ有効性検証
  const env = getAdminBootstrapConfig();
  if (!env.adminBootstrapEnabled) {
    throw new ServiceError('Bootstrap is not enabled', 403, 'BOOTSTRAP_DISABLED');
  }
  if (!compareSync(input.bootstrapSecret, env.adminBootstrapSecretHash)) {
    throw new ServiceError('Invalid bootstrap secret', 403, 'BOOTSTRAP_INVALID_SECRET');
  }
  if (env.adminBootstrapExpiresAt < new Date()) {
    throw new ServiceError('Bootstrap has expired', 403, 'BOOTSTRAP_EXPIRED');
  }

  // 5. 同一トランザクションでオペレーターと passkey を作成
  const operator = await input.adminPrisma.$transaction(async (tx) => {
    const op = await operatorModel.createInitialAdminOperator(tx, {
      email: input.email,
      displayName: input.displayName,
    });
    await passkeyModel.addOperatorPasskey(tx, {
      operatorId: op.id,
      credentialHandle: input.webAuthnCredential.credentialHandle,
      publicKey: input.webAuthnCredential.publicKey,
      signCount: BigInt(input.webAuthnCredential.signCount),
      aaguid: input.webAuthnCredential.aaguid,
      backupEligible: input.webAuthnCredential.backupEligible,
      backupState: input.webAuthnCredential.backupState,
      transports: input.webAuthnCredential.transports,
    });
    return op;
  });

  // 6. セッション作成
  const session = await createOperatorSession(
    { id: operator.id, email: operator.email, role: operator.role },
    input.valkey
  );

  // 7. JWT 署名
  const token = await signOperatorJwt(
    { id: operator.id, email: operator.email, role: operator.role },
    session
  );

  // 8. Cookie 構築
  const cookie = createSessionCookie(token);

  return { operator, cookie };
}
