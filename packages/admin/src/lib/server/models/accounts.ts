import type { AccountSummary, PasskeyInfo, SearchParams } from './types.js';
import type { PrismaClient as ProductPrismaClient } from '.prisma/product-client';

/**
 * アカウントを検索する（admin_view.account_summaries に対するパラメータ化クエリ）。
 *
 * @param productPrisma Product PrismaClient
 * @param params 検索パラメータ
 * @returns アカウント一覧と総数
 */
export async function searchAccounts(
  productPrisma: ProductPrismaClient,
  params: SearchParams
): Promise<{ items: AccountSummary[]; total: bigint }> {
  const query = params.query ?? '';
  const status = params.status ?? '';
  const limit = params.limit;
  const offset = params.offset;

  const countResult = await productPrisma.$queryRaw<{ count: bigint }[]>`
		SELECT COUNT(*) as count FROM admin_view.account_summaries
		WHERE (${query} = '' OR email ILIKE ${`%${query}%`})
			AND (${status} = '' OR status = ${status})
	`;

  const itemsResult = await productPrisma.$queryRaw<AccountSummaryRaw[]>`
		SELECT * FROM admin_view.account_summaries
		WHERE (${query} = '' OR email ILIKE ${`%${query}%`})
			AND (${status} = '' OR status = ${status})
		ORDER BY created_at DESC
		LIMIT ${limit} OFFSET ${offset}
	`;

  return {
    items: itemsResult.map(toAccountSummary),
    total: countResult[0]?.count ?? 0n,
  };
}

/**
 * ID でアカウントを取得する。
 *
 * @param productPrisma Product PrismaClient
 * @param id アカウント ID
 * @returns アカウント、または null
 */
export async function getAccountById(
  productPrisma: ProductPrismaClient,
  id: string
): Promise<AccountSummary | null> {
  const result = await productPrisma.$queryRaw<AccountSummaryRaw[]>`
		SELECT * FROM admin_view.account_summaries WHERE id = ${id}
	`;
  if (result.length === 0) return null;
  const first = result[0];
  if (first === undefined) return null;
  return toAccountSummary(first);
}

/**
 * アカウントを停止する（Product DB の admin_op.suspend_account を呼び出す）。
 *
 * @param productPrisma Product PrismaClient
 * @param accountId アカウント ID
 * @param operatorId 操作オペレーター ID
 * @param reason 停止理由
 * @param auditEventId 監査イベント ID
 */
export async function suspendAccountProduct(
  productPrisma: ProductPrismaClient,
  accountId: string,
  operatorId: string,
  reason: string,
  auditEventId: string
): Promise<void> {
  await productPrisma.$executeRaw`
		SELECT admin_op.suspend_account(${accountId}, ${operatorId}, ${reason}, ${auditEventId})
	`;
}

/**
 * アカウントを復旧する（Product DB の admin_op.restore_account を呼び出す）。
 *
 * @param productPrisma Product PrismaClient
 * @param accountId アカウント ID
 * @param operatorId 操作オペレーター ID
 * @param auditEventId 監査イベント ID
 */
export async function restoreAccountProduct(
  productPrisma: ProductPrismaClient,
  accountId: string,
  operatorId: string,
  auditEventId: string
): Promise<void> {
  await productPrisma.$executeRaw`
		SELECT admin_op.restore_account(${accountId}, ${operatorId}, ${auditEventId})
	`;
}

/**
 * アカウントに紐づく passkey 一覧を取得する（Product DB の admin_view.account_passkeys）。
 *
 * @param productPrisma Product PrismaClient
 * @param accountId アカウント ID
 * @returns passkey 配列
 */
export async function getAccountPasskeys(
  productPrisma: ProductPrismaClient,
  accountId: string
): Promise<PasskeyInfo[]> {
  const rows = await productPrisma.$queryRaw<AccountPasskeyRaw[]>`
		SELECT
			passkey_id as id,
			account_id as "operatorId",
			passkey_identifier as "credentialHandle",
			public_key as "publicKey",
			sign_count as "signCount",
			aaguid,
			backup_eligible as "backupEligible",
			backup_state as "backupState",
			transports,
			passkey_created_at as "createdAt"
		FROM admin_view.account_passkeys
		WHERE account_id = ${accountId}
		ORDER BY passkey_created_at DESC
	`;
  return rows.map(toPasskeyInfo);
}

// ヘルパー型
interface AccountSummaryRaw {
  id: string;
  email: string;
  status: string;
  status_reason: string | null;
  status_updated_at: Date | null;
  status_updated_by: string | null;
  session_revoked_after: Date | null;
  created_at: Date;
  passkey_count: bigint;
}

interface AccountPasskeyRaw {
  id: string;
  operatorId: string;
  credentialHandle: string;
  publicKey: Uint8Array;
  signCount: bigint;
  aaguid: Uint8Array;
  backupEligible: boolean;
  backupState: boolean;
  transports: unknown;
  createdAt: Date;
}

function toAccountSummary(row: AccountSummaryRaw): AccountSummary {
  return {
    id: row.id,
    email: row.email,
    status: row.status,
    statusReason: row.status_reason,
    statusUpdatedAt: row.status_updated_at,
    statusUpdatedBy: row.status_updated_by,
    sessionRevokedAfter: row.session_revoked_after,
    createdAt: row.created_at,
    passkeyCount: row.passkey_count,
  };
}

function toPasskeyInfo(row: AccountPasskeyRaw): PasskeyInfo {
  return {
    id: row.id,
    operatorId: row.operatorId,
    credentialHandle: row.credentialHandle,
    publicKey: row.publicKey,
    signCount: row.signCount,
    aaguid: row.aaguid,
    backupEligible: row.backupEligible,
    backupState: row.backupState,
    transports: row.transports,
    createdAt: row.createdAt,
  };
}
