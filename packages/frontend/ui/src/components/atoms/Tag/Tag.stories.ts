import Tag from './Tag.svelte';
import TagStory from './TagStory.svelte';

import type { Meta, StoryObj } from '@storybook/svelte-vite';

const meta = {
  title: 'Atoms/Tag',
  component: Tag,
  tags: ['autodocs'],
} satisfies Meta<typeof Tag>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {
  render: (() => ({
    Component: TagStory,
    props: { text: 'Design' },
  })) as unknown as Story['render'],
};

export const WithIcon: Story = {
  render: (() => ({
    Component: TagStory,
    props: { text: 'Fast', showIcon: true, variant: 'primary' },
  })) as unknown as Story['render'],
};
