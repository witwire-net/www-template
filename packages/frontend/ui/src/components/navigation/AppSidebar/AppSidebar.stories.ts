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

/** fixed（デフォルト）：画面左端固定ドロワー */
export const Default: Story = {
  args: {
    header: 'www-template UI',
    items: [
      { label: 'Overview', href: '/dashboard/overview', active: true },
      { label: 'Analytics', href: '/dashboard/analytics' },
      { label: 'Customers', href: '/dashboard/customers' },
      { label: 'Settings', href: '/dashboard/settings' },
    ],
    footer: '© 2024 www-template UI',
    isOpen: true,
  },
};

/** inline：フロー内に配置（旧 SideNav 相当） */
export const Inline: Story = {
  parameters: {
    layout: 'padded',
  },
  args: {
    variant: 'inline',
    header: 'Workspace',
    items: [
      { label: 'Overview', href: '/workspace/overview', active: true },
      { label: 'Analytics', href: '/workspace/analytics' },
      { label: 'Settings', href: '/workspace/settings' },
    ],
    footer: 'v1.0.0',
  },
};

/** 階層化ナビ（グループ折りたたみ） */
export const Nested: Story = {
  args: {
    header: 'Application',
    items: [
      { label: 'Dashboard', href: '/dashboard', active: false },
      {
        label: 'Analytics',
        defaultExpanded: true,
        children: [
          { label: 'Overview', href: '/analytics/overview', active: true },
          { label: 'Reports', href: '/analytics/reports' },
        ],
      },
      {
        label: 'Settings',
        children: [
          { label: 'General', href: '/settings/general' },
          { label: 'Security', href: '/settings/security' },
          { label: 'Billing', href: '/settings/billing' },
        ],
      },
    ],
    footer: '© 2024 www-template UI',
    isOpen: true,
  },
};

/** trailing スロット（バッジ等） */
export const WithTrailing: Story = {
  args: {
    header: 'Inbox',
    items: [
      { label: 'Inbox', href: '/inbox', active: true, trailing: '12' },
      { label: 'Sent', href: '/sent' },
      {
        label: 'Categories',
        trailing: '3',
        children: [
          { label: 'Work', href: '/categories/work', trailing: '5' },
          { label: 'Personal', href: '/categories/personal' },
        ],
      },
    ],
    isOpen: true,
  },
};

/** カスタム閉じるアイコン（モバイル） */
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
