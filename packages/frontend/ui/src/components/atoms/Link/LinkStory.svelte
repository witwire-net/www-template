<script lang="ts">
  import Link from './Link.svelte';

  let {
    href = '/learn-more',
    text = 'Learn more',
    variant = 'default',
    underline = 'hover',
  }: {
    href?: string;
    text?: string;
    underline?: 'none' | 'hover' | 'always';
    variant?: 'default' | 'primary' | 'muted' | 'ghost';
  } = $props();
</script>

<Link {href} {variant} {underline}>{text}</Link>
