import Accordion from './Accordion.svelte';

import type { Meta, StoryObj } from '@storybook/svelte-vite';

const meta = {
  title: 'Feedback/Accordion',
  component: Accordion,
  tags: ['autodocs'],
} satisfies Meta<typeof Accordion>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {
  args: {
    items: [
      {
        id: 'first',
        title: 'How does migration work',
        content:
          'Adopt each reusable component first, then remove section wrappers after migration is complete.',
      },
      {
        id: 'second',
        title: 'Can this be controlled from outside',
        content:
          'Use defaultOpenIndexes for initial state and rely on allowMultipleOpen to define toggle behavior.',
      },
      {
        id: 'third',
        title: 'Does it support accessibility',
        content:
          'Each item is rendered as a button with aria-expanded and associated region semantics.',
      },
    ],
    defaultOpenIndexes: [0],
  },
};
