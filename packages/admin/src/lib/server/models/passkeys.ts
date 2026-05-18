import { createRequire } from 'node:module';

import type { PasskeyInfo } from './types.js';
import type {
  PrismaClient as AdminPrismaClient,
  Prisma as PrismaNamespace,
} from '.prisma/admin-client';

const nodeRequire = createRequire(import.meta.url);
const { Prisma } = nodeRequire('.prisma/admin-client') as { Prisma: typeof PrismaNamespace };

/**
 * PrismaClient またはトランザクションクライアントを受け入れる型。
 */
type AdminPrismaLike = AdminPrismaClient | PrismaNamespace.TransactionClient;

/**
 * オペレーターの passkey 一覧を取得する。
 *
 * @param adminPrisma Admin PrismaClient
 * @param operatorId オペレーター ID
 * @returns passkey 配列
 */
export async function listOperatorPasskeys(
  adminPrisma: AdminPrismaLike,
  operatorId: string
): Promise<PasskeyInfo[]> {
  const rows = await (adminPrisma as AdminPrismaClient).adminOperatorPasskey.findMany({
    where: { operator_id: operatorId },
    orderBy: { createdAt: 'desc' },
  });
  return rows.map(toPasskeyInfo);
}

/**
 * credential handle で passkey を検索する。
 *
 * @param adminPrisma Admin PrismaClient
 * @param credentialHandle WebAuthn credential ID
 * @returns passkey 情報、または null
 */
export async function findOperatorPasskeyByCredentialHandle(
  adminPrisma: AdminPrismaLike,
  credentialHandle: string
): Promise<PasskeyInfo | null> {
  const row = await (adminPrisma as AdminPrismaClient).adminOperatorPasskey.findUnique({
    where: { credential_handle: credentialHandle },
  });
  if (row === null) return null;
  return toPasskeyInfo(row);
}

/**
 * 指定オペレーターに紐づく passkey を ID で検索する。
 *
 * @param adminPrisma Admin PrismaClient
 * @param operatorId 所有者オペレーター ID
 * @param passkeyId passkey ID
 * @returns passkey 情報、または null
 */
export async function findOperatorPasskeyForOperator(
  adminPrisma: AdminPrismaLike,
  operatorId: string,
  passkeyId: string
): Promise<PasskeyInfo | null> {
  const row = await (adminPrisma as AdminPrismaClient).adminOperatorPasskey.findFirst({
    where: { id: passkeyId, operator_id: operatorId },
  });
  if (row === null) return null;
  return toPasskeyInfo(row);
}

/**
 * オペレーターに passkey を追加する。
 *
 * @param adminPrisma Admin PrismaClient
 * @param data passkey データ
 * @returns 作成された passkey
 */
export async function addOperatorPasskey(
  adminPrisma: AdminPrismaLike,
  data: {
    operatorId: string;
    credentialHandle: string;
    publicKey: Uint8Array;
    signCount: bigint;
    aaguid: Uint8Array;
    backupEligible: boolean;
    backupState: boolean;
    transports: unknown;
  }
): Promise<PasskeyInfo> {
  const row = await (adminPrisma as AdminPrismaClient).adminOperatorPasskey.create({
    data: {
      operator_id: data.operatorId,
      credential_handle: data.credentialHandle,
      public_key: data.publicKey as Uint8Array<ArrayBuffer>,
      sign_count: data.signCount,
      aaguid: data.aaguid as Uint8Array<ArrayBuffer>,
      backup_eligible: data.backupEligible,
      backup_state: data.backupState,
      transports:
        data.transports === undefined || data.transports === null
          ? Prisma.DbNull
          : (data.transports as PrismaNamespace.InputJsonValue),
    },
  });
  return toPasskeyInfo(row);
}

/**
 * passkey を削除する。
 *
 * @param adminPrisma Admin PrismaClient
 * @param id passkey ID
 */
export async function deleteOperatorPasskey(
  adminPrisma: AdminPrismaLike,
  id: string
): Promise<void> {
  await (adminPrisma as AdminPrismaClient).adminOperatorPasskey.delete({ where: { id } });
}

/**
 * 認証成功後に passkey の sign count を更新する。
 *
 * @param adminPrisma Admin PrismaClient
 * @param id passkey ID
 * @param signCount 新しい sign count
 */
export async function updateOperatorPasskeySignCount(
  adminPrisma: AdminPrismaLike,
  id: string,
  signCount: number
): Promise<void> {
  await (adminPrisma as AdminPrismaClient).adminOperatorPasskey.update({
    where: { id },
    data: { sign_count: BigInt(signCount) },
  });
}

/**
 * オペレーターの passkey 数を取得する。
 *
 * @param adminPrisma Admin PrismaClient
 * @param operatorId オペレーター ID
 * @returns passkey 数
 */
export async function getPasskeyCount(
  adminPrisma: AdminPrismaLike,
  operatorId: string
): Promise<number> {
  return (adminPrisma as AdminPrismaClient).adminOperatorPasskey.count({
    where: { operator_id: operatorId },
  });
}

function toPasskeyInfo(row: {
  id: string;
  operator_id: string;
  credential_handle: string;
  public_key: Uint8Array;
  sign_count: bigint;
  aaguid: Uint8Array;
  backup_eligible: boolean;
  backup_state: boolean;
  transports: unknown;
  createdAt: Date;
}): PasskeyInfo {
  return {
    id: row.id,
    operatorId: row.operator_id,
    credentialHandle: row.credential_handle,
    publicKey: row.public_key,
    signCount: row.sign_count,
    aaguid: row.aaguid,
    backupEligible: row.backup_eligible,
    backupState: row.backup_state,
    transports: row.transports,
    createdAt: row.createdAt,
  };
}
