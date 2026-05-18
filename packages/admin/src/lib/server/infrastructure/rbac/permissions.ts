/**
 * 8 つの権限定数と 3 ロールの権限マッピング（OpenSpec 定義に準拠）。
 */
export const PERMISSIONS = [
  'accounts:read',
  'accounts:suspend',
  'accounts:restore',
  'audit:read',
  'operators:read',
  'operators:write',
  'operators:setup_token',
  'operators:deactivate',
] as const;

/**
 * 権限名の型。
 */
export type Permission = (typeof PERMISSIONS)[number];

/**
 * ロールごとの権限マップ。
 * - admin: 全権限あり
 * - operator: アカウント読取・停止・復元 + 監査読取（オペレーター管理なし）
 * - viewer: アカウント読取 + 監査読取のみ
 */
export const ROLE_PERMISSIONS = new Map<string, Map<Permission, boolean>>([
  [
    'admin',
    new Map([
      ['accounts:read', true],
      ['accounts:suspend', true],
      ['accounts:restore', true],
      ['audit:read', true],
      ['operators:read', true],
      ['operators:write', true],
      ['operators:setup_token', true],
      ['operators:deactivate', true],
    ]),
  ],
  [
    'operator',
    new Map([
      ['accounts:read', true],
      ['accounts:suspend', true],
      ['accounts:restore', true],
      ['audit:read', true],
      ['operators:read', false],
      ['operators:write', false],
      ['operators:setup_token', false],
      ['operators:deactivate', false],
    ]),
  ],
  [
    'viewer',
    new Map([
      ['accounts:read', true],
      ['accounts:suspend', false],
      ['accounts:restore', false],
      ['audit:read', true],
      ['operators:read', false],
      ['operators:write', false],
      ['operators:setup_token', false],
      ['operators:deactivate', false],
    ]),
  ],
]);
