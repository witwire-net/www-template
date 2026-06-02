/**
 * Product origin-local context index の管理モジュール。
 *
 * context index は browser reload 後に memory state を復元するための非 secret hint である。
 * accessToken、refreshToken、Cookie value、setup token、recovery token は一切含まない。
 * tamper された index は fail-close で破棄し、bootstrap 時に server refresh で再検証する。
 */

const CONTEXT_INDEX_KEY = 'www-template:product:context-index';
const CONTEXT_INDEX_VERSION = 1;
const MAX_ENTRIES = 10;

/** context index の entry。token/secret を含まない非 secret hint のみ。 */
export interface ContextIndexEntry {
  /** 対象 auth context の canonical ULID。 */
  authContextId: string;
  /** 対象 session の canonical ULID。 */
  sessionId: string;
  /** entry が Product account と Admin operator のどちらを表すか。 */
  identityKind: 'account';
  /** UI 表示用の非 secret label。 */
  displayHint?: {
    label: string;
    secondaryLabel?: string;
  };
  /** context が最後に server で確認された時刻（ISO 8601）。 */
  lastSeenAt: string;
  /** context index entry を削除する目安時刻（ISO 8601）。 */
  expiresHintAt: string;
}

/** Product origin-local localStorage に保存する context index の schema。 */
export interface ContextIndex {
  version: number;
  surface: 'product';
  /** 現在アクティブな auth context の ULID。 */
  activeAuthContextId: string | null;
  /** 非 secret context entry の一覧。 */
  entries: ContextIndexEntry[];
}

const ULID_PATTERN = /^[0-9A-HJKMNP-TV-Z]{26}$/u;

/** ULID 形式かどうかを検証する。 */
function isUlid(value: unknown): boolean {
  return typeof value === 'string' && ULID_PATTERN.test(value);
}

/** ISO 8601 形式かどうかを緩く検証する。 */
function isIso8601(value: unknown): boolean {
  return typeof value === 'string' && !Number.isNaN(Date.parse(value));
}

/** 未知の値が正当な ContextIndexEntry であるか検証する。 */
function isValidContextIndexEntry(value: unknown): value is ContextIndexEntry {
  if (typeof value !== 'object' || value === null) {
    return false;
  }
  const v = value as Record<string, unknown>;
  if (!isUlid(v.authContextId)) return false;
  if (!isUlid(v.sessionId)) return false;
  if (v.identityKind !== 'account') return false;
  if (!isIso8601(v.lastSeenAt)) return false;
  if (!isIso8601(v.expiresHintAt)) return false;
  if (v.displayHint !== undefined) {
    if (typeof v.displayHint !== 'object' || v.displayHint === null) return false;
    const dh = v.displayHint as Record<string, unknown>;
    if (typeof dh.label !== 'string') return false;
    if (dh.secondaryLabel !== undefined && typeof dh.secondaryLabel !== 'string') return false;
  }
  return true;
}

/** 未知の値が正当な ContextIndex であるか検証する（tamper detection）。 */
function isValidContextIndex(value: unknown): value is ContextIndex {
  if (typeof value !== 'object' || value === null) {
    return false;
  }
  const v = value as Record<string, unknown>;
  if (v.version !== CONTEXT_INDEX_VERSION) return false;
  if (v.surface !== 'product') return false;
  if (v.activeAuthContextId !== null && !isUlid(v.activeAuthContextId)) return false;
  if (!Array.isArray(v.entries)) return false;
  if (!v.entries.every(isValidContextIndexEntry)) return false;
  return true;
}

/** 空の context index を生成する。 */
function createEmptyContextIndex(): ContextIndex {
  return {
    version: CONTEXT_INDEX_VERSION,
    surface: 'product',
    activeAuthContextId: null,
    entries: [],
  };
}

/** localStorage から context index を読み出し、schema 検証を行う。tamper 時は null を返し、localStorage からも破棄する。 */
function readContextIndex(): ContextIndex | null {
  try {
    const raw = localStorage.getItem(CONTEXT_INDEX_KEY);
    if (raw === null) {
      return null;
    }
    const parsed = JSON.parse(raw) as unknown;
    if (!isValidContextIndex(parsed)) {
      localStorage.removeItem(CONTEXT_INDEX_KEY);
      return null;
    }
    return parsed;
  } catch {
    try {
      localStorage.removeItem(CONTEXT_INDEX_KEY);
    } catch {
      // localStorage 無効化時は無視
    }
    return null;
  }
}

/** context index を localStorage に書き込む。 */
function writeContextIndex(index: ContextIndex): void {
  try {
    localStorage.setItem(CONTEXT_INDEX_KEY, JSON.stringify(index));
  } catch {
    // localStorage が満杯または無効化されている場合は無視する。
    // context index は認証済み状態の証明ではないため、書き込み失敗は fatal でない。
  }
}

/** context index から指定 authContextId の entry を削除する。 */
function removeContextEntry(index: ContextIndex, authContextId: string): void {
  index.entries = index.entries.filter((e) => e.authContextId !== authContextId);
  if (index.activeAuthContextId === authContextId) {
    index.activeAuthContextId = null;
  }
}

/** context index に entry を追加または更新する。 */
function upsertContextEntry(
  index: ContextIndex,
  entry: ContextIndexEntry,
  setActive: boolean
): void {
  const filtered = index.entries.filter((e) => e.authContextId !== entry.authContextId);
  filtered.push(entry);
  // 過剰件数は LRU で削除（lastSeenAt が古い順）
  if (filtered.length > MAX_ENTRIES) {
    filtered.sort((a, b) => Date.parse(a.lastSeenAt) - Date.parse(b.lastSeenAt));
    filtered.splice(0, filtered.length - MAX_ENTRIES);
  }
  index.entries = filtered;
  if (setActive) {
    index.activeAuthContextId = entry.authContextId;
  }
}

/** context index を完全にクリアする。 */
function clearContextIndex(): void {
  try {
    localStorage.removeItem(CONTEXT_INDEX_KEY);
  } catch {
    // localStorage 無効化時は無視
  }
}

/**
 * 同一 origin の他 tab からの context index 変更を監視し、
 * 変更が検知されたら callback を呼び出す。
 * storage event を使用して multi-tab propagation を行う。
 *
 * @param callback - 他 tab で context index が変更された時に呼ばれるコールバック
 * @returns unsubscribe 関数
 */
function subscribeContextIndexChanges(callback: (index: ContextIndex | null) => void): () => void {
  const handler = (event: StorageEvent) => {
    if (event.key === CONTEXT_INDEX_KEY) {
      const index = readContextIndex();
      callback(index);
    }
  };
  window.addEventListener('storage', handler);
  return () => {
    window.removeEventListener('storage', handler);
  };
}

/**
 * auth session summary から context index entry を生成する。
 * accessToken などの secret は一切含めない。
 */
function toContextIndexEntry(
  session: { authContextId: string; sessionId: string; accountId: string },
  expiresAt: string
): ContextIndexEntry {
  return {
    authContextId: session.authContextId,
    sessionId: session.sessionId,
    identityKind: 'account',
    displayHint: {
      label: session.accountId,
    },
    lastSeenAt: new Date().toISOString(),
    expiresHintAt: expiresAt,
  };
}

export {
  clearContextIndex,
  createEmptyContextIndex,
  readContextIndex,
  removeContextEntry,
  subscribeContextIndexChanges,
  toContextIndexEntry,
  upsertContextEntry,
  writeContextIndex,
};
