import adapter from '@sveltejs/adapter-static';
import { vitePreprocess } from '@sveltejs/vite-plugin-svelte';

/** @type {import('@sveltejs/kit').Config} */
const config = {
  preprocess: vitePreprocess(),
  kit: {
    adapter: adapter({
      // Admin Console は Go Admin API と同一 origin で配信される静的 SPA なので、HTML と asset を dist に固定する。
      pages: 'dist',
      // Cloudflare の静的 asset 配信と Go `/api/v1/*` routing を分けやすいよう、asset 出力先も dist に集約する。
      assets: 'dist',
      // SSR/server route を持たないため、未知 path は client router が復元できる index.html にフォールバックする。
      fallback: 'index.html',
    }),
    alias: {
      $components: './src/lib/components',
    },
  },
};

export default config;
