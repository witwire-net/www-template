import {
  requestAdminAccount,
  requestAdminAccounts,
  requestCreateAdminAccount,
  type AdminAccountSummary,
  type WWWTemplateAccountLocale,
} from '@www-template/admin-api';

import { getAdminSession } from './auth';

/**
 * Admin account 一覧で UI に渡す read model です。
 *
 * Product Account の永続値を Admin API の read model から変換したもので、Admin frontend 独自の lifecycle state は持ちません。
 */
export interface AdminAccountListItem {
  id: string;
  email: string;
  status: string;
  createdAt: string;
  passkeyCount: number;
}

/**
 * Admin account 一覧取得の入力です。
 *
 * email は部分一致検索用、cursor は backend が返す opaque cursor、limit は 1 page の取得件数です。
 */
export interface AdminAccountSearchInput {
  email?: string;
  cursor?: string;
  limit?: number;
}

/**
 * Admin account 一覧取得の成功結果です。
 *
 * nextCursor は backend が返したときだけ次ページ UI に使います。
 */
export interface AdminAccountSearchResult {
  accounts: AdminAccountListItem[];
  nextCursor: string | null;
}

/**
 * Admin account 作成の入力です。
 *
 * email は customer account の email、locale は Product AccountSetting の初期 locale です。
 */
export interface AdminAccountCreateInput {
  email: string;
  locale?: WWWTemplateAccountLocale;
}

/**
 * Admin account domain function の失敗分類です。
 *
 * UI はこの分類だけを表示へ写像し、backend の詳細 error や auth failure reason を露出しません。
 */
export type AdminAccountDomainError =
  | 'unauthenticated'
  | 'forbidden'
  | 'invalid-input'
  | 'duplicate-email'
  | 'not-found'
  | 'unavailable'
  | 'unknown';

/**
 * Admin account domain function の Result 型です。
 *
 * success true の場合だけ data を読み、false の場合は error を UI 文言へ変換します。
 */
export type AdminAccountResult<T> =
  | { success: true; data: T }
  | { success: false; error: AdminAccountDomainError };

/**
 * Admin account 一覧を検索します。
 *
 * @param input email / cursor / limit の検索条件。
 * @returns account list と nextCursor、または秘匿的な error 分類。
 */
export async function searchAdminAccounts(
  input: AdminAccountSearchInput
): Promise<AdminAccountResult<AdminAccountSearchResult>> {
  // protected route 検証後の memory session だけを使い、refreshToken や Product SDK には触れない。
  const session = getAdminSession();
  if (session === null) return { success: false, error: 'unauthenticated' };

  // 空文字 query は backend へ送らず、無条件一覧として扱う。
  const email = input.email?.trim();
  const response = await requestAdminAccounts(
    {
      email: email === undefined || email === '' ? undefined : email,
      cursor: input.cursor,
      limit: input.limit,
    },
    session
  );

  // status code を domain error に畳み、transport 詳細を route component へ漏らさない。
  if (response.status !== 200) return { success: false, error: mapAccountStatus(response.status) };

  // generated DTO を Admin frontend の表示用 read model へ変換する。
  return {
    success: true,
    data: {
      accounts: response.data.accounts.map(toAccountListItem),
      nextCursor: response.data.nextCursor ?? null,
    },
  };
}

/**
 * Admin account 詳細を取得します。
 *
 * @param accountId 取得対象 Product Account の canonical ULID。
 * @returns account detail read model、または秘匿的な error 分類。
 */
export async function getAdminAccountDetail(
  accountId: string
): Promise<AdminAccountResult<AdminAccountListItem>> {
  // route parameter が空の場合は backend に到達せず、UI 入力の問題として扱う。
  const normalizedAccountId = accountId.trim();
  if (normalizedAccountId === '') return { success: false, error: 'invalid-input' };

  // current Admin session が無い場合は protected route から再ログインへ誘導できる error を返す。
  const session = getAdminSession();
  if (session === null) return { success: false, error: 'unauthenticated' };

  // Admin API wrapper 経由で same-origin `/api/v1/accounts/{id}` だけを呼ぶ。
  const response = await requestAdminAccount(normalizedAccountId, session);
  if (response.status !== 200) return { success: false, error: mapAccountStatus(response.status) };

  // detail response の account summary を UI 共通 read model へ変換する。
  return { success: true, data: toAccountListItem(response.data.account) };
}

/**
 * Admin operator として customer account を作成します。
 *
 * @param input email と任意 locale。
 * @returns 作成された account read model、または秘匿的な error 分類。
 */
export async function createCustomerAccount(
  input: AdminAccountCreateInput
): Promise<AdminAccountResult<AdminAccountListItem>> {
  // 明らかな空入力だけを UI 側で止め、正規化・重複・不変条件は Go backend domain object に委譲する。
  const email = input.email.trim();
  if (email === '' || !email.includes('@')) return { success: false, error: 'invalid-input' };

  // mutation は Admin accessToken と CSRF token が揃った memory session だけを使う。
  const session = getAdminSession();
  if (session === null) return { success: false, error: 'unauthenticated' };

  // Admin API wrapper が Authorization と X-CSRF-Token を付与し、backend が RBAC と CSRF binding を検証する。
  const response = await requestCreateAdminAccount({ email, locale: input.locale }, session);
  if (response.status !== 201) return { success: false, error: mapAccountStatus(response.status) };

  // 作成結果から auditEventId は UI に保持せず、作成済み account だけを次画面遷移に使う。
  return { success: true, data: toAccountListItem(response.data.account) };
}

function toAccountListItem(account: AdminAccountSummary): AdminAccountListItem {
  // generated DTO の `accountId` を UI component 既存 props の `id` に写像する。
  return {
    id: account.accountId,
    email: account.email,
    status: account.status,
    createdAt: account.createdAt,
    passkeyCount: account.passkeyCount,
  };
}

function mapAccountStatus(status: number): AdminAccountDomainError {
  // HTTP status を UI 向け分類に落とし込み、backend の error body 詳細を露出しない。
  if (status === 400) return 'invalid-input';
  if (status === 401) return 'unauthenticated';
  if (status === 403) return 'forbidden';
  if (status === 404) return 'not-found';
  if (status === 409) return 'duplicate-email';
  if (status === 503) return 'unavailable';
  return 'unknown';
}
