import { defineConfig } from 'vitest/config';

/**
 * Vitest monorepo projects.
 *
 * Run all tests: `pnpm test:run`
 * Run a single project: `vitest run --project frontend-app`
 */
export default defineConfig({
  test: {
    projects: [
      {
        extends: './packages/frontend/app/vitest.config.ts',
        root: './packages/frontend/app',
        test: {
          name: 'frontend-app',
        },
      },
      {
        extends: './packages/frontend/ui/vitest.config.ts',
        root: './packages/frontend/ui',
        test: {
          name: 'frontend-ui',
        },
      },
      {
        extends: './packages/admin/vitest.config.ts',
        root: './packages/admin',
        test: {
          name: 'frontend-admin',
        },
      },
    ],
  },
});
