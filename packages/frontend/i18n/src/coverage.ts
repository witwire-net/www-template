import { collectCatalogLeafPaths } from './internal';
import { SUPPORTED_LOCALES, type Locale } from './locales';

/**
 * 辞書カバレッジの差分 1 件を表します。
 */
export interface CoverageIssue {
  /**
   * 欠落が見つかった locale です。
   */
  readonly locale: Locale;

  /**
   * 対象 namespace です。
   */
  readonly namespace: string;

  /**
   * その locale に存在しない key path です。
   */
  readonly missingKeys: readonly string[];
}

/**
 * 辞書カバレッジの集計結果です。
 */
export interface CoverageReport {
  /**
   * すべての locale で key 差分がない場合に `true` になります。
   */
  readonly complete: boolean;

  /**
   * 不足している key の一覧です。
   */
  readonly issues: readonly CoverageIssue[];
}

const sortStrings = (values: Iterable<string>): string[] =>
  [...values].sort((left, right) => left.localeCompare(right));

const toLocaleMap = <TNamespaceMap extends object>(
  catalogs: Readonly<Record<Locale, TNamespaceMap>>
): ReadonlyMap<Locale, TNamespaceMap> =>
  new Map<Locale, TNamespaceMap>([
    ['ja', catalogs.ja],
    ['en', catalogs.en],
  ]);

const collectNamespaceNames = <TNamespaceMap extends object>(
  catalogMap: ReadonlyMap<Locale, TNamespaceMap>,
  locales: readonly Locale[]
): string[] => {
  const namespaceNameSet = new Set<string>();

  for (const locale of locales) {
    const localeCatalog = catalogMap.get(locale);
    if (localeCatalog === undefined) {
      continue;
    }

    for (const namespace of Object.keys(localeCatalog)) {
      namespaceNameSet.add(namespace);
    }
  }

  return sortStrings(namespaceNameSet);
};

const collectNamespaceKeys = <TNamespaceMap extends object>(
  catalogMap: ReadonlyMap<Locale, TNamespaceMap>,
  locales: readonly Locale[],
  namespace: string
): Set<string> => {
  const keys = new Set<string>();

  for (const locale of locales) {
    const localeCatalog = catalogMap.get(locale);
    if (localeCatalog === undefined) {
      continue;
    }

    const namespaceCatalog: unknown = Reflect.get(
      localeCatalog as Record<string, unknown>,
      namespace
    );
    if (typeof namespaceCatalog === 'string') {
      keys.add(namespace);
      continue;
    }

    if (namespaceCatalog !== undefined && typeof namespaceCatalog === 'object') {
      for (const key of collectCatalogLeafPaths(namespaceCatalog as Record<string, unknown>)) {
        keys.add(key);
      }
    }
  }

  return keys;
};

/**
 * locale 間で namespace ごとの key coverage を確認します。
 *
 * どの locale にどの key が不足しているかを一覧化し、辞書の非対称を検出します。
 */
export function getCatalogCoverage<const TNamespaceMap extends object>(
  catalogs: Readonly<Record<Locale, TNamespaceMap>>,
  options: {
    readonly locales?: readonly Locale[];
  } = {}
): CoverageReport {
  const locales = options.locales ?? SUPPORTED_LOCALES;
  const catalogMap = toLocaleMap(catalogs);
  const namespaceNames = collectNamespaceNames(catalogMap, locales);

  const issues: CoverageIssue[] = [];

  for (const namespace of namespaceNames) {
    const referenceKeys = collectNamespaceKeys(catalogMap, locales, namespace);

    for (const locale of locales) {
      const localeCatalog = catalogMap.get(locale);
      const namespaceCatalog: unknown =
        localeCatalog === undefined
          ? undefined
          : Reflect.get(localeCatalog as Record<string, unknown>, namespace);
      const localeKeys =
        namespaceCatalog !== undefined && typeof namespaceCatalog === 'object'
          ? new Set(collectCatalogLeafPaths(namespaceCatalog as Record<string, unknown>))
          : new Set<string>();
      const missingKeys = sortStrings([...referenceKeys].filter((key) => !localeKeys.has(key)));

      if (missingKeys.length > 0) {
        issues.push({
          locale,
          namespace,
          missingKeys,
        });
      }
    }
  }

  return Object.freeze({
    complete: issues.length === 0,
    issues: Object.freeze(issues),
  });
}

const formatCoverageIssues = (issues: readonly CoverageIssue[]): string =>
  issues
    .map((issue) => `${issue.locale}:${issue.namespace} -> ${issue.missingKeys.join(', ')}`)
    .join('\n');

/**
 * 辞書 coverage が不完全な場合に例外を投げます。
 */
export function assertCatalogCoverage<const TNamespaceMap extends object>(
  catalogs: Readonly<Record<Locale, TNamespaceMap>>,
  options: {
    readonly locales?: readonly Locale[];
  } = {}
): asserts catalogs is Readonly<Record<Locale, TNamespaceMap>> {
  const report = getCatalogCoverage(catalogs, options);
  if (report.complete) {
    return;
  }

  throw new Error(`i18n catalog coverage が不完全です:\n${formatCoverageIssues(report.issues)}`);
}
