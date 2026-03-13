import MetricGrid from './MetricGrid.svelte';

import type { Meta, StoryObj } from '@storybook/svelte-vite';

const meta = {
  title: 'Data/MetricGrid',
  component: MetricGrid,
  tags: ['autodocs'],
} satisfies Meta<typeof MetricGrid>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {
  args: {
    items: [
      {
        label: 'Activation rate',
        value: '62%',
        trend: '+4.1%',
        context: 'Last 30 days',
        tone: 'success',
      },
      {
        label: 'Pipeline influenced',
        value: '$1.8M',
        trend: '+12%',
        context: 'Quarter to date',
        tone: 'primary',
      },
      {
        label: 'Time to launch',
        value: '9 days',
        trend: '-2 days',
        context: 'vs previous quarter',
        tone: 'info',
      },
      {
        label: 'Churn risk alerts',
        value: '18',
        trend: 'Needs attention',
        context: 'Requires follow-up',
        tone: 'warning',
      },
    ],
  },
};
