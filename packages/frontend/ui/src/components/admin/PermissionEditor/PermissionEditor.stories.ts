import PermissionEditorStory from '@ui/components/admin/story-support/PermissionEditorStory.svelte';

import PermissionEditor from './PermissionEditor.svelte';

import type { Meta, StoryObj } from '@storybook/svelte-vite';

const meta = {
  title: 'Admin/PermissionEditor',
  component: PermissionEditor,
  tags: ['autodocs'],
} satisfies Meta<typeof PermissionEditor>;

export default meta;

type Story = StoryObj<Record<string, never>>;

export const Default: Story = {
  render: () => ({
    Component: PermissionEditorStory,
  }),
};
