import Snackbar from './Snackbar.svelte';
import SnackbarStory from './SnackbarStory.svelte';

import type { Meta, StoryObj } from '@storybook/svelte-vite';

const meta = {
  title: 'Feedback/Snackbar',
  component: Snackbar,
  tags: ['autodocs'],
} satisfies Meta<typeof Snackbar>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {
  args: {
    message: 'Snackbar message',
  },
  render: (() => ({
    Component: SnackbarStory,
  })) as unknown as Story['render'],
};
