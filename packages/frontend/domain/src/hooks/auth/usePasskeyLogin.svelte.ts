import { authApi } from '@www-template-frontend/api';

import { createPasskeyLoginInitialState, toPasskeyErrorMessage } from '../../auth/passkeyState';

import { useAuthSession } from './useAuthSession.svelte';

import type { AuthRouteIntent, PasskeyLoginState } from 'types';

interface PasskeyLoginData {
  state: PasskeyLoginState;
}

interface PasskeyLoginActions {
  setIdentifier: (identifier: string) => void;
  signInWithPasskey: (credential?: string) => Promise<AuthRouteIntent | '/app' | null>;
}

/** passkey start / finish と shared session 更新を扱う domain composable。 */
function usePasskeyLogin(): { data: PasskeyLoginData; actions: PasskeyLoginActions } {
  const state = $state<PasskeyLoginState>(createPasskeyLoginInitialState());
  const authSession = useAuthSession();

  const actions: PasskeyLoginActions = {
    setIdentifier: (identifier) => {
      state.identifier = identifier;
    },
    signInWithPasskey: async (credential) => {
      state.isSubmitting = true;
      state.error = null;

      try {
        const startResponse = await authApi.startPasskeyAuthentication(state.identifier.trim());

        if (startResponse.status !== 200) {
          return authSession.actions.handleFailure(
            startResponse.data.error,
            '認証開始に失敗しました。'
          );
        }

        state.lastChallengeRequestId = startResponse.data.requestId;
        state.lastCacheControl = startResponse.headers.get('cache-control');

        const finishResponse = await authApi.finishPasskeyAuthentication(
          credential ??
            JSON.stringify({
              requestId: startResponse.data.requestId,
              challenge: startResponse.data.challenge,
            })
        );

        state.lastCacheControl = finishResponse.headers.get('cache-control');

        if (finishResponse.status === 200) {
          authSession.actions.acceptSession(
            authApi.toSessionSummary(finishResponse.data),
            finishResponse.headers.get('cache-control')
          );
          state.lastSession = authSession.data.state.session;
          state.isSubmitting = false;
          return '/app';
        }

        if (finishResponse.status === 503) {
          return authSession.actions.handleFailure(
            finishResponse.data.error,
            '認証基盤を利用できませんでした。'
          );
        }

        state.error = finishResponse.data.error;
        state.isSubmitting = false;
        return null;
      } catch (error: unknown) {
        state.error = toPasskeyErrorMessage(error);
        state.isSubmitting = false;
        return null;
      } finally {
        state.isSubmitting = false;
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

export type { PasskeyLoginActions, PasskeyLoginData };
export { usePasskeyLogin };
