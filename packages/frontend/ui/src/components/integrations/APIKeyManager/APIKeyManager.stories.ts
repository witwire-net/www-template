import APIKeyManager from './APIKeyManager.svelte';
import APIKeyManagerSnippetStory from './APIKeyManagerSnippetStory.svelte';

import type { Meta, StoryObj } from '@storybook/svelte-vite';

const meta = {
  title: 'Integrations/APIKeyManager',
  component: APIKeyManager,
  tags: ['autodocs'],
  parameters: {
    docs: {
      description: {
        component:
          'API key rows now accept richer row metadata, labeled dates, and multiple actions. Legacy `label` / `maskedKey` / `createdAt` / `action` props still map through for compatibility.',
      },
    },
  },
  argTypes: {
    keys: {
      control: 'object',
      description:
        'API key rows. Prefer `name`, `maskedValue`, `date`, `metadata`, and `actions`; legacy row fields still resolve automatically.',
    },
  },
} satisfies Meta<typeof APIKeyManager>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {
  args: {
    keys: [
      {
        id: '1',
        name: 'Production',
        maskedValue: 'sk_live_****',
        description: 'Used by the main customer-facing environment.',
        date: { label: 'Rotated', value: '2025-01-03' },
        metadata: [{ label: 'Scope', value: 'deployments.read' }],
        actions: ['Rotate', 'Revoke'],
      },
      {
        id: '2',
        name: 'Staging',
        maskedValue: 'sk_test_****',
        description: 'Sandbox automation and QA smoke tests.',
        date: { label: 'Created', value: '2025-02-14' },
        metadata: [{ label: 'Owner', value: 'QA team' }],
        actions: ['Copy', 'Revoke'],
      },
    ],
  },
};

export const WithSnippetAction: Story = {
  render: (() => ({
    Component: APIKeyManagerSnippetStory,
  })) as unknown as Story['render'],
  args: {
    keys: [],
  },
};
