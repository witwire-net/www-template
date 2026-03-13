import InlineFormStory from '@ui/components/form/story-support/InlineFormStory.svelte';

import InlineForm from './InlineForm.svelte';

import type { Meta, StoryObj } from '@storybook/svelte-vite';

const meta = {
  title: 'Form/InlineForm',
  component: InlineForm,
  tags: ['autodocs'],
} satisfies Meta<typeof InlineForm>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {
  render: (args) => ({
    Component: InlineFormStory,
    props: args,
  }),
  args: {
    value: '',
    inputProps: {
      label: 'Search',
      name: 'query',
      placeholder: 'Search docs',
      type: 'search',
    },
    submitLabel: 'Run',
    submitButtonProps: {
      variant: 'secondary',
      size: 'sm',
    },
    trailingAction: 'Press enter to submit',
  },
};
