import AppSidebar from './AppSidebar.svelte';
import AppSidebarStory from './AppSidebarStory.svelte';

import type { Meta, StoryObj } from '@storybook/svelte-vite';

const meta = {
  title: 'Navigation/AppSidebar',
  component: AppSidebar,
  tags: ['autodocs'],
  parameters: {
    layout: 'fullscreen',
  },
} satisfies Meta<typeof AppSidebar>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {
  args: {
    logo: 'www-template UI',
    links: [
      { label: 'Overview', href: '/dashboard/overview', active: true },
      { label: 'Analytics', href: '/dashboard/analytics' },
      { label: 'Customers', href: '/dashboard/customers' },
      { label: 'Settings', href: '/dashboard/settings' },
    ],
    footer: '© 2024 www-template UI',
  },
};

export const CustomCloseIcon: Story = {
  render: (() => ({
    Component: AppSidebarStory,
    props: { customIcon: true },
  })) as unknown as Story['render'],
  parameters: {
    viewport: {
      defaultViewport: 'mobile1',
    },
  },
};
