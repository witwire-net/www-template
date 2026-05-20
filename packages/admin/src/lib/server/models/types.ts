import type { OperatorLocale } from './operator_locale.js';

/**
 * オペレータードメイン型
 */
export interface Operator {
  id: string;
  email: string;
  displayName: string;
  role: 'admin' | 'operator' | 'viewer';
  isActive: boolean;
  locale: OperatorLocale;
  setupTokenHash: string | null;
  setupTokenExpiresAt: Date | null;
  lastLoginAt: Date | null;
  createdAt: Date;
  updatedAt: Date;
}

/**
 * 監査イベントドメイン型
 */
export interface AuditEvent {
  id: string;
  operatorId: string;
  action: string;
  targetType: string;
  targetId: string;
  details: unknown;
  outcome: 'pending' | 'succeeded' | 'failed';
  errorCode: string | null;
  ipAddress: string | null;
  createdAt: Date;
  completedAt: Date | null;
}

/**
 * アカウント概要ドメイン型（Product DB ビュー由来）
 */
export interface AccountSummary {
  id: string;
  email: string;
  status: string;
  statusReason: string | null;
  statusUpdatedAt: Date | null;
  statusUpdatedBy: string | null;
  sessionRevokedAfter: Date | null;
  createdAt: Date;
  passkeyCount: bigint;
}

/**
 * Passkey 情報ドメイン型
 */
export interface PasskeyInfo {
  id: string;
  operatorId: string;
  credentialHandle: string;
  publicKey: Uint8Array;
  signCount: bigint;
  aaguid: Uint8Array;
  backupEligible: boolean;
  backupState: boolean;
  transports: unknown;
  createdAt: Date;
}

/**
 * 検索パラメータ
 */
export interface SearchParams {
  query?: string;
  status?: string;
  limit: number;
  offset: number;
}
