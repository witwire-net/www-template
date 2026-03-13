import { svelteTesting } from '@testing-library/svelte/vite';
import { mergeConfig } from 'vite';
import { defineConfig } from 'vitest/config';

import viteConfig from './vite.config';

/** Vitest 設定 (ui パッケージ) */
export default mergeConfig(
  viteConfig,
  defineConfig({
    plugins: [svelteTesting()],
    test: {
      globals: true,
      environment: 'jsdom',
      setupFiles: ['./src/tests/setup.ts'],
      exclude: ['node_modules/**', 'dist/**', 'tests/e2e/**'],
      include: ['src/**/*.test.ts'],
      coverage: {
        provider: 'v8',
        reporter: ['text', 'json', 'html'],
        exclude: [
          'node_modules/**',
          'src/tests/**',
          '**/*.d.ts',
          '**/*.config.*',
          'src/theme.ts', // テーマ設定は除外
        ],
        thresholds: {
          lines: 80,
          functions: 80,
          branches: 75,
          statements: 80,
        },
      },
    },
  })
);
