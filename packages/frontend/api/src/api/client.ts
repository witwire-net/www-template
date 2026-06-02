import {
  createApiSdk,
  getAccountSettings,
  updateAccountSettings,
  type AuthFailureClassification,
  type AuthFailureResponse,
  type AuthOperationErrorResponse,
  type DeviceLinkResponse,
  type PasskeyAddFinishRequest,
  type PasskeyAddStartResponse,
  type PasskeyItem,
  type PasskeyListResponse,
  type PasskeyRegisterStartRequest,
  type PasskeyStartResponse,
  type ProductAuthSessionResponse,
  type RecoveryAcceptedResponse,
  type RecoveryConsumeResponse,
  type ReauthenticationSessionKind,
  type ReauthenticationSessionResponse,
  type StatusResponse,
  type UlidId,
  type WebAuthnAssertionCredential,
  type WebAuthnAttestationCredential,
} from '../sdk';

import type { Status, AccountSetting, AccountLocale } from '../types';

const sdk = createApiSdk();

const toStatus = (dto: StatusResponse): Status => ({
  message: dto.message,
  timestamp: new Date(dto.timestamp),
});

/** Status API wrapper for the public sample endpoint. */
const statusApi = {
  get: async (): Promise<Status> => {
    const { data } = await sdk.status.get();
    return toStatus(data);
  },
};

/** AccountSetting API wrapper for authenticated account settings. */
const accountApi = {
  getSettings: async (
    options?: RequestInit
  ): Promise<{ setting: AccountSetting } | AuthFailureResponse> => {
    const response = await getAccountSettings(options);
    if (response.status === 200) {
      return { setting: response.data.setting };
    }
    return response.data;
  },
  updateSettings: async (
    locale: AccountLocale,
    options?: RequestInit
  ): Promise<{ setting: AccountSetting } | AuthFailureResponse | AuthOperationErrorResponse> => {
    const response = await updateAccountSettings({ locale }, options);
    if (response.status === 200) {
      return { setting: response.data.setting };
    }
    return response.data;
  },
};

interface AuthSuccess<T, S extends number = 200 | 202> {
  data: T;
  status: S;
  headers: Headers;
}

interface AuthFailure {
  data: AuthFailureResponse;
  status: 401 | 403 | 503;
  headers: Headers;
}

interface AuthOperationError {
  data: AuthOperationErrorResponse;
  status: 400 | 403 | 409;
  headers: Headers;
}

/** Auth API wrapper for passkey, recovery, and logout flows. */
const authApi = {
  logout: async (options?: RequestInit) => sdk.auth.logout(options),
  startPasskeyAuthentication: async (
    identifier: string,
    options?: RequestInit
  ): Promise<AuthSuccess<PasskeyStartResponse, 200> | AuthFailure> => {
    const response = await sdk.auth.startPasskeyAuthentication({ identifier }, options);
    if (response.status === 200 || response.status === 503) {
      return response;
    }

    const operationError = response as AuthOperationError;
    throw new Error(operationError.data.error);
  },
  finishPasskeyAuthentication: async (
    credential: WebAuthnAssertionCredential,
    options?: RequestInit
  ) => sdk.auth.finishPasskeyAuthentication({ credentialMode: 'cookie', credential }, options),
  startReauthentication: async (
    kind: ReauthenticationSessionKind,
    options?: RequestInit
  ): Promise<AuthSuccess<PasskeyStartResponse, 200> | AuthOperationError | AuthFailure> =>
    sdk.auth.startReauthentication({ kind }, options),
  finishReauthentication: async (
    requestId: UlidId,
    kind: ReauthenticationSessionKind,
    credential: WebAuthnAssertionCredential,
    options?: RequestInit
  ): Promise<
    AuthSuccess<ReauthenticationSessionResponse, 200> | AuthOperationError | AuthFailure
  > => sdk.auth.finishReauthentication({ requestId, kind, credential }, options),
  requestPasskeyRecovery: async (
    email: string,
    options?: RequestInit
  ): Promise<AuthSuccess<RecoveryAcceptedResponse, 202> | AuthFailure> => {
    const response = await sdk.auth.requestPasskeyRecovery({ email }, options);
    if (response.status === 202 || response.status === 503) {
      return response;
    }

    const operationError = response as AuthOperationError;
    throw new Error(operationError.data.error);
  },
  consumeRecoveryToken: async (
    token: string,
    options?: RequestInit
  ): Promise<AuthSuccess<RecoveryConsumeResponse, 200> | AuthOperationError | AuthFailure> =>
    sdk.auth.consumeRecoveryToken({ token }, options),
  startRecoveryPasskeyRegistration: async (
    recoverySession: string,
    options?: RequestInit
  ): Promise<AuthSuccess<PasskeyAddStartResponse, 200> | AuthOperationError | AuthFailure> => {
    const payload: PasskeyRegisterStartRequest = { recovery_session: recoverySession };
    const response = await sdk.auth.startPasskeyRegistration(payload, options);
    return response as AuthSuccess<PasskeyAddStartResponse, 200> | AuthOperationError | AuthFailure;
  },
  registerRecoveryPasskey: async (
    recoverySession: string,
    credential: WebAuthnAttestationCredential,
    options?: RequestInit
  ) =>
    sdk.auth.registerPasskey(
      { recovery_session: recoverySession, credentialMode: 'cookie', credential },
      options
    ),
  toFailureClassification: (failure: AuthFailureResponse): AuthFailureClassification =>
    failure.error,
  /**
   * Product auth session response から domain 層で使用する最小の session summary を抽出する。
   * Cookie mode / Bearer mode の共通フィールドだけを返し、refreshToken や Cookie command は含めない。
   * account subject payload は service artifact 境界で context を決定するため、明示的に account.accountId を抽出する。
   */
  toSessionSummary: (
    session: ProductAuthSessionResponse
  ): {
    requestId: string;
    authContextId: string;
    accountId: string;
    passkeyCredentialId?: string;
    sessionId: string;
    accessToken: string;
    expiresAt: string;
  } => {
    const account = session.account;
    return {
      requestId: session.requestId,
      authContextId: session.authContextId,
      accountId: account.accountId,
      passkeyCredentialId: account.passkeyCredentialId,
      sessionId: session.sessionId,
      accessToken: session.accessToken,
      expiresAt: session.expiresAt,
    };
  },

  // Passkey management (authenticated surface: /api/v1/passkeys)
  listPasskeys: async (
    options?: RequestInit
  ): Promise<AuthSuccess<PasskeyListResponse, 200> | AuthFailure> => sdk.auth.listPasskeys(options),
  startPasskeyAddition: async (
    options?: RequestInit
  ): Promise<AuthSuccess<PasskeyAddStartResponse, 200> | AuthFailure> =>
    sdk.auth.startPasskeyAddition(options),
  finishPasskeyAddition: async (
    credential: WebAuthnAttestationCredential,
    options?: RequestInit
  ): Promise<AuthSuccess<PasskeyListResponse, 200> | AuthFailure> => {
    const response = await sdk.auth.finishPasskeyAddition({ credential }, options);
    if (response.status === 400) {
      throw new Error(response.data.error);
    }
    return response;
  },
  deletePasskey: async (
    id: UlidId,
    reauthSession: string,
    options?: RequestInit
  ): Promise<AuthSuccess<void, 204> | AuthOperationError | AuthFailure> => {
    const mergedHeaders = new Headers(options?.headers);
    mergedHeaders.set('X-Reauth-Session', reauthSession);
    const response = await sdk.auth.deletePasskey(id, {
      ...options,
      headers: mergedHeaders,
    });
    if (response.status === 400 || response.status === 409 || response.status === 403) {
      throw new Error(response.data.error);
    }
    return response as unknown as AuthSuccess<void, 204> | AuthFailure;
  },
  sendDeviceLink: async (
    reauthSession: string,
    options?: RequestInit
  ): Promise<AuthSuccess<DeviceLinkResponse, 200> | AuthOperationError | AuthFailure> => {
    const mergedHeaders = new Headers(options?.headers);
    mergedHeaders.set('X-Reauth-Session', reauthSession);
    const response = await sdk.auth.sendDeviceLink({
      ...options,
      headers: mergedHeaders,
    });
    if (response.status === 400 || response.status === 403) {
      return response as AuthOperationError;
    }
    return response;
  },
};

export type { PasskeyAddFinishRequest, PasskeyItem, PasskeyListResponse, DeviceLinkResponse };
export { authApi, statusApi, accountApi };

// SDK types are internal; consumers should use domain types
