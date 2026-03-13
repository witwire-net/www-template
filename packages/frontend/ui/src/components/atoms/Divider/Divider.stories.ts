import Divider from './Divider.svelte';
import DividerVerticalStory from './DividerVerticalStory.svelte';

import type { Meta, StoryObj } from '@storybook/svelte-vite';

const meta = {
  title: 'Atoms/Divider',
  component: Divider,
  tags: ['autodocs'],
} satisfies Meta<typeof Divider>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Horizontal: Story = {
  args: {
    orientation: 'horizontal',
  },
};

export const WithLabel: Story = {
  args: {
    label: 'OR',
  },
};

export const Vertical: Story = {
  render: (() => ({
    Component: DividerVerticalStory,
  })) as unknown as Story['render'],
};
