import Stack from './Stack.svelte';
import StackPreview from './StackPreview.svelte';

import type { Meta, StoryObj } from '@storybook/svelte-vite';

const meta = {
  title: 'Atoms/Stack',
  component: Stack,
  tags: ['autodocs'],
} satisfies Meta<typeof Stack>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Column: Story = {
  render: () => ({
    Component: StackPreview,
    props: { direction: 'column' },
  }),
};

export const Row: Story = {
  render: () => ({
    Component: StackPreview,
    props: { direction: 'row' },
  }),
};
