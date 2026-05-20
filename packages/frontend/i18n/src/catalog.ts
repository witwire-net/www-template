import { type MaybePromise, cloneCatalogTree } from './internal';
import { SUPPORTED_LOCALES, type Locale } from './locales';

/**
 * catalog の leaf として許容する値です。
 */
export type CatalogValue = string | CatalogTree;

/**
 * 1 つの JSON catalog が持つ再帰的な tree 構造です。
 * interface を使うことで TypeScript の型エイリアス循環参照を回避します。
 */
export interface CatalogTree {
  readonly [key: string]: CatalogValue;
}

/**
 * locale ごとの namespace catalog を束ねた型です。
 */
export type LocaleCatalogMap<TNamespaceMap extends object> = Readonly<
  Record<Locale, TNamespaceMap>
>;

/**
 * namespace ごとの loader を束ねた型です。
 */
export type NamespaceCatalogLoaders<TNamespaceMap extends object> = {
  readonly [Namespace in keyof TNamespaceMap]: () => MaybePromise<TNamespaceMap[Namespace]>;
};

/**
 * locale ごとの namespace loader を束ねた型です。
 */
export type LocaleCatalogLoaders<TNamespaceMap extends object> = Readonly<
  Record<Locale, NamespaceCatalogLoaders<TNamespaceMap>>
>;

/**
 * dot 区切りの翻訳 key path を表します。
 */
export type CatalogKeyPath<TCatalog> = TCatalog extends string
  ? never
  : TCatalog extends object
    ? {
        [Key in keyof TCatalog & string]: TCatalog[Key] extends string
          ? Key
          : TCatalog[Key] extends object
            ? `${Key}.${CatalogKeyPath<TCatalog[Key]>}`
            : never;
      }[keyof TCatalog & string]
    : never;

/**
 * JSON catalog を安全に読み込むための loader です。
 */
export interface JsonCatalogLoader<TNamespaceMap extends object> {
  /**
   * 指定 locale の namespace catalog を一括で読み込みます。
   */
  load(locale: Locale): Promise<TNamespaceMap>;

  /**
   * すべての locale の namespace catalog を一括で読み込みます。
   */
  loadAll(): Promise<LocaleCatalogMap<TNamespaceMap>>;
}

/**
 * JSON catalog を安全な tree に変換します。
 */
export function loadJsonCatalog<const TCatalog extends CatalogTree>(source: TCatalog): TCatalog;

/**
 * JSON catalog を安全な tree に変換します。
 */
export function loadJsonCatalog(source: unknown): CatalogTree;

/**
 * JSON catalog を安全な tree に変換します。
 */
export function loadJsonCatalog(source: unknown): CatalogTree {
  return cloneCatalogTree(source) as CatalogTree;
}

/**
 * locale と namespace の組み合わせから JSON catalog loader を組み立てます。
 *
 * surface 側は `messages/{locale}/{namespace}.json` を個別に所有し、この loader に
 * 渡すだけで安全な読み込みと型付き bundle 化を共通化できます。
 */
export function createJsonCatalogLoader<const TNamespaceMap extends object>(
  loaders: LocaleCatalogLoaders<TNamespaceMap>
): JsonCatalogLoader<TNamespaceMap> {
  const localeLoaders = new Map<Locale, NamespaceCatalogLoaders<TNamespaceMap>>([
    ['ja', loaders.ja],
    ['en', loaders.en],
  ]);
  const localeCache = new Map<Locale, Promise<TNamespaceMap>>();

  const loadLocale = (locale: Locale): Promise<TNamespaceMap> => {
    const cached = localeCache.get(locale);
    if (cached !== undefined) {
      return cached;
    }

    const namespaceLoaders = localeLoaders.get(locale);
    if (namespaceLoaders === undefined) {
      throw new Error(`i18n の locale loader が見つかりません: ${locale}`);
    }

    const task = (async () => {
      const catalog = Object.create(null) as TNamespaceMap;

      for (const namespace of Object.keys(namespaceLoaders) as (keyof TNamespaceMap & string)[]) {
        const loader = Reflect.get(namespaceLoaders, namespace) as
          | (() => MaybePromise<CatalogTree>)
          | undefined;
        if (loader === undefined) {
          continue;
        }

        Reflect.set(catalog, namespace, loadJsonCatalog(await loader()));
      }

      return Object.freeze(catalog);
    })();

    localeCache.set(locale, task);
    return task;
  };

  return {
    load: loadLocale,
    loadAll: async () => {
      const catalogs = Object.create(null) as LocaleCatalogMap<TNamespaceMap>;

      for (const locale of SUPPORTED_LOCALES) {
        Reflect.set(catalogs, locale, await loadLocale(locale));
      }

      return Object.freeze(catalogs);
    },
  };
}
