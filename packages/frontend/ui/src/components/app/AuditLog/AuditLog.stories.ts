import AuditLog from './AuditLog.svelte';

import type { Meta, StoryObj } from '@storybook/svelte-vite';

const meta = {
  title: 'App/AuditLog',
  component: AuditLog,
  tags: ['autodocs'],
} satisfies Meta<typeof AuditLog>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {
  args: {
    entries: [
      {
        actor: 'Jordan Lee',
        action: 'updated',
        target: 'Workspace settings',
        time: 'Today 09:15',
      },
      {
        actor: 'Sam Park',
        action: 'invited',
        target: 'Alex Morgan',
        time: 'Yesterday 18:22',
      },
    ],
  },
};
