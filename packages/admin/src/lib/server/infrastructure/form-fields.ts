/**
 * FormData から安全に文字列値を取り出す。
 *
 * @param form 取得対象の FormData
 * @param name フィールド名
 * @param fallback 文字列以外だった場合の既定値（デフォルトは空文字）
 * @returns 安全な文字列
 */
export function getFormString(form: FormData, name: string, fallback = ''): string {
  const value = form.get(name);
  return typeof value === 'string' ? value : fallback;
}
