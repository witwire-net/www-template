import CommandPalette from './CommandPalette.svelte';
import CommandPaletteStory from './CommandPaletteStory.svelte';

import type { Meta, StoryObj } from '@storybook/svelte-vite';

const meta = {
  title: 'App/CommandPalette',
  component: CommandPalette,
  tags: ['autodocs'],
  parameters: {
    layout: 'fullscreen',
  },
} satisfies Meta<typeof CommandPalette>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {
  args: {
    open: false,
    onClose: () => undefined,
    commands: [],
    inputPlaceholder: 'Type a command',
  },
  render: (() => ({
    Component: CommandPaletteStory,
  })) as unknown as Story['render'],
};
