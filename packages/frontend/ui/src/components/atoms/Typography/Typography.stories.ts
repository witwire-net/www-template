import Typography from './Typography.svelte';
import TypographyShowcase from './TypographyShowcase.svelte';

import type { Meta, StoryObj } from '@storybook/svelte-vite';

const meta = {
  title: 'Atoms/Typography',
  component: Typography,
  tags: ['autodocs'],
} satisfies Meta<typeof Typography>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Showcase: Story = {
  render: (() => ({
    Component: TypographyShowcase,
  })) as unknown as Story['render'],
};
