import type { AuthRouteIntent, AuthSessionState, AuthSessionSummary } from '../types';

const ULID_PATTERN = /^[0-9A-HJKMNP-TV-Z]{26}$/u;

/** ULID 形式かどうかを検証する。 */
function isUlid(value: string): boolean {
  return ULID_PATTERN.test(value);
}

/** auth response の cache-control を no-store として扱えるか判定する。 */
function isNoStoreCacheControl(cacheControl: string | null): boolean {
  return cacheControl?.toLowerCase().includes('no-store') ?? false;
}

/** auth session state の初期値を作る。 */
function createAuthSessionInitialState(): AuthSessionState {
  return {
    phase: 'anonymous',
    session: null,
    sessions: [],
    activeSessionId: null,
    routeIntent: '/login',
    lastFailure: null,
    lastError: null,
    lastCacheControl: null,
    lastAccountSettingSnapshot: null,
  };
}

/**
 * active bearer session を state に反映する。
 * 受け取ったセッションを sessions 配列の唯一の要素として設定し、
 * activeSessionId や phase などの関連 state も同時に更新する。
 *
 * @param state - 更新対象の認証セッション state
 * @param session - 反映する認証セッション概要
 * @param cacheControl - レスポンスの cache-control 値（任意）
 */
function applyAuthenticatedSession(
  state: AuthSessionState,
  session: AuthSessionSummary,
  cacheControl: string | null
): void {
  state.phase = 'authenticated';
  state.session = session;
  state.sessions = [session];
  state.activeSessionId = session.sessionId;
  state.routeIntent = '/login';
  state.lastFailure = null;
  state.lastError = null;
  state.lastCacheControl = cacheControl;
}

/**
 * 新しいセッションを追加し、アクティブセッションとして設定する。
 *
 * - 同一 `sessionId` の既存 entry は新セッションで上書きする。
 * - 同一 `accountId` で異なる `sessionId` の既存 entry も新セッションで置換する。
 *   これにより同一アカウントがブラウザ内で重複表示されるのを防ぐ。
 * - 異なる `accountId` のセッションはマルチアカウント切替のため保持する。
 *
 * @param state - 更新対象の認証セッション state
 * @param session - 追加する認証セッション概要
 * @param cacheControl - レスポンスの cache-control 値
 */
function addAuthenticatedSession(
  state: AuthSessionState,
  session: AuthSessionSummary,
  cacheControl: string | null
): void {
  const sessions = state.sessions ?? [];
  // 同一 sessionId または同一 accountId の既存 entry を除去し、新セッションで置換する。
  // これにより同一アカウントの重複 entry を防止する。
  const filtered = sessions.filter(
    (s) => s.sessionId !== session.sessionId && s.accountId !== session.accountId
  );
  filtered.push(session);
  state.sessions = filtered;
  state.activeSessionId = session.sessionId;
  state.session = session;
  state.phase = 'authenticated';
  state.routeIntent = '/login';
  state.lastFailure = null;
  state.lastError = null;
  state.lastCacheControl = cacheControl;
}

/** アクティブセッションを指定した sessionId に切り替える。
 *  該当セッションが存在しない場合は何もしない。 */
function switchActiveSession(state: AuthSessionState, sessionId: string): boolean {
  const target = state.sessions?.find((s) => s.sessionId === sessionId);
  if (target == null) {
    return false;
  }
  state.session = target;
  state.activeSessionId = target.sessionId;
  return true;
}

/** アクティブセッションを除去する。
 *  残りのセッションがある場合は最初のセッションをアクティブにする。
 *  セッションが空になった場合は未認証状態に戻す。 */
function removeActiveSession(state: AuthSessionState): AuthRouteIntent | null {
  const remaining = (state.sessions ?? []).filter((s) => s.sessionId !== state.activeSessionId);
  state.sessions = remaining;

  if (remaining.length > 0) {
    const next = remaining[0];
    state.session = next;
    state.activeSessionId = next.sessionId;
    state.phase = 'authenticated';
    state.routeIntent = '/login';
    state.lastFailure = null;
    state.lastError = null;
    return null;
  }

  return clearAuthSession(state);
}

/**
 * 指定された sessionId をメモリ上の認証 state から除去する。
 *
 * @param state - 更新対象の認証セッション state
 * @param sessionId - 削除対象の sessionId
 * @param routeIntent - 対象がアクティブだった場合に設定する遷移先
 * @returns 対象がアクティブなら遷移先 route intent、非アクティブなら null
 */
function removeSessionById(
  state: AuthSessionState,
  sessionId: string,
  routeIntent: AuthRouteIntent = '/login'
): AuthRouteIntent | null {
  const remaining = (state.sessions ?? []).filter((s) => s.sessionId !== sessionId);
  state.sessions = remaining;

  if (state.activeSessionId !== sessionId) {
    return null;
  }

  state.session = null;
  state.activeSessionId = null;
  state.phase = routeIntent === '/account-suspended' ? 'account-suspended' : 'anonymous';
  state.routeIntent = routeIntent;
  state.lastFailure = routeIntent === '/account-suspended' ? 'account-suspended' : null;
  state.lastError = null;
  return routeIntent;
}

/** missing session を通常 login 導線へ正規化する。 */
function applyMissingSession(
  state: AuthSessionState,
  cacheControl: string | null = null
): AuthRouteIntent {
  state.phase = 'anonymous';
  state.session = null;
  state.sessions = [];
  state.activeSessionId = null;
  state.routeIntent = '/login';
  state.lastFailure = 'unauthenticated';
  state.lastError = null;
  state.lastCacheControl = cacheControl;
  return state.routeIntent;
}

/** expired / revoked session を session-expired 導線へ正規化する。 */
function applyExpiredSession(
  state: AuthSessionState,
  cacheControl: string | null = null
): AuthRouteIntent {
  state.phase = 'session-expired';
  state.session = null;
  state.sessions = [];
  state.activeSessionId = null;
  state.routeIntent = '/session-expired';
  state.lastFailure = 'session-expired';
  state.lastError = null;
  state.lastCacheControl = cacheControl;
  return state.routeIntent;
}

/**
 * suspended account の失敗を account-suspended 導線へ正規化する。
 *
 * @param state - 更新対象の認証セッション state
 * @param cacheControl - レスポンスの cache-control 値（任意）
 * @param targetSessionId - 削除対象 sessionId。未指定時は active session を対象にする
 * @returns account suspended 案内 route intent
 */
function applyAccountSuspended(
  state: AuthSessionState,
  cacheControl: string | null = null,
  targetSessionId: string | null = state.activeSessionId ?? null
): AuthRouteIntent {
  if (targetSessionId !== null) {
    removeSessionById(state, targetSessionId, '/account-suspended');
  }

  state.phase = 'account-suspended';
  state.session = null;
  state.activeSessionId = null;
  state.routeIntent = '/account-suspended';
  state.lastFailure = 'account-suspended';
  state.lastError = null;
  state.lastCacheControl = cacheControl;
  return state.routeIntent;
}

/** internal error を fail-close として保持する。 */
function applyInternalError(
  state: AuthSessionState,
  message: string,
  cacheControl: string | null = null
): void {
  state.lastFailure = 'internal-error';
  state.lastError = message;
  state.lastCacheControl = cacheControl;
}

/** logout や tab close 時に in-memory session を破棄する。 */
function clearAuthSession(
  state: AuthSessionState,
  cacheControl: string | null = null
): AuthRouteIntent {
  state.phase = 'anonymous';
  state.session = null;
  state.sessions = [];
  state.activeSessionId = null;
  state.routeIntent = '/login';
  state.lastFailure = null;
  state.lastError = null;
  state.lastCacheControl = cacheControl;
  return state.routeIntent;
}

/** current bearer token を Authorization header へ写像する。 */
function createAuthorizationHeaders(state: AuthSessionState): Record<string, string> {
  if (state.session === null) {
    return {};
  }

  return {
    Authorization: `Bearer ${state.session.accessToken}`,
  };
}

/** auth summary が ULID 方針を満たすか確認する。 */
function hasUlidAuthSessionShape(session: AuthSessionSummary): boolean {
  const required = [session.requestId, session.authContextId, session.accountId, session.sessionId];
  // passkeyCredentialId は refresh response で省略されることがあるため、
  // 存在する場合のみ ULID 検証を行う。
  if (session.passkeyCredentialId != null) {
    required.push(session.passkeyCredentialId);
  }
  return required.every(isUlid);
}

export {
  addAuthenticatedSession,
  applyAccountSuspended,
  applyAuthenticatedSession,
  applyExpiredSession,
  applyInternalError,
  applyMissingSession,
  clearAuthSession,
  createAuthSessionInitialState,
  createAuthorizationHeaders,
  hasUlidAuthSessionShape,
  isNoStoreCacheControl,
  isUlid,
  removeActiveSession,
  removeSessionById,
  switchActiveSession,
};
