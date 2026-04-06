/**
 * Protected passkey management route.
 * Declares this page as a no-store auth surface.
 * listPasskeys() is called from the page component on mount (CSR-only SPA; no server load).
 */
export const _AUTH_ROUTE_CACHE_POLICY = 'no-store' as const;
