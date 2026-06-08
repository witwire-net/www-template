import { render, screen } from '@testing-library/svelte';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import LanguagePage from '../../routes/(protected)/settings/general/language/+page.svelte';

// domain hooks のモック
vi.mock('@www-template/domain', () => ({
  useAccount: () => ({
    data: { state: { account: null, loading: false, error: null } },
    actions: {
      updateLocale: vi.fn().mockResolvedValue(true),
    },
  }),
}));

vi.mock('@www-template/domain/auth/session', () => ({
  useAuthSession: () => ({
    actions: {
      createAuthorizationHeaders: () => ({ Authorization: 'Bearer test' }),
    },
  }),
}));

describe('[LOCALIZATION-FE-S005] 設定画面の表示言語ページ', () => {
  beforeEach(() => {
    localStorage.setItem('www-template:locale', 'ja');
  });

  afterEach(() => {
    localStorage.clear();
  });

  it('表示言語ラベルと Select が表示される', async () => {
    render(LanguagePage);

    // 表示言語ラベルが表示されていることを確認
    expect(screen.getByText('表示言語')).toBeInTheDocument();

    // Select トリガーが存在することを確認
    const trigger = screen.getByRole('button', { name: '表示言語' });
    expect(trigger).toBeInTheDocument();
  });
});
