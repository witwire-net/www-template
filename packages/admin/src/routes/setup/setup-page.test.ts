import { beforeEach, describe, expect, it, vi } from 'vitest';

const setupPageMocks = vi.hoisted(() => ({
  countOperators: vi.fn(),
  getAdminPrisma: vi.fn(),
  getEnvConfig: vi.fn(),
  getPlatformConfig: vi.fn(() => ({ isProduction: false })),
}));

vi.mock('$lib/server/models/operators.js', () => ({
  countOperators: setupPageMocks.countOperators,
}));
vi.mock('$lib/server/infrastructure/db/prisma.js', () => ({
  getAdminPrisma: setupPageMocks.getAdminPrisma,
}));
vi.mock('$lib/server/infrastructure/config/env.js', () => ({
  getEnvConfig: setupPageMocks.getEnvConfig,
}));
vi.mock('$lib/server/infrastructure/config/platform.js', () => ({
  getPlatformConfig: setupPageMocks.getPlatformConfig,
}));

import { actions, load } from './+page.server.js';

describe('initial setup page server contract', () => {
  beforeEach(() => {
    // 各 scenario が operator count の呼び出し有無を独立して検証できるよう、mock 履歴を初期化する。
    vi.clearAllMocks();
  });

  it('初回 setup は start/finish action で passkey 登録後に session cookie を設定して root redirect する', async () => {
    setupPageMocks.countOperators.mockResolvedValue(0);
    setupPageMocks.getEnvConfig.mockReturnValue(enabledBootstrapGate());
    const startResponse = new Response(
      JSON.stringify({ challengeId: 'challenge-1', options: { challenge: 'public' } }),
      { status: 200 }
    );
    const startResult = await actions.start(
      createActionEvent({ form: setupForm(), fetchResponse: startResponse }) as never
    );
    expect(startResult).toEqual({ challengeId: 'challenge-1', options: { challenge: 'public' } });

    const cookiesSet = vi.fn();
    const finishEvent = createActionEvent({
      form: { challengeId: 'challenge-1', attestation: JSON.stringify({ id: 'credential-1' }) },
      fetchResponse: new Response(null, {
        status: 303,
        headers: { 'set-cookie': 'admin_session=jwt-token; Path=/; HttpOnly' },
      }),
      cookiesSet,
    });
    await expect(actions.finish(finishEvent as never)).rejects.toMatchObject({
      status: 303,
      location: '/',
    });
    expect(cookiesSet).toHaveBeenCalledWith(
      'admin_session',
      'jwt-token',
      expect.objectContaining({ httpOnly: true, path: '/', sameSite: 'lax' })
    );
  });

  it('operator が存在する場合は初回 setup フォームを表示せず login へ戻す', async () => {
    setupPageMocks.countOperators.mockResolvedValue(1);
    setupPageMocks.getEnvConfig.mockReturnValue(enabledBootstrapGate());

    await expect(load({} as never)).rejects.toMatchObject({ status: 303, location: '/login' });
  });

  it('bootstrap gate が無効または期限切れの場合は初回 setup フォームを表示しない', async () => {
    setupPageMocks.countOperators.mockResolvedValue(0);
    setupPageMocks.getEnvConfig.mockReturnValueOnce({
      adminBootstrapEnabled: false,
      adminBootstrapExpiresAt: new Date('2999-01-01T00:00:00.000Z'),
    });
    await expect(load({} as never)).rejects.toMatchObject({ status: 403 });
    expect(setupPageMocks.countOperators).not.toHaveBeenCalled();

    setupPageMocks.getEnvConfig.mockReturnValueOnce({
      adminBootstrapEnabled: true,
      adminBootstrapExpiresAt: new Date(0),
    });
    await expect(load({} as never)).rejects.toMatchObject({ status: 403 });
    expect(setupPageMocks.countOperators).not.toHaveBeenCalled();
  });
});

function createActionEvent(input: {
  form: Record<string, string>;
  fetchResponse: Response;
  cookiesSet?: ReturnType<typeof vi.fn>;
}) {
  // SvelteKit form action が利用する最小限の form/fetch/cookie API だけを持つ event fixture を作る。
  const form = new FormData();
  for (const [key, value] of Object.entries(input.form)) form.set(key, value);
  return {
    url: new URL('https://admin.example.test/setup'),
    request: { formData: async () => form },
    fetch: vi.fn(async () => input.fetchResponse),
    cookies: { set: input.cookiesSet ?? vi.fn() },
  };
}

function setupForm(): Record<string, string> {
  // bootstrap start action の正常入力を共通 fixture 化し、secret をテスト外へ漏らさない。
  return { email: 'admin@example.test', displayName: 'Admin', bootstrapSecret: 'bootstrap-secret' };
}

function enabledBootstrapGate(): { adminBootstrapEnabled: boolean; adminBootstrapExpiresAt: Date } {
  // Date.now に依存しない十分未来の期限で bootstrap gate 有効状態を表す。
  return {
    adminBootstrapEnabled: true,
    adminBootstrapExpiresAt: new Date('2999-01-01T00:00:00.000Z'),
  };
}
