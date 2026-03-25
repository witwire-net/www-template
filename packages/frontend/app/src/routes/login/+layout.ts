/**
 * auth route の no-store metadata。
 * SvelteKit CSR route は HTTP response header を直接制御できないため、
 * route-level metadata として no-store intent を宣言する。
 * Cloudflare/WAF 側の Cache-Control: no-store と併せて auth surface を保護する。
 */
export const _AUTH_ROUTE_CACHE_POLICY = 'no-store' as const;
