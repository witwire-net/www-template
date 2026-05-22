import Redis from 'ioredis';

import { verifyOperatorSession } from '$lib/server/infrastructure/auth/operator';
import { getAdminAuthConfig } from '$lib/server/infrastructure/config/env';
import {
  validateCsrf,
  requireSameOrigin,
  issueCsrfToken,
} from '$lib/server/infrastructure/csrf/guard';
import { getAdminPrisma } from '$lib/server/infrastructure/db/prisma';
import { findOperatorById } from '$lib/server/models/operators';

import type { Handle, RequestEvent } from '@sveltejs/kit';
import type { Redis as RedisClient } from 'ioredis';

const SESSION_COOKIE_NAME = 'admin_session';
const PRE_AUTH_EXACT_ROUTES = new Set(['/login', '/setup', '/operator-setup']);
const PRE_AUTH_API_PREFIXES = [
  '/api/admin/auth/passkey/',
  '/api/admin/auth/setup/',
  '/api/admin/auth/operator-setup/',
];
const SAFE_METHODS = new Set(['GET', 'HEAD', 'OPTIONS']);

let adminValkey: RedisClient | null = null;

function getAdminValkey(): RedisClient {
  // 共有 Valkey infrastructure の Admin 用 DB 番号を指す URL だけを使い、Product DB 番号への誤接続は env 検証で防ぐ。
  const { adminValkeyUrl } = getAdminAuthConfig();
  // request ごとの接続生成を避け、同一プロセス内では Admin Valkey client を再利用する。
  adminValkey ??= new Redis(adminValkeyUrl, { lazyConnect: true, maxRetriesPerRequest: 1 });
  return adminValkey;
}

function isPreAuthRoute(pathname: string): boolean {
  // 画面ルートは完全一致で許可し、`/login/extra` のような想定外 route は protected として扱う。
  if (PRE_AUTH_EXACT_ROUTES.has(pathname)) {
    return true;
  }
  // 認証前 API は singular `passkey` と setup 系だけを prefix 許可し、plural `passkeys` は除外する。
  return PRE_AUTH_API_PREFIXES.some((prefix) => pathname.startsWith(prefix));
}

function isRouteLevelProtectedPasskeysApi(pathname: string): boolean {
  // passkey 管理 API は route 側で権限を見ても、hook 段階で未認証を 401 に固定する。
  return (
    pathname === '/api/admin/auth/passkeys' || pathname.startsWith('/api/admin/auth/passkeys/')
  );
}

function isAdminBffRoute(pathname: string): boolean {
  // Admin package-local BFF は `/api/admin/*` に限定されるため、API 応答判定に使う。
  return pathname === '/api/admin' || pathname.startsWith('/api/admin/');
}

function isSafeMethod(method: string): boolean {
  // GET/HEAD/OPTIONS は状態変更を行わない前提のため、session-bound CSRF の対象外にする。
  return SAFE_METHODS.has(method.toUpperCase());
}

async function loadOperatorFromCookie(event: RequestEvent): Promise<App.Locals['operator']> {
  // cookie がない場合は未認証として扱い、DB/Valkey への不要なアクセスを行わない。
  const token = event.cookies.get(SESSION_COOKIE_NAME);
  if (token === undefined || token === '') {
    return null;
  }

  // JWT と Admin Valkey の active session（sessionId/jti binding）を infrastructure helper で検証する。
  const session = await verifyOperatorSession(token, getAdminValkey());
  if (session === null) {
    return null;
  }

  // JWT/Valkey 内の role を認可に使わず、Admin DB の現在値を必ず読み直す。
  const operator = await findOperatorById(getAdminPrisma(), session.operatorId);
  if (operator?.isActive !== true) {
    return null;
  }

  // 後続 load/action/BFF が DB current role と operator locale だけを参照できるよう locals に最小情報を設定する。
  return {
    id: operator.id,
    email: operator.email,
    role: operator.role,
    locale: operator.locale,
    sessionId: session.sessionId,
    jti: session.jti,
  };
}

function noStoreResponse(
  body: string | null,
  status: number,
  extraHeaders: HeadersInit = {}
): Response {
  // 早期終了する redirect / error 応答にも通常応答と同じ no-store を必ず付与する。
  const headers = new Headers(extraHeaders);
  headers.set('Cache-Control', 'no-store');
  return new Response(body, { status, headers });
}

function noStoreRedirect(location: string): Response {
  // SvelteKit の redirect helper は hook 内で header 後付けできないため、明示的な 303 応答を返す。
  return noStoreResponse(null, 303, { Location: location });
}

function clearInvalidSessionCookie(event: RequestEvent): void {
  // 失効・改ざん・無効化 operator の cookie をブラウザから即時削除し、同じ無効 cookie の再送を防ぐ。
  event.cookies.delete(SESSION_COOKIE_NAME, { path: '/' });
}

function loginRedirectLocation(event: RequestEvent): string {
  // deep link から未認証になった場合に戻り先を保存し、認証後に元の Admin 画面へ戻せるようにする。
  const pathAndQuery = `${event.url.pathname}${event.url.search}`;
  if (pathAndQuery === '/') return '/login';
  return `/login?redirectTo=${encodeURIComponent(pathAndQuery)}`;
}

function rejectUnauthenticated(event: RequestEvent): Response {
  // BFF と passkey 管理 API は HTML redirect ではなく API として 401 を返す。
  if (isAdminBffRoute(event.url.pathname)) {
    return noStoreResponse('Admin authentication required', 401);
  }
  // Admin 画面の protected route は login へ誘導し、未認証ユーザーの迷子を防ぐ。
  return noStoreRedirect(loginRedirectLocation(event));
}

async function csrfFailureResponse(validate: () => void | Promise<void>): Promise<Response | null> {
  // infrastructure helper が拒否した CSRF/Origin エラーを no-store 付き 403 応答に正規化する。
  try {
    await validate();
    return null;
  } catch {
    return noStoreResponse('Forbidden', 403);
  }
}

/**
 * Admin サーバーサイドフック。
 *
 * すべての Admin リクエストで session cookie、Admin Valkey active session、Admin DB の現在オペレーターを検証し、
 * 認証済み context を `event.locals.operator` に設定する。未認証の protected route は画面なら `/login`、
 * Admin BFF なら 401 にし、cookie 認証済みの非 GET 系 request には sessionId/jti-bound CSRF を要求する。
 *
 * @param input SvelteKit から渡される request event と resolve 関数
 * @returns Cache-Control: no-store と必要な CSRF cookie を付与した response
 */
export const handle: Handle = async ({ event, resolve }) => {
  // request ごとの認証状態を必ず初期化し、前 request の状態混入を防ぐ。
  event.locals.operator = null;
  const hadSessionCookie = event.cookies.get(SESSION_COOKIE_NAME) !== undefined;

  // cookie が存在する場合だけ、JWT → Admin Valkey → Admin DB の順に fail-close で検証する。
  event.locals.operator = await loadOperatorFromCookie(event);
  if (hadSessionCookie && event.locals.operator === null) {
    // 検証できなかった cookie は安全側で破棄し、期限切れ・改ざん・非 active 化を同じ処理に集約する。
    clearInvalidSessionCookie(event);
  }

  // 認証済みユーザーが login に戻った場合は console root に戻し、不要な再ログインを避ける。
  if (event.locals.operator !== null && event.url.pathname === '/login') {
    return noStoreRedirect('/');
  }

  // 認証前 API の非 safe method は session-bound CSRF ではなく Origin allowlist のみを要求する。
  if (
    event.locals.operator === null &&
    isPreAuthRoute(event.url.pathname) &&
    !isSafeMethod(event.request.method)
  ) {
    const originFailure = await csrfFailureResponse(() => {
      requireSameOrigin(event);
    });
    if (originFailure !== null) {
      return originFailure;
    }
  }

  // passkeys 管理 API は pre-auth route に含めず、未認証を route 実装前に必ず 401 へ固定する。
  if (event.locals.operator === null && isRouteLevelProtectedPasskeysApi(event.url.pathname)) {
    return noStoreResponse('Admin passkey management requires authentication', 401);
  }

  // protected route の未認証アクセスを止め、画面 route は login redirect、BFF は 401 に分岐する。
  if (event.locals.operator === null && !isPreAuthRoute(event.url.pathname)) {
    return rejectUnauthenticated(event);
  }

  // cookie 認証済みの状態変更 request では Origin と sessionId/jti-bound CSRF token を検証する。
  if (event.locals.operator !== null && !isSafeMethod(event.request.method)) {
    const csrfFailure = await csrfFailureResponse(() => validateCsrf(event));
    if (csrfFailure !== null) {
      return csrfFailure;
    }
  }

  // route/page/load/BFF の通常処理を実行し、その後で共通 security header を付与する。
  const response = await resolve(event);
  // Admin 画面・load・BFF の機微情報を browser/proxy に保存させない。
  response.headers.set('Cache-Control', 'no-store');

  // 認証済み GET 応答で sessionId/jti に紐づく double-submit CSRF cookie を配布する。
  if (event.locals.operator !== null && isSafeMethod(event.request.method)) {
    const { cookieValue } = issueCsrfToken(
      event.locals.operator.sessionId,
      event.locals.operator.jti
    );
    response.headers.append('Set-Cookie', cookieValue);
  }

  return response;
};
