import { type CatalogKeyPath, type CatalogTree, type LocaleCatalogMap } from './catalog';
import { type I18nConfig } from './config';
import { createFormatters, type I18nFormatters } from './formatters';
import { getCatalogLeaf, interpolateTemplate, type TranslationValue } from './internal';
import { type Locale } from './locales';

/**
 * 翻訳値の補間に使う入力です。
 */
export type TranslationValues = Readonly<Record<string, TranslationValue>>;

/**
 * typed translator が提供する操作です。
 */
export interface Translator<TNamespaceMap extends object> {
  /**
   * 現在選択されている locale です。
   */
  readonly locale: Locale;

  /**
   * fallback に使う locale です。
   */
  readonly fallbackLocale: Locale;

  /**
   * locale 固定の formatter 群です。
   */
  readonly formatters: I18nFormatters;

  /**
   * 指定 namespace の翻訳文を取得します。
   */
  t<Namespace extends keyof TNamespaceMap & string>(
    namespace: Namespace,
    key: CatalogKeyPath<TNamespaceMap[Namespace]>,
    values?: TranslationValues
  ): string;

  /**
   * 指定 key が存在するかどうかを判定します。
   */
  has<Namespace extends keyof TNamespaceMap & string>(
    namespace: Namespace,
    key: CatalogKeyPath<TNamespaceMap[Namespace]>
  ): boolean;
}

const resolveNamespaceCatalog = <TNamespaceMap extends object>(
  catalogs: ReadonlyMap<Locale, TNamespaceMap>,
  locale: Locale,
  fallbackLocale: Locale,
  namespace: keyof TNamespaceMap & string
): CatalogTree | undefined => {
  const localeCatalog = catalogs.get(locale) ?? catalogs.get(fallbackLocale);
  if (localeCatalog === undefined) {
    return undefined;
  }

  const namespaceCatalog: unknown = Reflect.get(
    localeCatalog as Record<string, unknown>,
    namespace
  );
  return typeof namespaceCatalog === 'object' && namespaceCatalog !== null
    ? (namespaceCatalog as CatalogTree)
    : undefined;
};

const resolveMessage = <TNamespaceMap extends object>(
  catalogs: ReadonlyMap<Locale, TNamespaceMap>,
  config: I18nConfig,
  namespace: keyof TNamespaceMap & string,
  key: string
): string => {
  const catalog = resolveNamespaceCatalog(
    catalogs,
    config.locale,
    config.fallbackLocale,
    namespace
  );
  if (catalog === undefined) {
    throw new Error(`i18n namespace が見つかりません: ${namespace}`);
  }

  const message = getCatalogLeaf(
    catalog as Record<string, unknown>,
    key,
    config.namespaceSeparator
  );
  if (message === undefined) {
    throw new Error(`i18n key が見つかりません: ${namespace}.${key}`);
  }

  return message;
};

/**
 * typed translator を生成します。
 *
 * 現在 locale と fallback locale の両方を保持し、namespace 単位で分割された
 * catalog から安全に文字列を引き当てます。
 */
export function createTranslator<const TNamespaceMap extends object>(
  config: I18nConfig,
  catalogs: LocaleCatalogMap<TNamespaceMap>
): Translator<TNamespaceMap> {
  const formatters = createFormatters(config.locale);
  const localeCatalogs = new Map<Locale, TNamespaceMap>([
    ['ja', catalogs.ja],
    ['en', catalogs.en],
  ]);

  return {
    locale: config.locale,
    fallbackLocale: config.fallbackLocale,
    formatters,
    t(namespace, key, values = {}) {
      const message = resolveMessage(localeCatalogs, config, namespace, key);
      return interpolateTemplate(message, values);
    },
    has(namespace, key) {
      try {
        resolveMessage(localeCatalogs, config, namespace, key);
        return true;
      } catch {
        return false;
      }
    },
  };
}
