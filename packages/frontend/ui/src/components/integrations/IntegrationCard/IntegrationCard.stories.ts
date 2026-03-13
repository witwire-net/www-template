import IntegrationCard from './IntegrationCard.svelte';
import IntegrationCardSnippetStory from './IntegrationCardSnippetStory.svelte';

import type { Meta, StoryObj } from '@storybook/svelte-vite';

const meta = {
  title: 'Integrations/IntegrationCard',
  component: IntegrationCard,
  tags: ['autodocs'],
  parameters: {
    docs: {
      description: {
        component:
          '`icon` and `action` accept fallback values or migrated Svelte 5 snippets. Use the `beta` status to document preview integrations.',
      },
    },
  },
  argTypes: {
    status: {
      control: 'select',
      options: ['available', 'connected', 'beta'],
      description: 'Integration availability state.',
    },
    icon: {
      control: 'text',
      description: 'Fallback icon text/number or a Svelte 5 snippet.',
      table: {
        type: {
          summary: 'Snippet | string | number',
        },
      },
    },
    action: {
      control: 'text',
      description: 'Fallback action text/number or a Svelte 5 snippet.',
      table: {
        type: {
          summary: 'Snippet | string | number',
        },
      },
    },
  },
} satisfies Meta<typeof IntegrationCard>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {
  args: {
    name: 'Slack',
    description: 'Send updates to your team channels.',
    status: 'available',
    icon: 'S',
    action: 'Connect',
  },
};

export const Connected: Story = {
  args: {
    name: 'GitHub',
    description: 'Sync repositories and deployment events.',
    status: 'connected',
    icon: 'G',
    action: 'Manage',
  },
};

export const WithSnippetContent: Story = {
  render: (args) => ({
    Component: IntegrationCardSnippetStory,
    props: args,
  }),
  args: {
    name: 'Linear',
    description: 'Keep issue status and incident follow-up in sync.',
    status: 'connected',
  },
};

export const Beta: Story = {
  render: (args) => ({
    Component: IntegrationCardSnippetStory,
    props: args,
  }),
  args: {
    name: 'Salesforce',
    description: 'Preview account signals before the CRM sync rollout is complete.',
    status: 'beta',
  },
};
