/**
 * 設定トップページの load function。
 * SSR 無効の CSR SPA ため、クライアントサイドで `/settings/general/language` へリダイレクトする。
 */
import { redirect } from '@sveltejs/kit';

/**
 * 設定トップページの load function。
 * `/settings/general/language` へ 302 リダイレクトする。
 */
export const load = () => {
  redirect(302, '/settings/general/language');
};
