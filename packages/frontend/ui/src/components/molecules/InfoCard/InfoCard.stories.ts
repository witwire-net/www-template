import InfoCardShowcase from '@ui/story-support/components/molecules/InfoCard/InfoCardShowcase.svelte';

import InfoCard from './InfoCard.svelte';

import type { Meta, StoryObj } from '@storybook/svelte-vite';

const meta = {
  title: 'Molecules/InfoCard',
  component: InfoCard,
  tags: ['autodocs'],
} satisfies Meta<typeof InfoCard>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {
  args: {
    title: 'Analytics workspace',
    description: 'Use the shared card shell for compact content clusters.',
    meta: 'Updated 2h ago',
  },
};

export const WithAtoms: Story = {
  render: (args) => ({
    Component: InfoCardShowcase as typeof InfoCard,
    props: args,
  }),
};
