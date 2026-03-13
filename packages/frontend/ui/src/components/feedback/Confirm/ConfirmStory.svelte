<svelte:options runes={true} />

<script lang="ts">
  import Button from '@ui/components/atoms/Button/Button.svelte';

  import Confirm from './Confirm.svelte';

  let open = $state(false);
  let status = $state('No action yet');
</script>

<div style="padding: 1.5rem;">
  <Button
    variant="danger"
    onclick={() => {
      open = true;
    }}
  >
    Delete project
  </Button>
  <p style="margin-top: 1rem; color: #666;">Result: {status}</p>
  <Confirm
    {open}
    title="Delete project"
    description="This action cannot be undone."
    onConfirm={() => {
      status = 'Project deleted';
      open = false;
    }}
    onCancel={() => {
      status = 'Deletion cancelled';
      open = false;
    }}
  />
</div>
