import { expect, test } from '@playwright/test';

test.describe('www-template auth surface', () => {
  test.skip(
    ({ browserName }) => browserName !== 'chromium',
    'surface smoke is validated in Chromium for stability'
  );

  test.beforeEach(async ({ page }) => {
    await page.goto('/');
  });

  test('公開トップから auth app へ移動できる', async ({ page }) => {
    await expect(
      page.getByRole('heading', { name: '公開面と認証面を再利用しやすい構成でまとめています。' })
    ).toBeVisible();

    await page.getByRole('link', { name: 'ログインを試す' }).click();
    await expect(page).toHaveURL(/localhost:5174\/login$/);
    await expect(page.getByRole('button', { name: 'パスキーでログイン' })).toBeVisible();
  });

  test('公開トップで主要メッセージと CTA を確認できる', async ({ page }) => {
    await expect(
      page.getByRole('heading', { name: '公開面と認証面を再利用しやすい構成でまとめています。' })
    ).toBeVisible();
    await expect(page.getByText('public route は SSR の SvelteKit として運用')).toBeVisible();
    await expect(page.getByRole('link', { name: 'ログインを試す' })).toBeVisible();
  });

  test('recovery route を表示できる', async ({ page }) => {
    await page.goto('http://localhost:5174/login/recovery');

    await expect(page.getByRole('heading', { name: 'パスキー復旧' })).toBeVisible();
    await expect(page.getByLabel('メールアドレス')).toBeVisible();
    await expect(page.getByRole('button', { name: '復旧メールを送信' })).toBeVisible();
  });
});
