<svelte:options runes={true} />

<script lang="ts">
  import Button from '@ui/components/atoms/Button/Button.svelte';

  import Dialog from './Dialog.svelte';

  let open = $state(false);
  let status = $state('No action yet');
</script>

<div style="padding: 1.5rem;">
  <Button
    onclick={() => {
      open = true;
    }}
  >
    Open Dialog
  </Button>
  <p style="margin-top: 1rem; color: #666;">Result: {status}</p>
  <Dialog
    {open}
    onClose={() => {
      status = 'Dialog closed';
      open = false;
    }}
    title="Invite member"
    description="Send an invitation to join your workspace."
  >
    <div>Dialog body content.</div>
    {#snippet actions()}
      <Button
        variant="ghost"
        onclick={() => {
          status = 'Invitation canceled';
          open = false;
        }}
      >
        Cancel
      </Button>
      <Button
        onclick={() => {
          status = 'Invitation sent';
          open = false;
        }}
      >
        Send invite
      </Button>
    {/snippet}
  </Dialog>
</div>
