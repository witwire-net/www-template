import { beforeEach, describe, expect, it, vi } from 'vitest';

import {
  clearAdminContextIndex,
  configureAdminContextIndexStorage,
  createEmptyAdminContextIndex,
  getAdminContextIndexStorageKey,
  readAdminContextIndex,
  removeAdminContextEntry,
  subscribeAdminContextIndexChanges,
  upsertAdminContextEntry,
  writeAdminContextIndex,
} from './context_index';

import type { AdminContextIndexEntry } from './context_index';

function installBrowserStorageMocks(): { emitStorage: (key: string) => void } {
  // Vitest の node environment でも browser origin-local storage の挙動を検証できるよう、最小 mock を用意する。
  const values = new Map<string, string>();
  let storageListener: ((event: { key: string | null }) => void) | null = null;
  const storage = {
    getItem: (key: string) => values.get(key) ?? null,
    setItem: (key: string, value: string) => {
      values.set(key, value);
    },
    removeItem: (key: string) => {
      values.delete(key);
    },
    clear: () => {
      values.clear();
    },
  };
  vi.stubGlobal('localStorage', storage);
  configureAdminContextIndexStorage(storage, {
    addStorageListener: (listener) => {
      storageListener = listener;
    },
    removeStorageListener: () => {
      storageListener = null;
    },
  });
  return {
    emitStorage: (key: string) => {
      storageListener?.({ key });
    },
  };
}

const adminContextEntry: AdminContextIndexEntry = {
  authContextId: '01ARZ3NDEKTSV4RRFFQ69G5FAV',
  operatorSessionId: '01ARZ3NDEKTSV4RRFFQ69G5FAY',
  displayHint: 'operator@example.com',
  roleHint: 'admin',
  lastSeenAt: '2026-03-21T00:00:00.000Z',
  expiresHintAt: '2026-03-21T01:00:00.000Z',
};

describe('Admin context index', () => {
  beforeEach(() => {
    // 各 test の前に Admin 専用 key を消し、token/secret の残留を検出しやすくする。
    installBrowserStorageMocks();
    clearAdminContextIndex();
  });

  it('[ADMIN-AUTH-FE-S041] context index stores only non-secret operator/session hints', () => {
    // login/bootstrap 後に保存される index は accessToken や refreshToken を含まない。
    const index = createEmptyAdminContextIndex();
    upsertAdminContextEntry(index, adminContextEntry, true);
    writeAdminContextIndex(index);

    const raw = localStorage.getItem(getAdminContextIndexStorageKey()) ?? '';
    const restored = readAdminContextIndex();

    expect(restored?.activeAuthContextId).toBe(adminContextEntry.authContextId);
    expect(raw).toContain('operator@example.com');
    expect(raw).not.toContain('accessToken');
    expect(raw).not.toContain('refreshToken');
    expect(raw).not.toContain('Cookie');
    expect(raw).not.toContain('cookieValue');
    expect(raw).not.toContain('setupToken');
  });

  it('[ADMIN-AUTH-FE-S041] tampered context index is removed and cannot bootstrap state', () => {
    // version/surface/ULID が壊れた index は fail-close として null にし、storage からも除去する。
    localStorage.setItem(
      getAdminContextIndexStorageKey(),
      JSON.stringify({
        version: 999,
        surface: 'product',
        activeAuthContextId: 'tampered',
        entries: [],
      })
    );

    expect(readAdminContextIndex()).toBeNull();
    expect(localStorage.getItem(getAdminContextIndexStorageKey())).toBeNull();
  });

  it('[ADMIN-AUTH-FE-S041] context index rejects unknown secret-like fields', () => {
    // schema にない accessToken / refreshToken / Cookie value が混ざる entry は tamper として拒否する。
    localStorage.setItem(
      getAdminContextIndexStorageKey(),
      JSON.stringify({
        version: 1,
        surface: 'admin',
        activeAuthContextId: adminContextEntry.authContextId,
        entries: [
          {
            ...adminContextEntry,
            accessToken: 'must-not-be-accepted',
            refreshToken: 'must-not-be-accepted',
            cookieValue: 'must-not-be-accepted',
          },
        ],
      })
    );

    expect(readAdminContextIndex()).toBeNull();
    expect(localStorage.getItem(getAdminContextIndexStorageKey())).toBeNull();
  });

  it('[ADMIN-AUTH-FE-S042] context index is limited to the Admin origin localStorage key', () => {
    // Admin package 固有 key に version / authContextId / operatorSessionId / display / role hint だけを保存する。
    const index = createEmptyAdminContextIndex();
    upsertAdminContextEntry(index, adminContextEntry, true);
    writeAdminContextIndex(index);

    const raw = localStorage.getItem(getAdminContextIndexStorageKey()) ?? '';

    expect(localStorage.getItem('www-template:admin:context-index')).not.toBeNull();
    expect(raw).toContain('"version":1');
    expect(raw).toContain('"surface":"admin"');
    expect(raw).toContain('"operatorSessionId"');
    expect(raw).toContain('"roleHint":"admin"');
    expect(raw).not.toContain('www-template:product:context-index');
  });

  it('[ADMIN-AUTH-FE-S042] storage event propagates Admin context index updates across tabs', () => {
    // storage event は同一 origin の別 tab 伝搬の入口なので、Admin key だけに反応する。
    const browser = installBrowserStorageMocks();
    const callback = vi.fn();
    const unsubscribe = subscribeAdminContextIndexChanges(callback);
    const index = createEmptyAdminContextIndex();
    upsertAdminContextEntry(index, adminContextEntry, true);
    writeAdminContextIndex(index);

    browser.emitStorage(getAdminContextIndexStorageKey());
    browser.emitStorage('unrelated');

    expect(callback).toHaveBeenCalledTimes(1);
    expect(callback).toHaveBeenCalledWith(
      expect.objectContaining({
        surface: 'admin',
        activeAuthContextId: adminContextEntry.authContextId,
      })
    );
    unsubscribe();
  });

  it('[ADMIN-AUTH-FE-S043] cleanup removes only the target auth context entry', () => {
    // logout / inactive / refresh failure は対象 context だけを削除し、別 context の hint を残す。
    const index = createEmptyAdminContextIndex();
    const secondEntry: AdminContextIndexEntry = {
      ...adminContextEntry,
      authContextId: '01ARZ3NDEKTSV4RRFFQ69G5FAZ',
      operatorSessionId: '01ARZ3NDEKTSV4RRFFQ69G5FB1',
      displayHint: 'operator-2@example.com',
    };
    upsertAdminContextEntry(index, adminContextEntry, true);
    upsertAdminContextEntry(index, secondEntry, false);

    removeAdminContextEntry(index, adminContextEntry.authContextId);

    expect(index.entries).toHaveLength(1);
    expect(index.entries[0]?.authContextId).toBe(secondEntry.authContextId);
    expect(index.activeAuthContextId).toBeNull();
  });
});
