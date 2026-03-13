import FeatureList from './FeatureList.svelte';

import type { Meta, StoryObj } from '@storybook/svelte-vite';

const meta = {
  title: 'Data/FeatureList',
  component: FeatureList,
  tags: ['autodocs'],
  parameters: {
    layout: 'padded',
  },
} satisfies Meta<typeof FeatureList>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {
  args: {
    columns: 2,
    items: [
      {
        title: 'Shared tokens',
        description: 'Centralize spacing, color, and typography decisions.',
      },
      {
        title: 'Story coverage',
        description: 'Keep visual regression work visible during migration.',
      },
      {
        title: 'Composable data views',
        description: 'Build lists and cards on top of reusable shells.',
      },
      {
        title: 'Package-local checks',
        description: 'Verify the UI package without editing neighboring areas.',
      },
    ],
  },
};
