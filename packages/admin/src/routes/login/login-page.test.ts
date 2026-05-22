import { readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';

import { describe, expect, it, vi } from 'vitest';

const loginPageMocks = vi.hoisted(() => ({
  createAdminI18n: vi.fn(() => ({
    locale: 'ja',
    t: (key: string) =>
      key === 'login.error'
        ? 'パスキー認証に失敗しました。入力内容を確認してもう一度お試しください。'
        : key === 'login.title'
          ? 'Admin Login'
          : key,
  })),
}));

vi.mock('$lib/i18n', () => ({
  createAdminI18n: loginPageMocks.createAdminI18n,
}));

import { load } from './+page.server.js';

const loginPageSource = readFileSync(
  fileURLToPath(new URL('./+page.svelte', import.meta.url)),
  'utf8'
);

describe('login page', () => {
  it('ログイン中は loading 表示と二重送信防止が source contract として維持される', () => {
    expect(loginPageSource).toContain('if (isSubmitting) return;');
    expect(loginPageSource).toContain('isSubmitting = true;');
    expect(loginPageSource).toContain("disabled={isSubmitting || email.trim() === ''}");
    expect(loginPageSource).toContain('<Spinner aria-hidden="true" />');
    expect(loginPageSource).toContain('{data.labels.submitting}');
    expect(loginPageSource).toContain('isSubmitting = false;');
  });

  it('LOCALIZATION-FE-S009 Admin 認証前画面は operator DB を要求せず fallback translator を使う', () => {
    const loaded = load({} as never);
    expect(loaded).toMatchObject({
      labels: {
        title: 'Admin Login',
        error: expect.stringContaining('パスキー認証に失敗しました'),
      },
    });
    expect(loginPageSource).toContain('{data.labels.title}');
    expect(loginPageSource).toContain('message = data.labels.error');
  });
});
