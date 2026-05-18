import {
  createApiSdk,
  type AuthFailureClassification,
  type AuthFailureResponse,
  type AuthOperationErrorResponse,
  type AuthSessionResponse,
  type DeviceLinkResponse,
  type PasskeyAddFinishRequest,
  type PasskeyAddStartResponse,
  type PasskeyItem,
  type PasskeyListResponse,
  type PasskeyRegisterStartRequest,
  type PasskeyStartResponse,
  type RecoveryAcceptedResponse,
  type RecoveryConsumeResponse,
  type ReauthenticationSessionKind,
  type ReauthenticationSessionResponse,
  type StatusResponse,
  type UlidId,
  type WebAuthnAssertionCredential,
  type WebAuthnAttestationCredential,
} from '../sdk';

import type { Status } from '../types';

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
  ) => sdk.auth.finishPasskeyAuthentication({ credential }, options),
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
  ) => sdk.auth.registerPasskey({ recovery_session: recoverySession, credential }, options),
  toFailureClassification: (failure: AuthFailureResponse): AuthFailureClassification =>
    failure.error,
  toSessionSummary: (session: AuthSessionResponse): AuthSessionResponse => session,

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
export { authApi, statusApi };

// SDK types are internal; consumers should use domain types
