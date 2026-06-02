import {
  createAdminAccount,
  createAdminOperator,
  finishAdminInitialSetup,
  finishAdminOperatorSetup,
  finishAdminPasskeyAuthentication,
  getAdminAccount,
  getCreateAdminAccountUrl,
  getCreateAdminOperatorUrl,
  getFinishAdminInitialSetupUrl,
  getCurrentAdminOperator,
  getDeleteAdminOperatorPasskeyUrl,
  getFinishAdminOperatorSetupUrl,
  getFinishAdminPasskeyAuthenticationUrl,
  getGetAdminAccountUrl,
  getGetCurrentAdminOperatorUrl,
  getListAdminAccountsUrl,
  getListAdminOperatorPasskeysUrl,
  getLogoutAdminOperatorUrl,
  getRefreshAdminOperatorSessionUrl,
  getStartAdminInitialSetupUrl,
  getStartAdminOperatorSetupUrl,
  getStartAdminPasskeyAuthenticationUrl,
  listAdminAccounts,
  listAdminOperatorPasskeys,
  logoutAdminOperator,
  refreshAdminOperatorSession,
  startAdminInitialSetup,
  startAdminOperatorSetup,
  startAdminPasskeyAuthentication,
} from './generated/client';

import type {
  AdminCreateOperatorRequest,
  AdminInitialSetupFinishRequest,
  AdminInitialSetupStartRequest,
  AdminOperatorSetupFinishRequest,
  AdminOperatorSetupStartRequest,
  AdminPasskeyFinishRequest,
  AdminPasskeyStartRequest,
  ListAdminAccountsParams,
  WWWTemplateBearerContextRefreshRequest,
  WWWTemplateCreateAccountRequest,
} from './generated/client';

const adminApiPrefix = '/api/v1/';
const forbiddenAdminBffPrefix = `/${['api', 'admin'].join('/')}/`;

/**
 * Admin API wrapper が付与する session header の入力です。
 *
 * - `accessToken`: Admin operator auth domain が発行した短命 bearer token です。
 * refreshToken は HttpOnly Cookie 専用であり、この型にも response body にも保持しません。
 */
export interface AdminApiSessionHeaders {
  accessToken?: string | null;
}

/**
 * Admin API wrapper が各 request へ渡す安全な RequestInit です。
 *
 * すべて same-origin Cookie を利用できるよう `credentials: 'same-origin'` を固定し、
 * Admin operator session がある場合だけ Authorization header を付与します。
 */
export type AdminApiRequestOptions = AdminApiSessionHeaders;

/**
 * Admin automation client が Bearer mode refresh へ渡す入力です。
 *
 * @property authContextId refresh 対象を path で選択する Admin auth context ID。
 * @property refreshToken automation client が自身の secret store で管理する opaque refresh token。
 */
export interface AdminAutomationRefreshInput {
  authContextId: string;
  refreshToken: string;
}

const adminAutomationBearerRequestInit = {
  credentials: 'omit',
} as const satisfies RequestInit;

/**
 * Admin generated SDK の URL が同一 origin の `/api/v1/*` だけを指すことを検証します。
 *
 * @param path generated SDK の `get*Url()` から得た request path。
 * @returns 検証済みの Admin API path。
 * @throws Product origin や旧 Admin BFF へ逸脱する absolute URL / `/api/v1/*` 外の path を拒否します。
 *
 * @example
 * ```ts
 * assertAdminApiPath('/api/v1/accounts');
 * ```
 */
export function assertAdminApiPath(path: string): string {
  // absolute URL や protocol-relative URL を拒否し、Product host へ送る逃げ道を wrapper 境界で閉じる。
  if (/^[a-z][a-z\d+.-]*:\/\//iu.test(path) || path.startsWith('//')) {
    throw new Error('admin-api-absolute-url-forbidden');
  }

  // package-local BFF の旧 prefix を明示的に拒否し、Go Admin API だけを呼び出す。
  if (path.startsWith(forbiddenAdminBffPrefix)) {
    throw new Error('admin-api-bff-path-forbidden');
  }

  // Admin / Product とも path は `/api/v1/*` だが、この wrapper は Admin SDK 生成 path だけを許可する。
  if (!path.startsWith(adminApiPrefix)) {
    throw new Error('admin-api-path-out-of-scope');
  }

  // 検証済み path を返し、呼び出し元が evidence として path policy を確認できるようにする。
  return path;
}

/**
 * Admin generated SDK へ渡す RequestInit を生成します。
 *
 * @param options Admin access token を含む request option。
 * @returns `credentials: 'same-origin'` と必要な header を含む RequestInit。
 */
export function createAdminRequestInit(options: AdminApiRequestOptions = {}): RequestInit {
  // Orval 生成関数は `options.headers` を object spread するため、Headers ではなく plain object で保持する。
  const headers: Record<string, string> = {};

  // accessToken はブラウザー可読 memory state だけから header に写し、refreshToken は扱わない。
  if (
    options.accessToken !== undefined &&
    options.accessToken !== null &&
    options.accessToken !== ''
  ) {
    headers.Authorization = `Bearer ${options.accessToken}`;
  }

  // generated SDK の各関数に共通する safe default として same-origin credential を固定する。
  return { credentials: 'same-origin', headers };
}

/**
 * Admin account 一覧を Admin generated SDK 経由で取得します。
 *
 * @param params email / cursor / limit の検索条件。
 * @param session Admin operator accessToken を含む session header。
 * @returns generated SDK の account list response。
 */
export async function requestAdminAccounts(
  params: ListAdminAccountsParams,
  session: AdminApiSessionHeaders
) {
  // generated URL の path policy を実行時にも確認し、Admin API wrapper の責務を明示する。
  assertAdminApiPath(getListAdminAccountsUrl(params));

  // read route は Authorization と Cookie だけで current operator session を検証する。
  return listAdminAccounts(params, createAdminRequestInit(session));
}

/**
 * Admin account 作成 request を送信します。
 *
 * @param body 作成対象 email と任意 locale。
 * @param session Admin accessToken を含む browser-readable session header。
 * @returns generated SDK の account create response。
 */
export async function requestCreateAdminAccount(
  body: WWWTemplateCreateAccountRequest,
  session: AdminApiSessionHeaders
) {
  // account mutation は `/api/v1/accounts` に限定し、旧 BFF route を通らないことを確認する。
  assertAdminApiPath(getCreateAdminAccountUrl());

  // mutation route でも refreshToken は送らず、Authorization と same-origin Cookie だけを wrapper で固定する。
  return createAdminAccount(body, createAdminRequestInit(session));
}

/**
 * Admin operator 作成 request を送信します。
 *
 * @param body 作成対象 operator の email と role。
 * @param session Admin accessToken を含む browser-readable session header。
 * @returns generated SDK の operator create response。
 */
export async function requestCreateAdminOperator(
  body: AdminCreateOperatorRequest,
  session: AdminApiSessionHeaders
) {
  // operator mutation は Admin auth namespace の `/api/v1/auth/operators` に限定し、旧 BFF route を経由しない。
  assertAdminApiPath(getCreateAdminOperatorUrl());

  // mutation route なので Authorization を必ず wrapper で付与し、page から generated SDK を直接呼ばせない。
  return createAdminOperator(body, createAdminRequestInit(session));
}

/**
 * Admin account 詳細を取得します。
 *
 * @param accountId Admin API が扱う Product Account の canonical ULID。
 * @param session Admin operator accessToken を含む session header。
 * @returns generated SDK の account detail response。
 */
export async function requestAdminAccount(accountId: string, session: AdminApiSessionHeaders) {
  // detail URL も same-origin `/api/v1/*` から逸脱しないことを wrapper で確認する。
  assertAdminApiPath(getGetAdminAccountUrl(accountId));

  // read route は bearer session validation だけを要求する。
  return getAdminAccount(accountId, createAdminRequestInit(session));
}

/**
 * Admin passkey login ceremony を開始します。
 *
 * @param body operator identifier を含む start request。
 * @returns generated SDK の passkey start response。
 */
export async function requestStartAdminLogin(body: AdminPasskeyStartRequest) {
  // login start は session 前の public auth route だが、Admin origin の `/api/v1/*` だけを許可する。
  assertAdminApiPath(getStartAdminPasskeyAuthenticationUrl());

  // Origin 検証は Go Admin API が行うため、wrapper は Cookie 同梱設定だけを固定する。
  return startAdminPasskeyAuthentication(body, createAdminRequestInit());
}

/**
 * Admin passkey login ceremony を完了します。
 *
 * @param body requestId と browser WebAuthn assertion credential。
 * @returns generated SDK の operator session response。
 */
export async function requestFinishAdminLogin(body: AdminPasskeyFinishRequest) {
  // finish route も BFF を経由せず、Go Admin API の session 発行へ直接委譲する。
  assertAdminApiPath(getFinishAdminPasskeyAuthenticationUrl());

  // response body には refreshToken が含まれず、generated SDK response をそのまま domain へ返す。
  return finishAdminPasskeyAuthentication(body, createAdminRequestInit());
}

/**
 * 現在の Admin operator を取得します。
 *
 * @param session Admin operator accessToken を含む session header。
 * @returns generated SDK の current operator response。
 */
export async function requestCurrentAdminOperator(session: AdminApiSessionHeaders) {
  // protected current route が `/api/v1/*` から逸脱しないことを確認する。
  assertAdminApiPath(getGetCurrentAdminOperatorUrl());

  // current は read route なので session validation へ委譲する。
  return getCurrentAdminOperator(createAdminRequestInit(session));
}

/**
 * HttpOnly refresh Cookie を使って Admin operator session を更新します。
 *
 * @param authContextId refresh Cookie の Path と対応する Admin auth context ID。
 * @returns generated SDK の refreshed operator session response。
 */
export async function requestRefreshAdminSession(authContextId: string) {
  // refresh route は accessToken なしで Cookie rotation を行う Admin auth context route に限定する。
  assertAdminApiPath(getRefreshAdminOperatorSessionUrl(authContextId));

  // refreshToken は Cookie に閉じるため、wrapper は request body を送らず context ID と same-origin Cookie だけを渡す。
  return refreshAdminOperatorSession(authContextId, undefined, createAdminRequestInit());
}

/**
 * Admin automation client の Bearer mode refresh を実行します。
 *
 * @param input authContextId と automation client が保管する refreshToken。
 * @returns generated SDK の Bearer/Cookie union response。automation caller は Bearer response を扱います。
 */
export async function requestRefreshAdminAutomationSession(input: AdminAutomationRefreshInput) {
  // automation flow は browser Cookie を使わないため、Console wrapper と分離して credentials を omit に固定する。
  assertAdminApiPath(getRefreshAdminOperatorSessionUrl(input.authContextId));

  // refreshToken は automation client の body credential としてだけ送信し、Authorization header には変換しない。
  const body: WWWTemplateBearerContextRefreshRequest = {
    credentialMode: 'bearer',
    refreshToken: input.refreshToken,
  };
  return refreshAdminOperatorSession(input.authContextId, body, adminAutomationBearerRequestInit);
}

/**
 * 初回 Admin operator setup ceremony を開始します。
 *
 * @param body email / displayName / bootstrapSecret を含む初回 setup 入力。
 * @returns generated SDK の初回 setup start response。
 */
export async function requestStartInitialAdminSetup(body: AdminInitialSetupStartRequest) {
  // 初回 setup も package-local BFF ではなく、同一 origin の Go Admin API へ限定する。
  assertAdminApiPath(getStartAdminInitialSetupUrl());

  // bootstrap secret は backend 検証にだけ渡し、wrapper state や storage には保存しない。
  return startAdminInitialSetup(body, createAdminRequestInit());
}

/**
 * 初回 Admin operator setup ceremony を完了します。
 *
 * @param body email / displayName / bootstrapSecret / requestId / attestation credential。
 * @returns generated SDK の operator session response。
 */
export async function requestFinishInitialAdminSetup(body: AdminInitialSetupFinishRequest) {
  // finish route でも `/api/v1/auth/setup/finish` から逸脱しないことを検証する。
  assertAdminApiPath(getFinishAdminInitialSetupUrl());

  // session response には refreshToken 平文を含めず、HttpOnly Cookie 管理を backend に委譲する。
  return finishAdminInitialSetup(body, createAdminRequestInit());
}

/**
 * 現在の Admin operator session を logout します。
 *
 * @param session Admin accessToken を含む browser-readable session header。
 * @returns generated SDK の logout response。
 */
export async function requestLogoutAdminOperator(session: AdminApiSessionHeaders) {
  // logout は session mutation なので `/api/v1/auth/operator/logout` と Authorization を固定する。
  assertAdminApiPath(getLogoutAdminOperatorUrl());

  // backend に refresh Cookie revoke と accessToken/session revoke を委譲する。
  return logoutAdminOperator(createAdminRequestInit(session));
}

/**
 * Admin operator setup ceremony を開始します。
 *
 * @param body setupToken を含む start request。
 * @returns generated SDK の registration options response。
 */
export async function requestStartOperatorSetup(body: AdminOperatorSetupStartRequest) {
  // setup token を package-local BFF に渡さず、Go Admin API で hash / expiry 検証させる。
  assertAdminApiPath(getStartAdminOperatorSetupUrl());

  // start route では bearer session を持たないため、same-origin credentials だけを固定する。
  return startAdminOperatorSetup(body, createAdminRequestInit());
}

/**
 * Admin operator setup ceremony を完了します。
 *
 * @param body setupToken / requestId / attestation credential。
 * @returns generated SDK の operator session response。
 */
export async function requestFinishOperatorSetup(body: AdminOperatorSetupFinishRequest) {
  // finish route が token 消費と passkey 保存を Go backend transaction へ委譲する境界を確認する。
  assertAdminApiPath(getFinishAdminOperatorSetupUrl());

  // response body に setup token や refreshToken を戻さず、session response だけを domain へ返す。
  return finishAdminOperatorSetup(body, createAdminRequestInit());
}

/**
 * 現在の Admin operator passkey 一覧を取得します。
 *
 * @param session Admin accessToken を含む browser-readable session header。
 * @returns generated SDK の passkey list response。
 */
export async function requestAdminOperatorPasskeys(session: AdminApiSessionHeaders) {
  // passkey 一覧は認証手段列挙なので、Authorization 付き same-origin request に限定する。
  assertAdminApiPath(getListAdminOperatorPasskeysUrl());

  // backend middleware の policy に合わせ、passkey route も wrapper 経由の session header だけを同梱する。
  return listAdminOperatorPasskeys(createAdminRequestInit(session));
}

/**
 * Admin operator passkey 削除 path を検証します。
 *
 * @param passkeyId 削除対象 passkey の canonical ULID。
 * @returns 検証済みの generated delete path。
 */
export function getVerifiedDeleteAdminOperatorPasskeyPath(passkeyId: string): string {
  // 削除の実装は後続 task だが、path policy だけを共有し future BFF 回帰を防ぐ。
  return assertAdminApiPath(getDeleteAdminOperatorPasskeyUrl(passkeyId));
}
