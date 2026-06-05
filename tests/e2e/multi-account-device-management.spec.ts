import { expect, test, type Page, type Route } from '@playwright/test';

import { mockWebAuthn } from './support/webauthn';

const NO_STORE_HEADERS = {
  'cache-control': 'private, no-store, max-age=0',
  'content-type': 'application/json',
} as const;

const TEST_ULID = {
  requestId: '01ARZ3NDEKTSV4RRFFQ69G5FAV',
  authContextIdA: '01ARZ3NDEKTSV4RRFFQ69G5FB1',
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
      credentialMode: 'cookie',
      authContextId: authContextIdForSession(sessionId),
      sessionId,
      accessToken: token,
      expiresAt: '2026-04-04T00:00:00.000Z',
      contextIndexUpdateHints: [],
      clearCookieCommands: [],
      account: {
        accountId,
        passkeyCredentialId: TEST_ULID.passkeyCredentialId,
      },
    });
  });
};

const accountIdB = '01ARZ3NDEKTSV4RRFFQ69G5FAZ';
const sessionIdB = '01ARZ3NDEKTSV4RRFFQ69G5FB0';
const authContextIdB = '01ARZ3NDEKTSV4RRFFQ69G5FB2';

/** session ID に対応する authContextId を返し、E2E の session/context 対応を安定化する。 */
function authContextIdForSession(sessionId: string): string {
  return sessionId === TEST_ULID.sessionIdA ? TEST_ULID.authContextIdA : authContextIdB;
}

/**
 * 既存セッションを保持したまま、別アカウントを追加ログインする。
 * ユーザーメニューから「アカウント追加」をクリックし、ログイン後 `/` へ戻る。
 */
const loginSecondAccountViaPasskeyUi = async (
  page: Page,
  accountId: string,
  sessionId: string,
  token: string
) => {
  await mockPasskeyLogin(page, accountId, sessionId, token);
  // ユーザーメニューを開いてからアカウント追加をクリック
  await page.getByLabel('ユーザーメニュー').click();
  await page.getByRole('menuitem', { name: 'アカウント追加' }).click();
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
   * AUTH-FE-S029: 複数セッション時にユーザーメニューにアカウント切替が表示される
   */
  test('[AUTH-FE-S029] Account switcher UI is visible in user menu', async ({ page }) => {
    const tokenA = buildJwt({
      sub: TEST_ULID.accountIdA,
      sid: TEST_ULID.sessionIdA,
      exp: Math.floor(Date.now() / 1000) + 900,
    });
    await loginViaPasskeyUi(page, TEST_ULID.accountIdA, TEST_ULID.sessionIdA, tokenA);

    const tokenB = buildJwtB(Math.floor(Date.now() / 1000) + 900);
    await loginSecondAccountViaPasskeyUi(page, accountIdB, sessionIdB, tokenB);

    // ユーザーメニューを開く
    const userMenu = page.getByLabel('ユーザーメニュー');
    await expect(userMenu).toBeVisible();
    await userMenu.click();

    // アカウント切替セクションが表示される
    await expect(page.getByText('アカウント切替')).toBeVisible();
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

    // ユーザーメニューから設定ページへ client-side navigation
    await page.getByLabel('ユーザーメニュー').click();
    await page.getByRole('menuitem', { name: '設定' }).click();
    await expect(page).toHaveURL(/app.localhost:5174\/settings$/);
    // 設定ページからログインと端末へ client-side navigation
    await page.getByRole('link', { name: 'ログインと端末' }).click();
    await expect(page).toHaveURL(/app.localhost:5174\/settings\/sign-in$/);
    await expect(page.getByText('Device-B')).toBeVisible();
    await expect(page.getByText('Device-A')).not.toBeVisible();

    // ユーザーメニューを開いてアカウント A に切り替える
    await page.getByLabel('ユーザーメニュー').click();
    await page.getByRole('menuitemradio', { name: 'アカウント 1' }).click();

    // {#key} によりページが remount し、A のトークンで再取得 → Device-A が表示される
    await expect(page.getByText('Device-A')).toBeVisible();
    await expect(page.getByText('Device-B')).not.toBeVisible();
  });

  /**
   * AUTH-FE-S030: 部分ログアウトでアクティブセッションのみが除去される
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

    // ユーザーメニューからログアウト
    await page.getByLabel('ユーザーメニュー').click();
    await page.getByRole('menuitem', { name: 'ログアウト' }).click();

    // 残りセッションがあるため `/` へ遷移し、ログイン画面には戻らない
    await expect(page).toHaveURL(/app.localhost:5174\/?$/);

    // logout API が B のトークンで呼ばれたことを検証
    const logoutRequest = await logoutRequestPromise;
    expect(logoutRequest.headers().authorization).toContain(tokenB);

    // ユーザーメニューから設定ページへ client-side navigation
    await page.getByLabel('ユーザーメニュー').click();
    await page.getByRole('menuitem', { name: '設定' }).click();
    await expect(page).toHaveURL(/app.localhost:5174\/settings$/);
    // 設定ページからログインと端末へ client-side navigation
    await page.getByRole('link', { name: 'ログインと端末' }).click();
    await expect(page).toHaveURL(/app.localhost:5174\/settings\/sign-in$/);
    await expect(page.getByText('Chrome on macOS')).toBeVisible();
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

    // ユーザーメニューから設定ページへ client-side navigation
    await page.getByLabel('ユーザーメニュー').click();
    await page.getByRole('menuitem', { name: '設定' }).click();
    await expect(page).toHaveURL(/app.localhost:5174\/settings$/);
    // 設定ページからログインと端末へ client-side navigation
    await page.getByRole('link', { name: 'ログインと端末' }).click();
    await expect(page).toHaveURL(/app.localhost:5174\/settings\/sign-in$/);

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

    // ユーザーメニューから設定ページへ client-side navigation
    await page.getByLabel('ユーザーメニュー').click();
    await page.getByRole('menuitem', { name: '設定' }).click();
    await expect(page).toHaveURL(/app.localhost:5174\/settings$/);
    // 設定ページからログインと端末へ client-side navigation
    await page.getByRole('link', { name: 'ログインと端末' }).click();
    await expect(page).toHaveURL(/app.localhost:5174\/settings\/sign-in$/);
    await expect(page.getByText('Safari on iOS')).toBeVisible();

    // Safari on iOS のログアウトボタンをクリック
    const revokeButton = page.getByRole('button', { name: 'Safari on iOS をログアウト' });
    await revokeButton.click();

    // Safari on iOS が一覧から消える
    await expect(page.getByText('Safari on iOS')).not.toBeVisible();
    await expect(page.getByText('Chrome on macOS')).toBeVisible();
  });

  /**
   * AUTH-FE-S032: access token 期限切れ時にリフレッシュ成功でセッションを継続する
   *
   * 期限切れ間近のトークンでログイン後、デバイス管理ページへ遷移する。
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
    await page.route('**/api/v1/auth/contexts/*/refresh', async (route) => {
      await fulfillJson(route, 200, {
        requestId: TEST_ULID.requestId,
        credentialMode: 'cookie',
        authContextId: TEST_ULID.authContextIdA,
        sessionId: TEST_ULID.sessionIdA,
        accessToken: newToken,
        expiresAt: new Date((Math.floor(Date.now() / 1000) + 900) * 1000).toISOString(),
        contextIndexUpdateHints: [],
        clearCookieCommands: [],
        account: {
          accountId: TEST_ULID.accountIdA,
          passkeyCredentialId: TEST_ULID.passkeyCredentialId,
        },
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

    // 設定 → ログインと端末ページへ client-side navigation
    await page.goto('http://app.localhost:5174/settings');
    await page.getByRole('link', { name: 'ログインと端末' }).click();

    // session-expired へ遷移しないことを確認
    await expect(page).not.toHaveURL(/app.localhost:5174\/session-expired$/);
    // デバイス一覧が表示されることを確認
    await expect(page.getByText('Chrome on macOS')).toBeVisible();
  });

  /**
   * AUTH-FE-S033: refresh 失敗時のみ session-expired へ遷移する
   *
   * 期限切れトークンでログイン後、デバイス管理ページへ遷移する。
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
    await page.route('**/api/v1/auth/contexts/*/refresh', async (route) => {
      await fulfillJson(route, 401, {
        requestId: TEST_ULID.requestId,
        error: 'session-expired',
      });
    });

    // ユーザーメニューから設定ページへ client-side navigation
    await page.getByLabel('ユーザーメニュー').click();
    await page.getByRole('menuitem', { name: '設定' }).click();
    await expect(page).toHaveURL(/app.localhost:5174\/settings$/);
    // 設定ページからログインと端末へ client-side navigation
    await page.getByRole('link', { name: 'ログインと端末' }).click();
    await expect(page).toHaveURL(/app.localhost:5174\/settings\/sign-in$/);
    await expect(page).toHaveURL(/app.localhost:5174\/session-expired$/);
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

    // ユーザーメニューから設定ページへ client-side navigation
    await page.getByLabel('ユーザーメニュー').click();
    await page.getByRole('menuitem', { name: '設定' }).click();
    await expect(page).toHaveURL(/app.localhost:5174\/settings$/);
    // 設定ページからログインと端末へ client-side navigation
    await page.getByRole('link', { name: 'ログインと端末' }).click();
    await expect(page).toHaveURL(/app.localhost:5174\/settings\/sign-in$/);
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
