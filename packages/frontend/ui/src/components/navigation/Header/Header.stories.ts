import AppHeaderStory from './AppHeaderStory.svelte';
import Header from './Header.svelte';
import SiteHeaderStory from './SiteHeaderStory.svelte';

import type { Meta, StoryObj } from '@storybook/svelte-vite';

const meta = {
  title: 'Navigation/Header',
  component: Header,
  tags: ['autodocs'],
  parameters: {
    layout: 'fullscreen',
  },
} satisfies Meta<typeof Header>;

export default meta;

type Story = StoryObj<typeof meta>;

/** Public site header with mobile drawer (variant="site") */
export const Site: Story = {
  render: (() => ({
    Component: SiteHeaderStory,
  })) as unknown as Story['render'],
};

export const SiteCustomIcon: Story = {
  render: (() => ({
    Component: SiteHeaderStory,
    props: { customIcon: true },
  })) as unknown as Story['render'],
};

export const SiteCustomLabels: Story = {
  render: (() => ({
    Component: SiteHeaderStory,
    props: { customLabels: true },
  })) as unknown as Story['render'],
};

/** Dashboard / app header with left-side menu button (variant="app") */
export const App: Story = {
  render: (() => ({
    Component: AppHeaderStory,
  })) as unknown as Story['render'],
};

export const AppCustomIcon: Story = {
  render: (() => ({
    Component: AppHeaderStory,
    props: { customIcon: true },
  })) as unknown as Story['render'],
};
