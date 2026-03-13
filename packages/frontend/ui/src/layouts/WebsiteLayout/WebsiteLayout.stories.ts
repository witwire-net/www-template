import WebsiteLayout from './WebsiteLayout.svelte';
import WebsiteLayoutStory from './WebsiteLayoutStory.svelte';

import type { Meta, StoryObj } from '@storybook/svelte-vite';

const meta = {
  title: 'Layouts/WebsiteLayout',
  component: WebsiteLayout,
  tags: ['autodocs'],
  parameters: {
    layout: 'fullscreen',
  },
} satisfies Meta<typeof WebsiteLayout>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {
  render: (() => ({
    Component: WebsiteLayoutStory,
  })) as unknown as Story['render'],
};

export const RichContent: Story = {
  render: (() => ({
    Component: WebsiteLayoutStory,
    props: { rich: true },
  })) as unknown as Story['render'],
};
