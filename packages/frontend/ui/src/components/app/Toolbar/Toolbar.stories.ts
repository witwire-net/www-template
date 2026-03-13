import Toolbar from './Toolbar.svelte';
import ToolbarStory from './ToolbarStory.svelte';

import type { Meta, StoryObj } from '@storybook/svelte-vite';

const meta = {
  title: 'App/Toolbar',
  component: Toolbar,
  tags: ['autodocs'],
  parameters: {
    layout: 'fullscreen',
  },
} satisfies Meta<typeof Toolbar>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {
  render: (() => ({
    Component: ToolbarStory,
  })) as unknown as Story['render'],
};
