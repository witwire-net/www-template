import Skeleton from './Skeleton.svelte';
import SkeletonCard from './SkeletonCard.svelte';

import type { Meta, StoryObj } from '@storybook/svelte-vite';

const meta = {
  title: 'Atoms/Skeleton',
  component: Skeleton,
  tags: ['autodocs'],
} satisfies Meta<typeof Skeleton>;

export default meta;

type Story = StoryObj<typeof meta>;

export const TextLine: Story = {
  args: {
    variant: 'text',
    width: '100%',
  },
};

export const Circle: Story = {
  args: {
    variant: 'circular',
    width: 40,
    height: 40,
  },
};

export const Rectangle: Story = {
  args: {
    variant: 'rectangular',
    width: 200,
    height: 100,
  },
};

export const CardLoading: Story = {
  render: (() => ({
    Component: SkeletonCard,
  })) as unknown as Story['render'],
};
