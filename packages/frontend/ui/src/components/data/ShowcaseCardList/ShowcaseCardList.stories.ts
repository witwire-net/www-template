import ShowcaseCardList from './ShowcaseCardList.svelte';

import type { Meta, StoryObj } from '@storybook/svelte-vite';

const meta = {
  title: 'Data/ShowcaseCardList',
  component: ShowcaseCardList,
  tags: ['autodocs'],
  parameters: {
    layout: 'padded',
  },
} satisfies Meta<typeof ShowcaseCardList>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {
  args: {
    items: [
      {
        title: 'Field Ops Portal',
        description: 'Operational dashboards for dispatch, maintenance, and reporting.',
        industry: 'Logistics',
        action: 'View case study',
      },
      {
        title: 'Member Platform',
        description: 'A subscription workspace that aligned acquisition, onboarding, and billing.',
        industry: 'Media',
        action: 'Read overview',
      },
    ],
  },
};
