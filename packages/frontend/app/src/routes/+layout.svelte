<script lang="ts">
  import type { Snippet } from 'svelte';
  import { useObservability } from '@www-template/domain';

  import '@www-template/ui/styles';

  let { children }: { children: Snippet } = $props();

  useObservability('www-template-app');
</script>

{@render children()}
