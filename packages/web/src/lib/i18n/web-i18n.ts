import {
  createTranslator,
  defineI18nConfig,
  loadJsonCatalog,
  type CatalogKeyPath,
  type LocaleCatalogMap,
  type TranslationValues,
  resolveLocale,
  SUPPORTED_LOCALES,
  type Locale,
} from '@www-template/i18n';

import commonEn from './messages/en/common.json';
import commonJa from './messages/ja/common.json';

interface WebNamespaceMap {
  common: typeof commonJa;
}
type WebTranslationKey = `common.${CatalogKeyPath<WebNamespaceMap['common']>}`;

const catalogs: LocaleCatalogMap<WebNamespaceMap> = {
  ja: {
    common: loadJsonCatalog(commonJa),
  },
  en: {
    common: loadJsonCatalog(commonEn),
  },
};

/**
 * URL path から locale を抽出する。
 * 対応ロケールでない場合は null を返す。
 */
export function extractLocaleFromPath(pathname: string): Locale | null {
  const firstSegment = pathname.split('/')[1];
  if (SUPPORTED_LOCALES.includes(firstSegment as Locale)) {
    return firstSegment as Locale;
  }
  return null;
}

/**
 * web 用の i18n entrypoint を生成する。
 *
 * @param locale - URL から決まる locale
 */
export function useI18n(locale: Locale) {
  const translator = createTranslator(defineI18nConfig({ locale }), catalogs);

  return {
    locale: translator.locale,
    formatters: translator.formatters,
    t(key: WebTranslationKey, values?: TranslationValues): string {
      const separatorIndex = key.indexOf('.');
      if (separatorIndex <= 0 || separatorIndex === key.length - 1) {
        throw new Error(`i18n key が不正です: ${key}`);
      }

      const namespace = key.slice(0, separatorIndex);
      if (namespace !== 'common') {
        throw new Error('i18n namespace が不正です。');
      }

      return translator.t(
        'common',
        key.slice(separatorIndex + 1) as CatalogKeyPath<WebNamespaceMap['common']>,
        values
      );
    },
  };
}

export { resolveLocale, SUPPORTED_LOCALES };
export type { Locale };
