import {
  consumeRecoveryToken,
  finishPasskeyAuthentication,
  getStatus,
  logout,
  registerPasskey,
  requestPasskeyRecovery,
  startPasskeyAuthentication,
  type consumeRecoveryTokenResponse,
  type finishPasskeyAuthenticationResponse,
  type getStatusResponse,
  type logoutResponse,
  type PasskeyFinishRequest,
  type PasskeyRegisterRequest,
  type PasskeyStartRequest,
  type RecoveryConsumeRequest,
  type RecoveryRequest,
  type registerPasskeyResponse,
  type requestPasskeyRecoveryResponse,
  type startPasskeyAuthenticationResponse,
} from './generated/client.js';

export type {
  AuthFailureClassification,
  AuthFailureResponse,
  AuthOperationErrorResponse,
  AuthSessionResponse,
  PasskeyFinishRequest,
  PasskeyRegisterRequest,
  PasskeyStartRequest,
  PasskeyStartResponse,
  ErrorResponse,
  RecoveryAcceptedResponse,
  RecoveryConsumeRequest,
  RecoveryConsumeResponse,
  RecoveryRequest,
  StatusResponse,
  consumeRecoveryTokenResponse,
  finishPasskeyAuthenticationResponse,
  getStatusResponse,
  logoutResponse,
  registerPasskeyResponse,
  requestPasskeyRecoveryResponse,
  startPasskeyAuthenticationResponse,
} from './generated/client.js';

/** Configuration for the API SDK default request settings. */
/** Configuration for the API SDK default request settings. */
interface ApiSdkConfig {
  defaultInit?: RequestInit;
}

interface AuthSdk {
  logout: (options?: RequestInit) => Promise<logoutResponse>;
  startPasskeyAuthentication: (
    payload: PasskeyStartRequest,
    options?: RequestInit
  ) => Promise<startPasskeyAuthenticationResponse>;
  finishPasskeyAuthentication: (
    payload: PasskeyFinishRequest,
    options?: RequestInit
  ) => Promise<finishPasskeyAuthenticationResponse>;
  requestPasskeyRecovery: (
    payload: RecoveryRequest,
    options?: RequestInit
  ) => Promise<requestPasskeyRecoveryResponse>;
  consumeRecoveryToken: (
    payload: RecoveryConsumeRequest,
    options?: RequestInit
  ) => Promise<consumeRecoveryTokenResponse>;
  registerPasskey: (
    payload: PasskeyRegisterRequest,
    options?: RequestInit
  ) => Promise<registerPasskeyResponse>;
}

const toHeaderObject = (headers?: HeadersInit): Record<string, string> => {
  if (headers == null) {
    return {};
  }
  // Normalize using the built-in Headers implementation to avoid unsafe casts
  const normalized = new Headers(headers);
  return Object.fromEntries(normalized.entries());
};

const withDefaultInit = (init: RequestInit | undefined, defaultInit: RequestInit | undefined) => {
  if (defaultInit == null) {
    return init;
  }
  return {
    ...defaultInit,
    ...init,
    headers: {
      ...toHeaderObject(defaultInit.headers),
      ...toHeaderObject(init?.headers),
    },
  };
};

const withJsonInit = (options: RequestInit | undefined, defaultInit: RequestInit | undefined) =>
  withDefaultInit(
    {
      ...options,
      headers: {
        'Content-Type': 'application/json',
        ...toHeaderObject(options?.headers),
      },
    },
    defaultInit
  );

const createAuthSdk = (defaultInit: RequestInit | undefined): AuthSdk => ({
  logout: (options?: RequestInit) => logout(withDefaultInit(options, defaultInit)),
  startPasskeyAuthentication: (payload: PasskeyStartRequest, options?: RequestInit) =>
    startPasskeyAuthentication(payload, withJsonInit(options, defaultInit)),
  finishPasskeyAuthentication: (payload: PasskeyFinishRequest, options?: RequestInit) =>
    finishPasskeyAuthentication(payload, withJsonInit(options, defaultInit)),
  requestPasskeyRecovery: (payload: RecoveryRequest, options?: RequestInit) =>
    requestPasskeyRecovery(payload, withJsonInit(options, defaultInit)),
  consumeRecoveryToken: (payload: RecoveryConsumeRequest, options?: RequestInit) =>
    consumeRecoveryToken(payload, withJsonInit(options, defaultInit)),
  registerPasskey: (payload: PasskeyRegisterRequest, options?: RequestInit) =>
    registerPasskey(payload, withJsonInit(options, defaultInit)),
});

/** Create a typed API SDK wrapper with optional default request init. */
/** Create a typed API SDK wrapper with optional default request init. */
const createApiSdk = (config?: ApiSdkConfig) => {
  const defaultInit = config?.defaultInit;

  return {
    status: {
      get: (options?: RequestInit): Promise<getStatusResponse> =>
        getStatus(withDefaultInit(options, defaultInit)),
    },
    auth: createAuthSdk(defaultInit),
  };
};

export type { ApiSdkConfig };
export { createApiSdk };
