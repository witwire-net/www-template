import RoleMatrix from './RoleMatrix.svelte';

import type { Meta, StoryObj } from '@storybook/svelte-vite';

const meta = {
  title: 'Admin/RoleMatrix',
  component: RoleMatrix,
  tags: ['autodocs'],
} satisfies Meta<typeof RoleMatrix>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {
  args: {
    rowHeaderLabel: 'Capability',
    trueLabel: 'Granted',
    falseLabel: 'Restricted',
    columns: [
      { id: 'owner', label: 'Owner' },
      { id: 'manager', label: 'Manager' },
      { id: 'support', label: 'Support' },
    ],
    rows: [
      {
        id: 'members',
        label: 'Manage members',
        cells: [{ value: true }, { value: true }, { value: false }],
      },
      {
        id: 'content',
        label: 'Edit content',
        cells: [{ value: true }, { value: true }, { value: false }],
      },
      {
        id: 'sla',
        label: 'Response SLA',
        cells: [
          { value: '1 hour', variant: 'info' },
          { value: '4 hours', variant: 'info' },
          { value: 'Next business day', variant: 'warning' },
        ],
      },
    ],
  },
};
