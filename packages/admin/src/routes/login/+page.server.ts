import { createAdminI18n } from '$lib/i18n';

import type { ServerLoad } from '@sveltejs/kit';

/**
 * Admin ログインページの読み込み処理。
 *
 * 認証前画面は operator DB locale を持たないため、Admin package-local fallback locale の文言だけを返す。
 */
export const load: ServerLoad = () => {
  const { t } = createAdminI18n(null);
  return {
    labels: {
      title: t('login.title'),
      eyebrow: t('login.eyebrow'),
      heading: t('login.heading'),
      description: t('login.description'),
      cardTitle: t('login.cardTitle'),
      cardDescription: t('login.cardDescription'),
      emailLabel: t('login.emailLabel'),
      error: t('login.error'),
      submitting: t('login.submitting'),
      submit: t('login.submit'),
      setupToken: t('login.setupToken'),
    },
  };
};
