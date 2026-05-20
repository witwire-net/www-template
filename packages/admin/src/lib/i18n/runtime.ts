import {
  createTranslator,
  defineI18nConfig,
  loadJsonCatalog,
  type CatalogKeyPath,
  type CatalogTree,
  type LocaleCatalogMap,
} from '@www-template/i18n';

import enCommonCatalog from './messages/en/common.json';
import jaCommonCatalog from './messages/ja/common.json';

/**
 * Admin Console が package-local に扱う表示 locale です。
 *
 * Product AccountSetting や生成 Product SDK の型を参照せず、Admin operator locale と
 * Admin-owned JSON catalog の選択だけに使います。
 */
export type AdminLocale = 'ja' | 'en';

/**
 * Admin Console の翻訳テンプレートに渡せる補間値です。
 *
 * JSON catalog の文字列テンプレートへ埋め込む目的に限定し、HTML や object を
 * 直接 UI に出さないよう primitive のみに絞ります。
 */
export type AdminTranslationValue = string | number | boolean;

/**
 * Admin Console の翻訳テンプレートへ渡す補間値の集合です。
 */
export type AdminTranslationValues = Readonly<Record<string, AdminTranslationValue>>;

/**
 * Admin Console の package-local i18n API です。
 *
 * `t` は common namespace に事前固定されているため、Admin route / component は
 * `$lib/i18n` からこの API を 1 回 import するだけで文言を取得できます。
 */
export interface AdminI18n {
  /** 実際に採用された Admin 表示 locale です。 */
  readonly locale: AdminLocale;
  /** 指定 key の Admin-owned 翻訳文を返します。 */
  readonly t: (key: string, values?: AdminTranslationValues) => string;
}

/** Admin Console が対応する locale の一覧です。 */
export const ADMIN_LOCALES = ['ja', 'en'] as const;

/** 認証前画面や未知候補に使う Admin fallback locale です。 */
export const ADMIN_FALLBACK_LOCALE: AdminLocale = 'ja';

type AdminCatalog = Readonly<{ common: CatalogTree }>;
type AdminTranslationKey = CatalogKeyPath<AdminCatalog['common']>;

const adminCatalogs: LocaleCatalogMap<AdminCatalog> = {
  ja: { common: loadJsonCatalog(jaCommonCatalog) },
  en: { common: loadJsonCatalog(enCommonCatalog) },
};

/**
 * Admin Console 用の事前設定済み i18n API を作成します。
 *
 * @param localeCandidate Admin operator locale または認証前 fallback 用の候補値です。
 * @returns common namespace に固定された `t` 関数と、採用済み locale です。
 */
export function createAdminI18n(localeCandidate: string | null | undefined = null): AdminI18n {
  // shared frontend i18n core で locale を正規化し、Admin の catalog を解決する。
  const runtime = createTranslator(
    defineI18nConfig({
      locale: localeCandidate,
      supportedLocales: ADMIN_LOCALES,
      defaultLocale: ADMIN_FALLBACK_LOCALE,
      fallbackLocale: ADMIN_FALLBACK_LOCALE,
    }),
    adminCatalogs
  );

  return {
    locale: runtime.locale as AdminLocale,
    t: (key, values = {}) => {
      // common namespace を shared translator で解決し、欠落時だけ key を返して surface contract を維持する。
      try {
        return runtime.t('common', key as AdminTranslationKey, values);
      } catch {
        return key;
      }
    },
  };
}
