import { expect, test, type Page, type Route } from '@playwright/test';

import { mockWebAuthn } from './support/webauthn';

const NO_STORE_HEADERS = {
  'cache-control': 'private, no-store, max-age=0',
  'content-type': 'application/json',
} as const;

const TEST_ULID = {
  requestId: '01ARZ3NDEKTSV4RRFFQ69G5FAV',
  authContextId: '01ARZ3NDEKTSV4RRFFQ69G5FB2',
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
  await mockWebAuthn(page);

  // passkey login mock
  await page.route('**/api/v1/auth/passkey/start', async (route) => {
    await fulfillJson(route, 200, {
      requestId: TEST_ULID.requestId,
      challenge: 'dGVzdC1jaGFsbGVuZ2U',
      rpId: 'app.localhost',
      userVerification: 'required',
    });
  });
  await page.route('**/api/v1/auth/passkey/finish', async (route) => {
    await fulfillJson(route, 200, {
      requestId: TEST_ULID.requestId,
      credentialMode: 'cookie',
      authContextId: TEST_ULID.authContextId,
      sessionId: TEST_ULID.sessionId,
      accessToken: 'jwt-access-token',
      expiresAt: '2026-04-04T00:00:00.000Z',
      contextIndexUpdateHints: [],
      clearCookieCommands: [],
      account: {
        accountId: TEST_ULID.accountId,
        passkeyCredentialId: TEST_ULID.passkeyCredentialId,
      },
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

  await page.goto('http://app.localhost:5174/login');
  await page.getByRole('button', { name: 'パスキーでログイン' }).click();
  await expect(page).toHaveURL(/app.localhost:5174\/?$/);

  // client-side navigation で passkeys ページへ遷移（in-memory session を維持）
  await page.getByRole('link', { name: 'パスキー管理' }).click();
  await expect(page).toHaveURL(/app.localhost:5174\/passkeys$/);
};

/** reauth 用の API モックをセットアップする。
 *  kind に応じたレスポンスを返し、request payload の検証も行う。
 */
const mockReauth = async (page: Page, kind: 'otp-issue' | 'passkey-delete' = 'otp-issue') => {
  await page.route('**/api/v1/auth/reauth/start', async (route) => {
    const body = route.request().postDataJSON() as { kind: string } | null;
    expect(body?.kind).toBe(kind);
    await fulfillJson(route, 200, {
      requestId: TEST_ULID.requestId,
      challenge: 'cmVhdXRoLWNoYWxsZW5nZQ',
      rpId: 'app.localhost',
      userVerification: 'required',
    });
  });
  await page.route('**/api/v1/auth/reauth/finish', async (route) => {
    const body = route.request().postDataJSON() as { kind: string } | null;
    expect(body?.kind).toBe(kind);
    await fulfillJson(route, 200, {
      requestId: TEST_ULID.requestId,
      reauthSessionId: '01ARZ3NDEKTSV4RRFFQ69G5FBA',
      kind,
      expiresAt: '2026-04-04T00:00:00.000Z',
    });
  });
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
        challenge: 'YWRkLWNoYWxsZW5nZQ',
        rpId: 'app.localhost',
        rpName: 'www-template',
        user: {
          id: 'dXNlcjE',
          name: 'test@example.com',
          displayName: 'Test User',
        },
        pubKeyCredParams: [
          { type: 'public-key', alg: -7 },
          { type: 'public-key', alg: -257 },
        ],
        residentKey: 'required',
        requireResidentKey: true,
        userVerification: 'required',
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

    await page.getByRole('button', { name: 'この端末でログインを有効にする' }).click();

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
    await mockReauth(page, 'passkey-delete');

    // 削除後に1件のみを返す
    await page.route(`**/api/v1/passkeys/${TEST_ULID.passkeyCredentialId}`, async (route) => {
      const reauthSession = route.request().headers()['x-reauth-session'];
      expect(reauthSession).toBe('01ARZ3NDEKTSV4RRFFQ69G5FBA');
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
    await mockWebAuthn(page);

    // 1件のみを返すようにモック上書き
    await page.route('**/api/v1/auth/passkey/start', async (route) => {
      await fulfillJson(route, 200, {
        requestId: TEST_ULID.requestId,
        challenge: 'dGVzdC1jaGFsbGVuZ2U',
        rpId: 'app.localhost',
        userVerification: 'required',
      });
    });
    await page.route('**/api/v1/auth/passkey/finish', async (route) => {
      await fulfillJson(route, 200, {
        requestId: TEST_ULID.requestId,
        credentialMode: 'cookie',
        authContextId: TEST_ULID.authContextId,
        sessionId: TEST_ULID.sessionId,
        accessToken: 'jwt-access-token',
        expiresAt: '2026-04-04T00:00:00.000Z',
        contextIndexUpdateHints: [],
        clearCookieCommands: [],
        account: {
          accountId: TEST_ULID.accountId,
          passkeyCredentialId: TEST_ULID.passkeyCredentialId,
        },
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

    await page.goto('http://app.localhost:5174/login');
    await page.getByRole('button', { name: 'パスキーでログイン' }).click();
    await expect(page).toHaveURL(/app.localhost:5174\/?$/);

    // client-side navigation で passkeys ページへ遷移
    await page.getByRole('link', { name: 'パスキー管理' }).click();
    await expect(page).toHaveURL(/app.localhost:5174\/passkeys$/);

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

    await page.getByRole('button', { name: 'この端末でログインを有効にする' }).click();

    await expect(page.getByRole('alert')).toBeVisible();
    await expect(page.getByRole('list')).toBeVisible();
  });

  /**
   * AUTH-FE-S015: パスキー削除フロー中にエラーが発生した場合は通知される (delete error)
   */
  test('パスキー削除でAPIエラーが返った場合にエラーメッセージが表示される', async ({ page }) => {
    await loginAndGoToPasskeys(page);
    await mockReauth(page, 'passkey-delete');

    await page.route(`**/api/v1/passkeys/${TEST_ULID.passkeyCredentialId}`, async (route) => {
      const reauthSession = route.request().headers()['x-reauth-session'];
      expect(reauthSession).toBe('01ARZ3NDEKTSV4RRFFQ69G5FBA');
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
  test('デバイスリンク発行ボタンを押すとメール送信済み guidance が表示される', async ({ page }) => {
    await loginAndGoToPasskeys(page);
    await mockReauth(page, 'device-link');

    await page.route('**/api/v1/passkeys/send-device-link', async (route) => {
      const reauthSession = route.request().headers()['x-reauth-session'];
      expect(reauthSession).toBe('01ARZ3NDEKTSV4RRFFQ69G5FBA');
      await fulfillJson(route, 200, {
        requestId: TEST_ULID.requestId,
        issued: true,
      });
    });

    await page.getByRole('button', { name: '新しい端末でログインを有効にする' }).click();

    // 平文 OTP ではなく、device-link の案内メッセージだけが表示される
    await expect(page.getByText(/ログイン有効化リンクを送信しました/)).toBeVisible();
    await expect(page.getByText(/登録済みのメールアドレス宛にリンクを送信しました/)).toBeVisible();
    await expect(page.getByText(/123456/)).not.toBeVisible();
  });

  /**
   * AUTH-FE-S022: Passkey deletion は再認証を要求する
   */
  test('パスキー削除時に再認証を経て削除が成功する', async ({ page }) => {
    await loginAndGoToPasskeys(page);
    await mockReauth(page, 'passkey-delete');

    await page.route(`**/api/v1/passkeys/${TEST_ULID.passkeyCredentialId}`, async (route) => {
      const reauthSession = route.request().headers()['x-reauth-session'];
      expect(reauthSession).toBe('01ARZ3NDEKTSV4RRFFQ69G5FBA');
      await route.fulfill({
        status: 204,
        headers: { 'cache-control': 'private, no-store, max-age=0' },
        body: '',
      });
    });

    await expect(page.getByText('MacBook Pro')).toBeVisible();

    const deleteButton = page.getByRole('button', { name: 'MacBook Pro を削除' });
    await expect(deleteButton).toBeEnabled();
    await deleteButton.click();

    await expect(page.getByText('MacBook Pro')).not.toBeVisible();
    await expect(page.getByText('iPhone 15')).toBeVisible();
  });
});
