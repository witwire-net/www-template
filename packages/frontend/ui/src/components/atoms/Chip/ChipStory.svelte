<script lang="ts">
  import Chip from './Chip.svelte';

  let {
    variant = 'default',
    size = 'md',
    removable = false,
    text = 'Chip',
  }: {
    removable?: boolean;
    size?: 'sm' | 'md';
    text?: string;
    variant?: 'default' | 'primary' | 'success' | 'warning' | 'error' | 'info';
  } = $props();
</script>

<Chip {variant} {size} onRemove={removable ? (() => undefined) : undefined}>{text}</Chip>
