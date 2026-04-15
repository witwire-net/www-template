import { authApi } from '@www-template/api';

import {
  applyPasskeyDeleted,
  applyPasskeyError,
  applyPasskeyList,
  createPasskeyManagementInitialState,
  createWebAuthnAttestation,
  normalizeWebAuthnError,
  toPasskeyManagementErrorMessage,
} from '../../auth';

import { useAuthSession } from './useAuthSession.svelte';

import type { PasskeyItem, PasskeyManagementState } from 'types';

type AuthSessionRef = ReturnType<typeof useAuthSession>;

interface PasskeyManagementData {
  passkeys: PasskeyItem[];
  loading: boolean;
  error: string | null;
}

interface PasskeyManagementActions {
  listPasskeys: () => Promise<void>;
  addPasskey: () => Promise<void>;
  deletePasskey: (id: string) => Promise<void>;
  issueOtp: () => Promise<string | null>;
}

/**
 * API エラーレスポンスを処理する。
 * session-expired / unauthenticated は auth session に委譲し、
 * それ以外（internal-error 等）は passkey state にエラーとして記録する。
 */
function handleApiError(
  errorCode: string,
  fallbackMessage: string,
  state: PasskeyManagementState,
  authSession: AuthSessionRef
): void {
  if (errorCode === 'session-expired' || errorCode === 'unauthenticated') {
    authSession.actions.handleFailure(errorCode, fallbackMessage);
  } else {
    applyPasskeyError(state, fallbackMessage);
  }
}

const createListPasskeys =
  (state: PasskeyManagementState, authSession: AuthSessionRef) => async () => {
    state.loading = true;
    state.error = null;
    try {
      const response = await authApi.listPasskeys({
        headers: authSession.actions.createAuthorizationHeaders(),
      });
      if (response.status === 401 || response.status === 503) {
        handleApiError(
          response.data.error,
          'パスキー一覧を取得できませんでした。',
          state,
          authSession
        );
        return;
      }
      if (response.status === 200) {
        applyPasskeyList(state, response.data.passkeys);
      }
    } catch (error: unknown) {
      applyPasskeyError(state, toPasskeyManagementErrorMessage(error));
    } finally {
      state.loading = false;
    }
  };

const createAddPasskey =
  (state: PasskeyManagementState, authSession: AuthSessionRef) => async (): Promise<void> => {
    state.loading = true;
    state.error = null;
    try {
      // Step 1: Start — get challenge from server
      const startResponse = await authApi.startPasskeyAddition({
        headers: authSession.actions.createAuthorizationHeaders(),
      });
      if (startResponse.status === 401 || startResponse.status === 503) {
        handleApiError(
          startResponse.data.error,
          'パスキー追加を開始できませんでした。',
          state,
          authSession
        );
        return;
      }
      if (startResponse.status !== 200) return;

      // Step 2: Call browser WebAuthn API — normalize browser/device errors only
      let credential;
      try {
        credential = await createWebAuthnAttestation(startResponse.data);
      } catch (webAuthnError: unknown) {
        applyPasskeyError(state, normalizeWebAuthnError(webAuthnError));
        return;
      }

      // Step 3: Finish — send attestation to server
      const finishResponse = await authApi.finishPasskeyAddition(credential, {
        headers: authSession.actions.createAuthorizationHeaders(),
      });
      if (finishResponse.status === 401 || finishResponse.status === 503) {
        handleApiError(
          finishResponse.data.error,
          'パスキー追加を完了できませんでした。',
          state,
          authSession
        );
        return;
      }
      if (finishResponse.status === 200) {
        applyPasskeyList(state, finishResponse.data.passkeys);
      }
    } catch (error: unknown) {
      applyPasskeyError(state, toPasskeyManagementErrorMessage(error));
    } finally {
      state.loading = false;
    }
  };

const createDeletePasskey =
  (state: PasskeyManagementState, authSession: AuthSessionRef) =>
  async (id: string): Promise<void> => {
    state.loading = true;
    state.error = null;
    try {
      const response = await authApi.deletePasskey(id, {
        headers: authSession.actions.createAuthorizationHeaders(),
      });
      if (response.status === 401 || response.status === 503) {
        handleApiError(response.data.error, 'パスキーを削除できませんでした。', state, authSession);
        return;
      }
      applyPasskeyDeleted(state, id);
    } catch (error: unknown) {
      applyPasskeyError(state, toPasskeyManagementErrorMessage(error));
    } finally {
      state.loading = false;
    }
  };

const createIssueOtp =
  (state: PasskeyManagementState, authSession: AuthSessionRef) =>
  async (): Promise<string | null> => {
    state.loading = true;
    state.error = null;
    try {
      const response = await authApi.issuePasskeyOtp({
        headers: authSession.actions.createAuthorizationHeaders(),
      });
      if (response.status === 401 || response.status === 503) {
        handleApiError(response.data.error, 'OTP を発行できませんでした。', state, authSession);
        return null;
      }
      if (response.status === 200) {
        return response.data.otp;
      }
      return null;
    } catch (error: unknown) {
      applyPasskeyError(state, toPasskeyManagementErrorMessage(error));
      return null;
    } finally {
      state.loading = false;
    }
  };

/** 認証済みユーザーのパスキー一覧・追加（WebAuthn）・削除・OTP 発行を扱う domain composable。 */
function usePasskeyManagement(): {
  data: PasskeyManagementData;
  actions: PasskeyManagementActions;
} {
  const state = $state<PasskeyManagementState>(createPasskeyManagementInitialState());
  const authSession = useAuthSession();

  const actions: PasskeyManagementActions = {
    listPasskeys: createListPasskeys(state, authSession),
    addPasskey: createAddPasskey(state, authSession),
    deletePasskey: createDeletePasskey(state, authSession),
    issueOtp: createIssueOtp(state, authSession),
  };

  return {
    data: {
      get passkeys() {
        return state.passkeys;
      },
      get loading() {
        return state.loading;
      },
      get error() {
        return state.error;
      },
    },
    actions,
  };
}

export type { PasskeyManagementActions, PasskeyManagementData };
export { usePasskeyManagement };
