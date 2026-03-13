import Alert from './Alert.svelte';

import type { Meta, StoryObj } from '@storybook/svelte-vite';

const meta = {
  title: 'Feedback/Alert',
  component: Alert,
  tags: ['autodocs'],
} satisfies Meta<typeof Alert>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {
  args: {
    title: 'System update',
    description: 'A new update is available for your workspace.',
  },
};

export const Success: Story = {
  args: {
    title: 'Saved',
    description: 'Your changes have been saved.',
    variant: 'success',
  },
};
