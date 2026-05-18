import { fail, redirect } from '@sveltejs/kit';

import { getAdminPrisma } from '$lib/server/infrastructure/db/prisma';
import { requirePermission } from '$lib/server/infrastructure/rbac/guard';
import { createOperatorSchema, updateRoleSchema } from '$lib/server/models/schemas';
import { listOperators } from '$lib/server/services/operators/list';
import {
  createOperator,
  deactivateOperator,
  rotateSetupToken,
  updateOperatorRole,
} from '$lib/server/services/operators/manage';

import type { Actions, ServerLoad } from '@sveltejs/kit';

function getFormString(form: FormData, name: string): string {
  const value = form.get(name);
  return typeof value === 'string' ? value : '';
}

/**
 * オペレーター管理ページの読み込み処理。
 *
 * @param locals 認証済みローカル情報
 * @returns 一覧表示に必要なデータ
 */
export const load: ServerLoad = async ({ locals }) => {
  // オペレーター一覧と管理操作は admin 権限だけに公開する。
  requirePermission(locals.operator, 'operators:read');
  return {
    operators: await listOperators(getAdminPrisma()),
    currentOperatorId: locals.operator?.id ?? '',
  };
};

/**
 * オペレーター管理ページの form action 群。
 */
export const actions: Actions = {
  create: async ({ locals, request }) => {
    requirePermission(locals.operator, 'operators:write');
    const form = await request.formData();
    const parsed = createOperatorSchema.safeParse({
      email: getFormString(form, 'email'),
      displayName: getFormString(form, 'displayName'),
      role: getFormString(form, 'role'),
    });
    if (!parsed.success) return fail(400, { message: '入力値を確認してください。' });
    const result = await createOperator(getAdminPrisma(), parsed.data, locals.operator?.id ?? '');
    return { setupToken: result.plaintextToken, setupTokenEmail: result.operator.email };
  },
  update: async ({ locals, request }) => {
    requirePermission(locals.operator, 'operators:write');
    const form = await request.formData();
    const operatorId = getFormString(form, 'operatorId');
    const parsed = updateRoleSchema.safeParse({ role: getFormString(form, 'role') });
    if (operatorId === '' || !parsed.success)
      return fail(400, { message: 'ロール更新の入力値を確認してください。' });
    await updateOperatorRole(
      getAdminPrisma(),
      operatorId,
      parsed.data.role,
      locals.operator?.id ?? ''
    );
    return redirect(303, '/settings/operators');
  },
  deactivate: async ({ locals, request }) => {
    requirePermission(locals.operator, 'operators:deactivate');
    const operatorId = getFormString(await request.formData(), 'operatorId');
    if (operatorId === '') return fail(400, { message: '対象オペレーターを指定してください。' });
    await deactivateOperator(getAdminPrisma(), operatorId, locals.operator?.id ?? '');
    return redirect(303, '/settings/operators');
  },
  rotate: async ({ locals, request }) => {
    requirePermission(locals.operator, 'operators:setup_token');
    const operatorId = getFormString(await request.formData(), 'operatorId');
    if (operatorId === '') return fail(400, { message: '対象オペレーターを指定してください。' });
    const result = await rotateSetupToken(getAdminPrisma(), operatorId, locals.operator?.id ?? '');
    return { setupToken: result.plaintextToken, setupTokenEmail: result.operator.email };
  },
};
