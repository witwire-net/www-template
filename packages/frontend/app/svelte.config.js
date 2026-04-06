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
      '@www-template/api': '../api/src/index.ts',
      '@www-template/api/*': '../api/src/*',
      '@www-template/domain': '../domain/src/index.ts',
      '@www-template/domain/hooks/auth/useAuthSession':
        '../domain/src/hooks/auth/useAuthSession.svelte.ts',
      '@www-template/domain/hooks/auth/usePasskeyLogin':
        '../domain/src/hooks/auth/usePasskeyLogin.svelte.ts',
      '@www-template/domain/hooks/auth/useRecoveryFlow':
        '../domain/src/hooks/auth/useRecoveryFlow.svelte.ts',
      '@www-template/domain/hooks/auth/useSessionGuard':
        '../domain/src/hooks/auth/useSessionGuard.svelte.ts',
      '@www-template/domain/hooks/auth/usePasskeyManagement':
        '../domain/src/hooks/auth/usePasskeyManagement.svelte.ts',
      '@www-template/domain/hooks/auth/usePasskeyAddByOtp':
        '../domain/src/hooks/auth/usePasskeyAddByOtp.svelte.ts',
      '@www-template/domain/hooks/status/useStatus':
        '../domain/src/hooks/status/useStatus.svelte.ts',
      '@www-template/domain/*': '../domain/src/*',
      '@www-template/ui': '../ui/src/index.ts',
      '@www-template/ui/components': '../ui/src/components/index.ts',
      '@www-template/ui/styles': '../ui/src/styles/index.ts',
      '@www-template/ui/*': '../ui/src/*',
      '@': '../ui/src',
      '@/*': '../ui/src/*',
      '@ui': '../ui/src',
      '@ui/*': '../ui/src/*',
      types: '../domain/src/types',
    },
  },
};

export default config;
