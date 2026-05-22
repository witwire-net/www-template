import { existsSync, readFileSync } from 'node:fs';
import { isAbsolute, resolve } from 'node:path';

type AdminTomlPrimitive = string | number | boolean;
type AdminTomlDocument = Map<string, Map<string, AdminTomlPrimitive>>;

/**
 * Admin 認証境界で使う設定を検証して返す。
 *
 * @returns Admin JWT 署名鍵、公開 Origin、Admin 用 Valkey URL
 * @throws Error 必須設定値が欠落している場合、または Valkey 分離契約に違反する場合
 */
export function getAdminAuthConfig(): {
  jwtSecret: string;
  adminOrigin: string;
  adminValkeyUrl: string;
} {
  // Admin auth は Product の JWT や session と共有しないため、Admin 専用 TOML の secret だけを信頼する。
  const document = loadAdminTomlDocument();
  const jwtSecret = requireTomlString(document, 'auth', 'jwt_secret');
  const adminOrigin = requireTomlString(document, 'server', 'origin');
  const adminValkeyUrl = requireTomlString(document, 'valkey', 'admin_url');
  const productValkeyUrl = requireTomlString(document, 'valkey', 'product_url');

  // Admin と Product は同一 Valkey infrastructure の別 logical DB でだけ共有を許可し、key 空間衝突を fail-close で防ぐ。
  validateSharedValkeyDbSeparation(adminValkeyUrl, productValkeyUrl);

  return { jwtSecret, adminOrigin, adminValkeyUrl };
}

/**
 * Admin DB 接続用の設定を検証して返す。
 *
 * @returns Admin DB 接続 URL
 * @throws Error `database.admin_url` が欠落している場合
 */
export function getAdminDatabaseConfig(): { adminDatabaseUrl: string } {
  // Admin operator / passkey / audit は Admin 専用 DB にだけ保存する。
  return { adminDatabaseUrl: requireTomlString(loadAdminTomlDocument(), 'database', 'admin_url') };
}

/**
 * Product DB 管理連携用の設定を検証して返す。
 *
 * @returns Product DB への最小権限接続 URL
 * @throws Error `database.product_url` が欠落している場合
 */
export function getProductDatabaseConfig(): { productDatabaseUrl: string } {
  // Product アカウント管理は admin_view/admin_op 用の最小権限 login role だけで接続する。
  return {
    productDatabaseUrl: requireTomlString(loadAdminTomlDocument(), 'database', 'product_url'),
  };
}

/**
 * Admin 初回 bootstrap 用の設定を検証して返す。
 *
 * @returns bootstrap gate と secret hash、有効期限
 * @throws Error bootstrap 関連の必須設定値が欠落している場合、または有効期限が不正な場合
 */
export function getAdminBootstrapConfig(): {
  adminBootstrapEnabled: boolean;
  adminBootstrapSecretHash: string;
  adminBootstrapExpiresAt: Date;
} {
  // 初回 admin 作成は DB seed ではなく、明示 enable flag と期限付き secret hash でだけ許可する。
  const document = loadAdminTomlDocument();
  const expiresAt = new Date(requireTomlString(document, 'bootstrap', 'expires_at'));
  if (Number.isNaN(expiresAt.getTime())) {
    throw new TypeError(
      'Invalid admin config value: bootstrap.expires_at must be an ISO-8601 date.'
    );
  }

  return {
    adminBootstrapEnabled: requireTomlBoolean(document, 'bootstrap', 'enabled'),
    adminBootstrapSecretHash: requireTomlString(document, 'bootstrap', 'secret_hash'),
    adminBootstrapExpiresAt: expiresAt,
  };
}

/**
 * Admin 監査 OpenSearch 用の設定を検証して返す。
 *
 * @returns OpenSearch 接続 URL と Admin / Product namespace prefix
 * @throws Error 必須設定値が欠落している場合、または namespace が衝突する場合
 */
export function getAdminSearchConfig(): {
  opensearchUrl: string;
  adminOpensearchAuditReplicas: number;
  adminOpensearchAuditIndexPrefix: string;
  productOpensearchIndexPrefix: string;
} {
  const document = loadAdminTomlDocument();
  const opensearchUrl = requireTomlString(document, 'opensearch', 'url');
  const adminOpensearchAuditReplicas = requireTomlInteger(
    document,
    'opensearch',
    'admin_audit_replicas'
  );
  const adminPrefix = requireTomlString(document, 'opensearch', 'admin_audit_index_prefix');
  const productPrefix = requireTomlString(document, 'opensearch', 'product_index_prefix');

  // 単一 OpenSearch cluster を許容しても、Admin audit と Production domain の namespace 混在は拒否する。
  if (adminPrefix === productPrefix) {
    throw new Error('Admin audit prefix must not equal Production prefix.');
  }
  if (adminPrefix.includes(productPrefix) || productPrefix.includes(adminPrefix)) {
    throw new Error('Admin audit prefix and Production prefix must not contain each other.');
  }

  return {
    opensearchUrl,
    adminOpensearchAuditReplicas,
    adminOpensearchAuditIndexPrefix: adminPrefix,
    productOpensearchIndexPrefix: productPrefix,
  };
}

/**
 * Admin WebAuthn RP 用の設定を検証して返す。
 *
 * @returns WebAuthn RP ID と RP 表示名
 * @throws Error `auth.rp_id` または `auth.rp_name` が欠落している場合
 */
export function getAdminPlatformConfig(): { adminRpId: string; adminRpName: string } {
  // WebAuthn の RP 情報も Admin surface 固有の値なので、Admin TOML から一貫して読み込む。
  const document = loadAdminTomlDocument();
  return {
    adminRpId: requireTomlString(document, 'auth', 'rp_id'),
    adminRpName: requireTomlString(document, 'auth', 'rp_name'),
  };
}

/**
 * Admin runtime 全体の設定をまとめて検証して返す。
 *
 * @returns Admin auth / DB / Product DB / OpenSearch / bootstrap / WebAuthn の統合設定
 * @throws Error いずれかの必須設定値が欠落している場合、またはセキュリティ制約に違反する場合
 */
export function getEnvConfig() {
  // 起動時・テスト時に全契約を一括検証したい箇所のため、用途別 getter の合成だけにする。
  return {
    ...getAdminAuthConfig(),
    ...getAdminDatabaseConfig(),
    ...getProductDatabaseConfig(),
    ...getAdminSearchConfig(),
    ...getAdminBootstrapConfig(),
    ...getAdminPlatformConfig(),
  };
}

function loadAdminTomlDocument(): AdminTomlDocument {
  // Admin runtime は個別 env ではなく Admin 専用 TOML を設定の正とし、環境差分はファイルパスだけで切り替える。
  const configPath = resolveAdminConfigPath();
  const source = readFileSync(configPath, 'utf8');
  return parseAdminToml(source, configPath);
}

function resolveAdminConfigPath(): string {
  // 明示パスが指定された場合は、そのパスだけを使用して誤った fallback を防ぐ。
  const configuredPath = process.env.ADMIN_CONFIG_PATH;
  if (configuredPath !== undefined && configuredPath !== '') {
    const resolvedPath = isAbsolute(configuredPath) ? configuredPath : resolve(configuredPath);
    if (existsSync(resolvedPath)) {
      return resolvedPath;
    }
    throw new Error(`Admin config file not found: ${resolvedPath}`);
  }

  // pnpm を root から実行する場合と package directory から実行する場合の両方で local admin config を見つける。
  const candidates = [
    resolve('.config/local.admin.toml'),
    resolve('../../.config/local.admin.toml'),
  ];
  const found = candidates.find((candidate) => existsSync(candidate));
  if (found !== undefined) {
    return found;
  }

  throw new Error(
    'Admin config file not found. Set ADMIN_CONFIG_PATH or place .config/local.admin.toml at the project root.'
  );
}

function parseAdminToml(source: string, configPath: string): AdminTomlDocument {
  // Admin 設定ファイルで使う section/key/value の最小 TOML subset を deterministic に解析する。
  const document: AdminTomlDocument = new Map();
  let currentSection = '';
  const lines = source.split(/\r?\n/u);

  for (const [index, rawLine] of lines.entries()) {
    const line = rawLine.trim();
    const lineNumber = index + 1;
    const location = formatConfigLocation(configPath, lineNumber);
    if (line === '' || line.startsWith('#')) {
      continue;
    }

    const section = parseSectionHeader(line);
    if (section !== null) {
      currentSection = section;
      if (!document.has(currentSection)) {
        document.set(currentSection, new Map());
      }
      continue;
    }

    if (currentSection === '') {
      throw new Error(`Invalid admin config ${location}: key must be inside a section.`);
    }

    const separatorIndex = line.indexOf('=');
    if (separatorIndex === -1) {
      throw new Error(`Invalid admin config ${location}: expected key = value.`);
    }

    const key = line.slice(0, separatorIndex).trim();
    const rawValue = line.slice(separatorIndex + 1).trim();
    if (!/^[-_a-zA-Z0-9]+$/u.test(key)) {
      throw new Error(`Invalid admin config ${location}: invalid key name.`);
    }
    const sectionValues = document.get(currentSection);
    if (sectionValues === undefined) {
      throw new Error(`Invalid admin config ${location}: section is not initialized.`);
    }
    sectionValues.set(key, parseTomlPrimitive(rawValue, location));
  }

  return document;
}

function parseSectionHeader(line: string): string | null {
  // dotted section は使わず、Admin 設定の責務単位を top-level section に限定する。
  const sectionMatch = /^\[([-_a-zA-Z0-9]+)\]$/u.exec(line);
  const section = sectionMatch?.[1];
  return section ?? null;
}

function parseTomlPrimitive(value: string, location: string): AdminTomlPrimitive {
  // double quoted string は JSON string として解釈し、escape の誤りを設定読み込み時に検出する。
  if (value.startsWith('"') && value.endsWith('"')) {
    try {
      return JSON.parse(value) as string;
    } catch (error) {
      throw new TypeError(`Invalid admin config ${location}: invalid quoted string.`, {
        cause: error,
      });
    }
  }

  // single quoted string は dev 用 bcrypt hash などをそのまま保持するために最小処理で受け付ける。
  if (value.startsWith("'") && value.endsWith("'")) {
    return value.slice(1, -1);
  }

  if (value === 'true') {
    return true;
  }
  if (value === 'false') {
    return false;
  }
  if (/^-?\d+$/u.test(value)) {
    return Number(value);
  }

  throw new TypeError(`Invalid admin config ${location}: unsupported value syntax.`);
}

function requireTomlString(document: AdminTomlDocument, section: string, key: string): string {
  // 空文字列は secret や URL の fail-open を招くため、未設定と同じ扱いで拒否する。
  const value = readTomlValue(document, section, key);
  if (typeof value !== 'string' || value.trim() === '') {
    throw new Error(`Missing required admin config value: ${section}.${key}`);
  }
  return value.trim();
}

function requireTomlBoolean(document: AdminTomlDocument, section: string, key: string): boolean {
  // bootstrap gate は文字列 truthy を許さず、TOML boolean の true/false だけを受け入れる。
  const value = readTomlValue(document, section, key);
  if (typeof value !== 'boolean') {
    throw new TypeError(`Missing required admin config value: ${section}.${key}`);
  }
  return value;
}

function requireTomlInteger(document: AdminTomlDocument, section: string, key: string): number {
  // replica 数は OpenSearch index 設定へ直接渡るため、非負整数だけを許可する。
  const value = readTomlValue(document, section, key);
  if (typeof value !== 'number' || !Number.isInteger(value) || value < 0) {
    throw new TypeError(`Missing required admin config value: ${section}.${key}`);
  }
  return value;
}

function readTomlValue(
  document: AdminTomlDocument,
  section: string,
  key: string
): AdminTomlPrimitive | undefined {
  // Map 経由で section/key を参照し、動的 object index を避けて lint と prototype 汚染リスクを同時に潰す。
  return document.get(section)?.get(key);
}

function formatConfigLocation(configPath: string, lineNumber: number): string {
  // template literal には文字列化済みの行番号だけを渡し、strict template expression ルールを満たす。
  return `${configPath}:${String(lineNumber)}`;
}

function validateSharedValkeyDbSeparation(adminValkeyUrl: string, productValkeyUrl: string): void {
  // Product と Admin は同じ Valkey endpoint を共有しながら、logical DB 番号で認証状態を完全分離する。
  const admin = parseValkeyUrl(adminValkeyUrl, 'valkey.admin_url');
  const product = parseValkeyUrl(productValkeyUrl, 'valkey.product_url');
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
    throw new Error(`Invalid admin config value: ${name} must be a valid redis URL.`);
  }
  const rawDb = parsed.pathname.replace(/^\//u, '');
  if (!/^\d+$/u.test(rawDb)) {
    throw new Error(
      `Invalid admin config value: ${name} must include an explicit logical DB number.`
    );
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
