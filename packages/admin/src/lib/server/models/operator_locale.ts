/**
 * Admin Console のオペレーター表示言語として保存できる値の一覧です。
 * Product AccountSetting とは独立した Admin package-local な定義として扱います。
 */
export const ADMIN_OPERATOR_LOCALES = ['ja', 'en'] as const;

/**
 * Admin Console のオペレーター表示言語です。
 */
export type OperatorLocale = (typeof ADMIN_OPERATOR_LOCALES)[number];

/**
 * 明示的な設定がない Admin operator に適用する既定 locale です。
 */
export const DEFAULT_OPERATOR_LOCALE: OperatorLocale = 'ja';

/**
 * 入力値が Admin operator locale として保存可能か検証します。
 *
 * @param value DB または form から渡された候補値
 * @returns 対応済み Admin operator locale
 * @throws 未対応値の場合。DB の未知値を既定値へ丸めず fail-closed するために投げます。
 */
export function parseOperatorLocale(value: string): OperatorLocale {
  // DB 由来・form 由来の値を同じ基準で正規化し、余分な空白だけを許容する。
  const normalized = value.trim();
  // 対応済み値だけを返し、未知値は Product AccountSetting に委譲せず Admin 境界内で拒否する。
  if (ADMIN_OPERATOR_LOCALES.some((locale) => locale === normalized)) {
    return normalized as OperatorLocale;
  }
  // 既定値へ黙って丸めると DB 破損や不正入力を隠すため、呼び出し側で 400/503 に変換できる error にする。
  throw new Error('unsupported admin operator locale');
}

/**
 * 入力値が Admin operator locale として有効か判定します。
 *
 * @param value form などから渡された候補値
 * @returns 対応済み locale の場合 true
 */
export function isOperatorLocale(value: string): value is OperatorLocale {
  // parseOperatorLocale と同じ受理範囲を boolean API として提供し、service 層の分岐を読みやすくする。
  return ADMIN_OPERATOR_LOCALES.some((locale) => locale === value.trim());
}
