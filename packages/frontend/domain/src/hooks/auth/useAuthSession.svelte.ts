import { authApi } from '@www-template/api';

import {
  applyAuthenticatedSession,
  applyExpiredSession,
  applyInternalError,
  applyMissingSession,
  clearAuthSession,
  createAuthorizationHeaders,
  createAuthSessionInitialState,
  hasUlidAuthSessionShape,
} from '../../auth';

import type {
  AuthFailureState,
  AuthRouteIntent,
  AuthSessionState,
  AuthSessionSummary,
} from 'types';

interface AuthSessionData {
  state: AuthSessionState;
}

interface AuthSessionActions {
  acceptSession: (session: AuthSessionSummary, cacheControl: string | null) => void;
  createAuthorizationHeaders: () => Record<string, string>;
  handleFailure: (classification: AuthFailureState, message?: string) => AuthRouteIntent;
  handleMissingSession: () => AuthRouteIntent;
  clearInMemorySession: () => AuthRouteIntent;
  logoutCurrentSession: () => Promise<AuthRouteIntent>;
}

const SESSION_STORAGE_KEY = 'www-template:auth-session';

/** sessionStorage からセッションを復元する。復元できなければ初期状態を返す。 */
function restoreStateFromStorage(): AuthSessionState {
  const initial = createAuthSessionInitialState();

  if (typeof sessionStorage === 'undefined') {
    return initial;
  }

  try {
    const raw = sessionStorage.getItem(SESSION_STORAGE_KEY);

    if (raw === null) {
      return initial;
    }

    const parsed: unknown = JSON.parse(raw);

    if (
      typeof parsed !== 'object' ||
      parsed === null ||
      (parsed as Record<string, unknown>).phase !== 'authenticated'
    ) {
      return initial;
    }

    const session = (parsed as Record<string, unknown>).session;

    if (typeof session !== 'object' || session === null) {
      return initial;
    }

    const s = session as Record<string, unknown>;

    if (
      typeof s.requestId !== 'string' ||
      typeof s.accountId !== 'string' ||
      typeof s.passkeyCredentialId !== 'string' ||
      typeof s.sessionId !== 'string' ||
      typeof s.sessionToken !== 'string' ||
      typeof s.expiresAt !== 'string'
    ) {
      return initial;
    }

    const summary: AuthSessionSummary = {
      requestId: s.requestId,
      accountId: s.accountId,
      passkeyCredentialId: s.passkeyCredentialId,
      sessionId: s.sessionId,
      sessionToken: s.sessionToken,
      expiresAt: s.expiresAt,
    };

    if (!hasUlidAuthSessionShape(summary)) {
      return initial;
    }

    return {
      ...initial,
      phase: 'authenticated',
      session: summary,
    };
  } catch {
    return initial;
  }
}

/** セッション状態を sessionStorage に保存する。 */
function persistStateToStorage(state: AuthSessionState): void {
  if (typeof sessionStorage === 'undefined') {
    return;
  }

  if (state.phase === 'authenticated' && state.session !== null) {
    sessionStorage.setItem(
      SESSION_STORAGE_KEY,
      JSON.stringify({ phase: state.phase, session: state.session })
    );
  } else {
    sessionStorage.removeItem(SESSION_STORAGE_KEY);
  }
}

const state = $state<AuthSessionState>(restoreStateFromStorage());

/** in-memory bearer session と route 分岐を共有する domain composable。 */
function useAuthSession(): { data: AuthSessionData; actions: AuthSessionActions } {
  const actions: AuthSessionActions = {
    acceptSession: (session, cacheControl) => {
      const nextSession = {
        requestId: session.requestId,
        accountId: session.accountId,
        passkeyCredentialId: session.passkeyCredentialId,
        sessionId: session.sessionId,
        sessionToken: session.sessionToken,
        expiresAt: session.expiresAt,
      };

      if (!hasUlidAuthSessionShape(nextSession)) {
        applyInternalError(state, '認証セッションの識別子形式が不正です。', cacheControl);
        return;
      }

      applyAuthenticatedSession(state, nextSession, cacheControl);
      persistStateToStorage(state);
    },
    createAuthorizationHeaders: () => createAuthorizationHeaders(state),
    handleFailure: (classification, message) => {
      if (classification === 'session-expired') {
        const intent = applyExpiredSession(state);
        persistStateToStorage(state);
        return intent;
      }

      if (classification === 'unauthenticated') {
        const intent = applyMissingSession(state);
        persistStateToStorage(state);
        return intent;
      }

      applyInternalError(state, message ?? '認証状態を確認できませんでした。');
      return '/login';
    },
    handleMissingSession: () => {
      const intent = applyMissingSession(state);
      persistStateToStorage(state);
      return intent;
    },
    clearInMemorySession: () => {
      const intent = clearAuthSession(state);
      persistStateToStorage(state);
      return intent;
    },
    logoutCurrentSession: async () => {
      state.phase = 'logging-out';

      if (state.session === null) {
        const intent = clearAuthSession(state);
        persistStateToStorage(state);
        return intent;
      }

      try {
        const response = await authApi.logout({
          headers: createAuthorizationHeaders(state),
        });

        if (response.status === 200) {
          const intent = clearAuthSession(state, response.headers.get('cache-control'));
          persistStateToStorage(state);
          return intent;
        }

        if (response.data.error === 'session-expired') {
          clearAuthSession(state, response.headers.get('cache-control'));
          persistStateToStorage(state);
          return '/login';
        }

        clearAuthSession(state, response.headers.get('cache-control'));
        persistStateToStorage(state);
        return '/login';
      } catch (error: unknown) {
        applyInternalError(
          state,
          error instanceof Error ? error.message : 'ログアウトに失敗しました。'
        );
        return clearAuthSession(state);
      }
    },
  };

  return {
    data: {
      state,
    },
    actions,
  };
}

export type { AuthSessionActions, AuthSessionData };
export { useAuthSession };
