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
    routeIntent: '/login',
    lastFailure: null,
    lastError: null,
    lastCacheControl: null,
  };
}

/** active bearer session を state に反映する。 */
function applyAuthenticatedSession(
  state: AuthSessionState,
  session: AuthSessionSummary,
  cacheControl: string | null
): void {
  state.phase = 'authenticated';
  state.session = session;
  state.routeIntent = '/login';
  state.lastFailure = null;
  state.lastError = null;
  state.lastCacheControl = cacheControl;
}

/** missing session を通常 login 導線へ正規化する。 */
function applyMissingSession(
  state: AuthSessionState,
  cacheControl: string | null = null
): AuthRouteIntent {
  state.phase = 'anonymous';
  state.session = null;
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
  state.routeIntent = '/session-expired';
  state.lastFailure = 'session-expired';
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
    Authorization: `Bearer ${state.session.sessionToken}`,
  };
}

/** auth summary が ULID 方針を満たすか確認する。 */
function hasUlidAuthSessionShape(session: AuthSessionSummary): boolean {
  return [
    session.requestId,
    session.accountId,
    session.passkeyCredentialId,
    session.sessionId,
  ].every(isUlid);
}

export {
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
};
