import FormFieldStory from '@ui/components/form/story-support/FormFieldStory.svelte';

import FormField from './FormField.svelte';

import type { Meta, StoryObj } from '@storybook/svelte-vite';

const meta = {
  title: 'Form/FormField',
  component: FormField,
  tags: ['autodocs'],
} satisfies Meta<typeof FormField>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {
  render: (args) => ({
    Component: FormFieldStory,
    props: args,
  }),
  args: {
    label: 'Email',
    helperText: 'We will never share your email',
  },
};
