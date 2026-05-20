import { render, screen } from '@testing-library/svelte';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import SettingsPage from '../../routes/(protected)/settings/+page.svelte';

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

describe('[LOCALIZATION-FE-S005] 設定画面の fallback locale 表示', () => {
  beforeEach(() => {
    localStorage.setItem('www-template:locale', 'ja');
  });

  afterEach(() => {
    localStorage.clear();
  });

  it('translator 読み込み前に日本語文字列が表示される', async () => {
    render(SettingsPage);

    // fallback 文字列が表示されていることを確認
    expect(screen.getByText('設定')).toBeInTheDocument();
    expect(screen.getByText('表示言語')).toBeInTheDocument();
    expect(screen.getByText('アプリの表示言語を選択します。')).toBeInTheDocument();
  });
});
