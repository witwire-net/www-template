import AppHeader from './AppHeader.svelte';
import AppHeaderStory from './AppHeaderStory.svelte';

import type { Meta, StoryObj } from '@storybook/svelte-vite';

const meta = {
  title: 'Navigation/AppHeader',
  component: AppHeader,
  tags: ['autodocs'],
  parameters: {
    layout: 'fullscreen',
  },
} satisfies Meta<typeof AppHeader>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {
  render: (() => ({
    Component: AppHeaderStory,
  })) as unknown as Story['render'],
};

export const CustomIcon: Story = {
  render: (() => ({
    Component: AppHeaderStory,
    props: { customIcon: true },
  })) as unknown as Story['render'],
};
