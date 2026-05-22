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

/**
 * FormData から JSON 文字列を安全にパースする。
 *
 * WebAuthn 応答 JSON の破損を明示的な結果で返し、未捕捉例外を避ける。
 *
 * @param value FormData から取得したエントリ値
 * @returns パース成功時は `{ ok: true, value }`、失敗時は `{ ok: false }`
 */
export function parseJsonFormField(
  value: FormDataEntryValue | null
): { ok: true; value: unknown } | { ok: false } {
  if (typeof value !== 'string') return { ok: false };
  try {
    return { ok: true, value: JSON.parse(value) as unknown };
  } catch {
    return { ok: false };
  }
}
