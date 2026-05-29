/** 認証導線が選択する route intent。 */
export type AuthRouteIntent = '/login' | '/session-expired' | '/account-suspended';

/** auth failure の安定分類。 */
export type AuthFailureState =
  | 'unauthenticated'
  | 'session-expired'
  | 'account-suspended'
  | 'internal-error';

/** in-memory bearer session の最小表現。 */
export interface AuthSessionSummary {
  requestId: string;
  accountId: string;
  passkeyCredentialId: string;
  sessionId: string;
  accessToken: string;
  expiresAt: string;
  // refreshToken はブラウザー可読 state に保持しない。
  // refresh flow は same-origin Cookie ベースで行う。
}

/** refresh response から取得した AccountSetting snapshot。 */
export interface AccountSettingSnapshot {
  locale: 'ja' | 'en';
}

/** 共有 auth session state。 */
export interface AuthSessionState {
  phase:
    | 'anonymous'
    | 'authenticating'
    | 'authenticated'
    | 'session-expired'
    | 'account-suspended'
    | 'logging-out';
  session: AuthSessionSummary | null;
  /** メモリ上の複数セッションリスト。マルチアカウント対応。 */
  sessions?: AuthSessionSummary[];
  /** 現在アクティブなセッションの sessionId。 */
  activeSessionId?: string | null;
  routeIntent: AuthRouteIntent;
  lastFailure: AuthFailureState | null;
  lastError: string | null;
  lastCacheControl: string | null;
  /** 最後の refresh で取得した AccountSetting snapshot。 */
  lastAccountSettingSnapshot: AccountSettingSnapshot | null;
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
  /** consumeRecoveryToken レスポンスの kind。recovery or device-link。 */
  kind?: 'recovery' | 'device-link';
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
  /** デバイスリンク送信済みフラグ。再認証後に sendDeviceLink が成功した場合に true になる。 */
  deviceLinkSent: boolean;
}
