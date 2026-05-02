/**
 * Public passkey add route (OTP handoff from authenticated device).
 * No auth required — user arrives from a QR code / link with an OTP.
 * SSR/CSR mode is managed by the root layout.
 *
 * auth route として no-store cache semantics を維持し、入力された email / OTP が
 * browser cache や history を経由して漏えいしないようにする。
 */
export const _AUTH_ROUTE_CACHE_POLICY = 'no-store' as const;
