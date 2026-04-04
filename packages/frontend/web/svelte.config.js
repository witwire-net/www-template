import adapter from '@sveltejs/adapter-cloudflare';
import { vitePreprocess } from '@sveltejs/vite-plugin-svelte';

const config = {
  preprocess: vitePreprocess(),
  kit: {
    adapter: adapter(),
    alias: {
      '@www-template-frontend/api': '../api/src/index.ts',
      '@www-template-frontend/api/*': '../api/src/*',
      '@www-template-frontend/ui': '../ui/src/index.ts',
      '@www-template-frontend/ui/components': '../ui/src/components/index.ts',
      '@www-template-frontend/ui/styles': '../ui/src/styles/index.ts',
      '@www-template-frontend/ui/*': '../ui/src/*',
      '@': '../ui/src',
      '@/*': '../ui/src/*',
      '@ui': '../ui/src',
      '@ui/*': '../ui/src/*',
    },
  },
};

export default config;
