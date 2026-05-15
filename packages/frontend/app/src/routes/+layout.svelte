<script lang="ts">
  import type { Snippet } from 'svelte';
  import { useObservability } from '@www-template/domain';
  import { env } from '$env/dynamic/public';

  import '@www-template/ui/styles.css';
  import '../app.css';

  let { children }: { children: Snippet } = $props();

  const otelCollectorUrl = env.PUBLIC_OTEL_COLLECTOR_URL ?? 'http://localhost:4318/v1/traces';

  useObservability('www-template-app', otelCollectorUrl);
</script>

{@render children()}
