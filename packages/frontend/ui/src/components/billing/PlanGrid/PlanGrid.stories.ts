import PlanGrid from './PlanGrid.svelte';
import PlanGridStory from './PlanGridStory.svelte';

import type { Meta, StoryObj } from '@storybook/svelte-vite';

const meta = {
  title: 'Billing/PlanGrid',
  component: PlanGrid,
  tags: ['autodocs'],
  parameters: {
    layout: 'padded',
  },
} satisfies Meta<typeof PlanGrid>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {
  render: (() => ({
    Component: PlanGridStory as typeof PlanGrid,
  })) as unknown as Story['render'],
};
