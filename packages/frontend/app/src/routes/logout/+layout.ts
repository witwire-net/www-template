/**
 * logout route の no-store metadata。
 * /logout は public utility route だが auth surface として no-store を維持する。
 */
export const _AUTH_ROUTE_CACHE_POLICY = 'no-store' as const;
