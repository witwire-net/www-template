import FileUpload from './FileUpload.svelte';

import type { Meta, StoryObj } from '@storybook/svelte-vite';

const meta = {
  title: 'Form/FileUpload',
  component: FileUpload,
  tags: ['autodocs'],
} satisfies Meta<typeof FileUpload>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {
  args: {
    label: 'Upload files',
    helperText: 'PNG, JPG up to 10MB',
    title: 'Drop product images here',
    subtitle: 'You can drag images in or browse your device.',
    buttonLabel: 'Browse images',
  },
};
