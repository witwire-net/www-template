<script lang="ts">
  import Button from '@ui/components/atoms/Button/Button.svelte';
  import type { Snippet } from 'svelte';

  import APIKeyManager from './APIKeyManager.svelte';

  interface ApiKeyStoryItem {
    id: string | number;
    name: string;
    maskedValue: string;
    date?: {
      label?: string;
      value: string;
    };
    metadata?: readonly {
      label?: string;
      value: string;
    }[];
    actions?: readonly (Snippet | string | number)[];
  }

  interface Props {
    keys?: readonly ApiKeyStoryItem[];
  }

  const defaultKeys: ApiKeyStoryItem[] = [
    {
      id: 'primary',
      name: 'Primary workspace',
      maskedValue: 'sk_live_****9X2P',
      date: { label: 'Rotated', value: '2026-02-18' },
      metadata: [{ label: 'Scope', value: 'deployments.write' }],
      actions: [rotatePrimaryKey, 'Audit'],
    },
    {
      id: 'sandbox',
      name: 'Sandbox',
      maskedValue: 'sk_test_****1A7M',
      date: { label: 'Created', value: '2026-03-01' },
      metadata: [{ label: 'Owner', value: 'Developer experience' }],
      actions: [copySandboxKey],
    },
  ];

  let { keys = [] }: Props = $props();

  const resolvedKeys = $derived(keys.length > 0 ? keys : defaultKeys);
</script>

<APIKeyManager keys={resolvedKeys} />

{#snippet rotatePrimaryKey()}
  <Button size="sm" type="button">Rotate key</Button>
{/snippet}

{#snippet copySandboxKey()}
  <Button size="sm" variant="ghost" type="button">Copy token</Button>
{/snippet}
