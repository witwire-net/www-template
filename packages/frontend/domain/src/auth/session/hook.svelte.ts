import { SvelteMap, SvelteSet } from 'svelte/reactivity';

import { authApi, refreshToken } from '@www-template/api';

import {
  addAuthenticatedSession,
  applyAccountSuspended,
  applyExpiredSession,
  applyInternalError,
  applyMissingSession,
  clearAuthSession,
  createAuthSessionInitialState,
  createAuthorizationHeaders,
  hasUlidAuthSessionShape,
  removeActiveSession,
  switchActiveSession,
} from './state';

import {
  clearContextIndex,
  createEmptyContextIndex,
  readContextIndex,
  removeContextEntry,
  toContextIndexEntry,
  upsertContextEntry,
  writeContextIndex,
} from './context_index';

import { fetchDevices, revokeDevice, revokeOtherDevices } from './session_api';

import { decodeAccessToken, expToIsoString, isRefreshNeeded } from './token_state';

import type {
  AuthFailureState,
  AuthRouteIntent,
  AuthSessionState,
  AuthSessionSummary,
} from '../types';
import type { DeviceSession } from './session_api';
import type { AccessTokenClaims } from './token_state';

const SESSION_EXPIRED_ERROR = 'session-expired';
const ACCOUNT_SUSPENDED_ERROR = 'account-suspended';

/** same-origin Product API へのみ Cookie を添付する。 */
const COOKIE_AUTH_REQUEST_INIT = { credentials: 'same-origin' } as const satisfies RequestInit;

/** refresh 応答の accessToken が更新対象 session と一致するか検証する。 */
function isAccessTokenForSession(claims: AccessTokenClaims, session: AuthSessionSummary): boolean {
  return claims.accountId === session.accountId && claims.sessionId === session.sessionId;
}

/** context index bootstrap の進行状態。 */
type BootstrapPhase = 'pending' | 'done';

interface AuthSessionData {
  state: AuthSessionState;
  /** context index bootstrap の進行状態。guard が redirect 判断に使う。 */
  bootstrapPhase: { value: BootstrapPhase };
}

interface AuthSessionActions {
  acceptSession: (session: AuthSessionSummary, cacheControl: string | null) => void;
  createAuthorizationHeaders: () => Record<string, string>;
  handleFailure: (classification: AuthFailureState, message?: string) => AuthRouteIntent;
  handleMissingSession: () => AuthRouteIntent;
  clearInMemorySession: () => AuthRouteIntent;
  logoutCurrentSession: () => Promise<AuthRouteIntent | null>;
  refreshActiveSession: () => Promise<AuthRouteIntent | null>;
  switchSession: (sessionId: string) => boolean;
  listDevices: () => Promise<DeviceSession[] | null>;
  revokeDevice: (sessionId: string) => Promise<boolean>;
  revokeOtherDevices: () => Promise<boolean>;
}

/** bearer token は persistent storage に保存せず、tab close で破棄する。 */
const state = $state<AuthSessionState>(createAuthSessionInitialState());

/** authContextId 単位で実行中の refresh Promise を保持する。 */
const refreshInFlight = new SvelteMap<string, Promise<AuthRouteIntent | null>>();

/** トークン期限間近なら自動リフレッシュし、最新 Authorization ヘッダーを返す。 */
async function ensureFreshAuthorizationHeaders(
  authState: AuthSessionState
): Promise<Record<string, string>> {
  const active = authState.session;
  if (active == null) {
    return {};
  }

  const claims = decodeAccessToken(active.accessToken);
  if (claims != null && isRefreshNeeded(claims.exp, Date.now())) {
    const authContextId = active.authContextId;
    let inflight = refreshInFlight.get(authContextId);
    if (inflight == null) {
      inflight = executeRefreshActiveSession(authState, active.sessionId).finally(() => {
        refreshInFlight.delete(authContextId);
      });
      refreshInFlight.set(authContextId, inflight);
    }
    await inflight;
  }

  return createAuthorizationHeaders(authState);
}

/** ログアウト前の軽量リフレッシュ。失敗時もセッション遷移しない。 */
async function attemptRefreshForLogout(authState: AuthSessionState): Promise<void> {
  const active = authState.session;
  if (active == null) {
    return;
  }

  const claims = decodeAccessToken(active.accessToken);
  if (claims == null || !isRefreshNeeded(claims.exp, Date.now())) {
    return;
  }

  try {
    // Cookie-only refresh は body を持たず、HttpOnly Cookie だけを送信する。
    const response = await refreshToken(active.authContextId, undefined, COOKIE_AUTH_REQUEST_INIT);

    if (response.status === 200 && 'accessToken' in response.data) {
      const { accessToken } = response.data;
      const newClaims = decodeAccessToken(accessToken);
      if (newClaims == null || !isAccessTokenForSession(newClaims, active)) {
        return;
      }

      const updatedSession: AuthSessionSummary = {
        ...active,
        accessToken,
        expiresAt: expToIsoString(newClaims.exp),
      };

      authState.sessions = (authState.sessions ?? []).map((s) =>
        s.sessionId === active.sessionId ? updatedSession : s
      );

      if (authState.activeSessionId === active.sessionId) {
        authState.session = updatedSession;
      }
    }
  } catch {
    // ネットワークエラー等: 現在のトークンのままログアウトを継続する
  }
}

/** 現在アクティブなセッションをリモートでログアウトし、サーバー・ローカル双方で失効させる。 */
async function executeLogoutCurrentSession(
  authState: AuthSessionState
): Promise<AuthRouteIntent | null> {
  authState.phase = 'logging-out';

  if (authState.session === null) {
    return clearAuthSession(authState);
  }

  await attemptRefreshForLogout(authState);

  try {
    const headers = createAuthorizationHeaders(authState);
    if (headers.Authorization == null) {
      applyInternalError(authState, 'ログアウトに必要な認証情報がありません。');
      return removeActiveSession(authState);
    }

    const response = await authApi.logout({ ...COOKIE_AUTH_REQUEST_INIT, headers });

    if (response.status === 200) {
      const data = response.data as {
        revoked?: boolean;
        contextIndexUpdateHints?: { action: string; authContextId?: string }[];
      };
      // logout response の contextIndexUpdateHints に従い context index を同期する
      if (Array.isArray(data.contextIndexUpdateHints)) {
        const index = readContextIndex() ?? createEmptyContextIndex();
        for (const hint of data.contextIndexUpdateHints) {
          if (hint.action === 'remove' && hint.authContextId != null) {
            removeContextEntry(index, hint.authContextId);
          } else if (hint.action === 'clear-surface') {
            index.entries = [];
            index.activeAuthContextId = null;
          }
        }
        writeContextIndex(index);
      } else {
        // hint がない場合は active session の authContextId のみ削除する。
        const activeSession = authState.session;
        if (activeSession != null) {
          const index = readContextIndex() ?? createEmptyContextIndex();
          removeContextEntry(index, activeSession.authContextId);
          writeContextIndex(index);
        }
      }
      return removeActiveSession(authState);
    }

    if (response.data.error === SESSION_EXPIRED_ERROR) {
      return removeActiveSession(authState);
    }

    applyInternalError(authState, 'ログアウトに失敗しました。');
    return removeActiveSession(authState);
  } catch (error: unknown) {
    applyInternalError(
      authState,
      error instanceof Error ? error.message : 'ログアウトに失敗しました。'
    );
    return removeActiveSession(authState);
  }
}

/** リフレッシュ失敗時に対象セッションを整理する。 */
function handleRefreshFailureForTarget(
  authState: AuthSessionState,
  targetSessionId: string
): AuthRouteIntent | null {
  const targetSession = authState.sessions?.find((s) => s.sessionId === targetSessionId);
  if (targetSession != null) {
    const index = readContextIndex() ?? createEmptyContextIndex();
    removeContextEntry(index, targetSession.authContextId);
    writeContextIndex(index);
  }

  if (authState.activeSessionId !== targetSessionId) {
    authState.sessions = authState.sessions?.filter((s) => s.sessionId !== targetSessionId) ?? [];
    return null;
  }
  return applyExpiredSession(authState);
}

/** suspended account 応答を対象セッション単位で処理する。 */
function handleAccountSuspendedForTarget(
  authState: AuthSessionState,
  targetSessionId: string,
  cacheControl: string | null = null
): AuthRouteIntent | null {
  const targetSession = authState.sessions?.find((s) => s.sessionId === targetSessionId);
  if (targetSession != null) {
    const index = readContextIndex() ?? createEmptyContextIndex();
    removeContextEntry(index, targetSession.authContextId);
    writeContextIndex(index);
  }

  if (authState.activeSessionId !== targetSessionId) {
    authState.sessions = authState.sessions?.filter((s) => s.sessionId !== targetSessionId) ?? [];
    return null;
  }

  return applyAccountSuspended(authState, cacheControl, targetSessionId);
}

/** HttpOnly Cookie でリフレッシュし新トークンを取得する。 */
async function executeRefreshActiveSession(
  authState: AuthSessionState,
  targetSessionId: string
): Promise<AuthRouteIntent | null> {
  const targetSession =
    authState.sessions?.find((s) => s.sessionId === targetSessionId) ?? authState.session;

  if (targetSession == null) {
    return handleRefreshFailureForTarget(authState, targetSessionId);
  }

  try {
    // authContextId を path parameter にし、HttpOnly Cookie だけを送信する。
    const response = await refreshToken(
      targetSession.authContextId,
      undefined,
      COOKIE_AUTH_REQUEST_INIT
    );

    if (response.status === 200 && 'accessToken' in response.data) {
      const { accessToken, accountSetting } = response.data;
      const claims = decodeAccessToken(accessToken);
      if (claims == null || !isAccessTokenForSession(claims, targetSession)) {
        return handleRefreshFailureForTarget(authState, targetSessionId);
      }

      const updatedSession: AuthSessionSummary = {
        ...targetSession,
        accessToken,
        expiresAt: expToIsoString(claims.exp),
      };

      // 対象セッションのみを sessions 配列内で更新する
      authState.sessions = (authState.sessions ?? []).map((s) =>
        s.sessionId === targetSessionId ? updatedSession : s
      );

      // activeSessionId が同一の場合のみ active session proxy を差し替える
      if (authState.activeSessionId === targetSessionId) {
        authState.session = updatedSession;
      }

      // AccountSetting snapshot が含まれていれば state に保存する
      if (accountSetting?.locale !== undefined) {
        authState.lastAccountSettingSnapshot = { locale: accountSetting.locale };
      }

      // refresh 成功時は context index を更新する
      const index = readContextIndex() ?? createEmptyContextIndex();
      upsertContextEntry(
        index,
        toContextIndexEntry(updatedSession, updatedSession.expiresAt),
        authState.activeSessionId === targetSessionId
      );
      writeContextIndex(index);

      return null;
    }

    if (response.status === 403 && response.data.error === ACCOUNT_SUSPENDED_ERROR) {
      return handleAccountSuspendedForTarget(
        authState,
        targetSessionId,
        response.headers.get('cache-control')
      );
    }

    return handleRefreshFailureForTarget(authState, targetSessionId);
  } catch {
    return handleRefreshFailureForTarget(authState, targetSessionId);
  }
}

/** protected API call をラップし、session-expired 時に refresh-once retry を行う。 */
async function withRefreshRetry<T>(
  authState: AuthSessionState,
  apiCall: (headers: Record<string, string>) => Promise<
    | { ok: true; data: T }
    | {
        ok: false;
        error: string;
        status?: number;
        failure?: 'session-expired' | 'unauthenticated' | 'account-suspended';
      }
  >
): Promise<T | null> {
  const headers = await ensureFreshAuthorizationHeaders(authState);
  if (headers.Authorization == null) {
    return null;
  }

  let result = await apiCall(headers);

  // session-expired の場合、active session を 1 回だけ refresh して retry する
  if (!result.ok && result.failure === SESSION_EXPIRED_ERROR) {
    const active = authState.session;
    if (active != null) {
      const refreshResult = await executeRefreshActiveSession(authState, active.sessionId);
      if (refreshResult == null) {
        // refresh 成功: 新しい Authorization header で retry
        const retryHeaders = createAuthorizationHeaders(authState);
        if (retryHeaders.Authorization != null) {
          result = await apiCall(retryHeaders);
        }
      }
      // refresh 失敗時は executeRefreshActiveSession 内で失効処理済み
    }
  }

  if (!result.ok) {
    if (result.failure === SESSION_EXPIRED_ERROR) {
      applyExpiredSession(authState);
      return null;
    }
    if (result.failure === ACCOUNT_SUSPENDED_ERROR) {
      applyAccountSuspended(authState);
      return null;
    }
    if (result.failure === 'unauthenticated') {
      applyMissingSession(authState);
      return null;
    }
    return null;
  }

  return result.data;
}

/** ログイン中の全セッション（デバイス）一覧を取得する。 */
async function executeListDevices(authState: AuthSessionState): Promise<DeviceSession[] | null> {
  return withRefreshRetry(authState, (headers) => fetchDevices(headers));
}

/** 指定されたセッションをリモートで無効化し、ローカル state と context index を更新する。 */
async function executeRevokeDevice(
  authState: AuthSessionState,
  sessionId: string
): Promise<boolean> {
  const result = await withRefreshRetry(authState, async (headers) => {
    const res = await revokeDevice(sessionId, headers);
    if (res.ok) {
      return { ok: true as const, data: true };
    }
    return { ok: false as const, error: res.error, status: 400, failure: res.failure };
  });
  if (result == null) {
    return false;
  }

  // 対象 session の context index entry を削除する
  const targetSession = authState.sessions?.find((s) => s.sessionId === sessionId);
  if (targetSession != null) {
    const index = readContextIndex() ?? createEmptyContextIndex();
    removeContextEntry(index, targetSession.authContextId);
    writeContextIndex(index);
  }

  if (sessionId === authState.activeSessionId) {
    removeActiveSession(authState);
  } else {
    authState.sessions = authState.sessions?.filter((s) => s.sessionId !== sessionId) ?? [];
  }
  return true;
}

/** 現在のセッション以外をすべてリモートで無効化し、ローカル state と context index を更新する。 */
async function executeRevokeOtherDevices(authState: AuthSessionState): Promise<boolean> {
  const result = await withRefreshRetry(authState, async (headers) => {
    const res = await revokeOtherDevices(headers);
    if (res.ok) {
      return { ok: true as const, data: true };
    }
    return { ok: false as const, error: res.error, status: 400, failure: res.failure };
  });
  if (result == null) {
    return false;
  }

  // 削除された session の context index entry を削除する
  const active = authState.session;
  if (active != null) {
    const index = readContextIndex() ?? createEmptyContextIndex();
    // active 以外の entry をすべて削除
    index.entries = index.entries.filter((e) => e.authContextId === active.authContextId);
    index.activeAuthContextId = active.authContextId;
    writeContextIndex(index);
  }

  authState.sessions = active != null ? [active] : [];
  return true;
}

/** context index から session bootstrap を試行し、成功 entry を復元する。 */
async function bootstrapSessionsFromContextIndex(): Promise<void> {
  try {
    const index = readContextIndex();
    if (index == null || index.entries.length === 0) {
      return;
    }

    const restoredSessions: AuthSessionSummary[] = [];
    let restoredActiveSession: AuthSessionSummary | null = null;

    for (const entry of index.entries) {
      try {
        const response = await refreshToken(
          entry.authContextId,
          undefined,
          COOKIE_AUTH_REQUEST_INIT
        );
        if (response.status === 200 && 'accessToken' in response.data) {
          const { accessToken, account, sessionId, expiresAt } = response.data;
          const accountId = account.accountId;
          const claims = decodeAccessToken(accessToken);
          if (claims == null || claims.accountId !== accountId || claims.sessionId !== sessionId) {
            continue;
          }
          const restoredSession: AuthSessionSummary = {
            requestId: response.data.requestId,
            authContextId: entry.authContextId,
            accountId,
            passkeyCredentialId: account.passkeyCredentialId,
            sessionId,
            accessToken,
            expiresAt,
          };
          restoredSessions.push(restoredSession);
          if (index.activeAuthContextId === entry.authContextId) {
            restoredActiveSession = restoredSession;
          }
        }
      } catch {
        // refresh failure: 該当 entry は authenticated state として採用しない
      }
    }

    if (restoredSessions.length > 0) {
      // 同一 accountId 重複を除去（後方 entry を優先）
      const dedupedSessions: AuthSessionSummary[] = [];
      const seenAccountIds = new SvelteSet<string>();
      for (let i = restoredSessions.length - 1; i >= 0; i--) {
        const s = restoredSessions[i];
        if (!seenAccountIds.has(s.accountId)) {
          seenAccountIds.add(s.accountId);
          dedupedSessions.unshift(s);
        }
      }

      state.sessions = dedupedSessions;
      // active が dedup 後も残っていれば維持、なければ先頭へ切替
      const active =
        restoredActiveSession != null &&
        dedupedSessions.some((s) => s.sessionId === restoredActiveSession.sessionId)
          ? restoredActiveSession
          : dedupedSessions[0];
      state.session = active;
      state.activeSessionId = active.sessionId;
      state.phase = 'authenticated';
      state.routeIntent = '/login';
      state.lastFailure = null;
      state.lastError = null;

      // bootstrap 後に index を再構築（失敗した entry と重複 entry を除去）
      const newIndex = createEmptyContextIndex();
      for (const s of dedupedSessions) {
        upsertContextEntry(
          newIndex,
          toContextIndexEntry(s, s.expiresAt),
          s.sessionId === active.sessionId
        );
      }
      writeContextIndex(newIndex);
    } else {
      // 復元できなかった場合は index をクリアする
      clearContextIndex();
    }
  } finally {
    // bootstrap 完了を記録し、guard が redirect 判断を再評価できるようにする
    bootstrapPhase.value = 'done';
  }
}

/** bootstrap が完了しているかどうかのフラグ。 */
let hasBootstrapped = false;

/** context index bootstrap の進行状態。guard が `/login` redirect の保留判断に使う。 */
const bootstrapPhase = $state<{ value: BootstrapPhase }>({ value: 'pending' });

/** in-memory bearer session と route 分岐を共有する domain composable。 */
function useAuthSession(): { data: AuthSessionData; actions: AuthSessionActions } {
  if (!hasBootstrapped) {
    hasBootstrapped = true;
    // non-blocking で context index から session を復元する
    void bootstrapSessionsFromContextIndex();
  }
  const actions: AuthSessionActions = {
    acceptSession: (session, cacheControl) => {
      const nextSession = {
        requestId: session.requestId,
        authContextId: session.authContextId,
        accountId: session.accountId,
        passkeyCredentialId: session.passkeyCredentialId,
        sessionId: session.sessionId,
        accessToken: session.accessToken,
        expiresAt: session.expiresAt,
      };

      if (!hasUlidAuthSessionShape(nextSession)) {
        applyInternalError(state, '認証セッションの識別子形式が不正です。', cacheControl);
        return;
      }

      addAuthenticatedSession(state, nextSession, cacheControl);

      // login/refresh 成功時は context index を更新する
      const index = readContextIndex() ?? createEmptyContextIndex();
      upsertContextEntry(index, toContextIndexEntry(nextSession, nextSession.expiresAt), true);
      writeContextIndex(index);
    },
    createAuthorizationHeaders: () => createAuthorizationHeaders(state),
    handleFailure: (classification, message) => {
      if (classification === SESSION_EXPIRED_ERROR) {
        return applyExpiredSession(state);
      }

      if (classification === ACCOUNT_SUSPENDED_ERROR) {
        return applyAccountSuspended(state);
      }

      if (classification === 'unauthenticated') {
        return applyMissingSession(state);
      }

      applyInternalError(state, message ?? '認証状態を確認できませんでした。');
      return '/login';
    },
    handleMissingSession: () => applyMissingSession(state),
    clearInMemorySession: () => {
      clearContextIndex();
      return clearAuthSession(state);
    },
    logoutCurrentSession: () => executeLogoutCurrentSession(state),
    refreshActiveSession: () => {
      const active = state.session;
      if (active == null) {
        return Promise.resolve(applyExpiredSession(state));
      }
      return executeRefreshActiveSession(state, active.sessionId);
    },
    switchSession: (sessionId) => {
      const switched = switchActiveSession(state, sessionId);
      if (switched) {
        const active = state.session;
        if (active != null) {
          const index = readContextIndex() ?? createEmptyContextIndex();
          index.activeAuthContextId = active.authContextId;
          writeContextIndex(index);
        }
      }
      return switched;
    },
    listDevices: () => executeListDevices(state),
    revokeDevice: (sessionId) => executeRevokeDevice(state, sessionId),
    revokeOtherDevices: () => executeRevokeOtherDevices(state),
  };

  return {
    data: {
      state,
      bootstrapPhase,
    },
    actions,
  };
}

export type { AuthSessionActions, AuthSessionData, BootstrapPhase, DeviceSession };
export { useAuthSession };
