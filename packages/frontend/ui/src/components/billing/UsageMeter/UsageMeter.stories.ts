import UsageMeter from './UsageMeter.svelte';

import type { Meta, StoryObj } from '@storybook/svelte-vite';

const meta = {
  title: 'Billing/UsageMeter',
  component: UsageMeter,
  tags: ['autodocs'],
  parameters: {
    layout: 'padded',
  },
} satisfies Meta<typeof UsageMeter>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {
  args: {
    label: 'Storage',
    used: 32,
    limit: 100,
    unit: 'GB',
  },
};
