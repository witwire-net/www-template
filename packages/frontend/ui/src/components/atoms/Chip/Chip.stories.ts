import Chip from './Chip.svelte';
import ChipStory from './ChipStory.svelte';

import type { Meta, StoryObj } from '@storybook/svelte-vite';

const meta = {
  title: 'Atoms/Chip',
  component: Chip,
  tags: ['autodocs'],
} satisfies Meta<typeof Chip>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {
  render: (() => ({
    Component: ChipStory,
    props: { text: 'Filter' },
  })) as unknown as Story['render'],
};

export const Removable: Story = {
  render: (() => ({
    Component: ChipStory,
    props: { text: 'Active', variant: 'primary', removable: true },
  })) as unknown as Story['render'],
};
