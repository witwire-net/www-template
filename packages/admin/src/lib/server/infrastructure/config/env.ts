/**
 * 必須環境変数を検証し、型付き設定オブジェクトを返す。
 * Product と Admin が共有する Valkey インフラの DB 番号分離と、OpenSearch prefix の衝突を防ぐ。
 *
 * @returns 型付き環境設定オブジェクト
 * @throws Error 必須環境変数が欠落している場合、またはセキュリティ制約に違反する場合
 */
export function getEnvConfig() {
  const jwtSecret = requireEnv('JWT_SECRET');
  const adminOrigin = requireEnv('ADMIN_ORIGIN');
  const adminDatabaseUrl = requireEnv('ADMIN_DATABASE_URL');
  const productDatabaseUrl = requireEnv('PRODUCT_DATABASE_URL');
  const adminValkeyUrl = requireEnv('ADMIN_VALKEY_URL');
  const opensearchUrl = requireEnv('OPENSEARCH_URL');
  const adminOpensearchAuditReplicas = requireEnv('ADMIN_OPENSEARCH_AUDIT_REPLICAS');
  const adminOpensearchAuditIndexPrefix = requireEnv('ADMIN_OPENSEARCH_AUDIT_INDEX_PREFIX');
  const productOpensearchIndexPrefix = requireEnv('PRODUCT_OPENSEARCH_INDEX_PREFIX');
  const adminBootstrapEnabled = requireEnv('ADMIN_BOOTSTRAP_ENABLED');
  const adminBootstrapSecretHash = requireEnv('ADMIN_BOOTSTRAP_SECRET_HASH');
  const adminBootstrapExpiresAt = requireEnv('ADMIN_BOOTSTRAP_EXPIRES_AT');

  // Product runtime と同一 Valkey infrastructure を使い、DB 番号だけを分ける契約を起動時に検証する。
  validateSharedValkeyDbSeparation(
    adminValkeyUrl,
    process.env.VALKEY_URL ?? process.env.PRODUCT_VALKEY_URL ?? ''
  );

  // OpenSearch prefix の衝突検証
  const adminPrefix = adminOpensearchAuditIndexPrefix;
  const productPrefix = productOpensearchIndexPrefix;
  if (adminPrefix === productPrefix) {
    throw new Error('Admin audit prefix must not equal Production prefix.');
  }
  if (adminPrefix.includes(productPrefix) || productPrefix.includes(adminPrefix)) {
    throw new Error('Admin audit prefix and Production prefix must not contain each other.');
  }

  return {
    jwtSecret,
    adminOrigin,
    adminDatabaseUrl,
    productDatabaseUrl,
    adminValkeyUrl,
    opensearchUrl,
    adminOpensearchAuditReplicas: parseInt(adminOpensearchAuditReplicas, 10),
    adminOpensearchAuditIndexPrefix: adminPrefix,
    productOpensearchIndexPrefix: productPrefix,
    adminBootstrapEnabled: adminBootstrapEnabled === 'true',
    adminBootstrapSecretHash,
    adminBootstrapExpiresAt: new Date(adminBootstrapExpiresAt),
  };
}

function validateSharedValkeyDbSeparation(adminValkeyUrl: string, productValkeyUrl: string): void {
  // Product 側 URL が注入されない runtime でも Admin URL 自体は明示 DB 番号を必須にする。
  const admin = parseValkeyUrl(adminValkeyUrl, 'ADMIN_VALKEY_URL');
  if (productValkeyUrl === '') {
    return;
  }
  const product = parseValkeyUrl(productValkeyUrl, 'VALKEY_URL');
  if (admin.infrastructureKey !== product.infrastructureKey) {
    throw new Error('Admin and Product Valkey must share the same infrastructure endpoint.');
  }
  if (admin.db === product.db) {
    throw new Error('Admin and Product Valkey must use different logical DB numbers.');
  }
}

function parseValkeyUrl(value: string, name: string): { infrastructureKey: string; db: number } {
  let parsed: URL;
  try {
    parsed = new URL(value);
  } catch {
    throw new Error(`Invalid ${name}: must be a valid redis URL.`);
  }
  const rawDb = parsed.pathname.replace(/^\//u, '');
  if (!/^\d+$/u.test(rawDb)) {
    throw new Error(`${name} must include an explicit logical DB number.`);
  }
  return {
    infrastructureKey: [
      parsed.protocol,
      parsed.username,
      parsed.password,
      parsed.hostname,
      parsed.port,
    ].join('|'),
    db: Number(rawDb),
  };
}

function requireEnv(name: string): string {
  const entry = Object.entries(process.env).find(([k]) => k === name);
  const value = entry?.[1];
  if (value === undefined || value === '') {
    throw new Error(`Missing required environment variable: ${name}`);
  }
  return value;
}
