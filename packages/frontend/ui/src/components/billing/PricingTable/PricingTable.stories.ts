import PricingTable from './PricingTable.svelte';

import type { Meta, StoryObj } from '@storybook/svelte-vite';

const meta = {
  title: 'Billing/PricingTable',
  component: PricingTable,
  tags: ['autodocs'],
  parameters: {
    layout: 'padded',
    docs: {
      description: {
        component:
          'Plan-first folder placement remains, but the component now accepts generic `columns` and `rows` for comparison matrices beyond strict billing-only feature tables.',
      },
    },
  },
} satisfies Meta<typeof PricingTable>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {
  args: {
    heading: 'Capability',
    columns: [
      { id: 'starter', title: 'Starter', subtitle: 'Self-serve', highlight: '$0/mo' },
      { id: 'growth', title: 'Growth', subtitle: 'For scaling teams', highlight: '$29/mo' },
      {
        id: 'enterprise',
        title: 'Enterprise',
        subtitle: 'Custom rollout',
        highlight: 'Talk to us',
      },
    ],
    rows: [
      {
        id: 'coverage',
        label: 'Coverage',
        description: 'Which team shape the offer is designed for.',
        values: ['Up to 3 workspaces', 'Multi-team workspace', 'Org-wide rollout'],
      },
      {
        id: 'support',
        label: 'Support model',
        values: [
          { value: 'Email', supportingText: '48h response' },
          { value: 'Priority', supportingText: 'Business hours', emphasis: true },
          { value: 'Dedicated', supportingText: 'Named success lead', emphasis: true },
        ],
      },
      {
        id: 'security',
        label: 'Security controls',
        values: ['Standard roles', 'SSO add-on', 'SSO + audit trails'],
      },
    ],
  },
};
