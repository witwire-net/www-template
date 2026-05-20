import { fail, redirect } from '@sveltejs/kit';

import {
  createAuditIntent,
  markAuditFailed,
  markAuditSucceeded,
} from '$lib/server/infrastructure/audit/logger';
import { getAdminPrisma, getProductPrisma } from '$lib/server/infrastructure/db/prisma';
import { requirePermission } from '$lib/server/infrastructure/rbac/guard';
import { suspendReasonSchema } from '$lib/server/models/schemas';
import { getAccountDetail } from '$lib/server/services/accounts/detail';
import { restoreAccount } from '$lib/server/services/accounts/restore';
import { suspendAccount } from '$lib/server/services/accounts/suspend';

import type { Actions, ServerLoad } from '@sveltejs/kit';

/**
 * フォームから文字列値を安全に取り出す。
 *
 * @param form 取得対象の FormData
 * @param name フィールド名
 * @param fallback 文字列以外だった場合の既定値
 * @returns 安全な文字列
 */
function getFormString(form: FormData, name: string, fallback = ''): string {
  const value = form.get(name);
  return typeof value === 'string' ? value : fallback;
}

function getClientIp(request: Request): string {
  // reverse proxy が付与する代表的なヘッダーを読み、存在しない場合は空文字で監査に渡す。
  return request.headers.get('x-forwarded-for')?.split(',')[0]?.trim() ?? '';
}

function getAccountId(params: Partial<Record<string, string>>): string {
  // SvelteKit の型上は optional のため、存在しない場合は 404 相当として一覧へ戻す。
  if (params.id === undefined || params.id === '') return redirect(303, '/accounts');
  return params.id;
}

/**
 * アカウント詳細ページの読み込み処理。
 *
 * @param locals 認証済みローカル情報
 * @param params ルートパラメータ
 * @returns アカウント詳細データ
 */
export const load: ServerLoad = async ({ locals, params }) => {
  // アカウント詳細には停止状況と passkey 一覧が含まれるため、読み取り権限を確認する。
  requirePermission(locals.operator, 'accounts:read');
  const accountId = getAccountId(params);
  const detail = await getAccountDetail(await getProductPrisma(), getAdminPrisma(), accountId);
  if (detail === null) {
    return redirect(303, '/accounts');
  }
  return detail;
};

/**
 * アカウント停止・復旧の form action 群。
 */
export const actions: Actions = {
  suspend: async ({ locals, params, request }) => {
    // 顧客の利用停止は高リスク操作のため、個別の suspend 権限を必須にする。
    requirePermission(locals.operator, 'accounts:suspend');
    const form = await request.formData();
    const accountId = getAccountId(params);
    const reason = suspendReasonSchema.safeParse(getFormString(form, 'reason'));
    if (!reason.success) return fail(400, { messageKey: 'accountDetail.suspendError' });
    await suspendAccount({
      adminPrisma: getAdminPrisma(),
      productPrisma: await getProductPrisma(),
      operatorId: locals.operator?.id ?? '',
      accountId,
      reason: reason.data,
      ipAddress: getClientIp(request),
      auditLogger: {
        createAuditIntent: (input) => createAuditIntent(getAdminPrisma(), input),
        markAuditSucceeded: (id) => markAuditSucceeded(getAdminPrisma(), id),
        markAuditFailed: (id, code) => markAuditFailed(getAdminPrisma(), id, code),
      },
    });
    return redirect(303, `/accounts/${accountId}`);
  },
  restore: async ({ locals, params, request }) => {
    // 復旧でも履歴を残すため、restore 権限を確認し同じ監査ロガーを使用する。
    requirePermission(locals.operator, 'accounts:restore');
    const form = await request.formData();
    const accountId = getAccountId(params);
    const reason = suspendReasonSchema.safeParse(
      getFormString(form, 'reason', 'restored by operator')
    );
    if (!reason.success) return fail(400, { messageKey: 'accountDetail.restoreError' });
    await restoreAccount({
      adminPrisma: getAdminPrisma(),
      productPrisma: await getProductPrisma(),
      operatorId: locals.operator?.id ?? '',
      accountId,
      reason: reason.data,
      ipAddress: getClientIp(request),
      auditLogger: {
        createAuditIntent: (input) => createAuditIntent(getAdminPrisma(), input),
        markAuditSucceeded: (id) => markAuditSucceeded(getAdminPrisma(), id),
        markAuditFailed: (id, code) => markAuditFailed(getAdminPrisma(), id, code),
      },
    });
    return redirect(303, `/accounts/${accountId}`);
  },
};
