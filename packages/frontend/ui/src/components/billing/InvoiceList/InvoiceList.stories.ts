import InvoiceList from './InvoiceList.svelte';
import InvoiceListStory from './InvoiceListStory.svelte';

import type { Meta, StoryObj } from '@storybook/svelte-vite';

const meta = {
  title: 'Billing/InvoiceList',
  component: InvoiceList,
  tags: ['autodocs'],
  parameters: {
    layout: 'padded',
    docs: {
      description: {
        component:
          'This stays intentionally invoice-specific. The row schema now supports labeled dates, richer status tones, and supplemental invoice metadata instead of assuming a single fixed billing export shape.',
      },
    },
  },
} satisfies Meta<typeof InvoiceList>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {
  render: (() => ({
    Component: InvoiceListStory as typeof InvoiceList,
  })) as unknown as Story['render'],
};
