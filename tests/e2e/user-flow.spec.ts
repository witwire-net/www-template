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

  test('公開トップで status card を操作できる', async ({ page }) => {
    await expect(
      page.getByRole('heading', { name: '公開面は SSR、データ更新は domain 経由です。' })
    ).toBeVisible();
    await expect(page.getByRole('button', { name: '公開 API を再取得' })).toBeVisible();
  });

  test('recovery route を表示できる', async ({ page }) => {
    await page.goto('http://localhost:5174/login/recovery');

    await expect(page.getByRole('heading', { name: 'パスキー復旧' })).toBeVisible();
    await expect(page.getByLabel('メールアドレス')).toBeVisible();
    await expect(page.getByRole('button', { name: '復旧メールを送信' })).toBeVisible();
  });
});
