<script lang="ts">
  import type { Snippet } from 'svelte';
  import { useObservability } from '@www-template/domain';
  import { PUBLIC_OTEL_COLLECTOR_URL } from '$env/static/public';

  import '@www-template/ui/styles.css';
  import '../app.css';

  let { children }: { children: Snippet } = $props();

  useObservability('www-template-app', PUBLIC_OTEL_COLLECTOR_URL);
</script>

{@render children()}
