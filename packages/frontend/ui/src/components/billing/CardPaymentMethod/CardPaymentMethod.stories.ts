import CardPaymentMethod from './CardPaymentMethod.svelte';
import CardPaymentMethodStory from './CardPaymentMethodStory.svelte';

import type { Meta, StoryObj } from '@storybook/svelte-vite';

const meta = {
  title: 'Billing/CardPaymentMethod',
  component: CardPaymentMethod,
  tags: ['autodocs'],
  parameters: {
    layout: 'padded',
    docs: {
      description: {
        component:
          'Renamed to match its actual scope: this component represents saved credit or debit card details, not every possible payment method type.',
      },
    },
  },
} satisfies Meta<typeof CardPaymentMethod>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {
  render: (args) => ({
    Component: CardPaymentMethodStory as typeof CardPaymentMethod,
    props: args,
  }),
  args: {
    brand: 'Visa',
    last4: '4242',
    expiry: '06/27',
    holder: 'Jordan Lee',
  },
};
