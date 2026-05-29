import { SvelteMap } from 'svelte/reactivity';

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

/**
 * Product frontend auth が Cookie を送信する際の共通 request init。
 *
 * same-origin の Product API だけに Cookie を添付し、cross-origin へ refresh Cookie が送られる
 * 余地を作らない。logout も同じ Cookie 境界で revoke を依頼する。
 */
const COOKIE_AUTH_REQUEST_INIT = { credentials: 'same-origin' } as const satisfies RequestInit;

/**
 * refresh 応答の accessToken が、更新対象 session と同一 account/session を指すか検証する。
 *
 * Cookie refresh は JavaScript から refreshToken を読めないため、サーバー応答の bearer token が
 * 期待した対象 session に属することを frontend domain でも確認し、不一致時は fail-close する。
 *
 * @param claims - refresh 応答 accessToken から decode した claim
 * @param session - 更新対象の in-memory session
 * @returns accountId と sessionId が一致する場合だけ `true`
 */
function isAccessTokenForSession(claims: AccessTokenClaims, session: AuthSessionSummary): boolean {
  return claims.accountId === session.accountId && claims.sessionId === session.sessionId;
}

interface AuthSessionData {
  state: AuthSessionState;
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

/**
 * bearer token を含む認証セッションは sessionStorage などの persistent client storage に
 * 保存しない。ブラウザタブを閉じた時点でセッションは破棄され、次回アクセス時は
 * 再認証を要求する。これにより token の漏えいリスクを最小化する。
 */
const state = $state<AuthSessionState>(createAuthSessionInitialState());

/** 並行リフレッシュを防止するため、sessionId 単位で実行中の refresh Promise を保持する。 */
const refreshInFlight = new SvelteMap<string, Promise<AuthRouteIntent | null>>();

/**
 * アクティブセッションのトークンが期限切れ間近の場合に自動リフレッシュを実行し、
 * 最新の Authorization ヘッダーを返す。
 * 同一セッションに対する並行リクエストは単一の refresh に集約される。
 * refresh 中にアクティブセッションが切り替わっても、対象セッションのトークンのみが更新される。
 *
 * @param authState - 認証セッション state
 * @returns 最新の Authorization ヘッダー（未認証時は空オブジェクト）
 */
async function ensureFreshAuthorizationHeaders(
  authState: AuthSessionState
): Promise<Record<string, string>> {
  const active = authState.session;
  if (active == null) {
    return {};
  }

  const claims = decodeAccessToken(active.accessToken);
  if (claims != null && isRefreshNeeded(claims.exp, Date.now())) {
    const sessionId = active.sessionId;
    let inflight = refreshInFlight.get(sessionId);
    if (inflight == null) {
      inflight = executeRefreshActiveSession(authState, sessionId).finally(() => {
        refreshInFlight.delete(sessionId);
      });
      refreshInFlight.set(sessionId, inflight);
    }
    await inflight;
  }

  return createAuthorizationHeaders(authState);
}

/**
 * ログアウト前にトークンをリフレッシュする軽量ヘルパー。
 * 成功時のみ対象セッションを更新し、いかなる失敗（ネットワーク含む）でも
 * セッションを失効させたり遷移させたりしない。
 *
 * @param authState - 認証セッション state
 */
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
    // Cookie-only refresh は request body を持たないため、生成 SDK の body 引数には
    // `undefined` を渡し、HttpOnly Cookie だけを same-origin 境界で送信する。
    const response = await refreshToken(undefined, COOKIE_AUTH_REQUEST_INIT);

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

/**
 * リフレッシュ失敗時に対象セッションの状態を整える。
 * 非アクティブなセッションが失敗した場合は配列からのみ除去し、
 * アクティブなセッションが失敗した場合は `applyExpiredSession` を呼んで
 * 失効状態に遷移させる。
 *
 * @param authState - 認証セッション state
 * @param targetSessionId - 失敗したセッションの ID
 * @returns 非アクティブセッション除去時は `null`、アクティブセッション失効時は遷移先 route intent
 */
function handleRefreshFailureForTarget(
  authState: AuthSessionState,
  targetSessionId: string
): AuthRouteIntent | null {
  if (authState.activeSessionId !== targetSessionId) {
    authState.sessions = authState.sessions?.filter((s) => s.sessionId !== targetSessionId) ?? [];
    return null;
  }
  return applyExpiredSession(authState);
}

/**
 * suspended account 応答を対象セッション単位で処理する。
 *
 * @param authState - 認証セッション state
 * @param targetSessionId - 停止対象と判定されたセッション ID
 * @param cacheControl - API 応答の cache-control 値
 * @returns アクティブセッションなら案内 route、非アクティブなら null
 */
function handleAccountSuspendedForTarget(
  authState: AuthSessionState,
  targetSessionId: string,
  cacheControl: string | null = null
): AuthRouteIntent | null {
  if (authState.activeSessionId !== targetSessionId) {
    authState.sessions = authState.sessions?.filter((s) => s.sessionId !== targetSessionId) ?? [];
    return null;
  }

  return applyAccountSuspended(authState, cacheControl, targetSessionId);
}

/**
 * 指定されたセッションを HttpOnly Cookie でリフレッシュし、新しいアクセストークンを取得する。
 * 成功時は対象セッションのみを更新し、現在アクティブなセッションが同じ場合に限り
 * `state.session` を差し替える。
 * いかなる失敗（ネットワーク含む）でも対象セッションは失効扱いとし、
 * `/session-expired` へ遷移する。
 *
 * @param authState - 認証セッション state
 * @param targetSessionId - リフレッシュ対象のセッション ID
 * @returns 成功時 `null`、失敗時は遷移先 route intent
 */
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
    // Cookie-only refresh は request body を持たないため、生成 SDK の body 引数には
    // `undefined` を渡し、HttpOnly Cookie だけを same-origin 境界で送信する。
    const response = await refreshToken(undefined, COOKIE_AUTH_REQUEST_INIT);

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

/** ログイン中の全セッション（デバイス）一覧を取得する。 */
async function executeListDevices(authState: AuthSessionState): Promise<DeviceSession[] | null> {
  const headers = await ensureFreshAuthorizationHeaders(authState);
  if (headers.Authorization == null) {
    return null;
  }
  const result = await fetchDevices(headers);
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

/** 指定されたセッションをリモートで無効化し、ローカル state を更新する。 */
async function executeRevokeDevice(
  authState: AuthSessionState,
  sessionId: string
): Promise<boolean> {
  const headers = await ensureFreshAuthorizationHeaders(authState);
  if (headers.Authorization == null) {
    return false;
  }
  const result = await revokeDevice(sessionId, headers);
  if (!result.ok) {
    if (result.failure === SESSION_EXPIRED_ERROR) {
      applyExpiredSession(authState);
      return false;
    }
    if (result.failure === ACCOUNT_SUSPENDED_ERROR) {
      applyAccountSuspended(authState);
      return false;
    }
    if (result.failure === 'unauthenticated') {
      applyMissingSession(authState);
      return false;
    }
    return false;
  }
  if (sessionId === authState.activeSessionId) {
    removeActiveSession(authState);
  } else {
    authState.sessions = authState.sessions?.filter((s) => s.sessionId !== sessionId) ?? [];
  }
  return true;
}

/** 現在のセッション以外をすべてリモートで無効化し、ローカル state を更新する。 */
async function executeRevokeOtherDevices(authState: AuthSessionState): Promise<boolean> {
  const headers = await ensureFreshAuthorizationHeaders(authState);
  if (headers.Authorization == null) {
    return false;
  }
  const result = await revokeOtherDevices(headers);
  if (!result.ok) {
    if (result.failure === SESSION_EXPIRED_ERROR) {
      applyExpiredSession(authState);
      return false;
    }
    if (result.failure === ACCOUNT_SUSPENDED_ERROR) {
      applyAccountSuspended(authState);
      return false;
    }
    if (result.failure === 'unauthenticated') {
      applyMissingSession(authState);
      return false;
    }
    return false;
  }
  const active = authState.session;
  authState.sessions = active != null ? [active] : [];
  return true;
}

/** in-memory bearer session と route 分岐を共有する domain composable。 */
function useAuthSession(): { data: AuthSessionData; actions: AuthSessionActions } {
  const actions: AuthSessionActions = {
    acceptSession: (session, cacheControl) => {
      const nextSession = {
        requestId: session.requestId,
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
    clearInMemorySession: () => clearAuthSession(state),
    logoutCurrentSession: () => executeLogoutCurrentSession(state),
    refreshActiveSession: () => {
      const active = state.session;
      if (active == null) {
        return Promise.resolve(applyExpiredSession(state));
      }
      return executeRefreshActiveSession(state, active.sessionId);
    },
    switchSession: (sessionId) => switchActiveSession(state, sessionId),
    listDevices: () => executeListDevices(state),
    revokeDevice: (sessionId) => executeRevokeDevice(state, sessionId),
    revokeOtherDevices: () => executeRevokeOtherDevices(state),
  };

  return {
    data: {
      state,
    },
    actions,
  };
}

export type { AuthSessionActions, AuthSessionData, DeviceSession };
export { useAuthSession };
