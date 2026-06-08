/**
 * 旧ログインと端末ページ。
 * `/settings/security/passkeys` へリダイレクトする。
 */
import { redirect } from '@sveltejs/kit';

/**
 * 旧ログインと端末ページの load function。
 * `/settings/security/passkeys` へ 302 リダイレクトする。
 */
export const load = () => {
  redirect(302, '/settings/security/passkeys');
};
