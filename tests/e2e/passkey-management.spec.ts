import { expect, test, type Page, type Route } from '@playwright/test';

const NO_STORE_HEADERS = {
  'cache-control': 'private, no-store, max-age=0',
  'content-type': 'application/json',
} as const;

const TEST_ULID = {
  requestId: '01ARZ3NDEKTSV4RRFFQ69G5FAV',
  accountId: '01ARZ3NDEKTSV4RRFFQ69G5FAW',
  passkeyCredentialId: '01ARZ3NDEKTSV4RRFFQ69G5FAX',
  passkeyCredentialId2: '01ARZ3NDEKTSV4RRFFQ69G5FB1',
  sessionId: '01ARZ3NDEKTSV4RRFFQ69G5FAY',
} as const;

const fulfillJson = async (route: Route, status: number, body: unknown) => {
  await route.fulfill({
    status,
    headers: NO_STORE_HEADERS,
    body: JSON.stringify(body),
  });
};

/** ログイン済み状態にして passkey 管理ページを開く */
const loginAndGoToPasskeys = async (page: Page) => {
  // passkey login mock
  await page.route('**/api/v1/auth/passkey/start', async (route) => {
    await fulfillJson(route, 200, {
      requestId: TEST_ULID.requestId,
      challenge: 'test-challenge-base64',
      rpId: 'localhost',
    });
  });
  await page.route('**/api/v1/auth/passkey/finish', async (route) => {
    await fulfillJson(route, 200, {
      requestId: TEST_ULID.requestId,
      accountId: TEST_ULID.accountId,
      passkeyCredentialId: TEST_ULID.passkeyCredentialId,
      sessionId: TEST_ULID.sessionId,
      sessionToken: 'opaque-bearer-token',
      expiresAt: '2026-04-04T00:00:00.000Z',
    });
  });
  // passkey list mock (2 items by default)
  await page.route('**/api/v1/passkeys', async (route) => {
    if (route.request().method() === 'GET') {
      await fulfillJson(route, 200, {
        requestId: TEST_ULID.requestId,
        passkeys: [
          {
            id: TEST_ULID.passkeyCredentialId,
            identifier: 'MacBook Pro',
            createdAt: '2026-01-01T00:00:00.000Z',
          },
          {
            id: TEST_ULID.passkeyCredentialId2,
            identifier: 'iPhone 15',
            createdAt: '2026-02-01T00:00:00.000Z',
          },
        ],
      });
    }
  });

  await page.goto('http://localhost:5174/login');
  await page.getByRole('button', { name: 'パスキーでログイン' }).click();
  await expect(page).toHaveURL(/localhost:5174\/?$/);

  await page.goto('http://localhost:5174/passkeys');
};

test.describe('passkey management', () => {
  test.skip(
    ({ browserName }) => browserName !== 'chromium',
    'passkey management tests run in Chromium only'
  );

  /**
   * AUTH-FE-S010: パスキー管理ページで登録済みパスキーを確認できる
   */
  test('パスキー管理ページで登録済みパスキーの一覧が表示される', async ({ page }) => {
    await loginAndGoToPasskeys(page);

    await expect(page.getByRole('list')).toBeVisible();
    await expect(page.getByText('MacBook Pro')).toBeVisible();
    await expect(page.getByText('iPhone 15')).toBeVisible();
  });

  /**
   * AUTH-FE-S011: 新しいパスキーを追加できる
   */
  test('パスキーを追加できる', async ({ page }) => {
    await loginAndGoToPasskeys(page);

    await page.route('**/api/v1/passkeys/start', async (route) => {
      await fulfillJson(route, 200, {
        requestId: TEST_ULID.requestId,
        challenge: 'add-challenge-base64',
        rpId: 'localhost',
      });
    });

    const newPasskeyId = '01ARZ3NDEKTSV4RRFFQ69G5FC0';
    const threePasskeys = [
      {
        id: TEST_ULID.passkeyCredentialId,
        identifier: 'MacBook Pro',
        createdAt: '2026-01-01T00:00:00.000Z',
      },
      {
        id: TEST_ULID.passkeyCredentialId2,
        identifier: 'iPhone 15',
        createdAt: '2026-02-01T00:00:00.000Z',
      },
      {
        id: newPasskeyId,
        identifier: 'New Device',
        createdAt: '2026-03-01T00:00:00.000Z',
      },
    ];

    await page.route('**/api/v1/passkeys/finish', async (route) => {
      await fulfillJson(route, 200, {
        requestId: TEST_ULID.requestId,
        passkeys: threePasskeys,
      });
    });

    // finish 後の listPasskeys() 再実行も 3件を返すよう GET mock を上書き
    await page.route('**/api/v1/passkeys', async (route) => {
      if (route.request().method() === 'GET') {
        await fulfillJson(route, 200, {
          requestId: TEST_ULID.requestId,
          passkeys: threePasskeys,
        });
      }
    });

    await page.getByRole('button', { name: '+ パスキーを追加' }).click();

    // 既存パスキーが保持されていることを確認
    await expect(page.getByText('MacBook Pro')).toBeVisible();
    await expect(page.getByText('iPhone 15')).toBeVisible();
    await expect(page.getByText('New Device')).toBeVisible();
  });

  /**
   * AUTH-FE-S012: パスキーを削除できる
   */
  test('2件以上ある場合にパスキーを削除できる', async ({ page }) => {
    await loginAndGoToPasskeys(page);

    // 削除後に1件のみを返す
    await page.route(`**/api/v1/passkeys/${TEST_ULID.passkeyCredentialId}`, async (route) => {
      await route.fulfill({
        status: 204,
        headers: { 'cache-control': 'private, no-store, max-age=0' },
        body: '',
      });
    });

    await expect(page.getByText('MacBook Pro')).toBeVisible();

    const deleteButton = page.getByRole('button', {
      name: 'MacBook Pro を削除',
    });
    await expect(deleteButton).toBeEnabled();
    await deleteButton.click();

    await expect(page.getByText('MacBook Pro')).not.toBeVisible();
    await expect(page.getByText('iPhone 15')).toBeVisible();
  });

  /**
   * AUTH-FE-S013: 最後の1件のパスキーは削除アクションが無効化される
   */
  test('最後の1件のパスキーの削除ボタンは無効化される', async ({ page }) => {
    // 1件のみを返すようにモック上書き
    await page.route('**/api/v1/auth/passkey/start', async (route) => {
      await fulfillJson(route, 200, {
        requestId: TEST_ULID.requestId,
        challenge: 'test-challenge-base64',
        rpId: 'localhost',
      });
    });
    await page.route('**/api/v1/auth/passkey/finish', async (route) => {
      await fulfillJson(route, 200, {
        requestId: TEST_ULID.requestId,
        accountId: TEST_ULID.accountId,
        passkeyCredentialId: TEST_ULID.passkeyCredentialId,
        sessionId: TEST_ULID.sessionId,
        sessionToken: 'opaque-bearer-token',
        expiresAt: '2026-04-04T00:00:00.000Z',
      });
    });
    await page.route('**/api/v1/passkeys', async (route) => {
      if (route.request().method() === 'GET') {
        await fulfillJson(route, 200, {
          requestId: TEST_ULID.requestId,
          passkeys: [
            {
              id: TEST_ULID.passkeyCredentialId,
              identifier: 'MacBook Pro',
              createdAt: '2026-01-01T00:00:00.000Z',
            },
          ],
        });
      }
    });

    await page.goto('http://localhost:5174/login');
    await page.getByRole('button', { name: 'パスキーでログイン' }).click();
    await expect(page).toHaveURL(/localhost:5174\/?$/);
    await page.goto('http://localhost:5174/passkeys');

    await expect(page.getByText('MacBook Pro')).toBeVisible();

    const deleteButton = page.getByRole('button', { name: 'MacBook Pro を削除' });
    await expect(deleteButton).toBeDisabled();
    await expect(page.getByText('最後のパスキーは削除できません')).toBeVisible();
  });

  /**
   * AUTH-FE-S014: パスキー追加フロー中にエラーが発生した場合は通知される
   */
  test('パスキー追加の start が失敗した場合にエラーメッセージが表示される', async ({ page }) => {
    await loginAndGoToPasskeys(page);

    await page.route('**/api/v1/passkeys/start', async (route) => {
      await fulfillJson(route, 503, {
        requestId: TEST_ULID.requestId,
        error: 'internal-error',
      });
    });

    await page.getByRole('button', { name: '+ パスキーを追加' }).click();

    await expect(page.getByRole('alert')).toBeVisible();
    await expect(page.getByRole('list')).toBeVisible();
  });

  /**
   * AUTH-FE-S015: パスキー削除フロー中にエラーが発生した場合は通知される (delete error)
   */
  test('パスキー削除でAPIエラーが返った場合にエラーメッセージが表示される', async ({ page }) => {
    await loginAndGoToPasskeys(page);

    await page.route(`**/api/v1/passkeys/${TEST_ULID.passkeyCredentialId}`, async (route) => {
      await fulfillJson(route, 503, {
        requestId: TEST_ULID.requestId,
        error: 'internal-error',
      });
    });

    const deleteButton = page.getByRole('button', { name: 'MacBook Pro を削除' });
    await deleteButton.click();

    await expect(page.getByRole('alert')).toBeVisible();
    // 一覧の状態が変化しないことを確認
    await expect(page.getByText('MacBook Pro')).toBeVisible();
    await expect(page.getByText('iPhone 15')).toBeVisible();
  });

  /**
   * AUTH-FE-S016: パスキー管理ページで OTP を発行できる
   */
  test('OTP 発行ボタンを押すと 6 桁の OTP が表示される', async ({ page }) => {
    await loginAndGoToPasskeys(page);

    await page.route('**/api/v1/passkeys/otp', async (route) => {
      await fulfillJson(route, 200, {
        requestId: TEST_ULID.requestId,
        otp: '123456',
      });
    });

    await page.getByRole('button', { name: 'OTPを発行' }).click();

    await expect(page.getByText('123456')).toBeVisible();
    await expect(page.getByText('このコードを新しい端末で入力してください')).toBeVisible();
  });
});

test.describe('passkey add by OTP (new device)', () => {
  test.skip(
    ({ browserName }) => browserName !== 'chromium',
    'passkey add-by-OTP tests run in Chromium only'
  );

  /**
   * AUTH-FE-S017: 新端末パスキー登録ページで有効な OTP を入力してパスキーを登録できる
   */
  test('有効な OTP を入力してパスキーを登録できる', async ({ page }) => {
    await page.route('**/api/v1/auth/passkey/add/start', async (route) => {
      await fulfillJson(route, 200, {
        requestId: TEST_ULID.requestId,
        challenge: 'otp-add-challenge-base64',
        rpId: 'localhost',
      });
    });

    await page.route('**/api/v1/auth/passkey/add/finish', async (route) => {
      await route.fulfill({
        status: 200,
        headers: { 'cache-control': 'private, no-store, max-age=0' },
        body: '',
      });
    });

    await page.goto('http://localhost:5174/passkeys/add');
    await expect(page.getByRole('heading', { name: 'パスキーを追加' })).toBeVisible();

    // OTP 入力フォームへの入力
    await page.getByLabel('ワンタイムパスワード').fill('123456');

    // ボタンが有効になっていること
    const submitButton = page.getByRole('button', { name: 'パスキーを登録' });
    await expect(submitButton).toBeEnabled();

    await submitButton.click();

    await expect(page.getByRole('heading', { name: 'パスキーを登録しました' })).toBeVisible();
    await expect(page.getByRole('link', { name: 'ログインページへ' })).toBeVisible();
  });

  /**
   * AUTH-FE-S018: 新端末パスキー登録ページで無効な OTP を入力した場合はエラーが表示される
   */
  test('無効な OTP を入力した場合にエラーメッセージが表示される', async ({ page }) => {
    await page.route('**/api/v1/auth/passkey/add/start', async (route) => {
      await fulfillJson(route, 400, {
        requestId: TEST_ULID.requestId,
        error: 'invalid_otp',
      });
    });

    await page.goto('http://localhost:5174/passkeys/add');
    await expect(page.getByRole('heading', { name: 'パスキーを追加' })).toBeVisible();

    await page.getByLabel('ワンタイムパスワード').fill('000000');

    const submitButton = page.getByRole('button', { name: 'パスキーを登録' });
    await expect(submitButton).toBeEnabled();

    await submitButton.click();

    await expect(page.getByRole('alert')).toBeVisible();
    // ページ状態が保持されていること（完了画面に遷移しない）
    await expect(page.getByRole('heading', { name: 'パスキーを追加' })).toBeVisible();
  });
});
