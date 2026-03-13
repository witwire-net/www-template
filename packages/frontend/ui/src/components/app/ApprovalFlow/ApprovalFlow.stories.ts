import ApprovalFlow from './ApprovalFlow.svelte';

import type { Meta, StoryObj } from '@storybook/svelte-vite';

const meta = {
  title: 'App/ApprovalFlow',
  component: ApprovalFlow,
  tags: ['autodocs'],
} satisfies Meta<typeof ApprovalFlow>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {
  args: {
    steps: [
      {
        label: 'Manager review',
        status: 'approved',
        approver: 'Jordan Lee',
      },
      {
        label: 'Security review',
        status: 'pending',
        approver: 'Sam Park',
      },
      { label: 'Finance review', status: 'pending' },
    ],
  },
};
