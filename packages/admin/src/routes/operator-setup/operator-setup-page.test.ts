import { describe, expect, it, vi } from 'vitest';

const operatorSetupPageMocks = vi.hoisted(() => ({
  getPlatformConfig: vi.fn(() => ({ isProduction: false })),
}));

vi.mock('$lib/server/infrastructure/config/platform.js', () => ({
  getPlatformConfig: operatorSetupPageMocks.getPlatformConfig,
}));

import { actions, load } from './+page.server.js';

describe('operator setup page server contract', () => {
  it('setup token は start/finish action で追加 operator の passkey 登録を完了する', async () => {
    const startResult = await actions.start(
      createActionEvent({
        form: { setupToken: 'one-time-token' },
        fetchResponse: new Response(
          JSON.stringify({ challengeId: 'challenge-1', options: { challenge: 'public' } }),
          { status: 200 }
        ),
      }) as never
    );
    expect(startResult).toEqual({ challengeId: 'challenge-1', options: { challenge: 'public' } });

    const cookiesSet = vi.fn();
    await expect(
      actions.finish(
        createActionEvent({
          form: { challengeId: 'challenge-1', attestation: JSON.stringify({ id: 'credential-1' }) },
          fetchResponse: new Response(null, {
            status: 303,
            headers: { 'set-cookie': 'admin_session=jwt-token; Path=/; HttpOnly' },
          }),
          cookiesSet,
        }) as never
      )
    ).rejects.toMatchObject({ status: 303, location: '/' });
    expect(cookiesSet).toHaveBeenCalledWith(
      'admin_session',
      'jwt-token',
      expect.objectContaining({ httpOnly: true, path: '/', sameSite: 'lax' })
    );
  });

  it('不正な setup token は non-revealing な固定エラーへ正規化する', async () => {
    const result = await actions.start(
      createActionEvent({
        form: { setupToken: 'bad-token' },
        fetchResponse: new Response('nope', { status: 401 }),
      }) as never
    );

    expect(result).toMatchObject({
      status: 401,
      data: { error: expect.stringContaining('セットアップトークンを確認できませんでした') },
    });
  });

  it('登録済み session を持つ operator は setup token 画面へ入れず Dashboard に戻る', async () => {
    await expect(
      Promise.resolve().then(() => load({ locals: { operator: authedOperator() } } as never))
    ).rejects.toMatchObject({
      status: 303,
      location: '/',
    });
    await expect(Promise.resolve(load({ locals: { operator: null } } as never))).resolves.toEqual(
      {}
    );
  });
});

function createActionEvent(input: {
  form: Record<string, string>;
  fetchResponse: Response;
  cookiesSet?: ReturnType<typeof vi.fn>;
}) {
  // operator setup action が参照する form/fetch/cookie API だけを持つ小さな event fixture を返す。
  const form = new FormData();
  for (const [key, value] of Object.entries(input.form)) form.set(key, value);
  return {
    url: new URL('https://admin.example.test/operator-setup'),
    request: { formData: async () => form },
    fetch: vi.fn(async () => input.fetchResponse),
    cookies: { set: input.cookiesSet ?? vi.fn() },
  };
}

function authedOperator(): NonNullable<App.Locals['operator']> {
  // load guard が見る認証済み operator locals の最小 shape を返す。
  return {
    id: 'op-1',
    email: 'admin@example.test',
    role: 'operator',
    locale: 'ja',
    sessionId: 'sess-1',
    jti: 'jti-1',
  };
}
