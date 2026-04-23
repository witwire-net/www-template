import adapter from '@sveltejs/adapter-cloudflare';
import { vitePreprocess } from '@sveltejs/vite-plugin-svelte';

const config = {
  preprocess: vitePreprocess(),
  kit: {
    adapter: adapter(),
    alias: {
      '@www-template/ui': '../frontend/ui/src/index.ts',
      '@www-template/ui/components': '../frontend/ui/src/components/index.ts',
      '@www-template/ui/styles': '../frontend/ui/src/styles/index.ts',
      '@www-template/ui/*': '../frontend/ui/src/*',
      '@': '../frontend/ui/src',
      '@/*': '../frontend/ui/src/*',
      '@ui': '../frontend/ui/src',
      '@ui/*': '../frontend/ui/src/*',
    },
  },
};

export default config;
