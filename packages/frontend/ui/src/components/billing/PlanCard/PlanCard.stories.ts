import PlanCard from './PlanCard.svelte';
import PlanCardStory from './PlanCardStory.svelte';

import type { Meta, StoryObj } from '@storybook/svelte-vite';

const meta = {
  title: 'Billing/PlanCard',
  component: PlanCard,
  tags: ['autodocs'],
  parameters: {
    layout: 'padded',
  },
} satisfies Meta<typeof PlanCard>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {
  render: (args) => ({
    Component: PlanCardStory as typeof PlanCard,
    props: args,
  }),
  args: {
    name: 'Pro',
    price: '$29',
    interval: 'month',
    description: 'For growing teams that need more control.',
    features: ['Unlimited projects', 'Advanced analytics', 'Priority support'],
  },
};
