import { getAdminPlatformConfig } from './env.js';

/**
 * ランタイムプラットフォーム設定の検証。
 * WebAuthn RP 設定は Admin TOML から、Node 環境は process env から検証する。
 *
 * @returns 型付きプラットフォーム設定オブジェクト
 * @throws Error 必須設定値が欠落している場合、または NODE_ENV が不正な場合
 */
export function getPlatformConfig() {
  // WebAuthn RP は Admin surface の固定契約なので、個別 env ではなく Admin TOML から取得する。
  const { adminRpId, adminRpName } = getAdminPlatformConfig();

  // NODE_ENV は Node/SvelteKit runtime が提供する実行モードなので、ここだけ process env を直接検証する。
  const rawNodeEnv = process.env.NODE_ENV ?? 'development';
  const allowedNodeEnvs = ['development', 'production', 'test'] as const;
  if (!allowedNodeEnvs.includes(rawNodeEnv as (typeof allowedNodeEnvs)[number])) {
    throw new Error(
      `Invalid NODE_ENV: "${rawNodeEnv}". Must be one of: ${allowedNodeEnvs.join(', ')}`
    );
  }
  const nodeEnv = rawNodeEnv as (typeof allowedNodeEnvs)[number];
  const isProduction = nodeEnv === 'production';

  return {
    adminRpId,
    adminRpName,
    nodeEnv,
    isProduction,
  };
}
