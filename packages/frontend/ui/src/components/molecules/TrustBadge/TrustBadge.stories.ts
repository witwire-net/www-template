import TrustBadgeWithIcon from '@ui/story-support/components/molecules/TrustBadge/TrustBadgeWithIcon.svelte';

import TrustBadge from './TrustBadge.svelte';

import type { Meta, StoryObj } from '@storybook/svelte-vite';

const meta = {
  title: 'Molecules/TrustBadge',
  component: TrustBadge,
  tags: ['autodocs'],
} satisfies Meta<typeof TrustBadge>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {
  args: {
    label: 'SOC2',
    description: 'Certified compliance',
  },
};

export const WithIcon: Story = {
  render: (args) => ({
    Component: TrustBadgeWithIcon as typeof TrustBadge,
    props: args,
  }),
};
