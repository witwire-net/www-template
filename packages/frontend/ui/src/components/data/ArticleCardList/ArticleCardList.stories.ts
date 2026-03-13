import ArticleCardList from './ArticleCardList.svelte';
import ArticleCardListSnippetStory from './ArticleCardListSnippetStory.svelte';

import type { Meta, StoryObj } from '@storybook/svelte-vite';

const meta = {
  title: 'Data/ArticleCardList',
  component: ArticleCardList,
  tags: ['autodocs'],
  parameters: {
    layout: 'padded',
    docs: {
      description: {
        component:
          'Each item can keep a simple `action` fallback or opt into the migrated `items[].actionContent` snippet for richer CTA content.',
      },
    },
  },
  argTypes: {
    items: {
      control: 'object',
      description:
        'Array of article cards. Each item supports `action` fallback text and the migrated `actionContent` snippet.',
    },
  },
} satisfies Meta<typeof ArticleCardList>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {
  args: {
    items: [
      {
        title: 'State of the product roadmap',
        excerpt: 'How the team balances platform work with customer-facing delivery.',
        date: 'Mar 10, 2026',
        tag: 'Product',
        action: 'Read more',
      },
      {
        title: 'Designing for internal launch speed',
        excerpt: 'Patterns that reduce handoff cost across design, frontend, and QA.',
        date: 'Mar 04, 2026',
        tag: 'Design Systems',
        action: 'Open article',
      },
      {
        title: 'What we learned from the migration',
        excerpt: 'A short retrospective on moving shared UI toward Svelte 5 runes.',
        date: 'Feb 28, 2026',
        tag: 'Engineering',
        action: 'View notes',
      },
    ],
  },
};

export const WithSnippetAction: Story = {
  render: (args) => ({
    Component: ArticleCardListSnippetStory,
    props: args,
  }),
};
