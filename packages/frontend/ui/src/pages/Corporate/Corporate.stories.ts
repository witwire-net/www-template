import Corporate from './Corporate.svelte';

import type { Meta, StoryObj } from '@storybook/svelte-vite';

const meta = {
  title: 'Pages/Corporate',
  component: Corporate,
  tags: ['autodocs'],
  parameters: {
    layout: 'fullscreen',
  },
} satisfies Meta<typeof Corporate>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {};
