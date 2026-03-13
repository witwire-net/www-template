import FilterBar from './FilterBar.svelte';
import FilterBarStory from './FilterBarStory.svelte';

import type { Meta, StoryObj } from '@storybook/svelte-vite';

const meta = {
  title: 'App/FilterBar',
  component: FilterBar,
  tags: ['autodocs'],
  parameters: {
    layout: 'fullscreen',
  },
} satisfies Meta<typeof FilterBar>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {
  render: (() => ({
    Component: FilterBarStory,
  })) as unknown as Story['render'],
};
