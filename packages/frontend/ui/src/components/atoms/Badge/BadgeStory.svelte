<script lang="ts">
  import Badge from './Badge.svelte';

  let {
    variant = 'neutral',
    size = 'md',
    text = 'Badge',
  }: {
    size?: 'sm' | 'md';
    text?: string;
    variant?: 'primary' | 'neutral' | 'success' | 'warning' | 'error' | 'info';
  } = $props();
</script>

<Badge {variant} {size}>{text}</Badge>
