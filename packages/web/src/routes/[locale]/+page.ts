import { error } from '@sveltejs/kit';

import { extractLocaleFromPath } from '../../lib/i18n/index';

/**
 * 公開 Web の locale 別ページを提供する。
 * URL から locale を抽出し、未対応 locale は 404 を返す。
 */
export const load = ({ url }: { url: URL }) => {
  const locale = extractLocaleFromPath(url.pathname);
  if (locale === null) {
    const notFound = error(404, 'Not Found') as Error;
    throw notFound;
  }

  return {
    locale,
  };
};
