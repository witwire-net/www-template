import NotificationCenter from './NotificationCenter.svelte';

import type { Meta, StoryObj } from '@storybook/svelte-vite';

const meta = {
  title: 'App/NotificationCenter',
  component: NotificationCenter,
  tags: ['autodocs'],
} satisfies Meta<typeof NotificationCenter>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {
  args: {
    items: [
      {
        title: 'Weekly report ready',
        description: 'Your report is available',
        time: '1h ago',
      },
      {
        title: 'New comment',
        description: 'Jordan commented on a task',
        time: 'Yesterday',
        read: true,
      },
    ],
  },
};
