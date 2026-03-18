import { fileURLToPath, URL } from 'node:url';

import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';

export default defineConfig({
  plugins: [sveltekit()],
  resolve: {
    alias: [
      {
        find: '@www-template-frontend/ui/components',
        replacement: fileURLToPath(new URL('../ui/src/components/index.ts', import.meta.url)),
      },
      {
        find: '@www-template-frontend/ui',
        replacement: fileURLToPath(new URL('../ui/src/index.ts', import.meta.url)),
      },
      {
        find: '@ui',
        replacement: fileURLToPath(new URL('../ui/src', import.meta.url)),
      },
      {
        find: '@',
        replacement: fileURLToPath(new URL('../ui/src', import.meta.url)),
      },
      {
        find: /^@www-template-frontend\/ui\/(.*)$/,
        replacement: `${fileURLToPath(new URL('../ui/src/', import.meta.url))}$1`,
      },
      {
        find: /^@\/(.*)$/,
        replacement: `${fileURLToPath(new URL('../ui/src/', import.meta.url))}$1`,
      },
      {
        find: /^@ui\/(.*)$/,
        replacement: `${fileURLToPath(new URL('../ui/src/', import.meta.url))}$1`,
      },
      {
        find: '@www-template-frontend/domain',
        replacement: fileURLToPath(new URL('../domain/src/index.ts', import.meta.url)),
      },
      {
        find: /^@www-template-frontend\/domain\/(.*)$/,
        replacement: `${fileURLToPath(new URL('../domain/src/', import.meta.url))}$1`,
      },
      {
        find: '@www-template-frontend/api',
        replacement: fileURLToPath(new URL('../api/src/index.ts', import.meta.url)),
      },
      {
        find: /^@www-template-frontend\/api\/(.*)$/,
        replacement: `${fileURLToPath(new URL('../api/src/', import.meta.url))}$1`,
      },
    ],
  },
  server: {
    host: '0.0.0.0',
    port: 5173,
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
});
