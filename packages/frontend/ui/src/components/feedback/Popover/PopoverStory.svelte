<svelte:options runes={true} />

<script lang="ts">
  import Button from '@ui/components/atoms/Button/Button.svelte';

  import Popover from './Popover.svelte';
</script>

<Popover>
  {#snippet trigger()}
    <Button size="sm">Open</Button>
  {/snippet}

  {#snippet content()}
    <div style="font-size: 0.875rem;">Popover content</div>
  {/snippet}
</Popover>
