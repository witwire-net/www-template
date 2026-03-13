import TooltipStory from './TooltipStory.svelte';

import type { Meta, StoryObj } from '@storybook/svelte-vite';

const meta = {
  title: 'Feedback/Tooltip',
  component: TooltipStory,
  tags: ['autodocs'],
} satisfies Meta<typeof TooltipStory>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {
  args: {},
};
