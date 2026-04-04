import { fileURLToPath, URL } from 'node:url';

import { sveltekit } from '@sveltejs/kit/vite';
import tailwindcss from '@tailwindcss/vite';
import { defineConfig } from 'vite';

const svelteRuntimeChunkPattern =
  /[/\\]svelte[/\\]src[/\\](?:internal[/\\]client[/\\]runtime|index-client)\.js$/;

export default defineConfig({
  plugins: [tailwindcss(), sveltekit()],
  build: {
    rollupOptions: {
      output: {
        manualChunks(id) {
          if (svelteRuntimeChunkPattern.test(id)) {
            return 'svelte-runtime';
          }

          return undefined;
        },
      },
    },
  },
  resolve: {
    alias: [
      {
        find: '@www-template/ui/styles',
        replacement: fileURLToPath(new URL('../ui/src/styles/index.ts', import.meta.url)),
      },
      {
        find: '@www-template/ui/components',
        replacement: fileURLToPath(new URL('../ui/src/components/index.ts', import.meta.url)),
      },
      {
        find: '@www-template/ui',
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
        find: /^@www-template\/ui\/(.*)$/,
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
        find: '@www-template/domain',
        replacement: fileURLToPath(new URL('../domain/src/index.ts', import.meta.url)),
      },
      {
        find: /^@www-template\/domain\/hooks\/(.*)$/,
        replacement: `${fileURLToPath(new URL('../domain/src/hooks/', import.meta.url))}$1.svelte.ts`,
      },
      {
        find: /^@www-template\/domain\/(.*)$/,
        replacement: `${fileURLToPath(new URL('../domain/src/', import.meta.url))}$1`,
      },
      {
        find: '@www-template/api',
        replacement: fileURLToPath(new URL('../api/src/index.ts', import.meta.url)),
      },
      {
        find: /^@www-template\/api\/(.*)$/,
        replacement: `${fileURLToPath(new URL('../api/src/', import.meta.url))}$1`,
      },
    ],
  },
  server: {
    host: '0.0.0.0',
    port: 5174,
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
  preview: {
    host: '0.0.0.0',
    port: 4174,
  },
});
