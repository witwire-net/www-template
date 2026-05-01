import type { RecoveryFlowState, RecoverySentView } from '../types';

/** enumeration-safe な recovery sent view を返す。 */
function createGenericRecoverySentView(): RecoverySentView {
  return {
    title: 'メールをご確認ください',
    description: '登録済みの宛先であれば、復旧用リンクをお送りします。',
    helper:
      '届かない場合は迷惑メールフォルダをご確認のうえ、しばらく待ってから再度お試しください。',
  };
}

/** recovery flow state の初期値を作る。 */
function createRecoveryFlowInitialState(): RecoveryFlowState {
  return {
    email: '',
    phase: 'idle',
    requestId: null,
    noticeId: null,
    recoveryTokenId: null,
    recoverySessionId: null,
    recoverySession: null,
    expiresAt: null,
    lastCacheControl: null,
    error: null,
    sentView: createGenericRecoverySentView(),
  };
}

/** recovery request accepted を sent state へ写像する。 */
function applyRecoveryAccepted(
  state: RecoveryFlowState,
  requestId: string,
  cacheControl: string | null
): void {
  state.phase = 'sent';
  state.requestId = requestId;
  state.noticeId = requestId;
  state.lastCacheControl = cacheControl;
  state.error = null;
  state.sentView = createGenericRecoverySentView();
}

/** valid recovery token を recovery-ready state へ写像する。 */
function applyRecoveryReady(
  state: RecoveryFlowState,
  payload: {
    requestId: string;
    recoveryTokenId: string;
    recoverySessionId: string;
    recoverySession: string;
    expiresAt: string;
  },
  cacheControl: string | null
): void {
  state.phase = 'ready';
  state.requestId = payload.requestId;
  state.noticeId = payload.requestId;
  state.recoveryTokenId = payload.recoveryTokenId;
  state.recoverySessionId = payload.recoverySessionId;
  state.recoverySession = payload.recoverySession;
  state.expiresAt = payload.expiresAt;
  state.lastCacheControl = cacheControl;
  state.error = null;
}

/** invalid / expired / consumed token を retry guidance state へ写像する。 */
function applyInvalidRecoveryToken(
  state: RecoveryFlowState,
  message: string,
  cacheControl: string | null
): void {
  state.phase = 'invalid';
  state.lastCacheControl = cacheControl;
  state.error = message;
  state.recoveryTokenId = null;
  state.recoverySessionId = null;
  state.recoverySession = null;
  state.expiresAt = null;
}

/** recovery register 成功後に transient state を片付ける。 */
function clearRecoveryState(state: RecoveryFlowState): void {
  state.phase = 'idle';
  state.requestId = null;
  state.noticeId = null;
  state.recoveryTokenId = null;
  state.recoverySessionId = null;
  state.recoverySession = null;
  state.expiresAt = null;
  state.lastCacheControl = null;
  state.error = null;
  state.sentView = createGenericRecoverySentView();
}

export {
  applyInvalidRecoveryToken,
  applyRecoveryAccepted,
  applyRecoveryReady,
  clearRecoveryState,
  createGenericRecoverySentView,
  createRecoveryFlowInitialState,
};
