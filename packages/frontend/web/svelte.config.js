import adapter from '@sveltejs/adapter-cloudflare';
import { vitePreprocess } from '@sveltejs/vite-plugin-svelte';

const config = {
  preprocess: vitePreprocess(),
  kit: {
    adapter: adapter(),
    alias: {
      '@www-template/domain': '../domain/src/index.ts',
      '@www-template/domain/*': '../domain/src/*',
      '@www-template/api': '../api/src/index.ts',
      '@www-template/api/*': '../api/src/*',
      '@www-template/ui': '../ui/src/index.ts',
      '@www-template/ui/components': '../ui/src/components/index.ts',
      '@www-template/ui/styles': '../ui/src/styles/index.ts',
      '@www-template/ui/*': '../ui/src/*',
      '@': '../ui/src',
      '@/*': '../ui/src/*',
      '@ui': '../ui/src',
      '@ui/*': '../ui/src/*',
    },
  },
};

export default config;
