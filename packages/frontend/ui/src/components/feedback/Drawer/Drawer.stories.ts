import Drawer from './Drawer.svelte';
import DrawerStory from './DrawerStory.svelte';

import type { Meta, StoryObj } from '@storybook/svelte-vite';

const meta = {
  title: 'Feedback/Drawer',
  component: Drawer,
  tags: ['autodocs'],
  parameters: {
    layout: 'fullscreen',
  },
} satisfies Meta<typeof Drawer>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {
  args: {
    open: false,
  },
  render: (() => ({
    Component: DrawerStory,
  })) as unknown as Story['render'],
};
