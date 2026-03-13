import SearchBar from './SearchBar.svelte';
import SearchBarStory from './SearchBarStory.svelte';

import type { Meta, StoryObj } from '@storybook/svelte-vite';

const meta = {
  title: 'App/SearchBar',
  component: SearchBar,
  tags: ['autodocs'],
} satisfies Meta<typeof SearchBar>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {
  render: (() => ({
    Component: SearchBarStory,
  })) as unknown as Story['render'],
};
