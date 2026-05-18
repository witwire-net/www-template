/**
 * セッション（デバイス）関連のAPIラッパーモジュール。
 *
 * このモジュールは、`@www-template/api` から生成されたセッション操作関数を
 * ドメイン層で使用しやすい形にラップし、エラーハンドリングと認証失敗分類を
 * 一元化する。エラー時は汎用的なメッセージを返し、機密情報を含まない。
 */

import {
  listSessions,
  revokeOtherSessions,
  revokeSession,
  type listSessionsResponse,
} from '@www-template/api';

const SESSION_EXPIRED_ERROR = 'session-expired';
const ACCOUNT_SUSPENDED_ERROR = 'account-suspended';

type SessionAuthFailure = 'session-expired' | 'unauthenticated' | 'account-suspended';

/** ドメイン層で使用するデバイスセッション表示モデル。 */
export interface DeviceSession {
  sessionId: string;
  deviceName: string;
  loginAt: string;
  lastActiveAt: string;
  ipHash: string;
  isCurrentSession: boolean;
}

/**
 * セッション（デバイス）一覧の取得結果。
 * 成功時は `data` にセッションリスト、失敗時は `error` に汎用メッセージを含む。
 * 認証エラー（401）の場合は `failure` に分類を含め、呼び元で状態遷移を委譲できる。
 */
export type ListDevicesResult =
  | { ok: true; data: DeviceSession[] }
  | { ok: false; error: string; status?: number; failure?: SessionAuthFailure };

/**
 * 認証操作の結果。
 * 成功時は `ok: true`、失敗時は `error` と任意で `failure` 分類を含む。
 */
export type AuthOperationResult =
  | { ok: true }
  | { ok: false; error: string; failure?: SessionAuthFailure };

/**
 * 401 レスポンスから認証失敗分類を抽出する共通ヘルパー。
 *
 * @param response - API レスポンスオブジェクト（status と data を含む）
 * @returns 分類文字列、または null
 */
function extractAuthFailure(response: {
  status: number;
  data: unknown;
}): SessionAuthFailure | null {
  if (
    (response.status === 401 || response.status === 403) &&
    typeof response.data === 'object' &&
    response.data !== null &&
    'error' in response.data
  ) {
    const err = (response.data as { error: string }).error;
    if (
      err === SESSION_EXPIRED_ERROR ||
      err === 'unauthenticated' ||
      err === ACCOUNT_SUSPENDED_ERROR
    ) {
      return err;
    }
  }
  return null;
}

/**
 * アクティブなアクセストークンを用いて、ログイン中のセッション一覧を取得する。
 * エラー時は汎用的なメッセージを返し、機密情報を含まない。
 *
 * @param headers - API 呼び出しに使用するヘッダー（Bearer 認証を含む）
 * @returns セッション一覧、またはエラー結果
 */
export async function fetchDevices(headers: Record<string, string>): Promise<ListDevicesResult> {
  try {
    const response: listSessionsResponse = await listSessions({ headers });
    if (response.status === 200 && 'sessions' in response.data) {
      return { ok: true, data: response.data.sessions as DeviceSession[] };
    }
    if ((response.status === 401 || response.status === 403) && 'error' in response.data) {
      const err = (response.data as { error: string }).error;
      if (
        err === SESSION_EXPIRED_ERROR ||
        err === 'unauthenticated' ||
        err === ACCOUNT_SUSPENDED_ERROR
      ) {
        return {
          ok: false,
          error: 'デバイス一覧の取得に失敗しました。',
          status: response.status,
          failure: err,
        };
      }
    }
    return { ok: false, error: 'デバイス一覧の取得に失敗しました。' };
  } catch {
    return { ok: false, error: 'デバイス一覧の取得に失敗しました。' };
  }
}

/**
 * 指定されたセッションをリモートで無効化する。
 *
 * @param sessionId - 無効化対象のセッション ID
 * @param headers - API 呼び出しに使用するヘッダー
 * @returns 成功/失敗と汎用メッセージ
 */
export async function revokeDevice(
  sessionId: string,
  headers: Record<string, string>
): Promise<AuthOperationResult> {
  try {
    const response = await revokeSession(sessionId, { headers });
    if (response.status === 204) {
      return { ok: true };
    }
    const failure = extractAuthFailure(response);
    return {
      ok: false,
      error: 'デバイスのログアウトに失敗しました。',
      failure: failure ?? undefined,
    };
  } catch {
    return { ok: false, error: 'デバイスのログアウトに失敗しました。' };
  }
}

/**
 * 現在のセッションを除くすべてのセッションを一括で無効化する。
 *
 * @param headers - API 呼び出しに使用するヘッダー
 * @returns 成功/失敗と汎用メッセージ
 */
export async function revokeOtherDevices(
  headers: Record<string, string>
): Promise<AuthOperationResult> {
  try {
    const response = await revokeOtherSessions({ headers });
    if (response.status === 204) {
      return { ok: true };
    }
    const failure = extractAuthFailure(response);
    return {
      ok: false,
      error: '他のデバイスのログアウトに失敗しました。',
      failure: failure ?? undefined,
    };
  } catch {
    return { ok: false, error: '他のデバイスのログアウトに失敗しました。' };
  }
}
