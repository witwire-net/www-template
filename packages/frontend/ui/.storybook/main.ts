import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

import { mergeConfig } from 'vite';

import type { StorybookConfig } from '@storybook/svelte-vite';

const config: StorybookConfig = {
  stories: ['../src/**/*.stories.@(ts|svelte)'],
  addons: [],
  framework: {
    name: '@storybook/svelte-vite',
    options: {},
  },
  viteFinal(config) {
    const storybookDir = dirname(fileURLToPath(import.meta.url));
    return mergeConfig(config, {
      resolve: {
        alias: {
          '@': resolve(storybookDir, '../src'),
          '@ui': resolve(storybookDir, '../src'),
        },
      },
    });
  },
};

export default config;
