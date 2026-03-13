import SideNav from './SideNav.svelte';

import type { Meta, StoryObj } from '@storybook/svelte-vite';

const meta = {
  title: 'Navigation/SideNav',
  component: SideNav,
  tags: ['autodocs'],
  parameters: {
    layout: 'fullscreen',
  },
} satisfies Meta<typeof SideNav>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {
  args: {
    header: 'Workspace',
    items: [
      { label: 'Overview', href: '/workspace/overview', active: true },
      { label: 'Analytics', href: '/workspace/analytics' },
      { label: 'Settings', href: '/workspace/settings' },
    ],
    footer: 'v1.0.0',
  },
};
