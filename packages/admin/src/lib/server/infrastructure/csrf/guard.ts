import { createHmac } from 'node:crypto';

import { error as skError } from '@sveltejs/kit';

import { getEnvConfig } from '../config/env.js';
import { getPlatformConfig } from '../config/platform.js';

import type { RequestEvent } from '@sveltejs/kit';

const CSRF_COOKIE_NAME = 'admin_csrf';
const CSRF_HEADER_NAME = 'x-csrf-token';

/**
 * sessionId と jti に紐づいた CSRF token を発行する。
 *
 * @param sessionId セッション識別子
 * @param jti JWT ID
 * @returns token と cookie 文字列
 */
export function issueCsrfToken(
  sessionId: string,
  jti: string
): { token: string; cookieValue: string } {
  const { jwtSecret } = getEnvConfig();
  const { isProduction } = getPlatformConfig();
  const message = `${sessionId}:${jti}`;
  const token = createHmac('sha256', jwtSecret).update(message).digest('hex');

  const cookieValue = [
    `${CSRF_COOKIE_NAME}=${token}`,
    'Path=/',
    'SameSite=Lax',
    isProduction ? 'Secure' : '',
  ]
    .filter(Boolean)
    .join('; ');

  return { token, cookieValue };
}

/**
 * 認証済みリクエストの CSRF token を検証する。
 * Origin ヘッダーと、X-CSRF-Token ヘッダーまたは form hidden token、admin_session cookie の整合性を確認する。
 *
 * @param event SvelteKit リクエストイベント
 * @throws 403 いずれかの検証に失敗した場合
 */
export async function validateCsrf(event: RequestEvent): Promise<void> {
  const { adminOrigin } = getEnvConfig();

  // Origin 検証
  const origin = event.request.headers.get('origin');
  if (origin === null || origin === '' || origin !== adminOrigin) {
    return skError(403, 'CSRF origin mismatch');
  }

  // Token 検証
  const csrfToken = await readRequestCsrfToken(event.request);
  const csrfCookie = event.cookies.get(CSRF_COOKIE_NAME);
  if (csrfToken === null || csrfToken === '' || csrfCookie === undefined || csrfCookie === '') {
    return skError(403, 'CSRF token missing');
  }
  if (csrfToken !== csrfCookie) {
    return skError(403, 'CSRF token mismatch');
  }

  // session-bound 検証
  const operator = event.locals.operator;
  if (operator === null) {
    return skError(403, 'CSRF session not found');
  }

  const { token: expectedToken } = issueCsrfToken(operator.sessionId, operator.jti);
  if (csrfToken !== expectedToken) {
    return skError(403, 'CSRF session binding mismatch');
  }
}

async function readRequestCsrfToken(request: Request): Promise<string | null> {
  // fetch/enhanced form は header、通常 HTML form は hidden field で同じ token を渡せるようにする。
  const headerToken = request.headers.get(CSRF_HEADER_NAME);
  if (headerToken !== null && headerToken !== '') return headerToken;
  const contentType = request.headers.get('content-type') ?? '';
  if (
    !contentType.includes('application/x-www-form-urlencoded') &&
    !contentType.includes('multipart/form-data')
  ) {
    return null;
  }
  const form = await request.clone().formData();
  const value = form.get('_csrf');
  return typeof value === 'string' && value !== '' ? value : null;
}

/**
 * 認証前ルート用の Origin 検証。
 * session-bound CSRF は要求しない。
 *
 * @param event SvelteKit リクエストイベント
 * @throws 403 Origin が一致しない場合
 */
export function requireSameOrigin(event: RequestEvent): void {
  const { adminOrigin } = getEnvConfig();
  const origin = event.request.headers.get('origin');
  if (origin === null || origin === '' || origin !== adminOrigin) {
    return skError(403, 'Origin mismatch');
  }
}
