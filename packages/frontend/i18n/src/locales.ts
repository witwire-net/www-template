/**
 * フロントエンド i18n で正式に扱う対応ロケールです。
 *
 * この配列は shared runtime の基準値であり、surface 側はここに含まれる
 * 値だけを辞書、resolver、loader の対象として扱います。
 */
export const SUPPORTED_LOCALES = ['ja', 'en'] as const;

/**
 * フロントエンド i18n で使う対応ロケールの型です。
 */
export type Locale = (typeof SUPPORTED_LOCALES)[number];

/**
 * 既定ロケールを表します。
 *
 * root レベルの fallback として使う前提で、決定的な初期値を提供します。
 */
export const DEFAULT_LOCALE: Locale = 'ja';

/**
 * 未選択時の fallback に使うロケールです。
 */
export const FALLBACK_LOCALE: Locale = DEFAULT_LOCALE;

/**
 * ロケール候補の入力型です。
 */
export type LocaleCandidate = string | readonly string[] | null | undefined;

/**
 * 受け取った文字列を `ja` / `en` へ正規化します。
 *
 * `ja-JP` や `en-US` のような派生タグは主ロケールへ丸め、未対応値は `null` を返します。
 */
export function normalizeLocale(candidate: string | null | undefined): Locale | null {
  if (candidate === null || candidate === undefined) {
    return null;
  }

  const normalized = candidate.trim().toLowerCase().replaceAll('_', '-');
  if (normalized.length === 0) {
    return null;
  }

  const [primary] = normalized.split('-');
  if (primary === 'ja' || primary === 'en') {
    return primary;
  }

  return null;
}

/**
 * 候補値の集合から対応ロケールを決定します。
 *
 * 複数候補が与えられた場合は先頭から評価し、対応ロケールが見つからなければ
 * 既定ロケールへ落とします。
 */
export function resolveLocale(
  candidate: LocaleCandidate,
  options: {
    readonly supportedLocales?: readonly Locale[];
    readonly defaultLocale?: Locale;
  } = {}
): Locale {
  const supportedLocales = options.supportedLocales ?? SUPPORTED_LOCALES;
  const defaultLocale = options.defaultLocale ?? DEFAULT_LOCALE;

  const candidates: readonly string[] = Array.isArray(candidate)
    ? candidate
    : candidate === null || candidate === undefined
      ? []
      : [candidate];

  for (const item of candidates) {
    const locale = normalizeLocale(item);
    if (locale !== null && supportedLocales.some((supportedLocale) => supportedLocale === locale)) {
      return locale;
    }
  }

  return supportedLocales.some((supportedLocale) => supportedLocale === defaultLocale)
    ? defaultLocale
    : (supportedLocales[0] ?? DEFAULT_LOCALE);
}

/**
 * 候補値が対応ロケールかどうかを判定します。
 */
export function isLocale(candidate: string | null | undefined): candidate is Locale {
  return normalizeLocale(candidate) !== null;
}
