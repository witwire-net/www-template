<svelte:options runes={true} />

<script lang="ts">
  import Button from '@ui/components/atoms/Button/Button.svelte';

  import Tooltip from './Tooltip.svelte';
</script>

<Tooltip content="Copied to clipboard">
  <Button size="sm">Hover me</Button>
</Tooltip>
