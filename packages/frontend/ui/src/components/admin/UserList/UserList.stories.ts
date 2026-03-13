import UserListStory from '@ui/components/admin/story-support/UserListStory.svelte';

import UserList from './UserList.svelte';

import type { Meta, StoryObj } from '@storybook/svelte-vite';

const meta = {
  title: 'Admin/UserList',
  component: UserList,
  tags: ['autodocs'],
} satisfies Meta<typeof UserList>;

export default meta;

type Story = StoryObj<Record<string, never>>;

export const Default: Story = {
  render: () => ({
    Component: UserListStory,
  }),
};
