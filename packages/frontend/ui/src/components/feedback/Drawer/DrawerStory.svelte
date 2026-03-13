<svelte:options runes={true} />

<script lang="ts">
  import Button from '@ui/components/atoms/Button/Button.svelte';

  import Drawer from './Drawer.svelte';

  let open = $state(false);
</script>

<div style="padding: 1.5rem;">
  <Button
    onclick={() => {
      open = true;
    }}
  >
    Open Drawer
  </Button>
</div>

<Drawer
  {open}
  onClose={() => {
    open = false;
  }}
>
  <div style="padding: 1.5rem; display: flex; flex-direction: column; gap: 1rem;">
    <h3 style="margin: 0;">Drawer title</h3>
    <p style="margin: 0;">Drawer content goes here.</p>
    <div>
      <Button
        variant="ghost"
        size="sm"
        onclick={() => {
          open = false;
        }}
      >
        Close
      </Button>
    </div>
  </div>
</Drawer>
