import { readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';

import { describe, expect, it, vi } from 'vitest';

const loginPageMocks = vi.hoisted(() => ({
  getPlatformConfig: vi.fn(() => ({ isProduction: false })),
}));

vi.mock('$lib/server/infrastructure/config/platform.js', () => ({
  getPlatformConfig: loginPageMocks.getPlatformConfig,
}));

import { actions } from './+page.server.js';

const loginPageSource = readFileSync(
  fileURLToPath(new URL('./+page.svelte', import.meta.url)),
  'utf8'
);

describe('login page passkey actions', () => {
  it('passkey finish が BFF cookie を外側 cookie に移して root redirect する', async () => {
    const cookiesSet = vi.fn();
    const event = createActionEvent({
      form: { challengeId: 'challenge-1', assertion: JSON.stringify({ id: 'cred-1' }) },
      fetchResponse: new Response(null, {
        status: 303,
        headers: { 'set-cookie': 'admin_session=jwt-token; Path=/; HttpOnly' },
      }),
      cookiesSet,
    });

    await expect(actions.finish(event as never)).rejects.toMatchObject({
      status: 303,
      location: '/',
    });
    expect(cookiesSet).toHaveBeenCalledWith(
      'admin_session',
      'jwt-token',
      expect.objectContaining({ httpOnly: true, path: '/', sameSite: 'lax' })
    );
  });

  it('unknown email の start 失敗は非列挙エラーに正規化する', async () => {
    const result = await actions.start(
      createActionEvent({
        form: { email: 'missing@example.test' },
        fetchResponse: new Response('nope', { status: 401 }),
      }) as never
    );

    expect(result).toMatchObject({
      status: 401,
      data: { error: expect.stringContaining('パスキー認証に失敗しました') },
    });
  });

  it('WebAuthn cancel 相当の空 assertion は cookie を設定せず非列挙エラーにする', async () => {
    const cookiesSet = vi.fn();
    const result = await actions.finish(
      createActionEvent({ form: { challengeId: 'challenge-1' }, cookiesSet }) as never
    );

    expect(result).toMatchObject({
      status: 400,
      data: { error: expect.stringContaining('パスキー認証に失敗しました') },
    });
    expect(cookiesSet).not.toHaveBeenCalled();
  });

  it('利用可能 credential なし相当の不正 JSON assertion は cookie を設定せず非列挙エラーにする', async () => {
    const cookiesSet = vi.fn();
    const result = await actions.finish(
      createActionEvent({
        form: { challengeId: 'challenge-1', assertion: '{' },
        cookiesSet,
      }) as never
    );

    expect(result).toMatchObject({
      status: 400,
      data: { error: expect.stringContaining('パスキー認証に失敗しました') },
    });
    expect(cookiesSet).not.toHaveBeenCalled();
  });

  it('ログイン中は loading 表示と二重送信防止が source contract として維持される', () => {
    // Admin Vitest は node 環境のため Svelte component を mount せず、isSubmitting に紐づく UI 契約を source 上で固定する。
    expect(loginPageSource).toContain('if (isSubmitting) return;');
    expect(loginPageSource).toContain('isSubmitting = true;');
    expect(loginPageSource).toContain("disabled={isSubmitting || email.trim() === ''}");
    expect(loginPageSource).toContain('<Spinner />');
    expect(loginPageSource).toContain('認証中…');
    expect(loginPageSource).toContain('isSubmitting = false;');
  });
});

function createActionEvent(input: {
  form: Record<string, string>;
  fetchResponse?: Response;
  cookiesSet?: ReturnType<typeof vi.fn>;
}) {
  // SvelteKit form action が参照する FormData / fetch / cookie API だけを持つ最小 event を作る。
  const form = new FormData();
  for (const [key, value] of Object.entries(input.form)) form.set(key, value);
  return {
    url: new URL('https://admin.example.test/login'),
    request: { formData: async () => form },
    fetch: vi.fn(async () => input.fetchResponse ?? new Response('{}', { status: 200 })),
    cookies: { set: input.cookiesSet ?? vi.fn() },
  };
}
