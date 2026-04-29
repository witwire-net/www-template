import { onMount } from 'svelte';

import { initObservability } from '../../lib/observability';

type ObservabilityData = object;

type ObservabilityActions = object;

/**
 * Initialize OpenTelemetry browser tracing on mount.
 * @param serviceName - The service name for tracing attribution.
 */
export function useObservability(serviceName: string): {
  data: ObservabilityData;
  actions: ObservabilityActions;
} {
  onMount(() => {
    initObservability(serviceName);
  });

  return { data: {}, actions: {} };
}
