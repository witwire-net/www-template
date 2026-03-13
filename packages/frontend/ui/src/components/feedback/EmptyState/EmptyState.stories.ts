import EmptyStateStory from './EmptyStateStory.svelte';

import type { Meta, StoryObj } from '@storybook/svelte-vite';

const meta = {
  title: 'Feedback/EmptyState',
  component: EmptyStateStory,
  tags: ['autodocs'],
} satisfies Meta<typeof EmptyStateStory>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {
  args: {},
};
