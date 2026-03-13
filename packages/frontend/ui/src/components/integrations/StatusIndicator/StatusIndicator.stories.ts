import StatusIndicator from './StatusIndicator.svelte';

import type { Meta, StoryObj } from '@storybook/svelte-vite';

const meta = {
  title: 'Integrations/StatusIndicator',
  component: StatusIndicator,
  tags: ['autodocs'],
} satisfies Meta<typeof StatusIndicator>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Active: Story = {
  args: {
    status: 'active',
    label: 'Connected',
  },
};

export const Warning: Story = {
  args: {
    status: 'warning',
    label: 'Needs attention',
  },
};
