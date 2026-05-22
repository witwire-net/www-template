import { redirect } from '@sveltejs/kit';

import type { ServerLoad } from '@sveltejs/kit';

/**
 * 追加 operator セットアップページの server load。
 *
 * 既に Admin session を持つ operator は setup token 登録を使う必要がないため、
 * Dashboard に戻して one-time token 入力 UI を表示しない。
 */
export const load: ServerLoad = (event) => {
  if (event.locals.operator !== null) {
    redirect(303, '/');
  }
  return {};
};
