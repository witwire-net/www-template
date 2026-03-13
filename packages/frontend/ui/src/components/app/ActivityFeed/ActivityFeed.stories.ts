import ActivityFeed from './ActivityFeed.svelte';

import type { Meta, StoryObj } from '@storybook/svelte-vite';

const meta = {
  title: 'App/ActivityFeed',
  component: ActivityFeed,
  tags: ['autodocs'],
} satisfies Meta<typeof ActivityFeed>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {
  args: {
    items: [
      {
        title: 'Jordan updated the roadmap',
        time: '2m ago',
        avatar: 'JL',
      },
      { title: 'New deployment completed', time: '1h ago' },
      { title: 'Billing invoice paid', time: 'Yesterday' },
    ],
  },
};
