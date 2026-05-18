import { fail, redirect } from '@sveltejs/kit';

import { getPlatformConfig } from '$lib/server/infrastructure/config/platform.js';

import type { Actions, ServerLoad } from '@sveltejs/kit';

const OPERATOR_SETUP_ERROR =
  'セットアップトークンを確認できませんでした。期限と入力内容を確認してください。';
const SESSION_COOKIE_MAX_AGE_SECONDS = 86400;

/**
 * 追加 operator セットアップページの server load。
 *
 * 既に Admin session を持つ operator は setup token 登録を使う必要がないため、
 * Dashboard に戻して one-time token 入力 UI を表示しない。
 */
export const load: ServerLoad = (event) => {
  // hooks.server.ts が検証した現在 session を唯一の判定材料にし、JWT claim を再解釈しない。
  if (event.locals.operator !== null) {
    redirect(303, '/');
  }

  // 未認証ユーザーには token 入力フォームを表示し、BFF 側で token の真偽を検証する。
  return {};
};

function parseJsonFormField(
  value: FormDataEntryValue | null
): { ok: true; value: unknown } | { ok: false } {
  // WebAuthn attestation JSON の破損を明示的な 400 にし、未捕捉例外を避ける。
  if (typeof value !== 'string') return { ok: false };
  try {
    return { ok: true, value: JSON.parse(value) as unknown };
  } catch {
    return { ok: false };
  }
}

function stringFormField(value: FormDataEntryValue | null): string {
  // FormData の File 値を暗黙 string 化せず、期待する文字列だけを BFF へ渡す。
  return typeof value === 'string' ? value : '';
}

function copySessionCookie(response: Response, event: Parameters<Actions['finish']>[0]): void {
  // BFF が返した Set-Cookie を action の cookie API へ移し、追加 operator setup action 単体でも完結させる。
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
 * 追加 Admin operator セットアップページの form actions。
 *
 * one-time setup token から登録対象 operator を特定し、WebAuthn passkey 登録を既存 BFF に委譲する。
 */
export const actions = {
  start: async (event) => {
    // token は bcrypt 検証されるため、平文を永続化せず BFF start route にだけ渡す。
    const form = await event.request.formData();
    const setupToken = stringFormField(form.get('setupToken'));
    const response = await event.fetch('/api/admin/auth/operator-setup/start', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', Origin: event.url.origin },
      body: JSON.stringify({ setupToken }),
    });
    if (!response.ok) return fail(response.status, { error: OPERATOR_SETUP_ERROR });
    return (await response.json()) as Record<string, unknown>;
  },
  finish: async (event) => {
    // challengeId と attestation の組だけを finish route へ渡し、token 再利用は BFF transaction で拒否する。
    const form = await event.request.formData();
    const challengeId = stringFormField(form.get('challengeId'));
    const attestation = parseJsonFormField(form.get('attestation'));
    if (!attestation.ok) return fail(400, { error: OPERATOR_SETUP_ERROR });
    const response = await event.fetch('/api/admin/auth/operator-setup/finish', {
      method: 'POST',
      redirect: 'manual',
      headers: { 'Content-Type': 'application/json', Origin: event.url.origin },
      body: JSON.stringify({ challengeId, attestation: attestation.value }),
    });
    if (response.status !== 303) return fail(response.status, { error: OPERATOR_SETUP_ERROR });
    copySessionCookie(response, event);
    redirect(303, '/');
  },
} satisfies Actions;
