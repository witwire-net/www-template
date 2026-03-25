<svelte:options runes={true} />

<script lang="ts">
  import Button from '@ui/components/atoms/Button/Button.svelte';

  import EmptyState from './EmptyState.svelte';

  let status = $state('Nothing refreshed yet');
</script>

<div style="display: flex; flex-direction: column; gap: 1rem; align-items: center;">
  <EmptyState title="No messages yet" description="New updates will appear here.">
    {#snippet icon()}
      <span role="img" aria-label="Inbox">📥</span>
    {/snippet}
    {#snippet action()}
      <Button
        size="sm"
        onclick={() => {
          status = 'Messages refreshed';
        }}
      >
        Refresh
      </Button>
    {/snippet}
  </EmptyState>
  <p style="margin: 0; color: var(--color-text-muted);">Status: {status}</p>
</div>
