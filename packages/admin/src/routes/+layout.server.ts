import { issueCsrfToken } from '$lib/server/infrastructure/csrf/guard';
import { hasPermission } from '$lib/server/infrastructure/rbac/guard';

import type { ServerLoad } from '@sveltejs/kit';

interface AdminNavItem {
  label: string;
  href: string;
  activePrefix: string;
}

const BASE_NAV_ITEMS: AdminNavItem[] = [
  { label: 'Dashboard', href: '/', activePrefix: '/' },
  { label: 'Accounts', href: '/accounts', activePrefix: '/accounts' },
  { label: 'Audit Log', href: '/audit', activePrefix: '/audit' },
];

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
  const settingsItems =
    operator !== null && hasPermission(operator.role, 'operators:read')
      ? [{ label: 'Settings', href: '/settings', activePrefix: '/settings' }]
      : [];

  return {
    operator:
      operator === null ? null : { id: operator.id, email: operator.email, role: operator.role },
    csrfToken: operator === null ? '' : issueCsrfToken(operator.sessionId, operator.jti).token,
    currentPath: url.pathname,
    navItems: [...BASE_NAV_ITEMS, ...settingsItems],
  };
};
