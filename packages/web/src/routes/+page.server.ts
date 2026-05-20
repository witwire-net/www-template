import { redirect } from '@sveltejs/kit';

import { resolveLocale } from '../lib/i18n/index';

/**
 * Accept-Language ヘッダーを locale 候補へ分解する。
 *
 * q 値は優先度判定の補助情報としてのみ扱い、実際の locale 値は除去する。
 *
 * @param header - リクエストヘッダーから取得した Accept-Language の生文字列。
 * @returns resolveLocale に渡せる locale 候補の配列。
 */
function parseAcceptLanguage(header: string | null): string[] {
  if (header === null) {
    return [];
  }

  return header
    .split(',')
    .map((value) => value.split(';')[0].trim())
    .filter((value) => value.length > 0);
}

/**
 * 公開 Web の root `/` から対応ロケール URL へ誘導する。
 *
 * SSR のリクエストヘッダーに含まれる Accept-Language を読み取り、
 * その結果に応じて locale 付き URL へ redirect する。
 *
 * @param event - SvelteKit が root page server load に渡すイベント。
 * @returns この関数は redirect を throw するため、正常終了しない。
 */
export const load = ({ request, url }: { request: Request; url: URL }) => {
  const locale = resolveLocale(parseAcceptLanguage(request.headers.get('accept-language')));

  const routeRedirect = redirect(302, `/${locale}${url.search}`) as Error;
  throw routeRedirect;
};
