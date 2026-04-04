import { defineConfig, devices } from '@playwright/test';

/**
 * Playwright E2E テスト設定
 * @see https://playwright.dev/docs/test-configuration
 */
export default defineConfig({
  testDir: './tests/e2e',
  /* テストを並列実行 */
  fullyParallel: false,
  /* CI環境でのリトライ設定 */
  retries: process.env.CI !== undefined ? 2 : 0,
  /* CI環境でのワーカー数 */
  workers: 1,
  /* レポーター設定 */
  reporter: 'html',
  /* 共通設定 */
  use: {
    /* ベースURL */
    baseURL: 'http://localhost:5173',
    /* 失敗時のスクリーンショット */
    screenshot: 'only-on-failure',
    /* 失敗時のビデオ */
    video: 'retain-on-failure',
    /* トレース設定 */
    trace: 'on-first-retry',
  },

  /* テスト前にサーバーを起動 */
  webServer: [
    {
      command: 'pnpm --filter @www-template/web dev',
      url: 'http://localhost:5173',
      reuseExistingServer: process.env.CI === undefined,
      timeout: 120 * 1000,
    },
    {
      command: 'pnpm --filter @www-template/app dev',
      url: 'http://localhost:5174/app',
      reuseExistingServer: process.env.CI === undefined,
      timeout: 120 * 1000,
    },
    {
      command: 'pnpm dev:server',
      url: 'http://localhost:8080/health',
      reuseExistingServer: process.env.CI === undefined,
      timeout: 120 * 1000,
    },
  ],

  /* ブラウザ設定 */
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },

    {
      name: 'firefox',
      use: { ...devices['Desktop Firefox'] },
    },

    {
      name: 'webkit',
      use: { ...devices['Desktop Safari'] },
    },

    /* モバイルブラウザテスト（オプション） */
    // {
    //   name: 'Mobile Chrome',
    //   use: { ...devices['Pixel 5'] },
    // },
    // {
    //   name: 'Mobile Safari',
    //   use: { ...devices['iPhone 12'] },
    // },
  ],
});
