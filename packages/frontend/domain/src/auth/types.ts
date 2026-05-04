/** 認証導線が選択する route intent。 */
export type AuthRouteIntent = '/login' | '/session-expired';

/** auth failure の安定分類。 */
export type AuthFailureState = 'unauthenticated' | 'session-expired' | 'internal-error';

/** in-memory bearer session の最小表現。 */
export interface AuthSessionSummary {
  requestId: string;
  accountId: string;
  passkeyCredentialId: string;
  sessionId: string;
  accessToken: string;
  expiresAt: string;
  /** セッション継続に使用するリフレッシュトークン。 */
  refreshToken?: string;
}

/** 共有 auth session state。 */
export interface AuthSessionState {
  phase: 'anonymous' | 'authenticating' | 'authenticated' | 'session-expired' | 'logging-out';
  session: AuthSessionSummary | null;
  /** メモリ上の複数セッションリスト。マルチアカウント対応。 */
  sessions?: AuthSessionSummary[];
  /** 現在アクティブなセッションの sessionId。 */
  activeSessionId?: string | null;
  routeIntent: AuthRouteIntent;
  lastFailure: AuthFailureState | null;
  lastError: string | null;
  lastCacheControl: string | null;
}

/** passkey login hook state。 */
export interface PasskeyLoginState {
  identifier: string;
  isSubmitting: boolean;
  lastChallengeRequestId: string | null;
  lastSession: AuthSessionSummary | null;
  lastCacheControl: string | null;
  error: string | null;
}

/** recovery sent view の共通 copy。 */
export interface RecoverySentView {
  title: string;
  description: string;
  helper: string;
}

/** recovery flow state。 */
export interface RecoveryFlowState {
  email: string;
  phase: 'idle' | 'submitting' | 'sent' | 'consuming' | 'ready' | 'invalid' | 'registering';
  requestId: string | null;
  noticeId: string | null;
  recoveryTokenId: string | null;
  recoverySessionId: string | null;
  recoverySession: string | null;
  expiresAt: string | null;
  lastCacheControl: string | null;
  error: string | null;
  sentView: RecoverySentView;
}

/** 登録済みパスキーの表示用モデル。 */
export interface PasskeyItem {
  id: string;
  identifier: string;
  createdAt: string;
}

/** passkey management hook state。 */
export interface PasskeyManagementState {
  passkeys: PasskeyItem[];
  loading: boolean;
  error: string | null;
  reauthSession: string | null;
}

/** passkey add-by-OTP hook state。 */
export interface PasskeyAddByOtpState {
  loading: boolean;
  error: string | null;
  done: boolean;
}
