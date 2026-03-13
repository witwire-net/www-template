import Radio from './Radio.svelte';

import type { Meta, StoryObj } from '@storybook/svelte-vite';

const meta = {
  title: 'Form/Radio',
  component: Radio,
  tags: ['autodocs'],
} satisfies Meta<typeof Radio>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {
  args: {
    label: 'Monthly billing',
    name: 'billing',
  },
};

export const Checked: Story = {
  args: {
    label: 'Annual billing',
    name: 'billing',
    checked: true,
  },
};
