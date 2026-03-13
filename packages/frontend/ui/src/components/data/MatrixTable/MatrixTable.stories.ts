import MatrixTable from './MatrixTable.svelte';

import type { Meta, StoryObj } from '@storybook/svelte-vite';

const meta = {
  title: 'Data/MatrixTable',
  component: MatrixTable,
  tags: ['autodocs'],
  parameters: {
    layout: 'padded',
  },
} satisfies Meta<typeof MatrixTable>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {
  args: {
    firstColumnHeader: 'Capability',
    headers: ['Starter', 'Growth', 'Enterprise'],
    rows: [
      { label: 'Seats', values: ['3', '15', 'Unlimited'] },
      { label: 'SSO', values: ['No', 'Optional add-on', 'Included'] },
      { label: 'Support SLA', values: ['Standard', 'Priority', 'Dedicated'] },
    ],
  },
};
