import Sparkline from './Sparkline.svelte';

import type { Meta, StoryObj } from '@storybook/svelte-vite';

const meta = {
  title: 'Data/Sparkline',
  component: Sparkline,
  tags: ['autodocs'],
} satisfies Meta<typeof Sparkline>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {
  args: {
    data: [4, 9, 6, 12, 8, 16, 10],
  },
};
