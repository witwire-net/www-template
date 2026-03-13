<svelte:options runes={true} />

<script lang="ts">
  import Button from '@ui/components/atoms/Button/Button.svelte';

  import Snackbar from './Snackbar.svelte';

  let visible = $state(false);
  let status = $state('No recent action');
</script>

<div style="padding: 1.5rem;">
  <Button
    onclick={() => {
      status = 'Changes saved';
      visible = true;
    }}
  >
    Save Changes
  </Button>
  <p style="margin-top: 1rem; color: #666;">Status: {status}</p>
  {#if visible}
    <div style="margin-top: 1rem;">
      <Snackbar
        message="Changes saved"
        onClose={() => {
          visible = false;
        }}
      >
        {#snippet action()}
          <Button
            size="sm"
            variant="ghost"
            onclick={() => {
              status = 'Changes reverted';
              visible = false;
            }}
          >
            Undo
          </Button>
        {/snippet}
      </Snackbar>
    </div>
  {/if}
</div>
