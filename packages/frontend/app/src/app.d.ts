declare global {
  namespace App {
    interface Error {
      message: string;
    }
  }
}

declare module '$env/static/public' {
  /** OpenTelemetry collector URL exposed to the browser. */
  export const PUBLIC_OTEL_COLLECTOR_URL: string;
}

export {};
