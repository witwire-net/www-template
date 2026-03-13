import Link from './Link.svelte';
import LinkStory from './LinkStory.svelte';

import type { Meta, StoryObj } from '@storybook/svelte-vite';

const meta = {
  title: 'Atoms/Link',
  component: Link,
  tags: ['autodocs'],
} satisfies Meta<typeof Link>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {
  render: (() => ({
    Component: LinkStory,
    props: { href: '/learn-more', text: 'Learn more' },
  })) as unknown as Story['render'],
};

export const Primary: Story = {
  render: (() => ({
    Component: LinkStory,
    props: { href: '/learn-more', text: 'Learn more', variant: 'primary' },
  })) as unknown as Story['render'],
};

export const AlwaysUnderline: Story = {
  render: (() => ({
    Component: LinkStory,
    props: { href: '/learn-more', text: 'Learn more', underline: 'always' },
  })) as unknown as Story['render'],
};
