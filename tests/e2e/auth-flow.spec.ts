import { expect, test, type Page, type Route } from '@playwright/test';

const NO_STORE_HEADERS = {
  'cache-control': 'private, no-store, max-age=0',
  'content-type': 'application/json',
} as const;

const TEST_ULID = {
  requestId: '01ARZ3NDEKTSV4RRFFQ69G5FAV',
  accountId: '01ARZ3NDEKTSV4RRFFQ69G5FAW',
  passkeyCredentialId: '01ARZ3NDEKTSV4RRFFQ69G5FAX',
  sessionId: '01ARZ3NDEKTSV4RRFFQ69G5FAY',
  recoveryTokenId: '01ARZ3NDEKTSV4RRFFQ69G5FAZ',
  recoverySessionId: '01ARZ3NDEKTSV4RRFFQ69G5FB0',
} as const;

const fulfillJson = async (route: Route, status: number, body: unknown) => {
  await route.fulfill({
    status,
    headers: NO_STORE_HEADERS,
    body: JSON.stringify(body),
  });
};

const mockPasskeyLogin = async (page: Page) => {
  await page.route('**/api/v1/auth/passkey/start', async (route) => {
    await fulfillJson(route, 200, {
      requestId: TEST_ULID.requestId,
      challenge: 'test-challenge-base64',
      rpId: 'www-template',
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
};

const loginViaPasskeyUi = async (page: Page) => {
  await mockPasskeyLogin(page);
  await page.goto('http://localhost:5174/login');
  await page.getByRole('button', { name: 'パスキーでログイン' }).click();
  await expect(page).toHaveURL(/localhost:5174\/?$/);
  await expect(page.getByRole('heading', { name: '認証済みアプリのエントリー' })).toBeVisible();
};

test.describe('auth flow', () => {
  test.skip(
    ({ browserName }) => browserName !== 'chromium',
    'auth flow is validated in Chromium for stability'
  );

  test('未認証で protected route に入ると login に戻る', async ({ page }) => {
    await page.goto('http://localhost:5174/');

    await expect(page).toHaveURL(/localhost:5174\/login$/);
    await expect(page.getByRole('heading', { name: 'ログイン' })).toBeVisible();
  });

  test('recovery request は sent 画面へ進む', async ({ page }) => {
    await page.goto('http://localhost:5174/login/recovery');

    await page.getByLabel('メールアドレス').fill('member@example.com');
    await page.getByRole('button', { name: '復旧メールを送信' }).click();

    await expect(page).toHaveURL(/localhost:5174\/login\/recovery\/sent$/);
    await expect(page.getByRole('heading', { name: 'メールをご確認ください' })).toBeVisible();
    await expect(
      page.getByText('登録済みの宛先であれば、復旧用リンクをお送りします。')
    ).toBeVisible();
  });

  test('無効な recovery token は retry guidance を表示する', async ({ page }) => {
    await page.goto('http://localhost:5174/login/recovery/consume?token=invalid-token');

    await expect(page.getByRole('heading', { name: '復旧リンクを確認できません' })).toBeVisible();
    await expect(page.getByRole('link', { name: '復旧をやり直す' })).toHaveAttribute(
      'href',
      '/login/recovery'
    );
  });

  test('register page は snapshot が無いと recovery に戻す', async ({ page }) => {
    await page.goto('http://localhost:5174/login/recovery/register');

    await expect(page).toHaveURL(/localhost:5174\/login\/recovery$/);
    await expect(page.getByRole('button', { name: '復旧メールを送信' })).toBeVisible();
  });

  test('session-expired route から login に戻れる', async ({ page }) => {
    await page.goto('http://localhost:5174/session-expired');

    await expect(page.getByRole('heading', { name: 'セッションが切れました' })).toBeVisible();
    await page.getByRole('button', { name: 'ログインへ' }).click();
    await expect(page).toHaveURL(/localhost:5174\/login$/);
  });

  test('passkey login 成功で protected app へ入れる', async ({ page }) => {
    await loginViaPasskeyUi(page);
    await expect(page.getByRole('link', { name: 'ログアウト', exact: true })).toBeVisible();
  });

  test('認証済み状態から logout 導線で login に戻れる', async ({ page }) => {
    await loginViaPasskeyUi(page);

    await page.route('**/api/v1/auth/logout', async (route) => {
      await fulfillJson(route, 200, {
        requestId: TEST_ULID.requestId,
        revoked: true,
      });
    });

    await page.getByRole('link', { name: 'ログアウト', exact: true }).click();
    await expect(page).toHaveURL(/localhost:5174\/login$/);
    await expect(page.getByRole('heading', { name: 'ログイン' })).toBeVisible();
  });

  test('valid recovery token から passkey 再登録を完了できる', async ({ page }) => {
    await page.route('**/api/v1/auth/recovery/consume', async (route) => {
      await fulfillJson(route, 200, {
        requestId: TEST_ULID.requestId,
        recoveryTokenId: TEST_ULID.recoveryTokenId,
        recoverySessionId: TEST_ULID.recoverySessionId,
        recovery_session: 'recovery-session-opaque',
        expiresAt: '2026-03-21T00:15:00.000Z',
      });
    });

    await page.route('**/api/v1/auth/passkey/register', async (route) => {
      await fulfillJson(route, 200, {
        requestId: TEST_ULID.requestId,
        accountId: TEST_ULID.accountId,
        passkeyCredentialId: TEST_ULID.passkeyCredentialId,
        sessionId: TEST_ULID.sessionId,
        sessionToken: 'opaque-bearer-token-recovery',
        expiresAt: '2026-04-04T00:00:00.000Z',
      });
    });

    await page.goto('http://localhost:5174/login/recovery/consume?token=valid-token');
    await expect(page).toHaveURL(/localhost:5174\/login\/recovery\/register$/);
    await expect(page.getByRole('heading', { name: 'パスキー再登録' })).toBeVisible();

    await page.getByRole('button', { name: '新しいパスキーを登録' }).click();

    await expect(page).toHaveURL(/localhost:5174\/?$/);
    await expect(page.getByRole('heading', { name: '認証済みアプリのエントリー' })).toBeVisible();
  });
});
