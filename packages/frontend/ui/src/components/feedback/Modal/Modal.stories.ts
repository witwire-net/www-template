import Modal from './Modal.svelte';
import ModalStory from './ModalStory.svelte';

import type { Meta, StoryObj } from '@storybook/svelte-vite';

const meta = {
  title: 'Feedback/Modal',
  component: Modal,
  tags: ['autodocs'],
  parameters: {
    layout: 'fullscreen',
  },
} satisfies Meta<typeof Modal>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {
  args: {
    open: false,
  },
  render: (() => ({
    Component: ModalStory,
  })) as unknown as Story['render'],
};
