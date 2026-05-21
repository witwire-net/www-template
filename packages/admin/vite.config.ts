import { fileURLToPath } from 'node:url';

import { sveltekit } from '@sveltejs/kit/vite';
import tailwindcss from '@tailwindcss/vite';
import { defineConfig } from 'vitest/config';

export default defineConfig({
  resolve: {
    alias: {
      '@www-template/i18n': fileURLToPath(
        new URL('../frontend/i18n/src/index.ts', import.meta.url)
      ),
    },
  },
  plugins: [tailwindcss(), sveltekit()],
  server: {
    // Devcontainer 外のブラウザから Admin Console に到達できるよう、Vite を全インターフェースで待ち受ける。
    host: '0.0.0.0',
    port: 5176,
  },
  test: { include: ['src/**/*.{test,spec}.{ts,js}'] },
});
