import CollectionShowcase from '@ui/story-support/components/organisms/Collection/CollectionShowcase.svelte';

import Collection from './Collection.svelte';

import type { Meta, StoryObj } from '@storybook/svelte-vite';

const meta = {
  title: 'Organisms/Collection',
  component: Collection,
  tags: ['autodocs'],
  parameters: {
    layout: 'padded',
  },
} satisfies Meta<typeof Collection>;

export default meta;

type Story = StoryObj<Record<string, never>>;

export const Default: Story = {
  render: () => ({
    Component: CollectionShowcase,
  }),
};
