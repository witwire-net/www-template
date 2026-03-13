import Badge from './Badge.svelte';
import BadgeStory from './BadgeStory.svelte';

import type { Meta, StoryObj } from '@storybook/svelte-vite';

const meta = {
  title: 'Atoms/Badge',
  component: Badge,
  tags: ['autodocs'],
} satisfies Meta<typeof Badge>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {
  render: (() => ({
    Component: BadgeStory,
    props: { text: 'Beta' },
  })) as unknown as Story['render'],
};

export const Success: Story = {
  render: (() => ({
    Component: BadgeStory,
    props: { text: 'Live', variant: 'success' },
  })) as unknown as Story['render'],
};
