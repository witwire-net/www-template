import {
  createApiSdk,
  type AuthFailureClassification,
  type AuthFailureResponse,
  type AuthOperationErrorResponse,
  type AuthSessionResponse,
  type PasskeyStartResponse,
  type Profile,
  type RecoveryAcceptedResponse,
  type RecoveryConsumeResponse,
  type StatusResponse,
} from '../sdk';

import type { CreateProfilePayload, Status } from '../types';

const sdk = createApiSdk();

const toStatus = (dto: StatusResponse): Status => ({
  message: dto.message,
  timestamp: new Date(dto.timestamp),
});

const toProfile = (dto: Profile) => ({
  id: dto.id,
  name: dto.name,
  email: dto.email,
  createdAt: new Date(dto.createdAt),
});

/** Status API wrapper for the public sample endpoint. */
const statusApi = {
  get: async (): Promise<Status> => {
    const { data } = await sdk.status.get();
    return toStatus(data);
  },
};

/** Profiles API wrapper for list/create/get operations. */
const profilesApi = {
  list: async () => {
    const response = (await sdk.profiles.list()) as { data: unknown; status: number };
    if (response.status !== 200) {
      const maybeError = response.data as { error?: string };
      throw new Error(maybeError.error ?? 'Failed to fetch profiles');
    }
    if (!Array.isArray(response.data)) {
      throw new TypeError('Invalid profiles response');
    }
    return response.data.map((user) => toProfile(user as Profile));
  },
  create: async (payload: CreateProfilePayload) => {
    const response = await sdk.profiles.create(payload);
    if (response.status !== 201) {
      const maybeError = response.data as { error?: string };
      throw new Error(maybeError.error ?? 'Failed to create profile');
    }
    return toProfile(response.data);
  },
  get: async (id: number) => {
    const response = await sdk.profiles.get(id);
    if (response.status !== 200) {
      return null;
    }
    return toProfile(response.data);
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
};

export { authApi, profilesApi, statusApi };

// SDK types are internal; consumers should use domain types
