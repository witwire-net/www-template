import Toast from './Toast.svelte';
import ToastStory from './ToastStory.svelte';

import type { Meta, StoryObj } from '@storybook/svelte-vite';

const meta = {
  title: 'Feedback/Toast',
  component: Toast,
  tags: ['autodocs'],
} satisfies Meta<typeof Toast>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {
  args: {
    title: 'Toast title',
  },
  render: (() => ({
    Component: ToastStory,
  })) as unknown as Story['render'],
};
