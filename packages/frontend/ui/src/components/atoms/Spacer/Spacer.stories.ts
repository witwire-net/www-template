import Spacer from './Spacer.svelte';
import SpacerShowcase from './SpacerShowcase.svelte';

import type { Meta, StoryObj } from '@storybook/svelte-vite';

const meta = {
  title: 'Atoms/Spacer',
  component: Spacer,
  tags: ['autodocs'],
} satisfies Meta<typeof Spacer>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Vertical: Story = {
  render: () => ({
    Component: SpacerShowcase,
    props: { axis: 'vertical' },
  }),
};

export const Horizontal: Story = {
  render: () => ({
    Component: SpacerShowcase,
    props: { axis: 'horizontal' },
  }),
};
