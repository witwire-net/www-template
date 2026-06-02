const CONTEXT_INDEX_KEY = 'www-template:admin:context-index';
const CONTEXT_INDEX_VERSION = 1;
const MAX_ENTRIES = 10;
const ULID_PATTERN = /^[0-9A-Z]{26}$/u;

/**
 * Admin context index が依存する最小 storage port です。
 *
 * domain は DOM global に直接触れず、Admin app の client hook が origin-local localStorage を注入します。
 */
export interface AdminContextIndexStoragePort {
  /** 指定 key の保存値を返します。存在しない場合は null を返します。 */
  getItem: (key: string) => string | null;
  /** 指定 key に文字列値を保存します。 */
  setItem: (key: string, value: string) => void;
  /** 指定 key の保存値を削除します。 */
  removeItem: (key: string) => void;
}

/**
 * Admin context index の storage event を購読する最小 event port です。
 *
 * 実体は browser の window ですが、domain 側は構造的な port としてだけ扱います。
 */
export interface AdminContextIndexEventPort {
  /** storage event listener を登録します。 */
  addStorageListener: (listener: (event: { key: string | null }) => void) => void;
  /** storage event listener を解除します。 */
  removeStorageListener: (listener: (event: { key: string | null }) => void) => void;
}

let storagePort: AdminContextIndexStoragePort | null = null;
let eventPort: AdminContextIndexEventPort | null = null;

const adminContextIndexEntryKeys = new Set([
  'authContextId',
  'operatorSessionId',
  'displayHint',
  'roleHint',
  'lastSeenAt',
  'expiresHintAt',
]);
const adminContextIndexKeys = new Set(['version', 'surface', 'activeAuthContextId', 'entries']);

/**
 * Admin origin-local context index の 1 entry です。
 *
 * accessToken、refreshToken、Cookie value、setup token は保持せず、
 * browser reload 後に context refresh を試すための非 secret hint だけを保存します。
 */
export interface AdminContextIndexEntry {
  /** refresh path と対応する Admin auth context の canonical ULID。 */
  authContextId: string;
  /** Admin operator session の canonical ULID。 */
  operatorSessionId: string;
  /** UI 表示だけに使う operator の非 secret label。 */
  displayHint: string;
  /** UI 表示だけに使う role hint。authorization 判定には使いません。 */
  roleHint: string;
  /** server refresh/current で最後に確認できた時刻。 */
  lastSeenAt: string;
  /** stale entry を削除する目安時刻。認証可否は server refresh で再検証します。 */
  expiresHintAt: string;
}

/**
 * Admin origin-local localStorage に保存する context index schema です。
 *
 * `surface` を固定して Product index との混入を検出し、tamper 時は fail-close で破棄します。
 */
export interface AdminContextIndex {
  /** schema version。互換性維持ではなく tamper / stale schema 検出に使います。 */
  version: number;
  /** Admin origin の index であることを固定する marker。 */
  surface: 'admin';
  /** 現在 active とみなす auth context。認証済み証明ではありません。 */
  activeAuthContextId: string | null;
  /** token/secret を含まない Admin operator context entry 一覧。 */
  entries: AdminContextIndexEntry[];
}

/**
 * Admin context index の保存 key を返します。
 *
 * @returns Admin origin-local localStorage で使う package 固有 key。
 */
export function getAdminContextIndexStorageKey(): string {
  // key を 1 箇所に集約し、Product index や旧 BFF state と混ざらないようにする。
  return CONTEXT_INDEX_KEY;
}

/**
 * Admin app から context index の browser adapter を注入します。
 *
 * @param storage origin-local localStorage 相当の storage port。null の場合は保存を無効化します。
 * @param events storage event 相当の購読 port。省略時は multi-tab 購読を無効化します。
 */
export function configureAdminContextIndexStorage(
  storage: AdminContextIndexStoragePort | null,
  events: AdminContextIndexEventPort | null = null
): void {
  // DOM global 参照を app 層へ閉じ込め、domain は注入済み port だけで context index を扱う。
  storagePort = storage;
  eventPort = events;
}

function isUlid(value: unknown): boolean {
  // ULID 以外の任意文字列を context selector として採用しない。
  return typeof value === 'string' && ULID_PATTERN.test(value);
}

function isIso8601(value: unknown): boolean {
  // 厳密な日付演算ではなく、tamper された非日付値を排除するために parse 可能性だけを見る。
  return typeof value === 'string' && !Number.isNaN(Date.parse(value));
}

function hasOnlyAllowedKeys(value: Record<string, unknown>, allowedKeys: Set<string>): boolean {
  // accessToken / refreshToken / Cookie value など未知 field の混入を tamper として拒否する。
  return Object.keys(value).every((key) => allowedKeys.has(key));
}

function isValidAdminContextIndexEntry(value: unknown): value is AdminContextIndexEntry {
  // object 以外は schema tamper とみなす。
  if (typeof value !== 'object' || value === null) return false;
  const candidate = value as Record<string, unknown>;
  if (!hasOnlyAllowedKeys(candidate, adminContextIndexEntryKeys)) return false;

  // token/secret ではなく識別子と非 secret hint だけを受け入れる。
  return (
    isUlid(candidate.authContextId) &&
    isUlid(candidate.operatorSessionId) &&
    typeof candidate.displayHint === 'string' &&
    typeof candidate.roleHint === 'string' &&
    isIso8601(candidate.lastSeenAt) &&
    isIso8601(candidate.expiresHintAt)
  );
}

function isValidAdminContextIndex(value: unknown): value is AdminContextIndex {
  // schema 全体を検証し、Product surface や unknown version の混入を拒否する。
  if (typeof value !== 'object' || value === null) return false;
  const candidate = value as Record<string, unknown>;
  if (!hasOnlyAllowedKeys(candidate, adminContextIndexKeys)) return false;
  return (
    candidate.version === CONTEXT_INDEX_VERSION &&
    candidate.surface === 'admin' &&
    (candidate.activeAuthContextId === null || isUlid(candidate.activeAuthContextId)) &&
    Array.isArray(candidate.entries) &&
    candidate.entries.every(isValidAdminContextIndexEntry)
  );
}

/**
 * 空の Admin context index を生成します。
 *
 * @returns token/secret を含まない初期 index。
 */
export function createEmptyAdminContextIndex(): AdminContextIndex {
  // active context なし・entry なしで初期化し、認証済み状態は server refresh 後にだけ作る。
  return {
    version: CONTEXT_INDEX_VERSION,
    surface: 'admin',
    activeAuthContextId: null,
    entries: [],
  };
}

/**
 * Admin context index を localStorage から読み取ります。
 *
 * @returns schema 検証済み index。存在しない、または tamper されている場合は null。
 */
export function readAdminContextIndex(): AdminContextIndex | null {
  const storage = storagePort;
  if (storage === null) return null;
  try {
    // localStorage は origin-local だが改竄可能なので、読み取り直後に schema 検証する。
    const raw = storage.getItem(CONTEXT_INDEX_KEY);
    if (raw === null) return null;
    const parsed = JSON.parse(raw) as unknown;
    if (!isValidAdminContextIndex(parsed)) {
      storage.removeItem(CONTEXT_INDEX_KEY);
      return null;
    }
    return parsed;
  } catch {
    try {
      // 壊れた JSON や storage 例外時は fail-close として index を破棄する。
      storage.removeItem(CONTEXT_INDEX_KEY);
    } catch {
      // storage 自体が無効な環境では、認証済み state を復元しないだけに留める。
    }
    return null;
  }
}

/**
 * Admin context index を localStorage へ保存します。
 *
 * @param index 保存する token/secret 非含有 index。
 */
export function writeAdminContextIndex(index: AdminContextIndex): void {
  const storage = storagePort;
  if (storage === null) return;
  try {
    // JSON 化対象は schema 化済みの非 secret hint だけに限定する。
    storage.setItem(CONTEXT_INDEX_KEY, JSON.stringify(index));
  } catch {
    // index は認証証明ではないため、quota などの保存失敗は session 発行失敗にしない。
  }
}

/**
 * Admin context index を localStorage から削除します。
 */
export function clearAdminContextIndex(): void {
  const storage = storagePort;
  if (storage === null) return;
  try {
    // logout / all-context revoke 時は Admin surface の index だけを削除する。
    storage.removeItem(CONTEXT_INDEX_KEY);
  } catch {
    // storage 無効時は削除済みと同等に扱う。
  }
}

/**
 * 指定 auth context の entry を index から削除します。
 *
 * @param index 更新対象 index。
 * @param authContextId 削除対象の Admin auth context ID。
 */
export function removeAdminContextEntry(index: AdminContextIndex, authContextId: string): void {
  // 対象 entry だけを消し、active が対象なら active marker も消す。
  index.entries = index.entries.filter((entry) => entry.authContextId !== authContextId);
  if (index.activeAuthContextId === authContextId) index.activeAuthContextId = null;
}

/**
 * Admin context entry を追加または更新します。
 *
 * @param index 更新対象 index。
 * @param entry 保存する非 secret entry。
 * @param setActive true の場合、この entry を active context にします。
 */
export function upsertAdminContextEntry(
  index: AdminContextIndex,
  entry: AdminContextIndexEntry,
  setActive: boolean
): void {
  // 同一 context の古い hint を置換し、過剰件数は lastSeenAt の古いものから削除する。
  const nextEntries = index.entries.filter((item) => item.authContextId !== entry.authContextId);
  nextEntries.push(entry);
  if (nextEntries.length > MAX_ENTRIES) {
    nextEntries.sort((left, right) => Date.parse(left.lastSeenAt) - Date.parse(right.lastSeenAt));
    nextEntries.splice(0, nextEntries.length - MAX_ENTRIES);
  }
  index.entries = nextEntries;
  if (setActive) index.activeAuthContextId = entry.authContextId;
}

/**
 * 同一 Admin origin の別 tab からの context index 更新を購読します。
 *
 * @param callback storage event 後の検証済み index、または null を受け取る callback。
 * @returns 購読解除関数。
 */
export function subscribeAdminContextIndexChanges(
  callback: (index: AdminContextIndex | null) => void
): () => void {
  const events = eventPort;
  if (events === null) {
    // browser 以外では multi-tab propagation が存在しないため、no-op unsubscribe を返す。
    return () => undefined;
  }
  const handler = (event: { key: string | null }) => {
    // Admin 専用 key の変更だけを multi-tab propagation として扱う。
    if (event.key === CONTEXT_INDEX_KEY) callback(readAdminContextIndex());
  };
  events.addStorageListener(handler);
  return () => {
    // component 破棄時に handler を外し、別 route の state 更新を残さない。
    events.removeStorageListener(handler);
  };
}
