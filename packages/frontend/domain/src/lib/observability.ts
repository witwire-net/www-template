import { propagation } from '@opentelemetry/api';
import { W3CTraceContextPropagator } from '@opentelemetry/core';
import { OTLPTraceExporter } from '@opentelemetry/exporter-trace-otlp-http';
import { registerInstrumentations } from '@opentelemetry/instrumentation';
import { FetchInstrumentation } from '@opentelemetry/instrumentation-fetch';
import { resourceFromAttributes } from '@opentelemetry/resources';
import { BatchSpanProcessor, WebTracerProvider } from '@opentelemetry/sdk-trace-web';
import { ATTR_SERVICE_NAME, ATTR_SERVICE_VERSION } from '@opentelemetry/semantic-conventions';

/**
 * Initialize OpenTelemetry browser tracing for the given service.
 * @param serviceName - The service name to attribute traces to.
 */
export function initObservability(serviceName: string): void {
  const collectorUrl = getCollectorUrl();

  const exporter = new OTLPTraceExporter({
    url: collectorUrl,
  });

  const provider = new WebTracerProvider({
    resource: resourceFromAttributes({
      [ATTR_SERVICE_NAME]: serviceName,
      [ATTR_SERVICE_VERSION]: '0.1.0',
      'deployment.environment': 'development',
    }),
    spanProcessors: [new BatchSpanProcessor(exporter)],
  });

  provider.register();

  propagation.setGlobalPropagator(new W3CTraceContextPropagator());

  registerInstrumentations({
    instrumentations: [
      new FetchInstrumentation({
        propagateTraceHeaderCorsUrls: [/.*/],
      }),
    ],
  });

  const tracer = provider.getTracer(serviceName);

  const reportError = (event: ErrorEvent | PromiseRejectionEvent) => {
    const span = tracer.startSpan('browser.error');
    span.setAttribute(
      'error.type',
      event instanceof ErrorEvent ? event.type : 'unhandledrejection'
    );

    if (event instanceof ErrorEvent) {
      const err = event.error as unknown;
      if (err instanceof Error) {
        span.recordException(err);
      } else {
        span.recordException(new Error(event.message));
      }
    } else {
      const reason = event.reason as unknown;
      let err: Error;
      if (typeof reason === 'object' && reason !== null && reason instanceof Error) {
        err = reason;
      } else {
        err = new Error(String(reason));
      }
      span.recordException(err);
    }

    span.end();
  };

  window.addEventListener('error', reportError);
  window.addEventListener('unhandledrejection', reportError as EventListener);
}

function getCollectorUrl(): string {
  if (typeof import.meta !== 'undefined') {
    const meta = import.meta as { env: Record<string, string | undefined> };
    return meta.env.PUBLIC_OTEL_COLLECTOR_URL ?? 'http://localhost:4318/v1/traces';
  }
  return 'http://localhost:4318/v1/traces';
}
