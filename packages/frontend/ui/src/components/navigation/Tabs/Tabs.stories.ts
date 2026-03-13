import Tabs from './Tabs.svelte';

import type { Meta, StoryObj } from '@storybook/svelte-vite';

const meta = {
  title: 'Navigation/Tabs',
  component: Tabs,
  tags: ['autodocs'],
} satisfies Meta<typeof Tabs>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {
  args: {
    items: [
      { label: 'Overview', value: 'overview' },
      { label: 'Activity', value: 'activity' },
      { label: 'Settings', value: 'settings' },
    ],
    defaultValue: 'overview',
  },
};
