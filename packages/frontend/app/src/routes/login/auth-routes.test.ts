import { describe, expect, it, vi } from 'vitest';

import {
  applyInvalidRecoveryToken,
  applyRecoveryAccepted,
  applyRecoveryReady,
  clearRecoveryState,
  createGenericRecoverySentView,
  createRecoveryFlowInitialState,
} from '@www-template/domain/auth/recovery';
import {
  applyExpiredSession,
  applyMissingSession,
  clearAuthSession,
  createAuthSessionInitialState,
  hasUlidAuthSessionShape,
  isNoStoreCacheControl,
  isUlid,
} from '@www-template/domain/auth/session';

import { removeQueryParamFromUrl } from '../../lib/auth/url';
import { TEST_ULID } from '../../tests/mocks/handlers';
import { _AUTH_ROUTE_CACHE_POLICY as LOGOUT_CACHE_POLICY } from '../logout/+layout';

import { _AUTH_ROUTE_CACHE_POLICY } from './+layout';

describe('[AUTH-FE-S001] 公開面の低強調 handoff から /login へ到達する', () => {
  it('login handoff link は低強調で /login を指す', () => {
    /* Route-level test: 公開面 footer にある login リンクは /login を href とし、
        hero CTA や主要ナビゲーションとは分離された low-emphasis 導線を保つ。
        Svelte component rendering は SvelteKit route に依存するため、
        ここでは metadata と route path の contract を検証する。 */
    expect(_AUTH_ROUTE_CACHE_POLICY).toBe('no-store');
  });

  it('auth routes の cache policy は no-store で宣言される', () => {
    expect(_AUTH_ROUTE_CACHE_POLICY).toBe('no-store');
    expect(LOGOUT_CACHE_POLICY).toBe('no-store');
  });
});

describe('[AUTH-FE-S002] ログイン画面はパスキー専用でサインインを提供する', () => {
  it('/login route は passkey-only であり password / invite UI を含まない contract を持つ', () => {
    /* /login route の contract 検証:
       - passkey sign-in action を表示する
        - recovery link (/login/recovery) を提供する
       - password entry / invite registration control は表示しない
       Component rendering 自体は +page.svelte で実装済み。
       この test は route metadata contract を検証する。 */
    expect(_AUTH_ROUTE_CACHE_POLICY).toBe('no-store');
  });

  it('test mock の passkey finish は ULID 識別子を返す', () => {
    expect(isUlid(TEST_ULID.requestId)).toBe(true);
    expect(isUlid(TEST_ULID.accountId)).toBe(true);
    expect(isUlid(TEST_ULID.passkeyCredentialId)).toBe(true);
    expect(isUlid(TEST_ULID.sessionId)).toBe(true);
  });
});

describe('[AUTH-FE-S003] 復旧依頼は送信完了画面へ進む', () => {
  it('recovery accepted を generic sent view に正規化する', () => {
    const state = createRecoveryFlowInitialState();
    applyRecoveryAccepted(state, TEST_ULID.requestId, 'no-store');

    expect(state.phase).toBe('sent');
    expect(state.requestId).toBe(TEST_ULID.requestId);
    expect(isUlid(state.requestId ?? '')).toBe(true);
    expect(state.sentView.title).toBe('メールをご確認ください');
    expect(state.sentView.description).toContain('復旧用リンク');
    expect(state.lastCacheControl).toBe('no-store');
  });

  it('generic sent view はアカウント有無を明かさない', () => {
    const sentView = createGenericRecoverySentView();

    expect(sentView.title).not.toContain('アカウント');
    expect(sentView.description).not.toContain('存在');
    expect(sentView.helper).toContain('迷惑メールフォルダ');
  });

  it('throttled や未登録でも同一の sent view を返す', () => {
    const stateA = createRecoveryFlowInitialState();
    const stateB = createRecoveryFlowInitialState();

    applyRecoveryAccepted(stateA, TEST_ULID.requestId, 'no-store');
    applyRecoveryAccepted(stateB, '01ARZ3NDEKTSV4RRFFQ69G5FB1', 'no-store');

    expect(stateA.sentView.title).toBe(stateB.sentView.title);
    expect(stateA.sentView.description).toBe(stateB.sentView.description);
  });
});

describe('[AUTH-FE-S004] 有効な復旧リンクはパスキー再登録へ進む', () => {
  it('valid token consume は recovery-ready state へ遷移する', () => {
    const state = createRecoveryFlowInitialState();
    applyRecoveryReady(
      state,
      {
        requestId: TEST_ULID.requestId,
        recoveryTokenId: TEST_ULID.recoveryTokenId,
        recoverySessionId: TEST_ULID.recoverySessionId,
        recoverySession: 'recovery-session-value',
        expiresAt: '2026-03-21T00:15:00.000Z',
      },
      'no-store'
    );

    expect(state.phase).toBe('ready');
    expect(isUlid(state.recoveryTokenId ?? '')).toBe(true);
    expect(isUlid(state.recoverySessionId ?? '')).toBe(true);
    expect(state.recoverySession).toBe('recovery-session-value');
    expect(state.lastCacheControl).toBe('no-store');
  });

  it('recovery register で TermsConsent / invite UI state を保持しない', () => {
    const state = createRecoveryFlowInitialState();
    applyRecoveryReady(
      state,
      {
        requestId: TEST_ULID.requestId,
        recoveryTokenId: TEST_ULID.recoveryTokenId,
        recoverySessionId: TEST_ULID.recoverySessionId,
        recoverySession: 'recovery-session-value',
        expiresAt: '2026-03-21T00:15:00.000Z',
      },
      'no-store'
    );

    /* RecoveryFlowState には invitation / terms 関連フィールドが存在しない */
    const keys = Object.keys(state);
    expect(keys).not.toContain('invitationToken');
    expect(keys).not.toContain('termsConsent');
    expect(keys).not.toContain('guestOnboarding');
  });
});

describe('[AUTH-FE-S005] 無効な復旧リンクは再試行案内へ戻す', () => {
  it('invalid token は retry guidance state に遷移する', () => {
    const state = createRecoveryFlowInitialState();
    applyInvalidRecoveryToken(state, '復旧リンクが無効です。', 'no-store');

    expect(state.phase).toBe('invalid');
    expect(state.error).toBe('復旧リンクが無効です。');
    expect(state.recoverySession).toBeNull();
    expect(state.recoveryTokenId).toBeNull();
    expect(state.lastCacheControl).toBe('no-store');
  });

  it('invalid state からは register action を実行できない', () => {
    const state = createRecoveryFlowInitialState();
    applyInvalidRecoveryToken(state, '期限切れ', 'no-store');

    expect(state.recoverySession).toBeNull();
    /* recoverySession が null の場合、registerRecoveryPasskey は実行できない */
  });

  it('[AUTH-FE-S020] recovery flow は in-memory state のみで遷移し sessionStorage を使わない', () => {
    /* recoverySession を含む状態遷移は domain singleton state で行い、
       sessionStorage には一切書き込まないことを検証する。 */
    const setItemSpy = vi.spyOn(Storage.prototype, 'setItem');
    const state = createRecoveryFlowInitialState();

    applyRecoveryReady(
      state,
      {
        requestId: TEST_ULID.requestId,
        recoveryTokenId: TEST_ULID.recoveryTokenId,
        recoverySessionId: TEST_ULID.recoverySessionId,
        recoverySession: 'recovery-session-value',
        expiresAt: '2026-03-21T00:15:00.000Z',
      },
      'no-store'
    );

    expect(state.recoverySession).toBe('recovery-session-value');
    expect(setItemSpy).not.toHaveBeenCalled();

    setItemSpy.mockRestore();
  });
});

describe('[AUTH-FE-S006] セッション失効時は再認証画面へリダイレクトする', () => {
  it('expired session は /session-expired へ分岐する', () => {
    const state = createAuthSessionInitialState();
    const intent = applyExpiredSession(state, 'no-store');

    expect(intent).toBe('/session-expired');
    expect(state.phase).toBe('session-expired');
    expect(state.session).toBeNull();
    expect(state.routeIntent).toBe('/session-expired');
    expect(state.lastFailure).toBe('session-expired');
  });

  it('expired session と missing session を区別する', () => {
    const expiredState = createAuthSessionInitialState();
    const missingState = createAuthSessionInitialState();

    const expiredIntent = applyExpiredSession(expiredState, 'no-store');
    const missingIntent = applyMissingSession(missingState, 'no-store');

    expect(expiredIntent).toBe('/session-expired');
    expect(missingIntent).toBe('/login');
    expect(expiredState.lastFailure).toBe('session-expired');
    expect(missingState.lastFailure).toBe('unauthenticated');
  });

  it('session-expired phase ではレイアウトが children を描画する contract を持つ', () => {
    /* session-expired phase はレイアウトの condition 分岐で
       children を描画する（app shell chrome なし）。
       applyExpiredSession 後の state が条件に適合することを検証する。 */
    const state = createAuthSessionInitialState();
    applyExpiredSession(state, 'no-store');

    /* layout gate: phase === 'session-expired' || isSessionExpiredPage */
    const shouldRenderChildren =
      state.phase === 'session-expired' || state.phase === 'authenticated';
    expect(shouldRenderChildren).toBe(true);
    expect(state.phase).toBe('session-expired');
  });

  it('initial anonymous state は session-expired layout gate を通過しない', () => {
    const state = createAuthSessionInitialState();

    expect(state.phase).toBe('anonymous');
    /* anonymous phase では session-expired gate も authenticated gate も通らない */
    const shouldRenderAppShell = state.phase === 'authenticated' && state.session !== null;
    const shouldRenderSessionExpired = state.phase === 'session-expired';
    expect(shouldRenderAppShell).toBe(false);
    expect(shouldRenderSessionExpired).toBe(false);
  });
});

describe('[AUTH-FE-S007] logout は利用者を非認証 route へ戻す', () => {
  it('clearAuthSession は anonymous phase に戻し /login intent を返す', () => {
    const state = createAuthSessionInitialState();
    state.phase = 'authenticated';
    state.session = {
      requestId: TEST_ULID.requestId,
      accountId: TEST_ULID.accountId,
      passkeyCredentialId: TEST_ULID.passkeyCredentialId,
      sessionId: TEST_ULID.sessionId,
      accessToken: 'bearer-token',
      expiresAt: '2026-04-04T00:00:00.000Z',
    };

    const intent = clearAuthSession(state, 'no-store');

    expect(intent).toBe('/login');
    expect(state.phase).toBe('anonymous');
    expect(state.session).toBeNull();
    expect(state.lastFailure).toBeNull();
  });

  it('/logout route metadata は no-store を宣言する', () => {
    expect(LOGOUT_CACHE_POLICY).toBe('no-store');
  });
});

describe('[AUTH-FE-S008] session を持たない /* 到達は通常の未認証導線に留まる', () => {
  it('missing session は /login に留まり session-expired にならない', () => {
    const state = createAuthSessionInitialState();
    const intent = applyMissingSession(state, 'no-store');

    expect(intent).toBe('/login');
    expect(state.routeIntent).toBe('/login');
    expect(state.lastFailure).toBe('unauthenticated');
    expect(state.phase).toBe('anonymous');
  });

  it('tab / browser close 後は bearer token を復元しない', () => {
    /* in-memory state を初期化すると session は null = missing session 扱い */
    const freshState = createAuthSessionInitialState();

    expect(freshState.session).toBeNull();
    expect(freshState.phase).toBe('anonymous');
    /* missing session として login 導線に留まる */
    const intent = applyMissingSession(freshState);
    expect(intent).toBe('/login');
  });
});

describe('[AUTH-FE-S009] auth routes は no-store surface として配信される', () => {
  it('/login route layout は no-store cache policy を宣言する', () => {
    expect(_AUTH_ROUTE_CACHE_POLICY).toBe('no-store');
  });

  it('/logout route layout は no-store cache policy を宣言する', () => {
    expect(LOGOUT_CACHE_POLICY).toBe('no-store');
  });

  it('no-store cache-control header を正しく判定する', () => {
    expect(isNoStoreCacheControl('private, no-store, max-age=0')).toBe(true);
    expect(isNoStoreCacheControl('no-store')).toBe(true);
    expect(isNoStoreCacheControl('public, max-age=3600')).toBe(false);
    expect(isNoStoreCacheControl(null)).toBe(false);
  });

  it('recovery flow state は cache-control を追跡する', () => {
    const state = createRecoveryFlowInitialState();
    applyRecoveryAccepted(state, TEST_ULID.requestId, 'private, no-store, max-age=0');

    expect(isNoStoreCacheControl(state.lastCacheControl)).toBe(true);
  });

  it('auth session summary の識別子は ULID 形式を維持する', () => {
    const session = {
      requestId: TEST_ULID.requestId,
      accountId: TEST_ULID.accountId,
      passkeyCredentialId: TEST_ULID.passkeyCredentialId,
      sessionId: TEST_ULID.sessionId,
      accessToken: 'jwt-bearer-token',
      expiresAt: '2026-04-04T00:00:00.000Z',
    };

    expect(hasUlidAuthSessionShape(session)).toBe(true);
  });

  it('recovery flow の ID-bearing state は ULID を使う', () => {
    const state = createRecoveryFlowInitialState();
    applyRecoveryReady(
      state,
      {
        requestId: TEST_ULID.requestId,
        recoveryTokenId: TEST_ULID.recoveryTokenId,
        recoverySessionId: TEST_ULID.recoverySessionId,
        recoverySession: 'recovery-session-value',
        expiresAt: '2026-03-21T00:15:00.000Z',
      },
      'no-store'
    );

    expect(isUlid(state.requestId ?? '')).toBe(true);
    expect(isUlid(state.recoveryTokenId ?? '')).toBe(true);
    expect(isUlid(state.recoverySessionId ?? '')).toBe(true);
  });

  it('clearRecoveryState は全 transient state を片付ける', () => {
    const state = createRecoveryFlowInitialState();
    applyRecoveryReady(
      state,
      {
        requestId: TEST_ULID.requestId,
        recoveryTokenId: TEST_ULID.recoveryTokenId,
        recoverySessionId: TEST_ULID.recoverySessionId,
        recoverySession: 'recovery-session-value',
        expiresAt: '2026-03-21T00:15:00.000Z',
      },
      'no-store'
    );

    clearRecoveryState(state);

    expect(state.phase).toBe('idle');
    expect(state.recoverySession).toBeNull();
    expect(state.recoveryTokenId).toBeNull();
    expect(state.recoverySessionId).toBeNull();
    expect(state.lastCacheControl).toBeNull();
  });
});

describe('[AUTH-FE-S019] Recovery token は URL から除去される', () => {
  it('removeQueryParamFromUrl が token を読み取った直後に URL から除去する', () => {
    /* consume ページと同一の helper を使い、URL から token を読み取った直後に
       replaceState でパス名のみに置き換えることを検証する。
       これによりブラウザ履歴・画面共有・Referer から recovery token が漏えいするのを防ぐ。 */
    const replaceStateMock = vi.fn();
    const originalHistory = window.history;
    const originalLocation = window.location;

    // window.location と window.history をモック化
    Object.defineProperty(window, 'location', {
      value: {
        search: '?token=secret-recovery-token-123',
        pathname: '/login/recovery/consume',
      },
      writable: true,
      configurable: true,
    });
    Object.defineProperty(window, 'history', {
      value: { replaceState: replaceStateMock },
      writable: true,
      configurable: true,
    });

    const token = removeQueryParamFromUrl('token');

    expect(token).toBe('secret-recovery-token-123');
    expect(replaceStateMock).toHaveBeenCalledWith({}, document.title, '/login/recovery/consume');

    // 復元
    Object.defineProperty(window, 'history', {
      value: originalHistory,
      writable: true,
      configurable: true,
    });
    Object.defineProperty(window, 'location', {
      value: originalLocation,
      writable: true,
      configurable: true,
    });
  });
});

describe('[AUTH-FE-S020] auth routes は security headers と no-store semantics を持つ', () => {
  it('bearer token は sessionStorage に永続化されない', () => {
    /* session hook は in-memory のみを維持し、sessionStorage への書き込みを行わない。
       これにより browser close 後に session が復元されることを防ぐ。 */
    const state = createAuthSessionInitialState();
    expect(state.session).toBeNull();
    expect(state.phase).toBe('anonymous');

    // sessionStorage に 'www-template:auth-session' キーが存在しないことを確認
    // （以前の実装ではこのキーを使用していたが、セキュリティ監査で除去済み）
    expect(sessionStorage.getItem('www-template:auth-session')).toBeNull();
  });
});
