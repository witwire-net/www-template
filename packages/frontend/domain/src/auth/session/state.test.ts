import { describe, expect, it, vi } from 'vitest';

import {
  applyAuthenticatedSession,
  applyExpiredSession,
  applyMissingSession,
  createAuthSessionInitialState,
  hasUlidAuthSessionShape,
  isNoStoreCacheControl,
  isUlid,
} from './state';

describe('authSessionState', () => {
  it('[AUTH-FE-S008] distinguishes missing sessions from expired sessions', () => {
    const missingState = createAuthSessionInitialState();
    const expiredState = createAuthSessionInitialState();

    expect(applyMissingSession(missingState, 'no-store')).toBe('/login');
    expect(applyExpiredSession(expiredState, 'no-store')).toBe('/session-expired');
    expect(missingState.routeIntent).toBe('/login');
    expect(expiredState.routeIntent).toBe('/session-expired');
  });

  it('[AUTH-FE-S009] treats no-store cache metadata as the auth route contract', () => {
    expect(isNoStoreCacheControl('private, no-store, max-age=0')).toBe(true);
    expect(isNoStoreCacheControl('public, max-age=60')).toBe(false);
  });

  it('keeps auth-owned identifiers ULID-formatted', () => {
    const session = {
      requestId: '01ARZ3NDEKTSV4RRFFQ69G5FAV',
      accountId: '01ARZ3NDEKTSV4RRFFQ69G5FAW',
      passkeyCredentialId: '01ARZ3NDEKTSV4RRFFQ69G5FAX',
      sessionId: '01ARZ3NDEKTSV4RRFFQ69G5FAY',
      sessionToken: 'opaque-token',
      expiresAt: '2026-03-21T00:00:00.000Z',
    };

    expect(isUlid(session.accountId)).toBe(true);
    expect(hasUlidAuthSessionShape(session)).toBe(true);
    expect(isUlid('not-a-ulid')).toBe(false);
  });

  it('[AUTH-FE-S020] applyAuthenticatedSession は sessionStorage に書き込まない', () => {
    /* bearer token を含むセッション state helper は persistent storage に触れない。
       hook 側で sessionStorage 書き込みが削除されているため、
       state helper 単体でも storage 操作を行わないことを検証する。 */
    const setItemSpy = vi.spyOn(Storage.prototype, 'setItem');
    const state = createAuthSessionInitialState();

    applyAuthenticatedSession(
      state,
      {
        requestId: '01ARZ3NDEKTSV4RRFFQ69G5FAV',
        accountId: '01ARZ3NDEKTSV4RRFFQ69G5FAW',
        passkeyCredentialId: '01ARZ3NDEKTSV4RRFFQ69G5FAX',
        sessionId: '01ARZ3NDEKTSV4RRFFQ69G5FAY',
        sessionToken: 'opaque-bearer-token',
        expiresAt: '2026-03-21T00:00:00.000Z',
      },
      'no-store'
    );

    expect(state.phase).toBe('authenticated');
    expect(state.session?.sessionToken).toBe('opaque-bearer-token');
    expect(setItemSpy).not.toHaveBeenCalled();

    setItemSpy.mockRestore();
  });
});
