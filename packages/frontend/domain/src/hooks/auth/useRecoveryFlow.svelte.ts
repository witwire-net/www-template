import { authApi } from '@www-template-frontend/api';

import {
  applyInvalidRecoveryToken,
  applyRecoveryAccepted,
  applyRecoveryReady,
  clearRecoveryState,
  createRecoveryFlowInitialState,
} from '../../auth/recoveryState';
import { toRecoveryErrorMessage } from '../../auth/passkeyState';

import { useAuthSession } from './useAuthSession.svelte';

import type { RecoveryFlowState } from 'types';

const CACHE_CONTROL_HEADER = 'cache-control';

interface RecoveryFlowData {
  state: RecoveryFlowState;
}

/** consume → register 間で保存する snapshot の型。 */
interface RecoveryReadySnapshot {
  requestId: string;
  recoveryTokenId: string;
  recoverySessionId: string;
  recoverySession: string;
  expiresAt: string;
}

interface RecoveryFlowActions {
  setEmail: (email: string) => void;
  submitRecoveryRequest: () => Promise<'/login/recovery/sent' | null>;
  consumeToken: (token: string) => Promise<'/login/recovery/register' | '/login/recovery' | null>;
  registerRecoveryPasskey: (credential?: string) => Promise<null>;
  /** ready state の snapshot を取得する。フルリロード遷移前に app 層が永続化に使う。 */
  getReadySnapshot: () => RecoveryReadySnapshot | null;
  /** app 層が永続化から復元した snapshot で ready state を再構成する。 */
  restoreReadyState: (snapshot: RecoveryReadySnapshot) => void;
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
  async (token: string): Promise<'/login/recovery/register' | '/login/recovery' | null> => {
    state.phase = 'consuming';
    state.error = null;

    try {
      const response = await authApi.consumeRecoveryToken(token);
      state.lastCacheControl = response.headers.get(CACHE_CONTROL_HEADER);

      if (response.status === 200) {
        applyRecoveryReady(
          state,
          {
            requestId: response.data.requestId,
            recoveryTokenId: response.data.recoveryTokenId,
            recoverySessionId: response.data.recoverySessionId,
            recoverySession: response.data.recovery_session,
            expiresAt: response.data.expiresAt,
          },
          response.headers.get(CACHE_CONTROL_HEADER)
        );
        return '/login/recovery/register';
      }

      if (response.status === 400) {
        applyInvalidRecoveryToken(
          state,
          '復旧リンクを確認できませんでした。再度復旧をお試しください。',
          response.headers.get(CACHE_CONTROL_HEADER)
        );
        return '/login/recovery';
      }

      if (response.status === 503) {
        authSession.actions.handleFailure(response.data.error, '復旧状態を確認できませんでした。');
      }

      state.phase = 'idle';
      return null;
    } catch (error: unknown) {
      applyInvalidRecoveryToken(state, toRecoveryErrorMessage(error), state.lastCacheControl);
      return '/login/recovery';
    }
  };

const createRegisterRecoveryPasskeyAction =
  (state: RecoveryFlowState, authSession: ReturnType<typeof useAuthSession>) =>
  async (credential?: string): Promise<null> => {
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
      const response = await authApi.registerRecoveryPasskey(
        state.recoverySession,
        credential ?? JSON.stringify({ recoverySessionId: state.recoverySessionId, recovery: true })
      );

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
    getReadySnapshot: () => {
      if (
        state.phase !== 'ready' ||
        state.requestId === null ||
        state.recoveryTokenId === null ||
        state.recoverySessionId === null ||
        state.recoverySession === null ||
        state.expiresAt === null
      ) {
        return null;
      }
      return {
        requestId: state.requestId,
        recoveryTokenId: state.recoveryTokenId,
        recoverySessionId: state.recoverySessionId,
        recoverySession: state.recoverySession,
        expiresAt: state.expiresAt,
      };
    },
    restoreReadyState: (snapshot) => {
      applyRecoveryReady(state, snapshot, null);
    },
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

export type { RecoveryFlowActions, RecoveryFlowData, RecoveryReadySnapshot };
export { useRecoveryFlow };
