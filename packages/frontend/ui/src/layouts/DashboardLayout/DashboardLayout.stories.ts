import DashboardLayout from './DashboardLayout.svelte';
import DashboardLayoutStory from './DashboardLayoutStory.svelte';

import type { Meta, StoryObj } from '@storybook/svelte-vite';

const meta = {
  title: 'Layouts/DashboardLayout',
  component: DashboardLayout,
  tags: ['autodocs'],
  parameters: {
    layout: 'fullscreen',
  },
} satisfies Meta<typeof DashboardLayout>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {
  render: (() => ({
    Component: DashboardLayoutStory,
  })) as unknown as Story['render'],
};

export const RichNavigation: Story = {
  render: (() => ({
    Component: DashboardLayoutStory,
    props: { rich: true },
  })) as unknown as Story['render'],
};
