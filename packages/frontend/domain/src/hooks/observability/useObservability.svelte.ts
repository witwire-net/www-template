import { onMount } from 'svelte';

import { initObservability } from '../../lib/observability';

type ObservabilityData = object;

type ObservabilityActions = object;

/**
 * Initialize OpenTelemetry browser tracing on mount.
 * @param serviceName - The service name for tracing attribution.
 * @param collectorUrl - The OTLP HTTP collector endpoint URL.
 */
export function useObservability(
  serviceName: string,
  collectorUrl: string
): {
  data: ObservabilityData;
  actions: ObservabilityActions;
} {
  onMount(() => {
    initObservability(serviceName, collectorUrl);
  });

  return { data: {}, actions: {} };
}
