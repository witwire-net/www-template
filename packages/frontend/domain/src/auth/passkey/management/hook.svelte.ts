import { authApi } from '@www-template/api';

import {
  applyDeviceLinkSent,
  applyPasskeyDeleted,
  applyPasskeyError,
  applyPasskeyList,
  applyReauthSession,
  createPasskeyManagementInitialState,
  toPasskeyManagementErrorMessage,
} from './state';

import {
  createWebAuthnAttestation,
  getWebAuthnAssertion,
  normalizeWebAuthnError,
} from '../../webauthn';

import { useAuthSession } from '../../session/hook.svelte';

import type { PasskeyItem, PasskeyManagementState } from '../../types';

type AuthSessionRef = ReturnType<typeof useAuthSession>;
type PasskeyAuthFailure = 'session-expired' | 'unauthenticated' | 'account-suspended';

const AUTH_FAILURE_CODES = new Set<string>([
  'session-expired',
  'unauthenticated',
  'account-suspended',
]);

interface PasskeyManagementData {
  passkeys: PasskeyItem[];
  loading: boolean;
  error: string | null;
  reauthSession: string | null;
  deviceLinkSent: boolean;
}

interface PasskeyManagementActions {
  listPasskeys: () => Promise<void>;
  addPasskey: () => Promise<void>;
  deletePasskey: (id: string, reauthSession: string) => Promise<void>;
  sendDeviceLink: (reauthSession: string) => Promise<boolean>;
  performReauth: (kind: 'device-link' | 'passkey-delete') => Promise<string | null>;
  clearReauthSession: () => void;
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
  if (AUTH_FAILURE_CODES.has(errorCode)) {
    const failure = errorCode as PasskeyAuthFailure;
    authSession.actions.handleFailure(failure, fallbackMessage);
  } else {
    applyPasskeyError(state, fallbackMessage);
  }
}

function handlePasskeyFailureResponse(
  response: { status: number; data: unknown },
  fallbackMessage: string,
  state: PasskeyManagementState,
  authSession: AuthSessionRef
): boolean {
  if (response.status === 401 || response.status === 403 || response.status === 503) {
    const error =
      typeof response.data === 'object' && response.data !== null && 'error' in response.data
        ? String((response.data as { error: unknown }).error)
        : 'internal-error';
    handleApiError(error, fallbackMessage, state, authSession);
    return true;
  }

  return false;
}

function handleAuthOnlyFailure(
  errorCode: string,
  fallbackMessage: string,
  state: PasskeyManagementState,
  authSession: AuthSessionRef
): boolean {
  if (AUTH_FAILURE_CODES.has(errorCode)) {
    const failure = errorCode as PasskeyAuthFailure;
    authSession.actions.handleFailure(failure, fallbackMessage);
    return true;
  }

  return false;
}

const createListPasskeys =
  (state: PasskeyManagementState, authSession: AuthSessionRef) => async () => {
    state.loading = true;
    state.error = null;
    try {
      const response = await authApi.listPasskeys({
        headers: authSession.actions.createAuthorizationHeaders(),
      });
      if (
        handlePasskeyFailureResponse(
          response,
          'パスキー一覧を取得できませんでした。',
          state,
          authSession
        )
      ) {
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
      if (
        handlePasskeyFailureResponse(
          startResponse,
          'パスキー追加を開始できませんでした。',
          state,
          authSession
        )
      ) {
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
      if (
        handlePasskeyFailureResponse(
          finishResponse,
          'パスキー追加を完了できませんでした。',
          state,
          authSession
        )
      ) {
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
  async (id: string, reauthSession: string): Promise<void> => {
    state.loading = true;
    state.error = null;
    try {
      const response = await authApi.deletePasskey(id, reauthSession, {
        headers: authSession.actions.createAuthorizationHeaders(),
      });
      if (
        handlePasskeyFailureResponse(
          response,
          'パスキーを削除できませんでした。',
          state,
          authSession
        )
      ) {
        return;
      }
      if (response.status === 409) {
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

const createSendDeviceLink =
  (state: PasskeyManagementState, authSession: AuthSessionRef) =>
  async (reauthSession: string): Promise<boolean> => {
    state.loading = true;
    state.error = null;
    try {
      const response = await authApi.sendDeviceLink(reauthSession, {
        headers: authSession.actions.createAuthorizationHeaders(),
      });
      if (response.status === 401 || response.status === 503) {
        handleApiError(
          response.data.error,
          'ログイン有効化リンクを送信できませんでした。',
          state,
          authSession
        );
        return false;
      }
      if (response.status === 400 || response.status === 403) {
        if (
          handleAuthOnlyFailure(
            response.data.error,
            'ログイン有効化リンクを送信できませんでした。',
            state,
            authSession
          )
        ) {
          return false;
        }
        applyPasskeyError(state, response.data.error);
        return false;
      }
      if (response.status === 200) {
        applyDeviceLinkSent(state, response.data.issued);
        return response.data.issued;
      }
      return false;
    } catch (error: unknown) {
      applyPasskeyError(state, toPasskeyManagementErrorMessage(error));
      return false;
    } finally {
      state.loading = false;
    }
  };

const createClearReauthSession = (state: PasskeyManagementState) => (): void => {
  applyReauthSession(state, null);
};

const createPerformReauth =
  (state: PasskeyManagementState, authSession: AuthSessionRef) =>
  async (kind: 'device-link' | 'passkey-delete'): Promise<string | null> => {
    state.loading = true;
    state.error = null;
    try {
      const startResponse = await authApi.startReauthentication(kind, {
        headers: authSession.actions.createAuthorizationHeaders(),
      });
      if (
        handlePasskeyFailureResponse(
          startResponse,
          '再認証を開始できませんでした。',
          state,
          authSession
        )
      ) {
        return null;
      }
      if (startResponse.status !== 200) return null;

      let credential;
      try {
        credential = await getWebAuthnAssertion(startResponse.data);
      } catch (webAuthnError: unknown) {
        applyPasskeyError(state, normalizeWebAuthnError(webAuthnError));
        return null;
      }

      const finishResponse = await authApi.finishReauthentication(
        startResponse.data.requestId,
        kind,
        credential,
        {
          headers: authSession.actions.createAuthorizationHeaders(),
        }
      );
      if (
        handlePasskeyFailureResponse(
          finishResponse,
          '再認証を完了できませんでした。',
          state,
          authSession
        )
      ) {
        return null;
      }
      if (finishResponse.status === 200) {
        applyReauthSession(state, finishResponse.data.reauthSessionId);
        return finishResponse.data.reauthSessionId;
      }
      return null;
    } catch (error: unknown) {
      applyPasskeyError(state, toPasskeyManagementErrorMessage(error));
      return null;
    } finally {
      state.loading = false;
    }
  };

/** 認証済みユーザーのパスキー一覧・追加（WebAuthn）・削除・デバイスリンク送信を扱う domain composable。 */
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
    sendDeviceLink: createSendDeviceLink(state, authSession),
    performReauth: createPerformReauth(state, authSession),
    clearReauthSession: createClearReauthSession(state),
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
      get reauthSession() {
        return state.reauthSession;
      },
      get deviceLinkSent() {
        return state.deviceLinkSent;
      },
    },
    actions,
  };
}

export type { PasskeyManagementActions, PasskeyManagementData };
export { usePasskeyManagement };
