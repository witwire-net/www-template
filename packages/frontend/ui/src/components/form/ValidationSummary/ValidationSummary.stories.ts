import ValidationSummary from './ValidationSummary.svelte';

import type { Meta, StoryObj } from '@storybook/svelte-vite';

const meta = {
  title: 'Form/ValidationSummary',
  component: ValidationSummary,
  tags: ['autodocs'],
} satisfies Meta<typeof ValidationSummary>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {
  args: {
    errors: ['Email is required', 'Password must be at least 8 characters'],
  },
};
