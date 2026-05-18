import { error as skError } from '@sveltejs/kit';

import { ROLE_PERMISSIONS } from './permissions.js';

import type { Permission } from './permissions.js';

/**
 * 指定ロールが指定権限を持つか確認する。
 *
 * @param role ロール名
 * @param permission 権限名
 * @returns 権限がある場合 true
 */
export function hasPermission(role: string, permission: Permission): boolean {
  const rolePerms = ROLE_PERMISSIONS.get(role);
  if (rolePerms === undefined) {
    return false;
  }
  return rolePerms.get(permission) ?? false;
}

/**
 * オペレーターに指定権限がない場合は 403 を throw する。
 *
 * @param operator 認証済みオペレーター情報
 * @param permission 要求権限
 * @throws 403 権限不足時
 */
export function requirePermission(
  operator: { id: string; email: string; role: string; sessionId: string; jti: string } | null,
  permission: Permission
): void {
  if (operator === null || !hasPermission(operator.role, permission)) {
    return skError(403, 'Insufficient permissions');
  }
}
