import { extractLocaleFromPath, type Locale } from '$lib/i18n';

/**
 * Layout 全体で使用する locale を URL から抽出する。
 */
export const load = ({ url }: { url: URL }): { locale: Locale } => {
  const locale = extractLocaleFromPath(url.pathname);
  return {
    locale: locale ?? 'ja',
  };
};
