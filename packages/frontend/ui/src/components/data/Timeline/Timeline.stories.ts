import Timeline from './Timeline.svelte';

import type { Meta, StoryObj } from '@storybook/svelte-vite';

const meta = {
  title: 'Data/Timeline',
  component: Timeline,
  tags: ['autodocs'],
} satisfies Meta<typeof Timeline>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {
  args: {
    events: [
      { title: 'Project created', time: '09:00', status: 'success' },
      { title: 'Data synced', time: '10:12', status: 'info' },
      { title: 'Review pending', time: '11:45', status: 'warning' },
    ],
  },
};
