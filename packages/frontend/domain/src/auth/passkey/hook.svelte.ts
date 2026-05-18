import { authApi } from '@www-template/api';

import { getWebAuthnAssertion, normalizeWebAuthnError } from '../webauthn';

import { createPasskeyLoginInitialState, toPasskeyErrorMessage } from './state';

import { useAuthSession } from '../session/hook.svelte';

import type { AuthRouteIntent, PasskeyLoginState } from '../types';

interface PasskeyLoginData {
  state: PasskeyLoginState;
}

interface PasskeyLoginActions {
  setIdentifier: (identifier: string) => void;
  signInWithPasskey: () => Promise<AuthRouteIntent | null>;
}

/** passkey start / navigator.credentials.get / finish と shared session 更新を扱う domain composable。 */
function usePasskeyLogin(): { data: PasskeyLoginData; actions: PasskeyLoginActions } {
  const state = $state<PasskeyLoginState>(createPasskeyLoginInitialState());
  const authSession = useAuthSession();

  const actions: PasskeyLoginActions = {
    setIdentifier: (identifier) => {
      state.identifier = identifier;
    },
    signInWithPasskey: async () => {
      state.isSubmitting = true;
      state.error = null;

      try {
        // 認証開始は public surface のため、account existence や suspended 状態を表示しない。
        const startResponse = await authApi.startPasskeyAuthentication(state.identifier.trim());

        if (startResponse.status !== 200) {
          return authSession.actions.handleFailure(
            startResponse.data.error,
            '認証開始に失敗しました。'
          );
        }

        state.lastChallengeRequestId = startResponse.data.requestId;
        state.lastCacheControl = startResponse.headers.get('cache-control');

        // ブラウザの WebAuthn API だけを呼び、端末起因の失敗は汎用文言へ正規化する。
        let credential;
        try {
          credential = await getWebAuthnAssertion(startResponse.data);
        } catch (webAuthnError: unknown) {
          state.error = normalizeWebAuthnError(webAuthnError);
          state.isSubmitting = false;
          return null;
        }

        // valid credential 確認後の finish 応答だけを認証済み失敗分類として扱う。
        const finishResponse = await authApi.finishPasskeyAuthentication(credential);

        state.lastCacheControl = finishResponse.headers.get('cache-control');

        if (finishResponse.status === 200) {
          authSession.actions.acceptSession(
            authApi.toSessionSummary(finishResponse.data),
            finishResponse.headers.get('cache-control')
          );
          state.lastSession = authSession.data.state.session;
          state.isSubmitting = false;
          return null;
        }

        if (finishResponse.status === 503) {
          return authSession.actions.handleFailure(
            finishResponse.data.error,
            '認証基盤を利用できませんでした。'
          );
        }

        if (finishResponse.status === 403 && finishResponse.data.error === 'account-suspended') {
          state.lastSession = null;
          return authSession.actions.handleFailure(finishResponse.data.error);
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
