import Pagination from './Pagination.svelte';
import PaginationStory from './PaginationStory.svelte';

import type { Meta, StoryObj } from '@storybook/svelte-vite';

const meta = {
  title: 'Navigation/Pagination',
  component: Pagination,
  tags: ['autodocs'],
} satisfies Meta<typeof Pagination>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {
  args: {
    ariaLabel: 'Results pages',
    nextAriaLabel: 'Go to next page',
    nextLabel: 'Next',
    page: 1,
    pageCount: 1,
    onPageChange: () => undefined,
    previousAriaLabel: 'Go to previous page',
    previousLabel: 'Previous',
  },
  render: (() => ({
    Component: PaginationStory,
  })) as unknown as Story['render'],
};
