import Container from './Container.svelte';
import GridShowcase from './GridShowcase.svelte';

import type { Meta, StoryObj } from '@storybook/svelte-vite';

const meta = {
  title: 'Atoms/Grid',
  component: Container,
  tags: ['autodocs'],
} satisfies Meta<typeof Container>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {
  render: () => ({
    Component: GridShowcase,
  }),
};

export const Fluid: Story = {
  render: () => ({
    Component: GridShowcase,
    props: { fluid: true },
  }),
};
