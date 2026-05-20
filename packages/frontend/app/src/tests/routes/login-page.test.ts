import { render, screen } from '@testing-library/svelte';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import LoginPage from '../../routes/login/+page.svelte';

// $app/navigation の goto をモック
vi.mock('$app/navigation', () => ({
  goto: vi.fn(),
}));

describe('[LOCALIZATION-FE-S006] ログイン画面の fallback locale 表示', () => {
  beforeEach(() => {
    localStorage.setItem('www-template:locale', 'ja');
  });

  afterEach(() => {
    localStorage.clear();
  });

  it('translator 読み込み前に日本語文字列が表示される', async () => {
    render(LoginPage);

    // fallback 文字列が表示されていることを確認
    expect(screen.getByText('ログイン')).toBeInTheDocument();
    expect(screen.getByText('パスキーを使ってサインインしてください。')).toBeInTheDocument();
    expect(screen.getByText('パスキーでログイン')).toBeInTheDocument();
    expect(screen.getByText('パスキーを紛失した場合')).toBeInTheDocument();
  });
});
