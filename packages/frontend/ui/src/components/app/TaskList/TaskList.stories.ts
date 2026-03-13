import TaskList from './TaskList.svelte';
import TaskListStory from './TaskListStory.svelte';

import type { Meta, StoryObj } from '@storybook/svelte-vite';

const meta = {
  title: 'App/TaskList',
  component: TaskList,
  tags: ['autodocs'],
} satisfies Meta<typeof TaskList>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {
  args: {
    tasks: [],
  },
  render: (() => ({
    Component: TaskListStory,
  })) as unknown as Story['render'],
};
