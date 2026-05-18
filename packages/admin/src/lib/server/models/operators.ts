import type { Operator } from './types.js';
import type { PrismaClient as AdminPrismaClient, Prisma } from '.prisma/admin-client';

/**
 * PrismaClient またはトランザクションクライアントを受け入れる型。
 * サービス層でのトランザクション使用をモデル層でサポートするため。
 */
type AdminPrismaLike = AdminPrismaClient | Prisma.TransactionClient;

/**
 * ID でオペレーターを検索する。
 *
 * @param adminPrisma Admin PrismaClient
 * @param id オペレーター ID
 * @returns オペレーター、または null
 */
export async function findOperatorById(
  adminPrisma: AdminPrismaLike,
  id: string
): Promise<Operator | null> {
  const row = await (adminPrisma as AdminPrismaClient).adminOperator.findUnique({ where: { id } });
  if (row === null) return null;
  return toOperator(row);
}

/**
 * Email でオペレーターを検索する。
 *
 * @param adminPrisma Admin PrismaClient
 * @param email メールアドレス
 * @returns オペレーター、または null
 */
export async function findOperatorByEmail(
  adminPrisma: AdminPrismaLike,
  email: string
): Promise<Operator | null> {
  const row = await (adminPrisma as AdminPrismaClient).adminOperator.findUnique({
    where: { email },
  });
  if (row === null) return null;
  return toOperator(row);
}

/**
 * オペレーター総数を取得する。
 *
 * @param adminPrisma Admin PrismaClient
 * @returns 総数
 */
export async function countOperators(adminPrisma: AdminPrismaLike): Promise<number> {
  return (adminPrisma as AdminPrismaClient).adminOperator.count();
}

/**
 * オペレーター一覧を取得する。
 *
 * @param adminPrisma Admin PrismaClient
 * @returns オペレーター配列
 */
export async function listOperators(adminPrisma: AdminPrismaLike): Promise<Operator[]> {
  const rows = await (adminPrisma as AdminPrismaClient).adminOperator.findMany({
    orderBy: { createdAt: 'desc' },
  });
  return rows.map(toOperator);
}

/**
 * 初回 admin オペレーターを作成する（bootstrap 用）。
 *
 * @param adminPrisma Admin PrismaClient
 * @param data 作成データ
 * @returns 作成されたオペレーター
 */
export async function createInitialAdminOperator(
  adminPrisma: AdminPrismaLike,
  data: { email: string; displayName: string }
): Promise<Operator> {
  const row = await (adminPrisma as AdminPrismaClient).adminOperator.create({
    data: {
      email: data.email,
      display_name: data.displayName,
      role: 'admin',
      is_active: true,
    },
  });
  return toOperator(row);
}

/**
 * 新規オペレーターを作成する。
 *
 * @param adminPrisma Admin PrismaClient
 * @param data 作成データ
 * @returns 作成されたオペレーター
 */
export async function createOperator(
  adminPrisma: AdminPrismaLike,
  data: { email: string; displayName: string; role: string }
): Promise<Operator> {
  const row = await (adminPrisma as AdminPrismaClient).adminOperator.create({
    data: {
      email: data.email,
      display_name: data.displayName,
      role: data.role,
      is_active: true,
    },
  });
  return toOperator(row);
}

/**
 * オペレーターのロールを変更する。
 *
 * @param adminPrisma Admin PrismaClient
 * @param id オペレーター ID
 * @param role 新しいロール
 * @returns 更新されたオペレーター
 */
export async function updateOperatorRole(
  adminPrisma: AdminPrismaLike,
  id: string,
  role: string
): Promise<Operator> {
  const row = await (adminPrisma as AdminPrismaClient).adminOperator.update({
    where: { id },
    data: { role },
  });
  return toOperator(row);
}

/**
 * オペレーターを無効化する。
 *
 * @param adminPrisma Admin PrismaClient
 * @param id オペレーター ID
 * @returns 更新されたオペレーター
 */
export async function deactivateOperator(
  adminPrisma: AdminPrismaLike,
  id: string
): Promise<Operator> {
  const row = await (adminPrisma as AdminPrismaClient).adminOperator.update({
    where: { id },
    data: { is_active: false },
  });
  return toOperator(row);
}

/**
 * オペレーターのセットアップトークンを更新する。
 *
 * @param adminPrisma Admin PrismaClient
 * @param operatorId オペレーター ID
 * @param hash bcrypt ハッシュ済みトークン
 * @param expiresAt 有効期限
 */
export async function updateOperatorSetupToken(
  adminPrisma: AdminPrismaLike,
  operatorId: string,
  hash: string,
  expiresAt: Date
): Promise<void> {
  await (adminPrisma as AdminPrismaClient).adminOperator.update({
    where: { id: operatorId },
    data: { setup_token_hash: hash, setup_token_expires_at: expiresAt },
  });
}

/**
 * オペレーターのセットアップトークンを条件付きで消費済みにする。
 *
 * @param adminPrisma Admin PrismaClient
 * @param operatorId オペレーター ID
 * @param now 有効期限検証に使う現在時刻
 * @returns token を消費できた場合 true
 */
export async function consumeOperatorSetupToken(
  adminPrisma: AdminPrismaLike,
  operatorId: string,
  now: Date
): Promise<boolean> {
  const result = await (adminPrisma as AdminPrismaClient).adminOperator.updateMany({
    where: {
      id: operatorId,
      is_active: true,
      setup_token_hash: { not: null },
      setup_token_expires_at: { gt: now },
    },
    data: { setup_token_hash: null, setup_token_expires_at: null },
  });
  return result.count === 1;
}

/**
 * アクティブな admin ロールのオペレーター数を取得する。
 *
 * @param adminPrisma Admin PrismaClient
 * @returns アクティブ admin 数
 */
export async function countActiveAdmins(adminPrisma: AdminPrismaLike): Promise<number> {
  return (adminPrisma as AdminPrismaClient).adminOperator.count({
    where: { role: 'admin', is_active: true },
  });
}

/**
 * ログイン時刻を更新する。
 *
 * @param adminPrisma Admin PrismaClient
 * @param id オペレーター ID
 */
export async function updateLoginTimestamp(
  adminPrisma: AdminPrismaLike,
  id: string
): Promise<void> {
  await (adminPrisma as AdminPrismaClient).adminOperator.update({
    where: { id },
    data: { last_login_at: new Date() },
  });
}

function toOperator(row: {
  id: string;
  email: string;
  display_name: string;
  role: string;
  is_active: boolean;
  setup_token_hash: string | null;
  setup_token_expires_at: Date | null;
  last_login_at: Date | null;
  createdAt: Date;
  updatedAt: Date;
}): Operator {
  return {
    id: row.id,
    email: row.email,
    displayName: row.display_name,
    role: row.role as Operator['role'],
    isActive: row.is_active,
    setupTokenHash: row.setup_token_hash,
    setupTokenExpiresAt: row.setup_token_expires_at,
    lastLoginAt: row.last_login_at,
    createdAt: row.createdAt,
    updatedAt: row.updatedAt,
  };
}
