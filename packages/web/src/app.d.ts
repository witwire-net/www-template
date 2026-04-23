declare global {
  namespace App {
    interface Error {
      message: string;
    }
  }
}

declare module '$env/static/public' {
  /** Base URL of the SvelteKit SPA (frontend/app). Separate domain in production. Does not include the /app base path suffix. */
  export const PUBLIC_APP_URL: string;
}

export {};
