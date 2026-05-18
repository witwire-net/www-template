/**
 * ランタイムプラットフォーム環境変数の検証。
 * WebAuthn RP 設定と Node 環境を検証する。
 *
 * @returns 型付きプラットフォーム設定オブジェクト
 * @throws Error 必須環境変数が欠落している場合
 */
export function getPlatformConfig() {
  const adminRpId = process.env.ADMIN_RP_ID;
  const adminRpName = process.env.ADMIN_RP_NAME;
  if (adminRpId === undefined || adminRpId === '') {
    throw new Error('Missing required platform environment variable: ADMIN_RP_ID');
  }
  if (adminRpName === undefined || adminRpName === '') {
    throw new Error('Missing required platform environment variable: ADMIN_RP_NAME');
  }
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
