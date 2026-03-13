import LP from './LP.svelte';

import type { Meta, StoryObj } from '@storybook/svelte-vite';

const meta = {
  title: 'Pages/LP',
  component: LP,
  tags: ['autodocs'],
  parameters: {
    layout: 'fullscreen',
  },
} satisfies Meta<typeof LP>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {};
