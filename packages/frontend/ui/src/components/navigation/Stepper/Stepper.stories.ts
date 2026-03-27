import Stepper from './Stepper.svelte';

import type { Meta, StoryObj } from '@storybook/svelte-vite';

const meta = {
  title: 'Navigation/Stepper',
  component: Stepper,
  tags: ['autodocs'],
} satisfies Meta<typeof Stepper>;

export default meta;

type Story = StoryObj<typeof meta>;

const defaultSteps = [
  { label: 'Details', description: 'Add basic info' },
  { label: 'Billing', description: 'Choose plan' },
  { label: 'Launch', description: 'Go live' },
];

export const Default: Story = {
  args: {
    steps: defaultSteps,
    activeStep: 1,
  },
};

export const Vertical: Story = {
  args: {
    steps: defaultSteps,
    activeStep: 1,
    orientation: 'vertical',
  },
};
