import { error, fail, redirect } from '@sveltejs/kit';

import { getAdminBootstrapConfig } from '$lib/server/infrastructure/config/env.js';
import { getPlatformConfig } from '$lib/server/infrastructure/config/platform.js';
import { getAdminPrisma } from '$lib/server/infrastructure/db/prisma.js';
import * as operatorModel from '$lib/server/models/operators.js';

import type { Actions, ServerLoad } from '@sveltejs/kit';

const SETUP_ERROR = '初回セットアップを完了できませんでした。';
const SESSION_COOKIE_MAX_AGE_SECONDS = 86400;

function parseJsonFormField(
  value: FormDataEntryValue | null
): { ok: true; value: unknown } | { ok: false } {
  // WebAuthn attestation JSON の破損は入力エラーとして扱い、action を 500 にしない。
  if (typeof value !== 'string') return { ok: false };
  try {
    return { ok: true, value: JSON.parse(value) as unknown };
  } catch {
    return { ok: false };
  }
}

function stringFormField(value: FormDataEntryValue | null): string {
  // File 値を暗黙 string 化せず、文字列入力だけを JSON BFF へ転送する。
  return typeof value === 'string' ? value : '';
}

function copySessionCookie(response: Response, event: Parameters<Actions['finish']>[0]): void {
  // BFF の session cookie を action response に設定し、form action 経由でもログイン済みにする。
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
 * 初回 Admin セットアップページの server load。
 *
 * admin.operators が 0 件の環境だけ bootstrap UI を表示し、既に初期化済みなら login へ戻す。
 */
export const load: ServerLoad = async () => {
  // bootstrap gate が無効または期限切れの場合は、DB 状態に依存せず secret 入力フォーム自体を表示しない。
  const { adminBootstrapEnabled, adminBootstrapExpiresAt } = getAdminBootstrapConfig();
  if (!adminBootstrapEnabled || adminBootstrapExpiresAt.getTime() <= Date.now()) {
    error(403, 'Admin bootstrap is not available');
  }

  // 初期 operator が存在する環境で bootstrap 画面を露出しないため、DB 件数を唯一の判定にする。
  if ((await operatorModel.countOperators(getAdminPrisma())) !== 0) {
    redirect(303, '/login');
  }

  // 画面には秘密値を返さず、bootstrap 可能である事実だけを返す。
  return { bootstrapAllowed: true };
};

/**
 * 初回 Admin セットアップの form actions。
 *
 * start は bootstrap secret と operator 情報から registration challenge を作り、finish は attestation を検証する。
 */
export const actions = {
  start: async (event) => {
    // FormData を JSON BFF の schema に合わせて正規化し、未検証値を直接 DB に渡さない。
    const form = await event.request.formData();
    const response = await event.fetch('/api/admin/auth/setup/start', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', Origin: event.url.origin },
      body: JSON.stringify({
        email: stringFormField(form.get('email')),
        displayName: stringFormField(form.get('displayName')),
        bootstrapSecret: stringFormField(form.get('bootstrapSecret')),
      }),
    });
    if (!response.ok)
      return fail(response.status, { error: '初回セットアップを開始できませんでした。' });
    return (await response.json()) as Record<string, unknown>;
  },
  finish: async (event) => {
    // WebAuthn attestation は既存 BFF に委譲し、page action には検証処理を重複実装しない。
    const form = await event.request.formData();
    const challengeId = stringFormField(form.get('challengeId'));
    const attestation = parseJsonFormField(form.get('attestation'));
    if (!attestation.ok) return fail(400, { error: SETUP_ERROR });
    const response = await event.fetch('/api/admin/auth/setup/finish', {
      method: 'POST',
      redirect: 'manual',
      headers: { 'Content-Type': 'application/json', Origin: event.url.origin },
      body: JSON.stringify({ challengeId, attestation: attestation.value }),
    });
    if (response.status !== 303) return fail(response.status, { error: SETUP_ERROR });
    copySessionCookie(response, event);
    redirect(303, '/');
  },
} satisfies Actions;
