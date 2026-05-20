import { createAdminI18n } from '$lib/i18n';
import { issueCsrfToken } from '$lib/server/infrastructure/csrf/guard';
import { hasPermission } from '$lib/server/infrastructure/rbac/guard';

import type { ServerLoad } from '@sveltejs/kit';

interface AdminNavItem {
  label: string;
  href: string;
  activePrefix: string;
}

/**
 * レイアウトサーバーロード
 * 認証済みオペレーター情報をレイアウトデータに引き渡す
 */

/**
 * Admin Console の共通 layout data を作成する。
 *
 * セッション識別子や jti は server-only の `locals` に閉じ込め、client へは表示に必要な operator 公開情報、
 * navigation、通常 form 用の CSRF token だけを返す。
 */
export const load: ServerLoad = ({ locals, url }) => {
  // 現在の DB ロールから表示可能な navigation だけを組み立て、権限外導線の表示を防ぐ。
  const operator = locals.operator;
  // 認証済み operator の保存済み locale、または認証前 fallback locale から layout 用 translator を生成する。
  const { locale, t } = createAdminI18n(operator?.locale ?? null);
  // navigation label は route 内の固定文字列ではなく Admin-owned 辞書から生成する。
  const baseNavItems: AdminNavItem[] = [
    { label: t('nav.dashboard'), href: '/', activePrefix: '/' },
    { label: t('nav.accounts'), href: '/accounts', activePrefix: '/accounts' },
    { label: t('nav.audit'), href: '/audit', activePrefix: '/audit' },
  ];
  const settingsItems =
    operator !== null && hasPermission(operator.role, 'operators:read')
      ? [{ label: t('nav.settings'), href: '/settings', activePrefix: '/settings' }]
      : [];

  return {
    operator:
      operator === null
        ? null
        : { id: operator.id, email: operator.email, role: operator.role, locale: operator.locale },
    csrfToken: operator === null ? '' : issueCsrfToken(operator.sessionId, operator.jti).token,
    currentPath: url.pathname,
    locale,
    labels: {
      title: t('layout.title'),
      brand: t('layout.brand'),
      admin: t('header.admin'),
      operatorFallback: t('header.operatorFallback'),
      logout: t('header.logout'),
      close: t('shared.close'),
    },
    navItems: [...baseNavItems, ...settingsItems],
  };
};
