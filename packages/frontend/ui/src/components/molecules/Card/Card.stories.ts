import CardDefaultStory from '@ui/story-support/components/molecules/Card/CardDefaultStory.svelte';
import CardSizingStory from '@ui/story-support/components/molecules/Card/CardSizingStory.svelte';

import Card from './Card.svelte';

import type { Meta, StoryObj } from '@storybook/svelte-vite';

const meta = {
  title: 'Molecules/Card',
  component: Card,
  tags: ['autodocs'],
  parameters: {
    layout: 'padded',
  },
} satisfies Meta<typeof Card>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {
  render: (args) => ({
    Component: CardDefaultStory as typeof Card,
    props: args,
  }),
  args: {
    width: '300px',
  },
};

export const CustomSizing: Story = {
  render: (args) => ({
    Component: CardSizingStory as typeof Card,
    props: args,
  }),
};
