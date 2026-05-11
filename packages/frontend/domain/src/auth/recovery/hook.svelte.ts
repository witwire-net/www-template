import { authApi } from '@www-template/api';

import {
  applyInvalidRecoveryToken,
  applyRecoveryAccepted,
  applyRecoveryReady,
  clearRecoveryState,
  createRecoveryFlowInitialState,
} from './state';

import { createWebAuthnAttestation, normalizeWebAuthnError } from '../webauthn';
import { toRecoveryErrorMessage } from '../passkey/state';

import { useAuthSession } from '../session/hook.svelte';

import type { AuthFailureState, RecoveryFlowState } from '../types';

const CACHE_CONTROL_HEADER = 'cache-control';

interface RecoveryFlowData {
  state: RecoveryFlowState;
}

interface RecoveryFlowActions {
  setEmail: (email: string) => void;
  submitRecoveryRequest: () => Promise<'/login/recovery/sent' | null>;
  consumeToken: (
    token: string
  ) => Promise<{
    path: '/login/recovery/register' | '/login/recovery';
    kind?: 'recovery' | 'device-link';
  } | null>;
  registerRecoveryPasskey: () => Promise<null>;
  reset: () => void;
}

const createSubmitRecoveryRequestAction =
  (state: RecoveryFlowState, authSession: ReturnType<typeof useAuthSession>) =>
  async (): Promise<'/login/recovery/sent' | null> => {
    state.phase = 'submitting';
    state.error = null;

    try {
      const response = await authApi.requestPasskeyRecovery(state.email.trim());
      state.lastCacheControl = response.headers.get(CACHE_CONTROL_HEADER);

      if (response.status === 202) {
        applyRecoveryAccepted(
          state,
          response.data.requestId,
          response.headers.get(CACHE_CONTROL_HEADER)
        );
        return '/login/recovery/sent';
      }

      if (response.status === 401 || response.status === 503) {
        authSession.actions.handleFailure(
          response.data.error,
          '復旧依頼を受け付けられませんでした。'
        );
      }

      state.phase = 'idle';
      return null;
    } catch (error: unknown) {
      state.phase = 'idle';
      state.error = toRecoveryErrorMessage(error);
      return null;
    }
  };

const createConsumeTokenAction =
  (state: RecoveryFlowState, authSession: ReturnType<typeof useAuthSession>) =>
  async (
    token: string
  ): Promise<{
    path: '/login/recovery/register' | '/login/recovery';
    kind?: 'recovery' | 'device-link';
  } | null> => {
    state.phase = 'consuming';
    state.error = null;

    try {
      const response = await authApi.consumeRecoveryToken(token);
      state.lastCacheControl = response.headers.get(CACHE_CONTROL_HEADER);

      if (response.status === 200) {
        const kind = response.data.kind as 'recovery' | 'device-link';
        applyRecoveryReady(
          state,
          {
            requestId: response.data.requestId,
            recoveryTokenId: response.data.recoveryTokenId,
            recoverySessionId: response.data.recoverySessionId,
            recoverySession: response.data.recovery_session,
            expiresAt: response.data.expiresAt,
            kind,
          },
          response.headers.get(CACHE_CONTROL_HEADER)
        );
        return { path: '/login/recovery/register', kind };
      }

      if (response.status === 400) {
        applyInvalidRecoveryToken(
          state,
          '復旧リンクを確認できませんでした。再度復旧をお試しください。',
          response.headers.get(CACHE_CONTROL_HEADER)
        );
        return { path: '/login/recovery' };
      }

      if (response.status === 503) {
        authSession.actions.handleFailure(response.data.error, '復旧状態を確認できませんでした。');
      }

      state.phase = 'idle';
      return null;
    } catch (error: unknown) {
      applyInvalidRecoveryToken(state, toRecoveryErrorMessage(error), state.lastCacheControl);
      return { path: '/login/recovery' };
    }
  };

const createRegisterRecoveryPasskeyAction =
  (state: RecoveryFlowState, authSession: ReturnType<typeof useAuthSession>) =>
  async (): Promise<null> => {
    if (state.recoverySession === null) {
      applyInvalidRecoveryToken(
        state,
        '復旧状態が見つかりません。もう一度やり直してください。',
        null
      );
      return null;
    }

    state.phase = 'registering';
    state.error = null;

    try {
      // Step 1: Start — get WebAuthn creation options from server
      const startResponse = await authApi.startRecoveryPasskeyRegistration(state.recoverySession);
      if (startResponse.status === 400) {
        applyInvalidRecoveryToken(
          state,
          '復旧リンクの有効期限が切れた可能性があります。再度復旧をお試しください。',
          startResponse.headers.get(CACHE_CONTROL_HEADER)
        );
        return null;
      }
      if (startResponse.status !== 200) {
        // status === 400 は上で処理済みのため、ここは 503 (AuthFailure) のみ
        authSession.actions.handleFailure(
          startResponse.data.error as AuthFailureState,
          'パスキー再登録を開始できませんでした。'
        );
        state.phase = 'idle';
        return null;
      }

      // Step 2: Call browser WebAuthn API — normalize browser/device errors only
      let credential;
      try {
        credential = await createWebAuthnAttestation(startResponse.data);
      } catch (webAuthnError: unknown) {
        state.phase = 'ready';
        state.error = normalizeWebAuthnError(webAuthnError);
        return null;
      }

      // Step 3: Finish — send attestation to server
      const response = await authApi.registerRecoveryPasskey(state.recoverySession, credential);

      state.lastCacheControl = response.headers.get(CACHE_CONTROL_HEADER);

      if (response.status === 200) {
        authSession.actions.acceptSession(
          authApi.toSessionSummary(response.data),
          response.headers.get(CACHE_CONTROL_HEADER)
        );
        clearRecoveryState(state);
        return null;
      }

      if (response.status === 400) {
        applyInvalidRecoveryToken(
          state,
          '復旧リンクの有効期限が切れた可能性があります。再度復旧をお試しください。',
          response.headers.get(CACHE_CONTROL_HEADER)
        );
        return null;
      }

      authSession.actions.handleFailure(
        response.data.error,
        'パスキー再登録を完了できませんでした。'
      );
      state.phase = 'idle';
      return null;
    } catch (error: unknown) {
      state.phase = 'ready';
      state.error = toRecoveryErrorMessage(error);
      return null;
    }
  };

/**
 * Module-level singleton state.
 * recovery flow はフルナビゲーション（consume → register）で route を跨ぐため、
 * SvelteKit client-side routing で同一 module instance の state を共有する。
 */
const state = $state<RecoveryFlowState>(createRecoveryFlowInitialState());

/** recovery request / consume / register を集約する domain composable。 */
function useRecoveryFlow(): { data: RecoveryFlowData; actions: RecoveryFlowActions } {
  const authSession = useAuthSession();

  const actions: RecoveryFlowActions = {
    setEmail: (email) => {
      state.email = email;
    },
    submitRecoveryRequest: createSubmitRecoveryRequestAction(state, authSession),
    consumeToken: createConsumeTokenAction(state, authSession),
    registerRecoveryPasskey: createRegisterRecoveryPasskeyAction(state, authSession),
    reset: () => {
      clearRecoveryState(state);
    },
  };

  return {
    data: {
      state,
    },
    actions,
  };
}

export type { RecoveryFlowActions, RecoveryFlowData };
export { useRecoveryFlow };
