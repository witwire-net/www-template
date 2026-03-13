import WebhookList from './WebhookList.svelte';

import type { Meta, StoryObj } from '@storybook/svelte-vite';

const meta = {
  title: 'Integrations/WebhookList',
  component: WebhookList,
  tags: ['autodocs'],
  parameters: {
    docs: {
      description: {
        component:
          'Webhook rows now support richer status objects, multiple actions, and arbitrary metadata instead of a single hard-coded events string.',
      },
    },
  },
} satisfies Meta<typeof WebhookList>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {
  args: {
    webhooks: [
      {
        id: '1',
        title: 'Production deploys',
        endpoint: 'https://example.com/webhooks/deploys',
        summary: 'Routes deployment notifications into the release pipeline.',
        status: { label: 'Healthy', tone: 'success' },
        metadata: [
          { label: 'Events', value: 'deploy, incident' },
          { label: 'Last delivery', value: '2 minutes ago' },
        ],
        actions: ['Pause', 'Replay'],
      },
      {
        id: '2',
        title: 'On-call alerts',
        endpoint: 'https://example.com/webhooks/alerts',
        summary: 'Posts incident escalations to the on-call rotation.',
        status: { label: 'Paused', tone: 'warning' },
        metadata: [
          { label: 'Events', value: 'alerts' },
          { label: 'Retries', value: '3 queued' },
        ],
        actions: ['Resume'],
      },
    ],
  },
};
