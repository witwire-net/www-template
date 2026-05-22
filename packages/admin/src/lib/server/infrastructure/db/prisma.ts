import { createRequire } from 'node:module';

import { getAdminDatabaseConfig, getProductDatabaseConfig } from '../config/env.js';

import type { PrismaClient as AdminPrismaClient } from '.prisma/admin-client';
import type { PrismaClient as ProductPrismaClient } from '.prisma/product-client';

const nodeRequire = createRequire(import.meta.url);

type AdminPrismaClientFactory = new (options: {
  datasources: { admin_db: { url: string | undefined } };
}) => AdminPrismaClient;
type ProductPrismaClientFactory = new (options: {
  datasources: { product_db: { url: string | undefined } };
}) => ProductPrismaClient;

let adminPrismaClientFactory: AdminPrismaClientFactory | null = null;
let productPrismaClientFactory: ProductPrismaClientFactory | null = null;

let adminPrisma: AdminPrismaClient | null = null;
let productPrisma: ProductPrismaClient | null = null;
let productPrismaValidated = false;

/**
 * テスト用に Prisma Client constructor を差し替える。
 *
 * @param factories Admin/Product Prisma Client constructor
 */
export function setPrismaClientFactoriesForTest(factories: {
  admin: AdminPrismaClientFactory;
  product: ProductPrismaClientFactory;
}): void {
  adminPrismaClientFactory = factories.admin;
  productPrismaClientFactory = factories.product;
}

function getAdminPrismaClientFactory(): AdminPrismaClientFactory {
  if (adminPrismaClientFactory !== null) return adminPrismaClientFactory;
  const { PrismaClient } = nodeRequire('.prisma/admin-client') as {
    PrismaClient: AdminPrismaClientFactory;
  };
  return PrismaClient;
}

function getProductPrismaClientFactory(): ProductPrismaClientFactory {
  if (productPrismaClientFactory !== null) return productPrismaClientFactory;
  const { PrismaClient } = nodeRequire('.prisma/product-client') as {
    PrismaClient: ProductPrismaClientFactory;
  };
  return PrismaClient;
}

/**
 * Admin DB 用 Prisma Client のシングルトンを取得する。
 *
 * @returns Admin PrismaClient インスタンス
 */
export function getAdminPrisma(): AdminPrismaClient {
  const AdminPrismaClient = getAdminPrismaClientFactory();
  const { adminDatabaseUrl } = getAdminDatabaseConfig();
  adminPrisma ??= new AdminPrismaClient({
    datasources: { admin_db: { url: adminDatabaseUrl } },
  });
  return adminPrisma;
}

/**
 * Product DB 用 Prisma Client のシングルトンを取得する。
 * 初回呼び出し時に `validateProductDbRuntimeRole()` を実行し、
 * 検証に失敗した場合は fail-close して Product DB query を実行しない。
 *
 * @returns Product PrismaClient インスタンス
 * @throws Error ランタイムロール検証に失敗した場合
 */
export async function getProductPrisma(): Promise<ProductPrismaClient> {
  const ProductPrismaClient = getProductPrismaClientFactory();
  const { productDatabaseUrl } = getProductDatabaseConfig();
  productPrisma ??= new ProductPrismaClient({
    datasources: { product_db: { url: productDatabaseUrl } },
  });

  if (!productPrismaValidated) {
    await validateProductDbRuntimeRole(productPrisma);
    productPrismaValidated = true;
  }

  return productPrisma;
}

/**
 * Product DB 接続時のランタイムロールを検証する。
 * current_user が admin_console_write のメンバーであり、superuser でなく、
 * base table owner でないことを確認する。
 *
 * @param productPrisma Product PrismaClient インスタンス
 * @throws Error いずれかの検証に失敗した場合
 */
export async function validateProductDbRuntimeRole(
  productPrisma: ProductPrismaClient
): Promise<void> {
  const result = await productPrisma.$queryRaw<
    { has_role: boolean; is_superuser: boolean; is_owner: boolean }[]
  >`
		SELECT 
			pg_has_role(current_user, 'admin_console_write', 'member') AS has_role,
			(SELECT rolsuper FROM pg_roles WHERE rolname = current_user) AS is_superuser,
			EXISTS (
				SELECT 1 FROM pg_tables 
				WHERE schemaname IN ('public', 'admin_view', 'admin_op')
					AND tableowner = current_user
			) AS is_owner
	`;

  const row = result[0];
  if (row === undefined) {
    throw new Error('Product DB runtime role validation returned no rows');
  }

  if (!row.has_role) {
    throw new Error('Product DB login role is not a member of admin_console_write');
  }
  if (row.is_superuser) {
    throw new Error('Product DB login role is superuser');
  }
  if (row.is_owner) {
    throw new Error('Product DB login role is base table owner');
  }
}

/**
 * テストクリーンアップ用に両方の Prisma Client を切断する。
 */
export async function disconnectPrisma(): Promise<void> {
  if (adminPrisma !== null) {
    await adminPrisma.$disconnect();
    adminPrisma = null;
  }
  if (productPrisma !== null) {
    await productPrisma.$disconnect();
    productPrisma = null;
    productPrismaValidated = false;
  }
}
