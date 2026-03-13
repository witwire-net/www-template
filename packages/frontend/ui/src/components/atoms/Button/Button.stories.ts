import ButtonStory from './ButtonStory.svelte';

import type { Meta, StoryObj } from '@storybook/svelte-vite';

const meta = {
  title: 'Atoms/Button',
  component: ButtonStory,
  tags: ['autodocs'],
  parameters: {
    layout: 'centered',
  },
} satisfies Meta<typeof ButtonStory>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Primary: Story = {
  args: {
    text: 'Primary Button',
    variant: 'primary',
  },
};

export const Loading: Story = {
  args: {
    text: 'Loading...',
    isLoading: true,
  },
};
