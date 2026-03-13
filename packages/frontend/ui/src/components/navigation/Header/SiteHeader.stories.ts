import SiteHeader from './SiteHeader.svelte';
import SiteHeaderStory from './SiteHeaderStory.svelte';

import type { Meta, StoryObj } from '@storybook/svelte-vite';

const meta = {
  title: 'Navigation/SiteHeader',
  component: SiteHeader,
  tags: ['autodocs'],
  parameters: {
    layout: 'fullscreen',
  },
} satisfies Meta<typeof SiteHeader>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {
  render: (() => ({
    Component: SiteHeaderStory,
  })) as unknown as Story['render'],
};

export const CustomIcon: Story = {
  render: (() => ({
    Component: SiteHeaderStory,
    props: { customIcon: true },
  })) as unknown as Story['render'],
};

export const CustomLabels: Story = {
  render: (() => ({
    Component: SiteHeaderStory,
    props: { customLabels: true },
  })) as unknown as Story['render'],
};
