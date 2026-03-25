import { authApi } from '@www-template-frontend/api';

import {
  applyAuthenticatedSession,
  applyExpiredSession,
  applyInternalError,
  applyMissingSession,
  clearAuthSession,
  createAuthorizationHeaders,
  createAuthSessionInitialState,
  hasUlidAuthSessionShape,
} from '../../auth/authSessionState';

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

const state = $state<AuthSessionState>(createAuthSessionInitialState());

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
    },
    createAuthorizationHeaders: () => createAuthorizationHeaders(state),
    handleFailure: (classification, message) => {
      if (classification === 'session-expired') {
        return applyExpiredSession(state);
      }

      if (classification === 'unauthenticated') {
        return applyMissingSession(state);
      }

      applyInternalError(state, message ?? '認証状態を確認できませんでした。');
      return '/app/login';
    },
    handleMissingSession: () => applyMissingSession(state),
    clearInMemorySession: () => clearAuthSession(state),
    logoutCurrentSession: async () => {
      state.phase = 'logging-out';

      if (state.session === null) {
        return clearAuthSession(state);
      }

      try {
        const response = await authApi.logout({
          headers: createAuthorizationHeaders(state),
        });

        if (response.status === 200) {
          return clearAuthSession(state, response.headers.get('cache-control'));
        }

        if (response.data.error === 'session-expired') {
          clearAuthSession(state, response.headers.get('cache-control'));
          return '/app/login';
        }

        clearAuthSession(state, response.headers.get('cache-control'));
        return '/app/login';
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
