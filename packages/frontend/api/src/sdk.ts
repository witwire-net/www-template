import {
  consumeRecoveryToken,
  deletePasskey,
  finishPasskeyAddition,
  finishPasskeyAuthentication,
  finishReauthentication,
  getStatus,
  listPasskeys,
  listSessions,
  logout,
  refreshToken,
  registerPasskey,
  requestPasskeyRecovery,
  revokeOtherSessions,
  revokeSession,
  sendDeviceLink,
  startPasskeyAddition,
  startPasskeyAuthentication,
  startPasskeyRegistration,
  startReauthentication,
  type consumeRecoveryTokenResponse,
  type deletePasskeyResponse,
  type finishPasskeyAdditionResponse,
  type finishPasskeyAuthenticationResponse,
  type finishReauthenticationResponse,
  type getStatusResponse,
  type listPasskeysResponse,
  type listSessionsResponse,
  type logoutResponse,
  type PasskeyAddFinishRequest,
  type PasskeyFinishRequest,
  type PasskeyRegisterRequest,
  type PasskeyRegisterStartRequest,
  type PasskeyStartRequest,
  type RecoveryConsumeRequest,
  type RecoveryRequest,
  type registerPasskeyResponse,
  type requestPasskeyRecoveryResponse,
  type ReauthenticationFinishRequest,
  type ReauthenticationStartRequest,
  type sendDeviceLinkResponse,
  type startPasskeyAdditionResponse,
  type startPasskeyAuthenticationResponse,
  type startPasskeyRegistrationResponse,
  type startReauthenticationResponse,
  type UlidId,
} from './generated/client.js';

export type {
  AuthFailureClassification,
  AuthFailureResponse,
  AuthOperationErrorResponse,
  AuthSessionResponse,
  PasskeyAddFinishRequest,
  PasskeyAddStartResponse,
  PasskeyFinishRequest,
  PasskeyItem,
  PasskeyListResponse,
  PasskeyRegisterRequest,
  PasskeyRegisterStartRequest,
  PasskeyStartRequest,
  PasskeyStartResponse,
  DeviceLinkResponse,
  ErrorResponse,
  RecoveryAcceptedResponse,
  RecoveryConsumeRequest,
  RecoveryConsumeResponse,
  RecoveryRequest,
  ReauthenticationFinishRequest,
  ReauthenticationSessionKind,
  ReauthenticationSessionResponse,
  ReauthenticationStartRequest,
  StatusResponse,
  UlidId,
  WebAuthnAssertionCredential,
  WebAuthnAssertionResponse,
  WebAuthnAttestationCredential,
  WebAuthnAttestationResponse,
  WebAuthnCredentialDescriptor,
  consumeRecoveryTokenResponse,
  deletePasskeyResponse,
  finishPasskeyAdditionResponse,
  finishPasskeyAuthenticationResponse,
  finishReauthenticationResponse,
  getStatusResponse,
  listPasskeysResponse,
  logoutResponse,
  registerPasskeyResponse,
  requestPasskeyRecoveryResponse,
  sendDeviceLinkResponse,
  startPasskeyAdditionResponse,
  startPasskeyAuthenticationResponse,
  startPasskeyRegistrationResponse,
  startReauthenticationResponse,
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
  sendDeviceLink: (options?: RequestInit) => Promise<sendDeviceLinkResponse>;
  startReauthentication: (
    payload: ReauthenticationStartRequest,
    options?: RequestInit
  ) => Promise<startReauthenticationResponse>;
  finishReauthentication: (
    payload: ReauthenticationFinishRequest,
    options?: RequestInit
  ) => Promise<finishReauthenticationResponse>;
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
  sendDeviceLink: (options?: RequestInit) => sendDeviceLink(withDefaultInit(options, defaultInit)),
  startReauthentication: (payload: ReauthenticationStartRequest, options?: RequestInit) =>
    startReauthentication(payload, withJsonInit(options, defaultInit)),
  finishReauthentication: (payload: ReauthenticationFinishRequest, options?: RequestInit) =>
    finishReauthentication(payload, withJsonInit(options, defaultInit)),
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

export { listSessions, refreshToken, revokeOtherSessions, revokeSession };

export type { listSessionsResponse };

// AccountSetting generated SDK exports
export {
  getAccountSettings,
  updateAccountSettings,
  type AccountLocale,
  type AccountSetting,
  type AccountSettingResponse,
  type AccountSettingSnapshot,
  type UpdateAccountSettingRequest,
} from './generated/client';
