import { expect, test, type Page, type Route } from '@playwright/test';

import { mockWebAuthn } from './support/webauthn';

const NO_STORE_HEADERS = {
  'cache-control': 'private, no-store, max-age=0',
  'content-type': 'application/json',
} as const;

const TEST_ULID = {
  requestId: '01ARZ3NDEKTSV4RRFFQ69G5FAV',
  accountIdA: '01ARZ3NDEKTSV4RRFFQ69G5FAW',
  passkeyCredentialId: '01ARZ3NDEKTSV4RRFFQ69G5FAX',
  sessionIdA: '01ARZ3NDEKTSV4RRFFQ69G5FAY',
} as const;

const fulfillJson = async (route: Route, status: number, body: unknown) => {
  await route.fulfill({
    status,
    headers: NO_STORE_HEADERS,
    body: JSON.stringify(body),
  });
};

/** JWT ペイロードを Base64url でエンコードする。 */
function encodeJwtPayload(claims: Record<string, unknown>): string {
  const json = JSON.stringify(claims);
  const base64 = btoa(json);
  return base64.replace(/\+/gu, '-').replace(/\//gu, '_').replace(/=/gu, '');
}

/** テスト用の短命 JWT を生成する。 */
function buildJwt(claims: { sub: string; sid: string; exp: number }): string {
  const header = btoa(JSON.stringify({ alg: 'HS256', typ: 'JWT' }))
    .replace(/\+/gu, '-')
    .replace(/\//gu, '_')
    .replace(/=/gu, '');
  const payload = encodeJwtPayload({
    iat: claims.exp - 900,
    ...claims,
  });
  return `${header}.${payload}.signature`;
}

/** パスキーログインの API モックを設定する。 */
const mockPasskeyLogin = async (
  page: Page,
  accountId: string,
  sessionId: string,
  token: string
) => {
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
      accountId,
      passkeyCredentialId: TEST_ULID.passkeyCredentialId,
      sessionId,
      accessToken: token,
      refreshToken: `refresh-${sessionId}`,
      expiresAt: '2026-04-04T00:00:00.000Z',
    });
  });
};

/**
 * 既存セッションを保持したまま、別アカウントを追加ログインする。
 * クライアントサイドナビゲーションで `/login` へ遷移し、ログイン後 `/` へ戻る。
 */
const loginSecondAccountViaPasskeyUi = async (
  page: Page,
  accountId: string,
  sessionId: string,
  token: string
) => {
  await mockPasskeyLogin(page, accountId, sessionId, token);
  await page.getByRole('link', { name: '別アカウントを追加' }).click();
  await expect(page).toHaveURL(/app.localhost:5174\/login$/);
  await page.getByRole('button', { name: 'パスキーでログイン' }).click();
  await expect(page).toHaveURL(/app.localhost:5174\/?$/);
};

/** UI からパスキーログインを実行する。 */
const loginViaPasskeyUi = async (
  page: Page,
  accountId: string,
  sessionId: string,
  token: string
) => {
  await mockWebAuthn(page);
  await mockPasskeyLogin(page, accountId, sessionId, token);
  await page.goto('http://app.localhost:5174/login');
  await page.getByRole('button', { name: 'パスキーでログイン' }).click();
  await expect(page).toHaveURL(/app.localhost:5174\/?$/);
};

const accountIdB = '01ARZ3NDEKTSV4RRFFQ69G5FAZ';
const sessionIdB = '01ARZ3NDEKTSV4RRFFQ69G5FB0';

/** アカウント B 用の JWT を生成する。 */
function buildJwtB(exp: number): string {
  return buildJwt({ sub: accountIdB, sid: sessionIdB, exp });
}

test.describe('multi-account and device management', () => {
  test.skip(
    ({ browserName }) => browserName !== 'chromium',
    'multi-account and device management tests run in Chromium only'
  );

  /**
   * AUTH-FE-S029: 複数セッション時に AccountSwitcher UI が表示される
   */
  test('[AUTH-FE-S029] Account switcher UI is visible', async ({ page }) => {
    const tokenA = buildJwt({
      sub: TEST_ULID.accountIdA,
      sid: TEST_ULID.sessionIdA,
      exp: Math.floor(Date.now() / 1000) + 900,
    });
    await loginViaPasskeyUi(page, TEST_ULID.accountIdA, TEST_ULID.sessionIdA, tokenA);

    const tokenB = buildJwtB(Math.floor(Date.now() / 1000) + 900);
    await loginSecondAccountViaPasskeyUi(page, accountIdB, sessionIdB, tokenB);

    // AccountSwitcher trigger が表示され、アクティブなアカウント（B）の短縮 ID を含む
    const switcher = page.getByLabel('アカウントを切り替える');
    await expect(switcher).toBeVisible();
    await expect(switcher).toContainText('01ARZ3ND…5FAZ');
  });

  /**
   * AUTH-FE-S028: アカウント切り替え後の API ヘッダー変更を検証する
   *
   * sessions API が Authorization ヘッダーに応じて異なるデバイス名を返すようにモックし、
   * アカウント切り替え後にページが remount して正しいトークンのデータが表示されることを確認する。
   */
  test('[AUTH-FE-S028] Account switching changes active token', async ({ page }) => {
    const tokenA = buildJwt({
      sub: TEST_ULID.accountIdA,
      sid: TEST_ULID.sessionIdA,
      exp: Math.floor(Date.now() / 1000) + 900,
    });
    await loginViaPasskeyUi(page, TEST_ULID.accountIdA, TEST_ULID.sessionIdA, tokenA);

    const tokenB = buildJwtB(Math.floor(Date.now() / 1000) + 900);
    await loginSecondAccountViaPasskeyUi(page, accountIdB, sessionIdB, tokenB);

    // Authorization ヘッダーに応じて異なるデバイス名を返す
    await page.route('**/api/v1/sessions', async (route) => {
      const auth = (await route.request().headerValue('authorization')) ?? '';
      const deviceName = auth.includes(tokenA) ? 'Device-A' : 'Device-B';
      const sessionId = auth.includes(tokenA) ? TEST_ULID.sessionIdA : sessionIdB;
      await fulfillJson(route, 200, {
        requestId: TEST_ULID.requestId,
        sessions: [
          {
            sessionId,
            deviceName,
            loginAt: '2026-01-01T00:00:00.000Z',
            lastActiveAt: '2026-01-01T00:00:00.000Z',
            ipHash: 'abc',
            isCurrentSession: true,
          },
        ],
      });
    });

    // デバイス管理ページへ遷移（アクティブは B、Device-B が表示される）
    await page.getByRole('link', { name: 'デバイス管理' }).click();
    await expect(page).toHaveURL(/app.localhost:5174\/sessions$/);
    await expect(page.getByText('Device-B')).toBeVisible();
    await expect(page.getByText('Device-A')).not.toBeVisible();

    // AccountSwitcher でアカウント A に切り替える
    await page.getByLabel('アカウントを切り替える').click();
    await page.getByRole('menuitemradio', { name: '01ARZ3ND…5FAW' }).click();

    // {#key} によりページが remount し、A のトークンで再取得 → Device-A が表示される
    await expect(page.getByText('Device-A')).toBeVisible();
    await expect(page.getByText('Device-B')).not.toBeVisible();
  });

  /**
   * AUTH-FE-S030: 部分ログアウトでアクティブセッションのみが除去される
   *
   * アカウント A・B の両方をログイン後、ログアウトするとアクティブな B のみが除去され、
   * A のセッションが維持されて認証状態が保たれることを検証する。
   */
  test('[AUTH-FE-S030] Logout affects only active session', async ({ page }) => {
    const tokenA = buildJwt({
      sub: TEST_ULID.accountIdA,
      sid: TEST_ULID.sessionIdA,
      exp: Math.floor(Date.now() / 1000) + 900,
    });
    await loginViaPasskeyUi(page, TEST_ULID.accountIdA, TEST_ULID.sessionIdA, tokenA);

    const tokenB = buildJwtB(Math.floor(Date.now() / 1000) + 900);
    await loginSecondAccountViaPasskeyUi(page, accountIdB, sessionIdB, tokenB);

    // logout API のリクエストをキャプチャして Authorization ヘッダーを検証
    const logoutRequestPromise = page.waitForRequest(
      (req) => req.url().includes('/api/v1/auth/logout') && req.method() === 'POST'
    );

    // sessions API をモックし、A のトークンのみ受け入れ、B のトークンは拒否
    await page.route('**/api/v1/sessions', async (route) => {
      if (route.request().method() === 'GET') {
        const auth = (await route.request().headerValue('authorization')) ?? '';
        if (auth.includes(tokenA)) {
          await fulfillJson(route, 200, {
            requestId: TEST_ULID.requestId,
            sessions: [
              {
                sessionId: TEST_ULID.sessionIdA,
                deviceName: 'Chrome on macOS',
                loginAt: '2026-01-01T00:00:00.000Z',
                lastActiveAt: '2026-01-01T00:00:00.000Z',
                ipHash: 'abc',
                isCurrentSession: true,
              },
            ],
          });
        } else {
          // B のトークンが残っている場合は 401 を返してテストを失敗させる
          await fulfillJson(route, 401, {
            requestId: TEST_ULID.requestId,
            error: 'session-expired',
          });
        }
      }
    });

    // ログアウト（アクティブな B のみが除去される）
    await page.getByRole('link', { name: 'ログアウト' }).click();

    // 残りセッションがあるため `/` へ遷移し、ログイン画面には戻らない
    await expect(page).toHaveURL(/app.localhost:5174\/?$/);

    // logout API が B のトークンで呼ばれたことを検証
    const logoutRequest = await logoutRequestPromise;
    expect(logoutRequest.headers().authorization).toContain(tokenB);

    // A のセッションが有効なままであることを確認（デバイス管理ページへ遷移して検証）
    await page.getByRole('link', { name: 'デバイス管理' }).click();
    await expect(page).toHaveURL(/app.localhost:5174\/sessions$/);
    await expect(page.getByText('Chrome on macOS')).toBeVisible();
  });

  /**
   * AUTH-FE-S032: access token 期限切れ時にリフレッシュ成功でセッションを継続する
   *
   * 期限切れ間近のトークンでログイン後、デバイス管理ページへ client-side ナビゲーションで遷移する。
   * listDevices() 内で ensureFreshAuthorizationHeaders() が自動リフレッシュをトリガーし、
   * 成功時にセッションが継続することを検証する。
   */
  test('[AUTH-FE-S032] Proactive refresh continues session', async ({ page }) => {
    const exp = Math.floor(Date.now() / 1000) + 30; // 30 秒後に期限切れ
    const tokenA = buildJwt({
      sub: TEST_ULID.accountIdA,
      sid: TEST_ULID.sessionIdA,
      exp,
    });

    await loginViaPasskeyUi(page, TEST_ULID.accountIdA, TEST_ULID.sessionIdA, tokenA);

    // refresh API をモック（成功）
    const newToken = buildJwt({
      sub: TEST_ULID.accountIdA,
      sid: TEST_ULID.sessionIdA,
      exp: Math.floor(Date.now() / 1000) + 900,
    });
    await page.route('**/api/v1/auth/refresh', async (route) => {
      await fulfillJson(route, 200, {
        accessToken: newToken,
        refreshToken: 'refresh-new',
      });
    });

    // sessions API をモック
    await page.route('**/api/v1/sessions', async (route) => {
      if (route.request().method() === 'GET') {
        await fulfillJson(route, 200, {
          requestId: TEST_ULID.requestId,
          sessions: [
            {
              sessionId: TEST_ULID.sessionIdA,
              deviceName: 'Chrome on macOS',
              loginAt: '2026-01-01T00:00:00.000Z',
              lastActiveAt: '2026-01-01T12:00:00.000Z',
              ipHash: 'abc123',
              isCurrentSession: true,
            },
          ],
        });
      }
    });

    // client-side ナビゲーションでデバイス管理ページへ遷移
    await page.getByRole('link', { name: 'デバイス管理' }).click();
    await expect(page).toHaveURL(/app.localhost:5174\/sessions$/);

    // session-expired へ遷移しないことを確認
    await expect(page).not.toHaveURL(/app.localhost:5174\/session-expired$/);
    // デバイス一覧が表示されることを確認
    await expect(page.getByText('Chrome on macOS')).toBeVisible();
  });

  /**
   * AUTH-FE-S033: refresh 失敗時のみ session-expired へ遷移する
   *
   * 期限切れトークンでログイン後、デバイス管理ページへ client-side ナビゲーションで遷移する。
   * listDevices() 内で ensureFreshAuthorizationHeaders() が自動リフレッシュをトリガーし、
   * 失敗時に session-expired へ遷移することを検証する。
   */
  test('[AUTH-FE-S033] Refresh failure redirects to session-expired', async ({ page }) => {
    const exp = Math.floor(Date.now() / 1000) - 60; // 既に期限切れ
    const tokenA = buildJwt({
      sub: TEST_ULID.accountIdA,
      sid: TEST_ULID.sessionIdA,
      exp,
    });

    await loginViaPasskeyUi(page, TEST_ULID.accountIdA, TEST_ULID.sessionIdA, tokenA);

    // refresh API を失敗でモック
    await page.route('**/api/v1/auth/refresh', async (route) => {
      await fulfillJson(route, 401, {
        requestId: TEST_ULID.requestId,
        error: 'session-expired',
      });
    });

    // client-side ナビゲーションでデバイス管理ページへ遷移
    await page.getByRole('link', { name: 'デバイス管理' }).click();
    await expect(page).toHaveURL(/app.localhost:5174\/session-expired$/);
  });

  /**
   * AUTH-FE-S034: デバイス管理ページでログイン中のデバイスを確認できる
   */
  test('[AUTH-FE-S034] Device manager page shows sessions', async ({ page }) => {
    const tokenA = buildJwt({
      sub: TEST_ULID.accountIdA,
      sid: TEST_ULID.sessionIdA,
      exp: Math.floor(Date.now() / 1000) + 900,
    });

    await loginViaPasskeyUi(page, TEST_ULID.accountIdA, TEST_ULID.sessionIdA, tokenA);

    // sessions API をモック
    await page.route('**/api/v1/sessions', async (route) => {
      if (route.request().method() === 'GET') {
        await fulfillJson(route, 200, {
          requestId: TEST_ULID.requestId,
          sessions: [
            {
              sessionId: TEST_ULID.sessionIdA,
              deviceName: 'Chrome on macOS',
              loginAt: '2026-01-01T00:00:00.000Z',
              lastActiveAt: '2026-01-01T12:00:00.000Z',
              ipHash: 'abc123',
              isCurrentSession: true,
            },
            {
              sessionId: '01ARZ3NDEKTSV4RRFFQ69G5FAZ',
              deviceName: 'Safari on iOS',
              loginAt: '2026-01-02T00:00:00.000Z',
              lastActiveAt: '2026-01-02T12:00:00.000Z',
              ipHash: 'def456',
              isCurrentSession: false,
            },
          ],
        });
      }
    });

    // client-side ナビゲーションでデバイス管理ページへ遷移
    await page.getByRole('link', { name: 'デバイス管理' }).click();
    await expect(page).toHaveURL(/app.localhost:5174\/sessions$/);

    // デバイス名が表示される
    await expect(page.getByText('Chrome on macOS')).toBeVisible();
    await expect(page.getByText('Safari on iOS')).toBeVisible();

    // 現在のデバイスインジケーターが表示される
    await expect(page.getByLabel('現在のデバイス')).toBeVisible();
  });

  /**
   * AUTH-FE-S035: デバイス管理ページで特定デバイスをログアウトできる
   */
  test('[AUTH-FE-S035] Device manager revokes specific device', async ({ page }) => {
    const tokenA = buildJwt({
      sub: TEST_ULID.accountIdA,
      sid: TEST_ULID.sessionIdA,
      exp: Math.floor(Date.now() / 1000) + 900,
    });

    await loginViaPasskeyUi(page, TEST_ULID.accountIdA, TEST_ULID.sessionIdA, tokenA);

    const otherSessionId = '01ARZ3NDEKTSV4RRFFQ69G5FAZ';

    // sessions API をモック
    await page.route('**/api/v1/sessions', async (route) => {
      if (route.request().method() === 'GET') {
        await fulfillJson(route, 200, {
          requestId: TEST_ULID.requestId,
          sessions: [
            {
              sessionId: TEST_ULID.sessionIdA,
              deviceName: 'Chrome on macOS',
              loginAt: '2026-01-01T00:00:00.000Z',
              lastActiveAt: '2026-01-01T12:00:00.000Z',
              ipHash: 'abc123',
              isCurrentSession: true,
            },
            {
              sessionId: otherSessionId,
              deviceName: 'Safari on iOS',
              loginAt: '2026-01-02T00:00:00.000Z',
              lastActiveAt: '2026-01-02T12:00:00.000Z',
              ipHash: 'def456',
              isCurrentSession: false,
            },
          ],
        });
      }
    });

    // revoke API をモック
    await page.route(`**/api/v1/sessions/${otherSessionId}`, async (route) => {
      if (route.request().method() === 'DELETE') {
        await route.fulfill({
          status: 204,
          headers: { 'cache-control': 'private, no-store, max-age=0' },
          body: '',
        });
      }
    });

    // client-side ナビゲーションでデバイス管理ページへ遷移
    await page.getByRole('link', { name: 'デバイス管理' }).click();
    await expect(page).toHaveURL(/app.localhost:5174\/sessions$/);
    await expect(page.getByText('Safari on iOS')).toBeVisible();

    // Safari on iOS のログアウトボタンをクリック
    const revokeButton = page.getByRole('button', { name: 'Safari on iOS をログアウト' });
    await revokeButton.click();

    // Safari on iOS が一覧から消える
    await expect(page.getByText('Safari on iOS')).not.toBeVisible();
    await expect(page.getByText('Chrome on macOS')).toBeVisible();
  });

  /**
   * AUTH-FE-S036: デバイス管理ページで他のすべてのデバイスをログアウトできる
   */
  test('[AUTH-FE-S036] Device manager revokes all other devices', async ({ page }) => {
    const tokenA = buildJwt({
      sub: TEST_ULID.accountIdA,
      sid: TEST_ULID.sessionIdA,
      exp: Math.floor(Date.now() / 1000) + 900,
    });

    await loginViaPasskeyUi(page, TEST_ULID.accountIdA, TEST_ULID.sessionIdA, tokenA);

    const otherSessionId = '01ARZ3NDEKTSV4RRFFQ69G5FAZ';

    // sessions API をモック
    await page.route('**/api/v1/sessions', async (route) => {
      if (route.request().method() === 'GET') {
        await fulfillJson(route, 200, {
          requestId: TEST_ULID.requestId,
          sessions: [
            {
              sessionId: TEST_ULID.sessionIdA,
              deviceName: 'Chrome on macOS',
              loginAt: '2026-01-01T00:00:00.000Z',
              lastActiveAt: '2026-01-01T12:00:00.000Z',
              ipHash: 'abc123',
              isCurrentSession: true,
            },
            {
              sessionId: otherSessionId,
              deviceName: 'Safari on iOS',
              loginAt: '2026-01-02T00:00:00.000Z',
              lastActiveAt: '2026-01-02T12:00:00.000Z',
              ipHash: 'def456',
              isCurrentSession: false,
            },
          ],
        });
      }
    });

    // revoke others API をモック
    await page.route('**/api/v1/sessions/others', async (route) => {
      if (route.request().method() === 'DELETE') {
        await route.fulfill({
          status: 204,
          headers: { 'cache-control': 'private, no-store, max-age=0' },
          body: '',
        });
      }
    });

    // client-side ナビゲーションでデバイス管理ページへ遷移
    await page.getByRole('link', { name: 'デバイス管理' }).click();
    await expect(page).toHaveURL(/app.localhost:5174\/sessions$/);
    await expect(page.getByText('Safari on iOS')).toBeVisible();

    // 「他のすべてのデバイスをログアウト」ボタンをクリック
    const revokeOthersButton = page.getByRole('button', {
      name: '他のすべてのデバイスをログアウト',
    });
    await revokeOthersButton.click();

    // Safari on iOS が一覧から消え、現在のデバイスのみが残る
    await expect(page.getByText('Safari on iOS')).not.toBeVisible();
    await expect(page.getByText('Chrome on macOS')).toBeVisible();
  });
});
