import {
  DEFAULT_LOCALE,
  FALLBACK_LOCALE,
  type Locale,
  type LocaleCandidate,
  SUPPORTED_LOCALES,
  resolveLocale,
} from './locales';

/**
 * i18n runtime の設定入力です。
 *
 * locale の決定、fallback、対応ロケール群、namespace の区切り文字を
 * まとめて扱います。
 */
export interface I18nConfigInput {
  readonly locale?: LocaleCandidate;
  readonly supportedLocales?: readonly Locale[];
  readonly defaultLocale?: Locale;
  readonly fallbackLocale?: Locale;
  readonly namespaceSeparator?: string;
}

/**
 * i18n runtime の確定済み設定です。
 */
export interface I18nConfig {
  readonly locale: Locale;
  readonly supportedLocales: readonly Locale[];
  readonly defaultLocale: Locale;
  readonly fallbackLocale: Locale;
  readonly namespaceSeparator: string;
}

const normalizeSupportedLocales = (supportedLocales: readonly Locale[]): readonly Locale[] => {
  const uniqueLocales = [...new Set(supportedLocales)];
  if (uniqueLocales.length === 0) {
    throw new Error('i18n の対応ロケールは 1 件以上必要です。');
  }

  return Object.freeze(uniqueLocales);
};

const ensureSupported = (
  locale: Locale,
  supportedLocales: readonly Locale[],
  label: string
): Locale => {
  if (!supportedLocales.includes(locale)) {
    throw new Error(`i18n ${label} が対応ロケール一覧に含まれていません: ${locale}`);
  }

  return locale;
};

/**
 * i18n runtime の確定設定を生成します。
 *
 * 未指定時は `ja` / `en` を対応ロケールとして扱い、locale は候補から安全に
 * 解決します。namespace separator は空文字を拒否し、key path の予測可能性を
 * 保ちます。
 */
export function defineI18nConfig(input: I18nConfigInput = {}): I18nConfig {
  const supportedLocales = normalizeSupportedLocales(input.supportedLocales ?? SUPPORTED_LOCALES);
  const defaultLocale = ensureSupported(
    input.defaultLocale ?? DEFAULT_LOCALE,
    supportedLocales,
    'defaultLocale'
  );
  const fallbackLocale = ensureSupported(
    input.fallbackLocale ?? FALLBACK_LOCALE,
    supportedLocales,
    'fallbackLocale'
  );
  const locale = resolveLocale(input.locale, { supportedLocales, defaultLocale });
  const namespaceSeparator = input.namespaceSeparator ?? '.';

  if (namespaceSeparator.trim().length === 0) {
    throw new Error('i18n の namespaceSeparator は空文字にできません。');
  }

  return Object.freeze({
    locale,
    supportedLocales,
    defaultLocale,
    fallbackLocale,
    namespaceSeparator,
  });
}

/**
 * 設定から locale resolver を生成します。
 *
 * URL、ブラウザ言語、localStorage のような候補列から、同じ基準で locale を
 * 決定したい呼び出し側に使います。
 */
export function createLocaleResolver(config: I18nConfig): (candidate: LocaleCandidate) => Locale {
  return (candidate) =>
    resolveLocale(candidate, {
      supportedLocales: config.supportedLocales,
      defaultLocale: config.defaultLocale,
    });
}
