/**
 * JWT アクセストークンのペイロードに含まれる最小クレーム。
 * クライアントは署名を検証せず、デコードのみを行う。
 */
export interface AccessTokenClaims {
  /** アカウント識別子（JWT 標準クレーム `sub`）。 */
  accountId: string;
  /** セッション識別子（カスタムクレーム `sid`）。 */
  sessionId: string;
  /** JWT の有効期限（Unix タイムスタンプ、秒）。 */
  exp: number;
  /** JWT の発行時刻（Unix タイムスタンプ、秒）。 */
  iat: number;
}

/**
 * メモリ上にのみ保持されるトークンペア。
 * localStorage / sessionStorage / cookie / IndexedDB / URL への永続化は禁止。
 */
export interface MemoryTokenPair {
  /** JWT アクセストークン。 */
  accessToken: string;
  /** リフレッシュトークン。 */
  refreshToken: string;
}

/**
 * 空のトークンペアを生成する。
 * セッションクリア時などに使用する。
 *
 * @returns accessToken と refreshToken が空文字の MemoryTokenPair
 */
export function createEmptyTokenPair(): MemoryTokenPair {
  return { accessToken: '', refreshToken: '' };
}

/**
 * Base64url 文字列を通常の Base64 に変換する。
 * `atob` は標準 Base64 を要求するため、パディング復元と `-`/`_` の置換が必要。
 *
 * @param base64url - Base64url エンコードされた文字列
 * @returns 標準 Base64 文字列
 */
function base64urlToBase64(base64url: string): string {
  let base64 = base64url.replace(/-/gu, '+').replace(/_/gu, '/');
  const padding = base64.length % 4;
  if (padding === 2) {
    base64 += '==';
  } else if (padding === 3) {
    base64 += '=';
  }
  return base64;
}

/**
 * JWT アクセストークンのペイロードをデコードし、必要なクレームを抽出する。
 * 署名検証は行わない。フォーマット不正や必須クレーム欠落時は `null` を返す。
 *
 * @param token - JWT 形式のトークン文字列（header.payload.signature）
 * @returns デコードされた AccessTokenClaims、または失敗時 `null`
 */
export function decodeAccessToken(token: string): AccessTokenClaims | null {
  try {
    const parts = token.split('.');
    if (parts.length !== 3) {
      return null;
    }
    const payloadBase64 = base64urlToBase64(parts[1] ?? '');
    const payloadJson = atob(payloadBase64);
    const payload = JSON.parse(payloadJson) as Record<string, unknown>;

    const accountId = typeof payload.sub === 'string' ? payload.sub : '';
    const sessionId = typeof payload.sid === 'string' ? payload.sid : '';
    const exp = typeof payload.exp === 'number' ? payload.exp : 0;
    const iat = typeof payload.iat === 'number' ? payload.iat : 0;

    if (accountId === '' || sessionId === '' || exp === 0) {
      return null;
    }

    return { accountId, sessionId, exp, iat };
  } catch {
    return null;
  }
}

/**
 * アクセストークンのリフレッシュが必要かどうかを判定する。
 * 現在時刻から指定マージン（デフォルト 1 分）を加味し、
 * 有効期限が切れている、または間近の場合に `true` を返す。
 *
 * @param exp - JWT の有効期限（Unix タイムスタンプ、秒）
 * @param now - 現在時刻（Unix タイムスタンプ、ミリ秒）
 * @param marginMs - マージン（ミリ秒）。デフォルトは 60_000（1 分）
 * @returns リフレッシュが必要な場合 `true`
 */
export function isRefreshNeeded(exp: number, now: number, marginMs = 60_000): boolean {
  const expMs = exp * 1000;
  return expMs - now < marginMs;
}

/**
 * Unix タイムスタンプ（秒）を ISO 8601 文字列に変換する。
 * `.svelte.ts` 内で `new Date()` を直接使用すると lint エラーになるため、
 * 純粋なユーティリティ関数として切り出す。
 *
 * @param exp - Unix タイムスタンプ（秒）
 * @returns ISO 8601 形式の日時文字列
 */
export function expToIsoString(exp: number): string {
  return new Date(exp * 1000).toISOString();
}
