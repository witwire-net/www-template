import {
  type AdminOperatorAuthSessionResponse,
  type AdminOperatorContextRefreshResponse,
  type AdminOperatorProfile,
  type WWWTemplateContextIndexUpdateHint,
  requestCurrentAdminOperator,
  requestFinishInitialAdminSetup,
  requestFinishAdminLogin,
  requestFinishOperatorSetup,
  requestLogoutAdminOperator,
  requestRefreshAdminSession,
  requestStartInitialAdminSetup,
  requestStartAdminLogin,
  requestStartOperatorSetup,
  type WWWTemplatePasskeyAddStartResponse,
  type WWWTemplatePasskeyStartResponse,
  type WWWTemplateWebAuthnAssertionCredential,
  type WWWTemplateWebAuthnAttestationCredential,
} from '@www-template/admin-api';

import {
  clearAdminContextIndex,
  createEmptyAdminContextIndex,
  readAdminContextIndex,
  removeAdminContextEntry,
  upsertAdminContextEntry,
  writeAdminContextIndex,
} from './context_index';

import type { AdminContextIndexEntry } from './context_index';

type AdminSessionResponseLike =
  | AdminOperatorAuthSessionResponse
  | AdminOperatorContextRefreshResponse;

/**
 * Admin frontend domain が保持する browser-readable session state です。
 *
 * - accessToken は Admin backend から返る短命値だけを memory に保持します。
 * - refreshToken は HttpOnly Cookie 専用のため、この型には存在しません。
 * - operator は UI 表示に使いますが、最終 authorization は常に Go Admin API が行います。
 */
export interface AdminSessionState {
  operator: AdminOperatorProfile;
  sessionId: string;
  authContextId: string;
  accessToken: string;
  expiresAt: string;
}

/**
 * Admin protected route の検証結果です。
 *
 * authenticated は current operator 検証済み、unauthenticated は login 誘導、forbidden は active/role 不整合などの拒否を表します。
 */
export type AdminProtectedRouteState =
  | { status: 'authenticated'; session: AdminSessionState }
  | { status: 'unauthenticated' }
  | { status: 'forbidden' };

/**
 * Admin passkey login ceremony の開始結果です。
 *
 * requestId は finish request と対応させ、options は browser WebAuthn API へ渡す public challenge だけを含みます。
 */
export interface AdminLoginStartResult {
  requestId: string;
  options: WWWTemplatePasskeyStartResponse;
}

/**
 * Admin operator setup ceremony の開始結果です。
 *
 * setup token の平文は caller が保持する form 入力だけに残し、domain state には保存しません。
 */
export interface AdminOperatorSetupStartResult {
  requestId: string;
  options: WWWTemplatePasskeyAddStartResponse;
}

/**
 * 初回 Admin operator setup ceremony の開始入力です。
 *
 * email / displayName は作成する最初の admin operator の表示値で、bootstrapSecret は Admin backend の gate 検証だけに使います。
 */
export interface AdminInitialSetupStartInput {
  email: string;
  displayName: string;
  bootstrapSecret: string;
}

/**
 * 初回 Admin operator setup ceremony の開始結果です。
 *
 * `started` は finish request と対応する requestId と browser WebAuthn API 用の public challenge を保持します。
 * それ以外は setup form を閉じるための秘匿的な UI 分類で、backend error reason は保持しません。
 */
export type AdminInitialSetupStartResult =
  | { status: 'started'; requestId: string; options: WWWTemplatePasskeyAddStartResponse }
  | { status: 'invalid' | 'operator-exists' | 'bootstrap-disabled' | 'unavailable' };

let currentSession: AdminSessionState | null = null;

function toAdminContextIndexEntry(session: AdminSessionState): AdminContextIndexEntry {
  // context index は reload 後の refresh 対象発見だけに使うため、token/secret を含めない。
  return {
    authContextId: session.authContextId,
    operatorSessionId: session.sessionId,
    displayHint: session.operator.email,
    roleHint: session.operator.role,
    lastSeenAt: new Date().toISOString(),
    expiresHintAt: session.expiresAt,
  };
}

function persistAdminSessionContext(session: AdminSessionState): void {
  // login/setup/refresh 成功時は、server が返した session metadata だけを Admin 専用 index に反映する。
  const index = readAdminContextIndex() ?? createEmptyAdminContextIndex();
  upsertAdminContextEntry(index, toAdminContextIndexEntry(session), true);
  writeAdminContextIndex(index);
}

function removeAdminSessionContext(authContextId: string): void {
  // refresh failure / logout / inactive response では対象 context だけを index から削除する。
  const index = readAdminContextIndex() ?? createEmptyAdminContextIndex();
  removeAdminContextEntry(index, authContextId);
  writeAdminContextIndex(index);
}

function applyAdminContextIndexHints(hints: WWWTemplateContextIndexUpdateHint[]): void {
  // backend の logout/revoke hint を正として index を同期し、client 側の推測で Cookie 対象を決めない。
  const index = readAdminContextIndex() ?? createEmptyAdminContextIndex();
  for (const hint of hints) {
    if (hint.action === 'clear-surface') {
      clearAdminContextIndex();
      return;
    }
    if (hint.action === 'remove' && hint.authContextId !== undefined) {
      removeAdminContextEntry(index, hint.authContextId);
    }
    if (
      hint.action === 'upsert' &&
      hint.authContextId !== undefined &&
      hint.sessionId !== undefined &&
      hint.displayHint !== undefined &&
      hint.lastSeenAt !== undefined &&
      hint.expiresHintAt !== undefined
    ) {
      upsertAdminContextEntry(
        index,
        {
          authContextId: hint.authContextId,
          operatorSessionId: hint.sessionId,
          displayHint: hint.displayHint.label,
          roleHint: hint.displayHint.secondaryLabel ?? '',
          lastSeenAt: hint.lastSeenAt,
          expiresHintAt: hint.expiresHintAt,
        },
        true
      );
    }
  }
  writeAdminContextIndex(index);
}

/**
 * 現在 memory にある Admin session を読み取ります。
 *
 * @returns session がある場合は accessToken / operator を含む state、ない場合は null。
 */
export function getAdminSession(): AdminSessionState | null {
  // refreshToken を読める storage から復元しないことで、Cookie-only refresh の不変条件を守る。
  return currentSession;
}

/**
 * Admin session の browser-readable memory state を破棄します。
 *
 * @returns 何も返しません。副作用として module-local session state を null にします。
 */
export function clearAdminSession(): void {
  // logout 失敗時や protected route 拒否時にも、古い accessToken を UI 層へ残さない。
  currentSession = null;
}

/**
 * Admin passkey login ceremony を開始します。
 *
 * @param identifier operator email など、Admin backend が operator を識別する入力。
 * @returns requestId と WebAuthn assertion options。失敗時は null。
 */
export async function startAdminLogin(identifier: string): Promise<AdminLoginStartResult | null> {
  // UI の余分な空白だけを取り除き、operator の存在可否や正規化判断は backend に委譲する。
  const normalizedIdentifier = identifier.trim();
  if (normalizedIdentifier === '') return null;

  // package-local BFF ではなく Admin API wrapper 経由で start challenge を取得する。
  const response = await requestStartAdminLogin({ identifier: normalizedIdentifier });
  if (response.status !== 200) return null;

  // requestId と public challenge だけを返し、秘密値は domain state に保持しない。
  return { requestId: response.data.requestId, options: response.data };
}

/**
 * Admin passkey login ceremony を完了して session state を更新します。
 *
 * @param requestId start response と対応する requestId。
 * @param credential browser WebAuthn API が返した assertion credential。
 * @returns 認証済み session。失敗時は null。
 */
export async function finishAdminLogin(
  requestId: string,
  credential: WWWTemplateWebAuthnAssertionCredential
): Promise<AdminSessionState | null> {
  // finish request では cookie credential mode を明示し、Admin backend の Cookie session 発行へ固定する。
  const response = await requestFinishAdminLogin({
    requestId,
    credentialMode: 'cookie',
    credential,
  });
  if (response.status !== 200) {
    clearAdminSession();
    return null;
  }

  // Admin backend の session response から browser-readable 値だけを memory state に反映する。
  currentSession = toSessionState(response.data);
  if (currentSession === null) return null;
  persistAdminSessionContext(currentSession);
  return currentSession;
}

/**
 * Admin operator setup ceremony を開始します。
 *
 * @param setupToken operator が受け取った one-time setup token。
 * @returns requestId と WebAuthn registration options。失敗時は null。
 */
export async function startOperatorSetup(
  setupToken: string
): Promise<AdminOperatorSetupStartResult | null> {
  // setup token は form 入力から直接 API へ渡し、domain module へ永続保存しない。
  const normalizedSetupToken = setupToken.trim();
  if (normalizedSetupToken === '') return null;

  // Go Admin API が setup token の hash / expiry / consumed state を秘匿的に検証する。
  const response = await requestStartOperatorSetup({ setupToken: normalizedSetupToken });
  if (response.status !== 200) return null;

  // registration options は public challenge だけなので、browser WebAuthn API へ渡してよい。
  return { requestId: response.data.requestId, options: response.data };
}

/**
 * Admin operator setup ceremony を完了して session state を更新します。
 *
 * @param setupToken operator が入力した one-time setup token。
 * @param requestId start response と対応する requestId。
 * @param credential browser WebAuthn API が返した attestation credential。
 * @returns 認証済み session。失敗時は null。
 */
export async function finishOperatorSetup(
  setupToken: string,
  requestId: string,
  credential: WWWTemplateWebAuthnAttestationCredential
): Promise<AdminSessionState | null> {
  // finish request でも token 平文を memory session へ移さず、backend transaction の入力に限定する。
  const response = await requestFinishOperatorSetup({
    setupToken: setupToken.trim(),
    requestId,
    credentialMode: 'cookie',
    credential,
  });
  if (response.status !== 200) {
    clearAdminSession();
    return null;
  }

  // setup 成功時は login と同じ accessToken-only browser-readable state へ揃える。
  currentSession = toSessionState(response.data);
  if (currentSession === null) return null;
  persistAdminSessionContext(currentSession);
  return currentSession;
}

/**
 * 初回 Admin operator setup ceremony を開始します。
 *
 * @param input email / displayName / bootstrapSecret を含む初回 setup 入力。
 * @returns started の場合は requestId と WebAuthn registration options、失敗時は秘匿的な分類。
 */
export async function startInitialAdminSetup(
  input: AdminInitialSetupStartInput
): Promise<AdminInitialSetupStartResult> {
  // 画面入力の余白だけを削り、operator 件数や bootstrap secret の可否は backend に委譲する。
  const email = input.email.trim();
  const displayName = input.displayName.trim();
  const bootstrapSecret = input.bootstrapSecret.trim();
  if (email === '' || displayName === '' || bootstrapSecret === '') return { status: 'invalid' };

  // 初回 setup start は `/api/v1/auth/setup/start` だけを Admin API wrapper 経由で呼び出す。
  const response = await requestStartInitialAdminSetup({ email, displayName, bootstrapSecret });
  if (response.status !== 200) return mapInitialSetupStartStatus(response.status);

  // challenge は public data のみなので、requestId と一緒に UI の WebAuthn 呼び出しへ返す。
  return { status: 'started', requestId: response.data.requestId, options: response.data };
}

/**
 * 初回 Admin operator setup ceremony を完了して session state を更新します。
 *
 * @param input email / displayName / bootstrapSecret を含む初回 setup 入力。
 * @param requestId start response と対応する requestId。
 * @param credential browser WebAuthn API が返した attestation credential。
 * @returns 認証済み session。失敗時は null。
 */
export async function finishInitialAdminSetup(
  input: AdminInitialSetupStartInput,
  requestId: string,
  credential: WWWTemplateWebAuthnAttestationCredential
): Promise<AdminSessionState | null> {
  // bootstrap secret は finish API の入力に限定し、domain session state には決して保存しない。
  const response = await requestFinishInitialAdminSetup({
    email: input.email.trim(),
    displayName: input.displayName.trim(),
    bootstrapSecret: input.bootstrapSecret.trim(),
    requestId,
    credentialMode: 'cookie',
    credential,
  });
  if (response.status !== 200) {
    clearAdminSession();
    return null;
  }

  // login / operator setup と同じ accessToken-only browser-readable state に正規化する。
  currentSession = toSessionState(response.data);
  if (currentSession === null) return null;
  persistAdminSessionContext(currentSession);
  return currentSession;
}

/**
 * HttpOnly Cookie refresh を使って Admin session を更新します。
 *
 * @returns 更新後の session。refresh できない場合は null。
 */
export async function refreshAdminSession(): Promise<AdminSessionState | null> {
  // context-scoped refresh は Cookie Path と authContextId が一致する場合だけ有効なので、memory state が無い場合は安全側に login へ戻す。
  const index = readAdminContextIndex();
  const bootstrapEntry =
    currentSession === null && index !== null && index.activeAuthContextId !== null
      ? index.entries.find((entry) => entry.authContextId === index.activeAuthContextId)
      : undefined;
  const authContextId = currentSession?.authContextId ?? bootstrapEntry?.authContextId;
  if (authContextId === undefined || authContextId === '') {
    clearAdminSession();
    return null;
  }

  // refreshToken は JavaScript から読まず、same-origin Cookie として backend へだけ送る。
  const response = await requestRefreshAdminSession(authContextId);
  if (response.status !== 200) {
    removeAdminSessionContext(authContextId);
    clearAdminSession();
    return null;
  }

  // backend が再発行した accessToken だけを memory state に保存する。
  currentSession = toSessionState(response.data);
  if (currentSession?.authContextId !== authContextId) {
    removeAdminSessionContext(authContextId);
    clearAdminSession();
    return null;
  }
  persistAdminSessionContext(currentSession);
  return currentSession;
}

/**
 * protected route 表示前に current operator を検証します。
 *
 * @returns authenticated / unauthenticated / forbidden の route state。
 */
export async function verifyProtectedAdminRoute(): Promise<AdminProtectedRouteState> {
  // memory session が無い場合は、HttpOnly Cookie refresh で accessToken を再取得できるか確認する。
  const session = currentSession ?? (await refreshAdminSession());
  if (session === null) return { status: 'unauthenticated' };

  // current operator endpoint で accessToken が現在も有効かを Go Admin API に検証させる。
  const response = await requestCurrentAdminOperator(session);
  if (response.status === 200) {
    currentSession = { ...session, operator: response.data.operator };
    return { status: 'authenticated', session: currentSession };
  }

  // 401 は login 誘導、403 は権限/active state 拒否として扱い、どちらも詳細理由は UI に出さない。
  clearAdminSession();
  return response.status === 403 ? { status: 'forbidden' } : { status: 'unauthenticated' };
}

/**
 * Admin operator logout を実行し、browser-readable session state を破棄します。
 *
 * @returns logout API が成功した場合 true、session が無い場合や失敗時は false。
 */
export async function logoutAdminSession(): Promise<boolean> {
  // logout に必要な accessToken が無い場合は、local state だけを安全側に破棄する。
  const session = currentSession;
  if (session === null) {
    clearAdminSession();
    return false;
  }

  // Go Admin API に Cookie revoke と session revoke を委譲し、結果に関わらず local token は破棄する。
  const response = await requestLogoutAdminOperator(session);
  if (response.status === 200) {
    applyAdminContextIndexHints(response.data.contextIndexUpdateHints);
  } else {
    removeAdminSessionContext(session.authContextId);
  }
  clearAdminSession();
  return response.status === 200;
}

function toSessionState(response: AdminSessionResponseLike): AdminSessionState | null {
  // generated response から refreshToken を探さず、表示と bearer header に必要な値だけへ写像する。
  // Cookie mode response だけを Admin Console の browser session として採用し、automation Bearer response は browser state に混ぜない。
  if (response.credentialMode !== 'cookie') return null;
  return {
    operator: response.operator,
    sessionId: response.sessionId,
    authContextId: response.authContextId,
    accessToken: response.accessToken,
    expiresAt: response.expiresAt,
  };
}

function mapInitialSetupStartStatus(status: number): AdminInitialSetupStartResult {
  // HTTP status を UI 表示用の秘匿的分類へ落とし込み、backend の詳細 reason は捨てる。
  if (status === 400) return { status: 'invalid' };
  if (status === 409) return { status: 'operator-exists' };
  if (status === 403) return { status: 'bootstrap-disabled' };
  return { status: 'unavailable' };
}
