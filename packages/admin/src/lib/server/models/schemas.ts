import { z } from 'zod';

/**
 * ログイン email スキーマ
 */
export const loginEmailSchema = z.string().email().min(1).max(254);

/**
 * 停止理由スキーマ
 */
export const suspendReasonSchema = z.string().min(1).max(500);

/**
 * 検索パラメータスキーマ
 */
export const searchParamsSchema = z.object({
  query: z.string().optional(),
  status: z.string().optional(),
  limit: z.number().int().min(1).max(100),
  offset: z.number().int().min(0),
});

/**
 * オペレーター作成スキーマ
 */
export const createOperatorSchema = z.object({
  email: z.string().email().min(1).max(254),
  displayName: z.string().min(1).max(100),
  role: z.enum(['admin', 'operator', 'viewer']),
});

/**
 * ロール更新スキーマ
 */
export const updateRoleSchema = z.object({
  role: z.enum(['admin', 'operator', 'viewer']),
});
