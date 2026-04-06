import {
  createApiSdk,
  type AuthFailureClassification,
  type AuthFailureResponse,
  type AuthOperationErrorResponse,
  type AuthSessionResponse,
  type PasskeyAddByOtpFinishRequest,
  type PasskeyAddByOtpStartRequest,
  type PasskeyAddFinishRequest,
  type PasskeyAddStartResponse,
  type PasskeyItem,
  type PasskeyListResponse,
  type PasskeyOtpResponse,
  type PasskeyStartResponse,
  type RecoveryAcceptedResponse,
  type RecoveryConsumeResponse,
  type StatusResponse,
  type UlidId,
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
  status: 401 | 503;
  headers: Headers;
}

interface AuthOperationError {
  data: AuthOperationErrorResponse;
  status: 400;
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
  finishPasskeyAuthentication: async (credential: string, options?: RequestInit) =>
    sdk.auth.finishPasskeyAuthentication({ credential }, options),
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
  registerRecoveryPasskey: async (
    recoverySession: string,
    credential: string,
    options?: RequestInit
  ) => sdk.auth.registerPasskey({ recovery_session: recoverySession, credential }, options),
  toFailureClassification: (failure: AuthFailureResponse): AuthFailureClassification =>
    failure.error,
  toSessionSummary: (session: AuthSessionResponse): AuthSessionResponse => session,

  // Passkey management (authenticated surface: /api/v1/app/passkeys)
  listPasskeys: async (
    options?: RequestInit
  ): Promise<AuthSuccess<PasskeyListResponse, 200> | AuthFailure> => sdk.auth.listPasskeys(options),
  startPasskeyAddition: async (
    options?: RequestInit
  ): Promise<AuthSuccess<PasskeyAddStartResponse, 200> | AuthFailure> =>
    sdk.auth.startPasskeyAddition(options),
  finishPasskeyAddition: async (
    credential: string,
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
    options?: RequestInit
  ): Promise<AuthSuccess<void, 204> | AuthFailure> => {
    const response = await sdk.auth.deletePasskey(id, options);
    if (response.status === 409 || response.status === 403) {
      throw new Error(response.data.error);
    }
    return response as unknown as AuthSuccess<void, 204> | AuthFailure;
  },
  issuePasskeyOtp: async (
    options?: RequestInit
  ): Promise<AuthSuccess<PasskeyOtpResponse, 200> | AuthFailure> => {
    const response = await sdk.auth.issuePasskeyOtp(options);
    if (response.status === 400) {
      throw new Error(response.data.error);
    }
    return response;
  },

  // OTP-based passkey addition (public surface: /api/v1/auth/passkey/add/*)
  startPasskeyAdditionByOtp: async (
    otp: string,
    options?: RequestInit
  ): Promise<PasskeyAddStartResponse> => {
    const response = await sdk.auth.startPasskeyAdditionByOtp({ otp }, options);
    if (response.status === 200) {
      return response.data;
    }
    if (response.status === 400) {
      throw new Error(response.data.error);
    }
    throw new Error('passkey_add_by_otp_start_failed');
  },
  finishPasskeyAdditionByOtp: async (
    otp: string,
    credential: string,
    options?: RequestInit
  ): Promise<void> => {
    const response = await sdk.auth.finishPasskeyAdditionByOtp({ otp, credential }, options);
    if (response.status === 200) {
      return;
    }
    if (response.status === 400) {
      throw new Error(response.data.error);
    }
    throw new Error('passkey_add_by_otp_finish_failed');
  },
};

export type {
  PasskeyAddByOtpFinishRequest,
  PasskeyAddByOtpStartRequest,
  PasskeyAddFinishRequest,
  PasskeyItem,
  PasskeyListResponse,
  PasskeyOtpResponse,
};
export { authApi, statusApi };

// SDK types are internal; consumers should use domain types
