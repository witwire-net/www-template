import { describe, expect, it } from 'vitest';

import {
  applyExpiredSession,
  applyMissingSession,
  createAuthSessionInitialState,
  hasUlidAuthSessionShape,
  isNoStoreCacheControl,
  isUlid,
} from './authSessionState';

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
});
