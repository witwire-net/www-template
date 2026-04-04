import adapter from '@sveltejs/adapter-static';
import { vitePreprocess } from '@sveltejs/vite-plugin-svelte';

const config = {
  preprocess: vitePreprocess(),
  kit: {
    adapter: adapter({
      pages: 'dist',
      assets: 'dist',
      fallback: 'index.html',
    }),
    alias: {
      '@www-template-frontend/api': '../api/src/index.ts',
      '@www-template-frontend/api/*': '../api/src/*',
      '@www-template-frontend/domain': '../domain/src/index.ts',
      '@www-template-frontend/domain/hooks/auth/useAuthSession':
        '../domain/src/hooks/auth/useAuthSession.svelte.ts',
      '@www-template-frontend/domain/hooks/auth/usePasskeyLogin':
        '../domain/src/hooks/auth/usePasskeyLogin.svelte.ts',
      '@www-template-frontend/domain/hooks/auth/useRecoveryFlow':
        '../domain/src/hooks/auth/useRecoveryFlow.svelte.ts',
      '@www-template-frontend/domain/hooks/auth/useSessionGuard':
        '../domain/src/hooks/auth/useSessionGuard.svelte.ts',
      '@www-template-frontend/domain/hooks/status/useStatus':
        '../domain/src/hooks/status/useStatus.svelte.ts',
      '@www-template-frontend/domain/*': '../domain/src/*',
      '@www-template-frontend/ui': '../ui/src/index.ts',
      '@www-template-frontend/ui/components': '../ui/src/components/index.ts',
      '@www-template-frontend/ui/styles': '../ui/src/styles/index.ts',
      '@www-template-frontend/ui/*': '../ui/src/*',
      '@': '../ui/src',
      '@/*': '../ui/src/*',
      '@ui': '../ui/src',
      '@ui/*': '../ui/src/*',
      types: '../domain/src/types',
    },
  },
};

export default config;
