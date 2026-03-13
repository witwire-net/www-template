import PopoverStory from './PopoverStory.svelte';

import type { Meta, StoryObj } from '@storybook/svelte-vite';

const meta = {
  title: 'Feedback/Popover',
  component: PopoverStory,
  tags: ['autodocs'],
} satisfies Meta<typeof PopoverStory>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {
  args: {},
};
