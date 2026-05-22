import { fail, redirect } from '@sveltejs/kit';

import { createAdminI18n } from '$lib/i18n';
import { getAdminPrisma } from '$lib/server/infrastructure/db/prisma';
import { getFormString } from '$lib/server/infrastructure/form-fields';
import { hasPermission } from '$lib/server/infrastructure/rbac/guard';
import { requireAuthenticatedOperator } from '$lib/server/services/auth/routes';
import { ServiceError } from '$lib/server/services/errors';
import { listOperators } from '$lib/server/services/operators/list';
import { updateOwnOperatorLocale } from '$lib/server/services/operators/locale';

import type { Actions, ServerLoad } from '@sveltejs/kit';

/**
 * 設定ページの読み込み処理。
 *
 * @param locals 認証済みローカル情報
 * @returns オペレーター数の集計値
 */
export const load: ServerLoad = async ({ locals, url }) => {
  // 言語設定は全認証済み operator が扱える本人設定なので、operators:read ではなく認証済み境界だけを要求する。
  const operator = requireAuthenticatedOperator({ locals } as never);
  // 保存済み operator locale から settings page 専用 label を生成し、route/component 内の ad hoc 翻訳を避ける。
  const { t } = createAdminI18n(operator.locale);
  // operator 管理 summary は権限保有者にだけ計算・表示し、viewer/operator に管理情報を漏らさない。
  const canManageOperators = hasPermission(operator.role, 'operators:read');
  const operators = canManageOperators ? await listOperators(getAdminPrisma()) : [];
  return {
    locale: operator.locale,
    localeUpdated: url.searchParams.get('localeUpdated') === '1',
    canManageOperators,
    operatorCount: canManageOperators ? operators.length : 0,
    activeOperatorCount: canManageOperators
      ? operators.filter((currentOperator) => currentOperator.isActive).length
      : 0,
    labels: {
      title: t('settings.title'),
      description: t('settings.description'),
      languageTitle: t('settings.languageTitle'),
      languageDescription: t('settings.languageDescription'),
      languageLabel: t('settings.languageLabel'),
      languageJapanese: t('settings.languageJapanese'),
      languageEnglish: t('settings.languageEnglish'),
      languageSubmit: t('settings.languageSubmit'),
      languageSuccess: t('settings.languageSuccess'),
      languageError: t('settings.languageError'),
      managementTitle: t('settings.managementTitle'),
      managementDescription: t('settings.managementDescription', {
        active: operators.filter((currentOperator) => currentOperator.isActive).length,
        total: operators.length,
      }),
      managementBody: t('settings.managementBody'),
      managementButton: t('settings.managementButton'),
    },
  };
};

/**
 * 設定ページの form actions。
 */
export const actions: Actions = {
  locale: async (event) => {
    // hooks が検証した本人 operator ID だけを使い、form から別 operator ID を受け取らない。
    const operator = requireAuthenticatedOperator(event);
    const form = await event.request.formData();
    try {
      await updateOwnOperatorLocale(getAdminPrisma(), operator.id, getFormString(form, 'locale'));
    } catch (error) {
      if (error instanceof ServiceError && error.statusCode === 400) {
        return fail(400, { localeError: true });
      }
      throw error;
    }
    // PRG で再度 hook/load を通し、更新後の DB locale を layout と settings page の両方に反映する。
    return redirect(303, '/settings?localeUpdated=1');
  },
};
