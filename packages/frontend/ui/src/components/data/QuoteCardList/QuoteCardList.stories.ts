import QuoteCardList from './QuoteCardList.svelte';

import type { Meta, StoryObj } from '@storybook/svelte-vite';

const meta = {
  title: 'Data/QuoteCardList',
  component: QuoteCardList,
  tags: ['autodocs'],
  parameters: {
    layout: 'padded',
  },
} satisfies Meta<typeof QuoteCardList>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {
  args: {
    items: [
      {
        quote: 'We shipped the new dashboard without losing sight of the system constraints.',
        name: 'Aoi Tanaka',
        role: 'Product Designer',
      },
      {
        quote: 'Having the same card shell across views made the migration much easier to verify.',
        name: 'Ken Ito',
        role: 'Frontend Engineer',
      },
      {
        quote: 'The local story smoke test helped us catch regressions before they spread.',
        name: 'Mika Sato',
        role: 'QA Lead',
      },
    ],
  },
};
