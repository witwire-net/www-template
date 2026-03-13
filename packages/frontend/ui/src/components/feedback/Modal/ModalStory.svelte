<svelte:options runes={true} />

<script lang="ts">
  import Button from '@ui/components/atoms/Button/Button.svelte';

  import Modal from './Modal.svelte';

  let open = $state(true);
</script>

<Button
  onclick={() => {
    open = true;
  }}
>
  Open Modal
</Button>

<Modal
  {open}
  onClose={() => {
    open = false;
  }}
>
  <div style="display: flex; flex-direction: column; gap: 1rem;">
    <h3 style="margin: 0;">Modal title</h3>
    <p style="margin: 0;">Modal content goes here.</p>
    <div>
      <Button
        onclick={() => {
          open = false;
        }}
      >
        Close
      </Button>
    </div>
  </div>
</Modal>
