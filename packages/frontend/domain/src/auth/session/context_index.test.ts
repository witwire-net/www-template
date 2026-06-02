import { describe, expect, it, vi } from 'vitest';

import {
  createEmptyContextIndex,
  readContextIndex,
  removeContextEntry,
  subscribeContextIndexChanges,
  toContextIndexEntry,
  upsertContextEntry,
  writeContextIndex,
} from './context_index';

import type { ContextIndex } from './context_index';

describe('context_index', () => {
  beforeEach(() => {
    localStorage.clear();
  });

  afterEach(() => {
    localStorage.clear();
  });

  it('[AUTH-FE-S056] reads a valid context index from localStorage', () => {
    const index: ContextIndex = {
      version: 1,
      surface: 'product',
      activeAuthContextId: '01ARZ3NDEKTSV4RRFFQ69G5FAV',
      entries: [
        {
          authContextId: '01ARZ3NDEKTSV4RRFFQ69G5FAV',
          sessionId: '01ARZ3NDEKTSV4RRFFQ69G5FAY',
          identityKind: 'account',
          displayHint: { label: 'test@example.com' },
          lastSeenAt: '2026-03-21T00:00:00.000Z',
          expiresHintAt: '2026-03-21T01:00:00.000Z',
        },
      ],
    };
    localStorage.setItem('www-template:product:context-index', JSON.stringify(index));

    const result = readContextIndex();
    expect(result).not.toBeNull();
    expect(result?.activeAuthContextId).toBe('01ARZ3NDEKTSV4RRFFQ69G5FAV');
    expect(result?.entries).toHaveLength(1);
  });

  it('[AUTH-FE-S056] tampered context index returns null and fails bootstrap', () => {
    localStorage.setItem(
      'www-template:product:context-index',
      JSON.stringify({
        version: 999,
        surface: 'product',
        activeAuthContextId: 'invalid',
        entries: [],
      })
    );

    const result = readContextIndex();
    expect(result).toBeNull();
  });

  it('[AUTH-FE-S056] context index does not contain tokens or secrets', () => {
    const index = createEmptyContextIndex();
    const entry = toContextIndexEntry(
      {
        authContextId: '01ARZ3NDEKTSV4RRFFQ69G5FAV',
        sessionId: '01ARZ3NDEKTSV4RRFFQ69G5FAY',
        accountId: '01ARZ3NDEKTSV4RRFFQ69G5FAW',
      },
      '2026-03-21T01:00:00.000Z'
    );
    upsertContextEntry(index, entry, true);
    writeContextIndex(index);

    const raw = localStorage.getItem('www-template:product:context-index') ?? '';
    expect(raw).not.toContain('accessToken');
    expect(raw).not.toContain('refreshToken');
    expect(raw).not.toContain('cookie');
    expect(raw).not.toContain('secret');
  });

  it('[AUTH-FE-S058] context index is stored in origin-local localStorage', () => {
    const index = createEmptyContextIndex();
    const entry = toContextIndexEntry(
      {
        authContextId: '01ARZ3NDEKTSV4RRFFQ69G5FAV',
        sessionId: '01ARZ3NDEKTSV4RRFFQ69G5FAY',
        accountId: '01ARZ3NDEKTSV4RRFFQ69G5FAW',
      },
      '2026-03-21T01:00:00.000Z'
    );
    upsertContextEntry(index, entry, true);
    writeContextIndex(index);

    const keys = Object.keys(localStorage);
    expect(keys).toContain('www-template:product:context-index');
  });

  it('[AUTH-FE-S059] removeContextEntry deletes only the target entry', () => {
    const index = createEmptyContextIndex();
    const entryA = toContextIndexEntry(
      {
        authContextId: '01ARZ3NDEKTSV4RRFFQ69G5FAV',
        sessionId: '01ARZ3NDEKTSV4RRFFQ69G5FAY',
        accountId: '01ARZ3NDEKTSV4RRFFQ69G5FAW',
      },
      '2026-03-21T01:00:00.000Z'
    );
    const entryB = toContextIndexEntry(
      {
        authContextId: '01ARZ3NDEKTSV4RRFFQ69G5FAZ',
        sessionId: '01ARZ3NDEKTSV4RRFFQ69G5FB1',
        accountId: '01ARZ3NDEKTSV4RRFFQ69G5FB2',
      },
      '2026-03-21T02:00:00.000Z'
    );
    upsertContextEntry(index, entryA, true);
    upsertContextEntry(index, entryB, false);
    expect(index.entries).toHaveLength(2);

    removeContextEntry(index, entryA.authContextId);
    expect(index.entries).toHaveLength(1);
    expect(index.entries[0]?.authContextId).toBe(entryB.authContextId);
    expect(index.activeAuthContextId).toBeNull();
  });

  it('[AUTH-FE-S058] subscribeContextIndexChanges propagates changes across tabs', () => {
    const callback = vi.fn();
    const unsubscribe = subscribeContextIndexChanges(callback);

    const index = createEmptyContextIndex();
    const entry = toContextIndexEntry(
      {
        authContextId: '01ARZ3NDEKTSV4RRFFQ69G5FAV',
        sessionId: '01ARZ3NDEKTSV4RRFFQ69G5FAY',
        accountId: '01ARZ3NDEKTSV4RRFFQ69G5FAW',
      },
      '2026-03-21T01:00:00.000Z'
    );
    upsertContextEntry(index, entry, true);
    writeContextIndex(index);

    // storage event を手動で発火させる
    const event = new StorageEvent('storage', {
      key: 'www-template:product:context-index',
      newValue: localStorage.getItem('www-template:product:context-index'),
    });
    window.dispatchEvent(event);

    expect(callback).toHaveBeenCalledTimes(1);
    expect(callback).toHaveBeenCalledWith(
      expect.objectContaining({
        version: 1,
        surface: 'product',
        activeAuthContextId: '01ARZ3NDEKTSV4RRFFQ69G5FAV',
      })
    );

    unsubscribe();
  });

  it('[AUTH-FE-S058] subscribeContextIndexChanges ignores unrelated storage keys', () => {
    const callback = vi.fn();
    const unsubscribe = subscribeContextIndexChanges(callback);

    const event = new StorageEvent('storage', {
      key: 'unrelated-key',
      newValue: 'some-value',
    });
    window.dispatchEvent(event);

    expect(callback).not.toHaveBeenCalled();
    unsubscribe();
  });
});
