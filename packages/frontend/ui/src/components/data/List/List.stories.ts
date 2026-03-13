import List from './List.svelte';

import type { Meta, StoryObj } from '@storybook/svelte-vite';

const meta = {
  title: 'Data/List',
  component: List,
  tags: ['autodocs'],
} satisfies Meta<typeof List>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {
  args: {
    items: [
      {
        title: 'New deployment',
        description: 'Production build finished',
        meta: '2m ago',
        action: 'Open',
      },
      {
        title: 'Invoice paid',
        description: 'Team plan renewed',
        meta: '1h ago',
        action: 'Review',
      },
      {
        title: 'New member',
        description: 'Jordan joined the workspace',
        meta: 'Yesterday',
        action: 'Welcome',
      },
    ],
  },
};
