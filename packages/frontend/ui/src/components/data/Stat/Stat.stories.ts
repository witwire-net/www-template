import Stat from './Stat.svelte';

import type { Meta, StoryObj } from '@storybook/svelte-vite';

const meta = {
  title: 'Data/Stat',
  component: Stat,
  tags: ['autodocs'],
} satisfies Meta<typeof Stat>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {
  args: {
    label: 'Active Users',
    value: '12,430',
    change: '+12%',
    trend: 'up',
    description: 'vs last week',
  },
};
