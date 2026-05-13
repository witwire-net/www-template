import { describe, expect, it, vi } from 'vitest';

import {
  addAuthenticatedSession,
  applyAuthenticatedSession,
  applyExpiredSession,
  applyMissingSession,
  clearAuthSession,
  createAuthSessionInitialState,
  hasUlidAuthSessionShape,
  isNoStoreCacheControl,
  isUlid,
  removeActiveSession,
  switchActiveSession,
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
      accessToken: 'jwt-access-token',
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
        accessToken: 'jwt-bearer-token',
        expiresAt: '2026-03-21T00:00:00.000Z',
      },
      'no-store'
    );

    expect(state.phase).toBe('authenticated');
    expect(state.session?.accessToken).toBe('jwt-bearer-token');
    expect(setItemSpy).not.toHaveBeenCalled();

    setItemSpy.mockRestore();
  });

  it('[AUTH-FE-S027] addAuthenticatedSession appends a new session and keeps existing ones', () => {
    const state = createAuthSessionInitialState();
    const sessionA = {
      requestId: '01ARZ3NDEKTSV4RRFFQ69G5FAV',
      accountId: '01ARZ3NDEKTSV4RRFFQ69G5FAW',
      passkeyCredentialId: '01ARZ3NDEKTSV4RRFFQ69G5FAX',
      sessionId: '01ARZ3NDEKTSV4RRFFQ69G5FAY',
      accessToken: 'token-a',
      expiresAt: '2026-03-21T00:00:00.000Z',
    };

    addAuthenticatedSession(state, sessionA, 'no-store');
    expect(state.sessions).toHaveLength(1);
    expect(state.activeSessionId).toBe(sessionA.sessionId);

    const sessionB = {
      requestId: '01ARZ3NDEKTSV4RRFFQ69G5FAZ',
      accountId: '01ARZ3NDEKTSV4RRFFQ69G5FB1',
      passkeyCredentialId: '01ARZ3NDEKTSV4RRFFQ69G5FB2',
      sessionId: '01ARZ3NDEKTSV4RRFFQ69G5FB3',
      accessToken: 'token-b',
      expiresAt: '2026-03-21T01:00:00.000Z',
    };

    addAuthenticatedSession(state, sessionB, 'no-store');
    expect(state.sessions).toHaveLength(2);
    expect(state.activeSessionId).toBe(sessionB.sessionId);
    expect(state.session?.accessToken).toBe('token-b');
  });

  it('[AUTH-FE-S028] switchActiveSession changes the active session', () => {
    const state = createAuthSessionInitialState();
    const sessionA = {
      requestId: '01ARZ3NDEKTSV4RRFFQ69G5FAV',
      accountId: '01ARZ3NDEKTSV4RRFFQ69G5FAW',
      passkeyCredentialId: '01ARZ3NDEKTSV4RRFFQ69G5FAX',
      sessionId: '01ARZ3NDEKTSV4RRFFQ69G5FAY',
      accessToken: 'token-a',
      expiresAt: '2026-03-21T00:00:00.000Z',
    };
    const sessionB = {
      requestId: '01ARZ3NDEKTSV4RRFFQ69G5FAZ',
      accountId: '01ARZ3NDEKTSV4RRFFQ69G5FB1',
      passkeyCredentialId: '01ARZ3NDEKTSV4RRFFQ69G5FB2',
      sessionId: '01ARZ3NDEKTSV4RRFFQ69G5FB3',
      accessToken: 'token-b',
      expiresAt: '2026-03-21T01:00:00.000Z',
    };

    addAuthenticatedSession(state, sessionA, 'no-store');
    addAuthenticatedSession(state, sessionB, 'no-store');

    const switched = switchActiveSession(state, sessionA.sessionId);
    expect(switched).toBe(true);
    expect(state.activeSessionId).toBe(sessionA.sessionId);
    expect(state.session?.accessToken).toBe('token-a');

    const missing = switchActiveSession(state, 'nonexistent');
    expect(missing).toBe(false);
  });

  it('[AUTH-FE-S030] removeActiveSession clears only the active session and promotes another', () => {
    const state = createAuthSessionInitialState();
    const sessionA = {
      requestId: '01ARZ3NDEKTSV4RRFFQ69G5FAV',
      accountId: '01ARZ3NDEKTSV4RRFFQ69G5FAW',
      passkeyCredentialId: '01ARZ3NDEKTSV4RRFFQ69G5FAX',
      sessionId: '01ARZ3NDEKTSV4RRFFQ69G5FAY',
      accessToken: 'token-a',
      expiresAt: '2026-03-21T00:00:00.000Z',
    };
    const sessionB = {
      requestId: '01ARZ3NDEKTSV4RRFFQ69G5FAZ',
      accountId: '01ARZ3NDEKTSV4RRFFQ69G5FB1',
      passkeyCredentialId: '01ARZ3NDEKTSV4RRFFQ69G5FB2',
      sessionId: '01ARZ3NDEKTSV4RRFFQ69G5FB3',
      accessToken: 'token-b',
      expiresAt: '2026-03-21T01:00:00.000Z',
    };

    addAuthenticatedSession(state, sessionA, 'no-store');
    addAuthenticatedSession(state, sessionB, 'no-store'); // active = B

    const intent = removeActiveSession(state);
    expect(state.sessions).toHaveLength(1);
    expect(state.activeSessionId).toBe(sessionA.sessionId);
    expect(state.session?.accessToken).toBe('token-a');
    expect(state.phase).toBe('authenticated');
    expect(intent).toBeNull();
  });

  it('[AUTH-FE-S031] removeActiveSession returns to unauthenticated when no sessions remain', () => {
    const state = createAuthSessionInitialState();
    const sessionA = {
      requestId: '01ARZ3NDEKTSV4RRFFQ69G5FAV',
      accountId: '01ARZ3NDEKTSV4RRFFQ69G5FAW',
      passkeyCredentialId: '01ARZ3NDEKTSV4RRFFQ69G5FAX',
      sessionId: '01ARZ3NDEKTSV4RRFFQ69G5FAY',
      accessToken: 'token-a',
      expiresAt: '2026-03-21T00:00:00.000Z',
    };

    addAuthenticatedSession(state, sessionA, 'no-store');
    const intent = removeActiveSession(state);
    expect(state.sessions).toHaveLength(0);
    expect(state.activeSessionId).toBeNull();
    expect(state.session).toBeNull();
    expect(state.phase).toBe('anonymous');
    expect(intent).toBe('/login');
  });

  it('[AUTH-FE-S006] applyExpiredSession routes to session-expired and clears all sessions', () => {
    const state = createAuthSessionInitialState();
    addAuthenticatedSession(
      state,
      {
        requestId: '01ARZ3NDEKTSV4RRFFQ69G5FAV',
        accountId: '01ARZ3NDEKTSV4RRFFQ69G5FAW',
        passkeyCredentialId: '01ARZ3NDEKTSV4RRFFQ69G5FAX',
        sessionId: '01ARZ3NDEKTSV4RRFFQ69G5FAY',
        accessToken: 'token-a',
        expiresAt: '2026-03-21T00:00:00.000Z',
      },
      'no-store'
    );

    const intent = applyExpiredSession(state, 'no-store');
    expect(intent).toBe('/session-expired');
    expect(state.phase).toBe('session-expired');
    expect(state.sessions).toHaveLength(0);
    expect(state.activeSessionId).toBeNull();
  });

  it('[AUTH-FE-S007] clearAuthSession returns to anonymous route', () => {
    const state = createAuthSessionInitialState();
    addAuthenticatedSession(
      state,
      {
        requestId: '01ARZ3NDEKTSV4RRFFQ69G5FAV',
        accountId: '01ARZ3NDEKTSV4RRFFQ69G5FAW',
        passkeyCredentialId: '01ARZ3NDEKTSV4RRFFQ69G5FAX',
        sessionId: '01ARZ3NDEKTSV4RRFFQ69G5FAY',
        accessToken: 'token-a',
        expiresAt: '2026-03-21T00:00:00.000Z',
      },
      'no-store'
    );

    const intent = clearAuthSession(state, 'no-store');
    expect(intent).toBe('/login');
    expect(state.phase).toBe('anonymous');
    expect(state.session).toBeNull();
    expect(state.sessions).toHaveLength(0);
  });

  it('[AUTH-FE-S008] applyMissingSession stays on normal login flow', () => {
    const state = createAuthSessionInitialState();
    const intent = applyMissingSession(state, 'no-store');
    expect(intent).toBe('/login');
    expect(state.routeIntent).toBe('/login');
    expect(state.lastFailure).toBe('unauthenticated');
    expect(state.phase).toBe('anonymous');
  });
});
