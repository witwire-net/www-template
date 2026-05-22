import { getPlatformConfig } from '$lib/server/infrastructure/config/platform.js';

import type { RequestEvent } from '@sveltejs/kit';

/** 管理者 session cookie の有効期間（秒）。24 時間。 */
export const SESSION_COOKIE_MAX_AGE_SECONDS = 86400;

/**
 * BFF finish route が発行した Set-Cookie を、SvelteKit action の response cookie として設定する。
 *
 * progressive enhancement / form action 利用時も session cookie が成立するよう、
 * 内側 BFF の cookie を外側 action response へ移す。
 *
 * @param response BFF finish route の fetch 応答
 * @param event SvelteKit の RequestEvent
 */
export function copySessionCookie(response: Response, event: RequestEvent): void {
  const rawCookie = response.headers.get('set-cookie');
  const token = rawCookie?.match(/^admin_session=([^;]+)/)?.[1];
  if (token === undefined || token === '') return;
  const { isProduction } = getPlatformConfig();
  event.cookies.set('admin_session', token, {
    httpOnly: true,
    maxAge: SESSION_COOKIE_MAX_AGE_SECONDS,
    path: '/',
    sameSite: 'lax',
    secure: isProduction,
  });
}
