import PageHeader from './PageHeader.svelte';
import PageHeaderStory from './PageHeaderStory.svelte';

import type { Meta, StoryObj } from '@storybook/svelte-vite';

const meta = {
  title: 'App/PageHeader',
  component: PageHeader,
  tags: ['autodocs'],
  parameters: {
    layout: 'fullscreen',
  },
} satisfies Meta<typeof PageHeader>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {
  args: {
    title: 'Analytics',
  },
  render: (() => ({
    Component: PageHeaderStory,
  })) as unknown as Story['render'],
};
