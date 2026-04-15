import {
  consumeRecoveryToken,
  deletePasskey,
  finishPasskeyAddition,
  finishPasskeyAdditionByOtp,
  finishPasskeyAuthentication,
  getStatus,
  issuePasskeyOtp,
  listPasskeys,
  logout,
  registerPasskey,
  requestPasskeyRecovery,
  startPasskeyAddition,
  startPasskeyAdditionByOtp,
  startPasskeyAuthentication,
  startPasskeyRegistration,
  type consumeRecoveryTokenResponse,
  type deletePasskeyResponse,
  type finishPasskeyAdditionByOtpResponse,
  type finishPasskeyAdditionResponse,
  type finishPasskeyAuthenticationResponse,
  type getStatusResponse,
  type issuePasskeyOtpResponse,
  type listPasskeysResponse,
  type logoutResponse,
  type PasskeyAddByOtpFinishRequest,
  type PasskeyAddByOtpStartRequest,
  type PasskeyAddFinishRequest,
  type PasskeyFinishRequest,
  type PasskeyRegisterRequest,
  type PasskeyRegisterStartRequest,
  type PasskeyStartRequest,
  type RecoveryConsumeRequest,
  type RecoveryRequest,
  type registerPasskeyResponse,
  type requestPasskeyRecoveryResponse,
  type startPasskeyAdditionByOtpResponse,
  type startPasskeyAdditionResponse,
  type startPasskeyAuthenticationResponse,
  type startPasskeyRegistrationResponse,
  type UlidId,
} from './generated/client.js';

export type {
  AuthFailureClassification,
  AuthFailureResponse,
  AuthOperationErrorResponse,
  AuthSessionResponse,
  PasskeyAddByOtpFinishRequest,
  PasskeyAddByOtpStartRequest,
  PasskeyAddFinishRequest,
  PasskeyAddStartResponse,
  PasskeyFinishRequest,
  PasskeyItem,
  PasskeyListResponse,
  PasskeyOtpResponse,
  PasskeyRegisterRequest,
  PasskeyRegisterStartRequest,
  PasskeyStartRequest,
  PasskeyStartResponse,
  ErrorResponse,
  RecoveryAcceptedResponse,
  RecoveryConsumeRequest,
  RecoveryConsumeResponse,
  RecoveryRequest,
  StatusResponse,
  UlidId,
  WebAuthnAssertionCredential,
  WebAuthnAssertionResponse,
  WebAuthnAttestationCredential,
  WebAuthnAttestationResponse,
  WebAuthnCredentialDescriptor,
  consumeRecoveryTokenResponse,
  deletePasskeyResponse,
  finishPasskeyAdditionByOtpResponse,
  finishPasskeyAdditionResponse,
  finishPasskeyAuthenticationResponse,
  getStatusResponse,
  issuePasskeyOtpResponse,
  listPasskeysResponse,
  logoutResponse,
  registerPasskeyResponse,
  requestPasskeyRecoveryResponse,
  startPasskeyAdditionByOtpResponse,
  startPasskeyAdditionResponse,
  startPasskeyAuthenticationResponse,
  startPasskeyRegistrationResponse,
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
  startPasskeyRegistration: (
    payload: PasskeyRegisterStartRequest,
    options?: RequestInit
  ) => Promise<startPasskeyRegistrationResponse>;
  registerPasskey: (
    payload: PasskeyRegisterRequest,
    options?: RequestInit
  ) => Promise<registerPasskeyResponse>;
  listPasskeys: (options?: RequestInit) => Promise<listPasskeysResponse>;
  startPasskeyAddition: (options?: RequestInit) => Promise<startPasskeyAdditionResponse>;
  finishPasskeyAddition: (
    payload: PasskeyAddFinishRequest,
    options?: RequestInit
  ) => Promise<finishPasskeyAdditionResponse>;
  deletePasskey: (id: UlidId, options?: RequestInit) => Promise<deletePasskeyResponse>;
  issuePasskeyOtp: (options?: RequestInit) => Promise<issuePasskeyOtpResponse>;
  startPasskeyAdditionByOtp: (
    payload: PasskeyAddByOtpStartRequest,
    options?: RequestInit
  ) => Promise<startPasskeyAdditionByOtpResponse>;
  finishPasskeyAdditionByOtp: (
    payload: PasskeyAddByOtpFinishRequest,
    options?: RequestInit
  ) => Promise<finishPasskeyAdditionByOtpResponse>;
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
  startPasskeyRegistration: (payload: PasskeyRegisterStartRequest, options?: RequestInit) =>
    startPasskeyRegistration(payload, withJsonInit(options, defaultInit)),
  registerPasskey: (payload: PasskeyRegisterRequest, options?: RequestInit) =>
    registerPasskey(payload, withJsonInit(options, defaultInit)),
  listPasskeys: (options?: RequestInit) => listPasskeys(withDefaultInit(options, defaultInit)),
  startPasskeyAddition: (options?: RequestInit) =>
    startPasskeyAddition(withDefaultInit(options, defaultInit)),
  finishPasskeyAddition: (payload: PasskeyAddFinishRequest, options?: RequestInit) =>
    finishPasskeyAddition(payload, withJsonInit(options, defaultInit)),
  deletePasskey: (id: UlidId, options?: RequestInit) =>
    deletePasskey(id, withDefaultInit(options, defaultInit)),
  issuePasskeyOtp: (options?: RequestInit) =>
    issuePasskeyOtp(withDefaultInit(options, defaultInit)),
  startPasskeyAdditionByOtp: (payload: PasskeyAddByOtpStartRequest, options?: RequestInit) =>
    startPasskeyAdditionByOtp(payload, withJsonInit(options, defaultInit)),
  finishPasskeyAdditionByOtp: (payload: PasskeyAddByOtpFinishRequest, options?: RequestInit) =>
    finishPasskeyAdditionByOtp(payload, withJsonInit(options, defaultInit)),
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
