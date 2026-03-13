import Avatar from './Avatar.svelte';
import AvatarGallery from './AvatarGallery.svelte';

import type { Meta, StoryObj } from '@storybook/svelte-vite';

const meta = {
  title: 'Atoms/Avatar',
  component: Avatar,
  tags: ['autodocs'],
} satisfies Meta<typeof Avatar>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {
  args: {
    name: 'Alex Morgan',
  },
};

export const WithStatus: Story = {
  args: {
    name: 'Alex Morgan',
    status: 'online',
  },
};

export const Rounded: Story = {
  args: {
    name: 'AM',
    shape: 'rounded',
    size: 'lg',
  },
};

export const AllSizesWithStatus: Story = {
  render: (args) => ({
    Component: AvatarGallery,
    props: args,
  }),
  args: {
    name: 'Alex Morgan',
  },
};
