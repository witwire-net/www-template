import DataGrid from './DataGrid.svelte';
import DataGridStory from './DataGridStory.svelte';

import type { Meta, StoryObj } from '@storybook/svelte-vite';

const meta = {
  title: 'Data/DataGrid',
  component: DataGrid,
  tags: ['autodocs'],
  parameters: {
    layout: 'padded',
  },
} satisfies Meta<typeof DataGrid>;

export default meta;

type Story = StoryObj<{ compact?: boolean }>;

export const Default: Story = {
  render: (args) => ({
    Component: DataGridStory,
    props: args,
  }),
  args: {
    compact: false,
  },
};

export const Compact: Story = {
  render: (args) => ({
    Component: DataGridStory,
    props: args,
  }),
  args: {
    compact: true,
  },
};
