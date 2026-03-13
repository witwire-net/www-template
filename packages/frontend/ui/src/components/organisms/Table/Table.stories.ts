import TableShowcase from '@ui/story-support/components/organisms/Table/TableShowcase.svelte';

import Table from './Table.svelte';

import type { Meta, StoryObj } from '@storybook/svelte-vite';

const meta = {
  title: 'Organisms/Table',
  component: Table,
  tags: ['autodocs'],
  parameters: {
    layout: 'padded',
  },
} satisfies Meta<typeof Table>;

export default meta;

type Story = StoryObj<Record<string, never>>;

export const Default: Story = {
  render: () => ({
    Component: TableShowcase,
  }),
};
