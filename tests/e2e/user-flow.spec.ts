import { test, expect, type Page } from '@playwright/test';

const waitForTableOrEmptyState = async (page: Page) => {
  const tableLocator = page.locator('table');
  const emptyMessageLocator = page.getByText(/no users found/i);

  await Promise.any([
    tableLocator.waitFor({ state: 'visible', timeout: 10000 }),
    emptyMessageLocator.waitFor({ state: 'visible', timeout: 10000 }),
  ]).catch((error: unknown) => {
    throw new Error('Unable to find users table or empty state within the expected time.', {
      cause: error,
    });
  });
};

test.describe('ユーザー管理フロー', () => {
  test.beforeEach(async ({ page }) => {
    // ユーザーページに移動
    await page.goto('/');
  });

  test('ページが正しく表示される', async ({ page }) => {
    // ページタイトルの確認
    await expect(page.getByRole('heading', { name: 'Users' })).toBeVisible();

    // フォームの確認
    await expect(page.getByPlaceholder('Name')).toBeVisible();
    await expect(page.getByPlaceholder('Email')).toBeVisible();
    await expect(page.getByRole('button', { name: /create user/i })).toBeVisible();
  });

  test('ユーザー一覧が表示される', async ({ page }) => {
    // ユーザー一覧が表示されるまで待つ
    await waitForTableOrEmptyState(page);

    // テーブルまたは空メッセージのいずれかが表示されることを確認
    const hasTable = await page
      .locator('table')
      .isVisible()
      .catch(() => false);
    const hasEmptyMessage = await page
      .getByText(/no users found/i)
      .isVisible()
      .catch(() => false);

    expect(hasTable || hasEmptyMessage).toBeTruthy();
  });

  test('新しいユーザーを作成できる', async ({ page }) => {
    // データの初期状態を待つ
    await waitForTableOrEmptyState(page);

    // フォームに入力
    const timestamp = Date.now();
    const testName = `Test User ${String(timestamp)}`;
    const testEmail = `testuser${String(timestamp)}@example.com`;

    await page.getByPlaceholder('Name').fill(testName);
    await page.getByPlaceholder('Email').fill(testEmail);

    // フォームを送信
    await page.getByRole('button', { name: /create user/i }).click();

    // ローディング状態を待つ
    await expect(page.getByRole('button', { name: /create user/i })).toBeDisabled();
    await expect(page.getByRole('button', { name: /create user/i })).not.toBeDisabled({
      timeout: 5000,
    });

    // フォームがクリアされることを確認
    await expect(page.getByPlaceholder('Name')).toHaveValue('');
    await expect(page.getByPlaceholder('Email')).toHaveValue('');

    // 作成したユーザーがリストに表示されることを確認
    await expect(page.getByText(testName)).toBeVisible({ timeout: 5000 });
    await expect(page.getByText(testEmail)).toBeVisible();
  });

  test('フォームのバリデーションが機能する', async ({ page }) => {
    // 空のフォームで送信を試みる
    await page.getByRole('button', { name: /create user/i }).click();

    // ブラウザのバリデーションメッセージが表示される（HTML5 required属性）
    const nameInput = page.getByPlaceholder('Name');
    await expect(nameInput).toHaveAttribute('required');

    const emailInput = page.getByPlaceholder('Email');
    await expect(emailInput).toHaveAttribute('required');
    await expect(emailInput).toHaveAttribute('type', 'email');
  });

  test('複数のユーザーを連続して作成できる', async ({ page }) => {
    // データの初期状態を待つ
    await waitForTableOrEmptyState(page);

    const timestamp = Date.now();

    // 1人目のユーザーを作成
    await page.getByPlaceholder('Name').fill(`User 1 ${String(timestamp)}`);
    await page.getByPlaceholder('Email').fill(`user1-${String(timestamp)}@example.com`);
    await page.getByRole('button', { name: /create user/i }).click();

    // 完了を待つ
    await expect(page.getByRole('button', { name: /create user/i })).not.toBeDisabled({
      timeout: 5000,
    });
    await expect(page.getByPlaceholder('Name')).toHaveValue('');

    // 2人目のユーザーを作成
    await page.getByPlaceholder('Name').fill(`User 2 ${String(timestamp)}`);
    await page.getByPlaceholder('Email').fill(`user2-${String(timestamp)}@example.com`);
    await page.getByRole('button', { name: /create user/i }).click();

    // 完了を待つ
    await expect(page.getByRole('button', { name: /create user/i })).not.toBeDisabled({
      timeout: 5000,
    });

    // 両方のユーザーが表示されることを確認
    await expect(page.getByText(`User 1 ${String(timestamp)}`)).toBeVisible({ timeout: 5000 });
    await expect(page.getByText(`User 2 ${String(timestamp)}`)).toBeVisible();
  });

  test('ユーザーリストがテーブル形式で表示される', async ({ page }) => {
    // ユーザーが存在する場合のみテストを実行
    const hasTable = await page.waitForSelector('table', { timeout: 10000 }).catch(() => null);

    if (hasTable !== null) {
      // テーブルヘッダーの確認
      await expect(page.getByRole('columnheader', { name: /id/i })).toBeVisible();
      await expect(page.getByRole('columnheader', { name: /name/i })).toBeVisible();
      await expect(page.getByRole('columnheader', { name: /email/i })).toBeVisible();
      await expect(page.getByRole('columnheader', { name: /created at/i })).toBeVisible();

      // データ行が存在することを確認
      const rows = page.getByRole('row');
      const rowCount = await rows.count();
      expect(rowCount).toBeGreaterThan(1); // ヘッダー行 + データ行
    }
  });
});
