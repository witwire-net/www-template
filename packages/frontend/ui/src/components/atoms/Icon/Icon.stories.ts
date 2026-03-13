import { IconBolt } from '@tabler/icons-svelte';

import Icon from './Icon.svelte';
import IconSet from './IconSet.svelte';

import type { Meta, StoryObj } from '@storybook/svelte-vite';

const meta = {
  title: 'Atoms/Icon',
  component: Icon,
  tags: ['autodocs'],
} satisfies Meta<typeof Icon>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {
  args: {
    icon: IconBolt,
    size: 24,
    title: 'Fast',
  },
};

export const Set: Story = {
  render: (() => ({
    Component: IconSet,
  })) as unknown as Story['render'],
};
