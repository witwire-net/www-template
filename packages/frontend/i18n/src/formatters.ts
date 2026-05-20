import { type Locale } from './locales';

/**
 * i18n で共通利用する formatter 群です。
 *
 * locale を固定し、日付・数値・リストの描画を呼び出し側から注入された
 * 一貫した設定で行います。
 */
export interface I18nFormatters {
  /**
   * 日付文字列を整形します。
   */
  date(value: Date | number | string, options?: Intl.DateTimeFormatOptions): string;

  /**
   * 日時文字列を整形します。
   */
  dateTime(value: Date | number | string, options?: Intl.DateTimeFormatOptions): string;

  /**
   * 数値を locale に合わせて整形します。
   */
  number(value: number, options?: Intl.NumberFormatOptions): string;

  /**
   * リストを locale に合わせて整形します。
   */
  list(value: readonly string[], options?: Intl.ListFormatOptions): string;
}

/**
 * formatter の初期設定です。
 */
export interface FormatterDefaults {
  readonly date?: Intl.DateTimeFormatOptions;
  readonly dateTime?: Intl.DateTimeFormatOptions;
  readonly number?: Intl.NumberFormatOptions;
  readonly list?: Intl.ListFormatOptions;
}

const toDate = (value: Date | number | string): Date => {
  const date = value instanceof Date ? value : new Date(value);
  if (Number.isNaN(date.getTime())) {
    throw new TypeError('i18n formatter に無効な日付値が渡されました。');
  }

  return date;
};

/**
 * locale 固定の formatter 群を生成します。
 *
 * `timeZone` を含めた既定値を呼び出し側で渡せるので、画面・テスト・メール文面の
 * いずれでも同じ設定を再利用できます。
 */
export function createFormatters(locale: Locale, defaults: FormatterDefaults = {}): I18nFormatters {
  return {
    date(value, options = {}) {
      return new Intl.DateTimeFormat(locale, { ...defaults.date, ...options }).format(
        toDate(value)
      );
    },
    dateTime(value, options = {}) {
      return new Intl.DateTimeFormat(locale, { ...defaults.dateTime, ...options }).format(
        toDate(value)
      );
    },
    number(value, options = {}) {
      return new Intl.NumberFormat(locale, { ...defaults.number, ...options }).format(value);
    },
    list(value, options = {}) {
      return new Intl.ListFormat(locale, { ...defaults.list, ...options }).format([...value]);
    },
  };
}
