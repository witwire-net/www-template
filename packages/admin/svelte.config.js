import adapter from '@sveltejs/adapter-node';
import { vitePreprocess } from '@sveltejs/vite-plugin-svelte';

/** @type {import('@sveltejs/kit').Config} */
const config = {
  preprocess: vitePreprocess(),
  kit: {
    adapter: adapter(),
    csrf: {
      // Admin Console は独自の CSRF 防御機構（Origin 検証 + signed double-submit token）を使用するため、
      // SvelteKit 標準の CSRF チェックを無効化する
      checkOrigin: false,
    },
    alias: {
      $components: './src/lib/components',
      $server: './src/lib/server',
    },
  },
};

export default config;
