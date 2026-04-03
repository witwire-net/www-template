import '../src/styles/index';

import type { Preview } from '@storybook/svelte-vite';

const preview: Preview = {
  parameters: {
    controls: {
      matchers: {
        color: /(background|color)$/i,
        date: /date$/,
      },
    },
    layout: 'centered',
  },
};

export default preview;
