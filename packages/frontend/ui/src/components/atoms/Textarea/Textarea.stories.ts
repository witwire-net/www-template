import Textarea from './Textarea.svelte';

import type { Meta, StoryObj } from '@storybook/svelte-vite';

const meta = {
  title: 'Atoms/Textarea',
  component: Textarea,
  tags: ['autodocs'],
} satisfies Meta<typeof Textarea>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {
  args: {
    placeholder: 'Enter long text...',
  },
};

export const WithLabel: Story = {
  args: {
    label: 'Description',
    placeholder: 'Describe yourself',
  },
};
