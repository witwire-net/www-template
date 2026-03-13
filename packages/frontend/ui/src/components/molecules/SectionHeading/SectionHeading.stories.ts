import SectionHeadingSnippetStory from '@ui/story-support/components/molecules/SectionHeading/SectionHeadingSnippetStory.svelte';

import SectionHeading from './SectionHeading.svelte';

import type { Meta, StoryObj } from '@storybook/svelte-vite';

const meta = {
  title: 'Molecules/SectionHeading',
  component: SectionHeading,
  tags: ['autodocs'],
  parameters: {
    docs: {
      description: {
        component:
          'Use `eyebrow`, `title`, and `description` for simple fallback text, or switch to the migrated Svelte 5 `*Content` snippet props for rich content.',
      },
    },
  },
  argTypes: {
    eyebrow: {
      control: 'text',
      description: 'Fallback eyebrow text or number.',
      table: {
        category: 'Fallback props',
      },
    },
    title: {
      control: 'text',
      description: 'Fallback title text or number.',
      table: {
        category: 'Fallback props',
      },
    },
    description: {
      control: 'text',
      description: 'Fallback description text or number.',
      table: {
        category: 'Fallback props',
      },
    },
    eyebrowContent: {
      control: false,
      description: 'Migrated Svelte 5 snippet override for the eyebrow content.',
      table: {
        category: 'Snippet props',
        type: {
          summary: 'Snippet',
        },
      },
    },
    titleContent: {
      control: false,
      description: 'Migrated Svelte 5 snippet override for the title content.',
      table: {
        category: 'Snippet props',
        type: {
          summary: 'Snippet',
        },
      },
    },
    descriptionContent: {
      control: false,
      description: 'Migrated Svelte 5 snippet override for the description content.',
      table: {
        category: 'Snippet props',
        type: {
          summary: 'Snippet',
        },
      },
    },
  },
} satisfies Meta<typeof SectionHeading>;

export default meta;

type Story = StoryObj<typeof meta>;

const defaultArgs = {
  eyebrow: 'Overview',
  title: 'Reusable heading for section-level content',
  description: 'Use this component to keep heading hierarchy and spacing consistent.',
};

export const Default: Story = {
  args: defaultArgs,
};

export const CenterAligned: Story = {
  args: {
    ...defaultArgs,
    align: 'center',
  },
};

export const WithSnippetContent: Story = {
  render: (args) => ({
    Component: SectionHeadingSnippetStory,
    props: args,
  }),
  args: {
    align: 'left',
    titleAs: 'h2',
  },
};
