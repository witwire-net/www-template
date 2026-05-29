export type {
  PasskeyAddStartOptions,
  PasskeyStartOptions,
  WebAuthnAssertionResult,
  WebAuthnAttestationResult,
} from './webauthn';
export {
  base64urlToBuffer,
  bufferToBase64url,
  createWebAuthnAttestation,
  getWebAuthnAssertion,
  normalizeWebAuthnError,
} from './webauthn';
export {
  createPasskeyLoginInitialState,
  toPasskeyErrorMessage,
  toRecoveryErrorMessage,
} from './passkey/state';
export {
  applyPasskeyDeleted,
  applyPasskeyError,
  applyPasskeyList,
  createPasskeyManagementInitialState,
  toPasskeyManagementErrorMessage,
} from './passkey/management/state';
export {
  applyInvalidRecoveryToken,
  applyRecoveryAccepted,
  applyRecoveryReady,
  clearRecoveryState,
  createRecoveryFlowInitialState,
} from './recovery/state';
export {
  addAuthenticatedSession,
  applyAccountSuspended,
  applyAuthenticatedSession,
  applyExpiredSession,
  applyInternalError,
  applyMissingSession,
  clearAuthSession,
  createAuthSessionInitialState,
  createAuthorizationHeaders,
  hasUlidAuthSessionShape,
  isNoStoreCacheControl,
  isUlid,
  removeActiveSession,
  removeSessionById,
  switchActiveSession,
} from './session/state';
export {
  createEmptyAccessTokenState,
  decodeAccessToken,
  isRefreshNeeded,
} from './session/token_state';
export type { AccessTokenClaims, MemoryAccessTokenState } from './session/token_state';
export { createGenericRecoverySentView } from './recovery/state';
export type {
  AuthFailureState,
  AuthRouteIntent,
  AuthSessionState,
  AuthSessionSummary,
  PasskeyItem,
  PasskeyLoginState,
  PasskeyManagementState,
  RecoveryFlowState,
  RecoverySentView,
} from './types';
export { useAuthSession } from './session/hook.svelte';
export type { AuthSessionActions, AuthSessionData } from './session/hook.svelte';
export type { DeviceSession, ListDevicesResult } from './session/session_api';
export { usePasskeyLogin } from './passkey/hook.svelte';
export type { PasskeyLoginActions, PasskeyLoginData } from './passkey/hook.svelte';
export { usePasskeyManagement } from './passkey/management/hook.svelte';
export type {
  PasskeyManagementActions,
  PasskeyManagementData,
} from './passkey/management/hook.svelte';
export { useRecoveryFlow } from './recovery/hook.svelte';
export type { RecoveryFlowActions, RecoveryFlowData } from './recovery/hook.svelte';
export { useSessionGuard } from './guard/hook.svelte';
export type {
  SessionGuardActions,
  SessionGuardData,
  SessionGuardOptions,
} from './guard/hook.svelte';
