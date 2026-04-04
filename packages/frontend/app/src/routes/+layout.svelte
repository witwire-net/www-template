<script lang="ts">
  import type { Snippet } from 'svelte';

  import '@www-template/ui/styles';

  let { children }: { children: Snippet } = $props();
</script>

{@render children()}
