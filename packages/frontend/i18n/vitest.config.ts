import { defineConfig } from 'vitest/config';

/**
 * i18n パッケージ向けの Vitest 設定です。
 *
 * Node 環境で純粋な文字列処理と検証ロジックだけを実行し、
 * shared runtime の振る舞いを package 単位で確認します。
 */
export default defineConfig({
  test: {
    globals: true,
    environment: 'node',
    include: ['src/**/*.test.ts'],
    exclude: ['dist/**', 'node_modules/**'],
  },
});
