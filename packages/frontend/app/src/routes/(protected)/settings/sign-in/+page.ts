/**
 * 認証済みパスキー・デバイス管理ルート。
 * このページを no-store 認証サーフェスとして宣言する。
 * PasskeyList と DeviceManager はページコンポーネントのマウント時に呼び出される（CSR 専用 SPA。サーバーロードなし）。
 */
export const _AUTH_ROUTE_CACHE_POLICY = 'no-store' as const;
