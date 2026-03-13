import ResourceCardList from './ResourceCardList.svelte';

import type { Meta, StoryObj } from '@storybook/svelte-vite';

const meta = {
  title: 'Data/ResourceCardList',
  component: ResourceCardList,
  tags: ['autodocs'],
  parameters: {
    layout: 'padded',
  },
} satisfies Meta<typeof ResourceCardList>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {
  args: {
    items: [
      {
        title: 'Component guide',
        description: 'Notes for composing reusable cards in the design system.',
        meta: 'Docs',
        action: 'Open',
      },
      {
        title: 'Quality gate checklist',
        description: 'A short checklist for codegen, lint, test, build, and Storybook smoke tests.',
        meta: 'Operations',
        action: 'Review',
      },
    ],
  },
};
