import { describe, expect, it } from 'vitest';

import { createEmptyAccessTokenState, decodeAccessToken, isRefreshNeeded } from './token_state';

describe('token_state', () => {
  it('[AUTH-FE-S023] isRefreshNeeded returns true when token expires within margin', () => {
    const now = Date.now();
    const exp = Math.floor((now + 30_000) / 1000); // 残り 30 秒
    expect(isRefreshNeeded(exp, now, 60_000)).toBe(true);
  });

  it('[AUTH-FE-S024] isRefreshNeeded returns true when token is already expired', () => {
    const now = Date.now();
    const exp = Math.floor((now - 1_000) / 1000); // 1 秒前に期限切れ
    expect(isRefreshNeeded(exp, now, 60_000)).toBe(true);
  });

  it('[AUTH-FE-S025] createEmptyAccessTokenState holds accessToken in memory only', () => {
    const state = createEmptyAccessTokenState();
    expect(state.accessToken).toBe('');
  });

  it('[AUTH-FE-S026] initial token state is empty and does not restore previous tokens', () => {
    const state = createEmptyAccessTokenState();
    expect(state.accessToken).toBe('');
    expect('refreshToken' in state).toBe(false);
  });

  it('decodeAccessToken extracts claims from a valid JWT', () => {
    const header = btoa(JSON.stringify({ alg: 'HS256', typ: 'JWT' }));
    const payload = btoa(
      JSON.stringify({
        sub: '01ARZ3NDEKTSV4RRFFQ69G5FAV',
        sid: '01ARZ3NDEKTSV4RRFFQ69G5FAW',
        exp: 1_893_456_000,
        iat: 1_893_452_400,
      })
    );
    const token = `${header}.${payload}.signature`;

    const claims = decodeAccessToken(token);
    expect(claims).not.toBeNull();
    expect(claims?.accountId).toBe('01ARZ3NDEKTSV4RRFFQ69G5FAV');
    expect(claims?.sessionId).toBe('01ARZ3NDEKTSV4RRFFQ69G5FAW');
    expect(claims?.exp).toBe(1_893_456_000);
    expect(claims?.iat).toBe(1_893_452_400);
  });

  it('decodeAccessToken returns null for malformed token', () => {
    expect(decodeAccessToken('not-a-jwt')).toBeNull();
    expect(decodeAccessToken('only.two.parts')).toBeNull();
  });

  it('decodeAccessToken returns null for missing required claims', () => {
    const header = btoa(JSON.stringify({ alg: 'HS256', typ: 'JWT' }));
    const payload = btoa(JSON.stringify({ sub: '', sid: 'x', exp: 0 }));
    const token = `${header}.${payload}.sig`;
    expect(decodeAccessToken(token)).toBeNull();
  });
});
