import {
  requestCreateAdminOperator,
  type AdminOperatorProfile,
  type AdminOperatorRole,
} from '@www-template/admin-api';

import { getAdminSession, verifyProtectedAdminRoute } from './auth';

/**
 * Admin operator を UI 表示へ渡すための read model です。
 *
 * role / active は表示制御にだけ使い、backend authorization の代替にはしません。
 */
export interface AdminOperatorPresentation {
  id: string;
  email: string;
  role: string;
  active: boolean;
  canCreateAccounts: boolean;
}

/**
 * Admin operator 作成画面から渡す入力です。
 *
 * email は作成対象 operator の連絡先兼識別子で、role は backend RBAC が受け付ける role 値です。
 */
export interface AdminOperatorCreateInput {
  email: string;
  role: AdminOperatorRole;
}

/**
 * Admin operator 作成後に UI へ返す read model です。
 *
 * setup token 平文は backend delivery に閉じるため、この型には含めません。
 */
export interface AdminOperatorCreateResult {
  id: string;
  email: string;
  role: AdminOperatorRole;
  active: boolean;
  deliveryStatus: string;
}

/**
 * Admin operator domain function の失敗分類です。
 *
 * UI はこの分類だけを表示へ写像し、token 配送や RBAC の詳細理由を露出しません。
 */
export type AdminOperatorDomainError =
  | 'unauthenticated'
  | 'forbidden'
  | 'invalid-input'
  | 'duplicate-email'
  | 'unavailable'
  | 'unknown';

/**
 * Admin operator domain function の Result 型です。
 */
export type AdminOperatorResult<T> =
  | { success: true; data: T }
  | { success: false; error: AdminOperatorDomainError };

/**
 * 現在の Admin operator 表示 state を返します。
 *
 * @returns current operator の表示用 read model。未認証または拒否時は null。
 */
export async function getCurrentOperatorPresentation(): Promise<AdminOperatorPresentation | null> {
  // protected route state を先に検証し、古い memory session を表示に使わない。
  const routeState = await verifyProtectedAdminRoute();
  if (routeState.status !== 'authenticated') return null;

  // backend が検証した operator profile だけを UI 表示用に変換する。
  return toOperatorPresentation(routeState.session.operator);
}

/**
 * memory session から現在の operator 表示 state を同期的に読みます。
 *
 * @returns memory session がある場合は表示用 read model、ない場合は null。
 */
export function getCachedOperatorPresentation(): AdminOperatorPresentation | null {
  // layout の初期描画では network 前の cached state だけを使い、storage から復元しない。
  const session = getAdminSession();
  if (session === null) return null;

  // role は UI 表示に限定し、mutation の可否は Admin backend の RBAC が最終判断する。
  return toOperatorPresentation(session.operator);
}

/**
 * Admin operator を作成し、setup token delivery を backend に委譲します。
 *
 * @param input 作成対象 operator の email と role。
 * @returns 作成済み operator の表示用 read model、または秘匿的な error 分類。
 */
export async function createAdminOperator(
  input: AdminOperatorCreateInput
): Promise<AdminOperatorResult<AdminOperatorCreateResult>> {
  // UI の空白だけを取り除き、email 正規化・role/RBAC・delivery は Go Admin API に委譲する。
  const email = input.email.trim();
  if (email === '' || !email.includes('@')) return { success: false, error: 'invalid-input' };

  // mutation は memory にある Admin accessToken だけで実行し、refresh token は Cookie 境界に残す。
  const session = getAdminSession();
  if (session === null) return { success: false, error: 'unauthenticated' };

  // API wrapper が same-origin path / Authorization を保証し、page から generated SDK を直接呼ばせない。
  const response = await requestCreateAdminOperator({ email, role: input.role }, session);
  if (response.status !== 201) return { success: false, error: mapOperatorStatus(response.status) };

  // setup token 平文は response に存在しないため、operator summary と配送状態だけを UI state に返す。
  return {
    success: true,
    data: {
      id: response.data.operator.operatorId,
      email: response.data.operator.email,
      role: response.data.operator.role,
      active: response.data.operator.active,
      deliveryStatus: response.data.deliveryStatus,
    },
  };
}

function toOperatorPresentation(operator: AdminOperatorProfile): AdminOperatorPresentation {
  // generated DTO の operatorId を UI component の `id` へ写像する。
  return {
    id: operator.operatorId,
    email: operator.email,
    role: operator.role,
    active: operator.active,
    canCreateAccounts:
      operator.active && (operator.role === 'admin' || operator.role === 'operator'),
  };
}

function mapOperatorStatus(status: number): AdminOperatorDomainError {
  // HTTP status を UI 向け分類に落とし込み、setup token 配送失敗などの内部 reason は表示しない。
  if (status === 400) return 'invalid-input';
  if (status === 401) return 'unauthenticated';
  if (status === 403) return 'forbidden';
  if (status === 409) return 'duplicate-email';
  if (status === 503) return 'unavailable';
  return 'unknown';
}
