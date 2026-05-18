import { fail, redirect } from '@sveltejs/kit';

import { getPlatformConfig } from '$lib/server/infrastructure/config/platform.js';

import type { Actions } from '@sveltejs/kit';

const NON_REVEALING_LOGIN_ERROR =
  'パスキー認証に失敗しました。入力内容を確認してもう一度お試しください。';
const SESSION_COOKIE_MAX_AGE_SECONDS = 86400;

function parseJsonFormField(
  value: FormDataEntryValue | null
): { ok: true; value: unknown } | { ok: false } {
  // WebAuthn 応答 JSON が壊れている場合も 500 にせず、利用者へ安全な 400 を返す。
  if (typeof value !== 'string') return { ok: false };
  try {
    return { ok: true, value: JSON.parse(value) as unknown };
  } catch {
    return { ok: false };
  }
}

function stringFormField(value: FormDataEntryValue | null): string {
  // File など想定外の FormData 値を文字列化せず、空入力として扱って schema 側の検証に渡す。
  return typeof value === 'string' ? value : '';
}

function copySessionCookie(response: Response, event: Parameters<Actions['finish']>[0]): void {
  // BFF finish が発行した Set-Cookie から token 値だけを取り出し、外側の action response cookie として設定する。
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

/**
 * Admin ログインページの form actions。
 *
 * start は email から WebAuthn challenge を開始し、finish はブラウザが返した assertion を既存 BFF に渡す。
 * どちらも account / operator の存在有無を推測できないよう、公開エラーは同じ文言へ正規化する。
 */
export const actions = {
  start: async (event) => {
    // ブラウザフォームから受け取った email だけを BFF へ渡し、認証 material は page action に保存しない。
    const form = await event.request.formData();
    const email = stringFormField(form.get('email'));
    const response = await event.fetch('/api/admin/auth/passkey/start', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', Origin: event.url.origin },
      body: JSON.stringify({ email }),
    });

    // start 失敗時も operator の存在有無が分からない固定文言で返す。
    if (!response.ok) {
      return fail(response.status, { error: NON_REVEALING_LOGIN_ERROR });
    }

    // WebAuthn options は JSON-safe な BFF 応答のみを返し、サーバー内部状態を露出しない。
    return (await response.json()) as Record<string, unknown>;
  },
  finish: async (event) => {
    // assertion は client-side WebAuthn から JSON 文字列として受け、BFF の検証処理に委譲する。
    const form = await event.request.formData();
    const challengeId = stringFormField(form.get('challengeId'));
    const assertion = parseJsonFormField(form.get('assertion'));
    if (!assertion.ok) return fail(400, { error: NON_REVEALING_LOGIN_ERROR });
    const response = await event.fetch('/api/admin/auth/passkey/finish', {
      method: 'POST',
      redirect: 'manual',
      headers: { 'Content-Type': 'application/json', Origin: event.url.origin },
      body: JSON.stringify({ challengeId, assertion: assertion.value }),
    });

    // finish 失敗時も unknown / inactive / invalid assertion を同じ文言へ集約する。
    if (response.status !== 303) {
      return fail(response.status, { error: NON_REVEALING_LOGIN_ERROR });
    }

    // progressive enhancement / form action 利用時も session cookie が成立するよう、内側 BFF の cookie を外側へ移す。
    copySessionCookie(response, event);
    redirect(303, '/');
  },
} satisfies Actions;
