/**
 * Admin Console 全体を SSR 無効の静的 SPA として配信します。
 *
 * SvelteKit の server load / actions / hooks を使わず、Admin backend との通信は
 * browser 上の domain/api layer から same-origin `/api/v1/*` へ委譲します。
 */
export const ssr = false;

/**
 * Admin Console 全体で client-side routing を有効にします。
 *
 * Cloudflare は `/api/v1/*` を Go Admin API、それ以外を静的 asset へ振り分けるため、
 * UI 側の画面遷移は client router が担当します。
 */
export const csr = true;
