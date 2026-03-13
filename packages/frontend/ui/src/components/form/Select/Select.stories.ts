import Select from './Select.svelte';

import type { Meta, StoryObj } from '@storybook/svelte-vite';

const meta = {
  title: 'Form/Select',
  component: Select,
  tags: ['autodocs'],
} satisfies Meta<typeof Select>;

export default meta;

type Story = StoryObj<typeof meta>;

const defaultArgs = {
  label: 'Plan',
  placeholder: 'Select a plan',
  options: [
    { label: 'Starter', value: 'starter' },
    { label: 'Pro', value: 'pro' },
    { label: 'Enterprise', value: 'enterprise' },
  ],
};

export const Default: Story = {
  args: defaultArgs,
};

export const WithError: Story = {
  args: {
    ...defaultArgs,
    error: 'Please select a plan',
  },
};
