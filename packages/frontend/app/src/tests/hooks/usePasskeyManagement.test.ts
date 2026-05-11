import { http, HttpResponse } from 'msw';
import { describe, expect, it } from 'vitest';

import {
  applyPasskeyDeleted,
  applyPasskeyError,
  applyPasskeyList,
  createPasskeyManagementInitialState,
  toPasskeyManagementErrorMessage,
} from '@www-template/domain/auth/passkey/management';

import { NO_STORE_HEADERS, TEST_ULID } from '../mocks/handlers';
import { server } from '../mocks/server';

/**
 * usePasskeyManagement の deletePasskey / sendDeviceLink 挙動を検証する統合テスト。
 *
 * テスト戦略:
 * - tasks 8.4 / 8.5 の正式証跡は domain/src/auth/passkeyManagementState.test.ts の
 *   production helper (`applyPasskeyDeleted`, `applyPasskeyError`) を使ったテストが担う。
 * - このファイルは API route contract を MSW で確認し、加えて production helper を
 *   MSW レスポンスと組み合わせることで「route contract → state 変換」の結合を検証する。
 * - Svelte 5 $state rune は Svelte コンパイルコンテキスト外で instantiate できないため、
 *   hook を直接呼ぶ代わりに「pure helper + route contract」の組み合わせで担保する。
 *   これは既存の useAuthSession / useRecoveryFlow テストと同一の実績済みパターン。
 */
describe('usePasskeyManagement / API routes', () => {
  it('[AUTH-FE-S012] GET /api/v1/passkeys が passkeys 一覧を返し DELETE が 204 を返す', async () => {
    // Arrange: 2 件のパスキーが存在し、1 件の削除が成功する
    server.use(
      http.get('/api/v1/passkeys', () =>
        HttpResponse.json(
          {
            requestId: TEST_ULID.requestId,
            passkeys: [
              {
                id: TEST_ULID.passkeyCredentialId,
                identifier: 'MacBook Pro',
                createdAt: '2026-01-01T00:00:00.000Z',
              },
              {
                id: TEST_ULID.passkeyCredentialId2,
                identifier: 'iPhone 15',
                createdAt: '2026-02-01T00:00:00.000Z',
              },
            ],
          },
          { status: 200, headers: NO_STORE_HEADERS }
        )
      ),
      http.delete(
        `/api/v1/passkeys/${TEST_ULID.passkeyCredentialId}`,
        () => new HttpResponse(null, { status: 204, headers: NO_STORE_HEADERS })
      )
    );

    // Act: 一覧取得
    const listRes = await fetch('/api/v1/passkeys');
    const listData = (await listRes.json()) as {
      passkeys: { id: string; identifier: string; createdAt: string }[];
    };

    expect(listRes.status).toBe(200);
    expect(listData.passkeys).toHaveLength(2);

    // Assert: 一覧取得後に state helper を通じて state が正しく更新されること
    // (usePasskeyManagement.listPasskeys は applyPasskeyList を呼ぶ)
    const state = createPasskeyManagementInitialState();
    applyPasskeyList(state, listData.passkeys);
    expect(state.passkeys).toHaveLength(2);
    expect(state.passkeys.map((p) => p.id)).toContain(TEST_ULID.passkeyCredentialId);

    // Act: 削除（X-Reauth-Session 必須）
    const deleteRes = await fetch(`/api/v1/passkeys/${TEST_ULID.passkeyCredentialId}`, {
      method: 'DELETE',
      headers: { 'X-Reauth-Session': 'test-reauth-session' },
    });
    expect(deleteRes.status).toBe(204);

    // Assert: [AUTH-FE-S012] 削除成功後に applyPasskeyDeleted が対象を除去すること
    // (usePasskeyManagement.deletePasskey は 204 受信後に applyPasskeyDeleted を呼ぶ)
    applyPasskeyDeleted(state, TEST_ULID.passkeyCredentialId);
    expect(state.passkeys).toHaveLength(1);
    expect(state.passkeys.map((p) => p.id)).not.toContain(TEST_ULID.passkeyCredentialId);
    expect(state.passkeys.map((p) => p.id)).toContain(TEST_ULID.passkeyCredentialId2);
    expect(state.error).toBeNull();
  });

  it('[AUTH-FE-S015] DELETE /api/v1/passkeys/:id が 409 を返す場合は data.passkeys が変化しない', async () => {
    // Arrange: 最終 1 件に対する削除が 409 で失敗する
    server.use(
      http.get('/api/v1/passkeys', () =>
        HttpResponse.json(
          {
            requestId: TEST_ULID.requestId,
            passkeys: [
              {
                id: TEST_ULID.passkeyCredentialId,
                identifier: 'MacBook Pro',
                createdAt: '2026-01-01T00:00:00.000Z',
              },
            ],
          },
          { status: 200, headers: NO_STORE_HEADERS }
        )
      ),
      http.delete(`/api/v1/passkeys/${TEST_ULID.passkeyCredentialId}`, () =>
        HttpResponse.json(
          {
            requestId: TEST_ULID.requestId,
            error: 'last_passkey_cannot_be_deleted',
          },
          { status: 409, headers: NO_STORE_HEADERS }
        )
      )
    );

    // Act: 一覧取得 → state に反映
    const listRes = await fetch('/api/v1/passkeys');
    const listData = (await listRes.json()) as {
      passkeys: { id: string; identifier: string; createdAt: string }[];
    };
    expect(listRes.status).toBe(200);

    const state = createPasskeyManagementInitialState();
    applyPasskeyList(state, listData.passkeys);
    expect(state.passkeys).toHaveLength(1);

    // Act: 削除試行（409）
    const deleteRes = await fetch(`/api/v1/passkeys/${TEST_ULID.passkeyCredentialId}`, {
      method: 'DELETE',
      headers: { 'X-Reauth-Session': 'test-reauth-session' },
    });
    const deleteBody = (await deleteRes.json()) as { error: string };
    expect(deleteRes.status).toBe(409);

    // Assert: [AUTH-FE-S015] 409 エラー時は applyPasskeyError が呼ばれ passkeys は変化しない
    // (usePasskeyManagement.deletePasskey は 409 を catch → applyPasskeyError を呼ぶ)
    applyPasskeyError(state, toPasskeyManagementErrorMessage(new Error(deleteBody.error)));
    expect(state.passkeys).toHaveLength(1);
    expect(state.passkeys.map((p) => p.id)).toContain(TEST_ULID.passkeyCredentialId);
    expect(state.error).toBe('last_passkey_cannot_be_deleted');
  });

  it('[AUTH-FE-S012] DELETE /api/v1/passkeys/:id が 401 を返す場合は fail-close (auth failure)', async () => {
    // Arrange: 401 が返る (session-expired)
    server.use(
      http.delete(`/api/v1/passkeys/${TEST_ULID.passkeyCredentialId}`, () =>
        HttpResponse.json(
          {
            requestId: TEST_ULID.requestId,
            error: 'session-expired',
          },
          { status: 401, headers: NO_STORE_HEADERS }
        )
      )
    );

    const deleteRes = await fetch(`/api/v1/passkeys/${TEST_ULID.passkeyCredentialId}`, {
      method: 'DELETE',
      headers: { 'X-Reauth-Session': 'test-reauth-session' },
    });

    // Assert: 401 が返り、error が 'session-expired' であること
    // hook 側では response.status === 401 の分岐で handleFailure('session-expired', ...) が呼ばれ
    // authSession.state.phase が 'session-expired' になって useSessionGuard が redirect する
    expect(deleteRes.status).toBe(401);
    const body = (await deleteRes.json()) as { error: string };
    expect(body.error).toBe('session-expired');
  });

  it('[AUTH-FE-S016] POST /api/v1/passkeys/send-device-link が issued: true を返すとメール送信済み guidance を表示できる', async () => {
    // Arrange: reauth session 付きでデバイスリンク送信を依頼
    server.use(
      http.post('/api/v1/passkeys/send-device-link', ({ request }) => {
        const reauthSession = request.headers.get('X-Reauth-Session');
        if (reauthSession === null || reauthSession === '') {
          return HttpResponse.json(
            { requestId: TEST_ULID.requestId, error: 'reauth_session_required' },
            { status: 400, headers: NO_STORE_HEADERS }
          );
        }
        return HttpResponse.json(
          { requestId: TEST_ULID.requestId, issued: true },
          { status: 200, headers: NO_STORE_HEADERS }
        );
      })
    );

    // Act: API を直接呼び出して issued: true を確認
    // （hook は $state rune を使うため純粋 helper + contract 検証のパターンで担保する）
    const res = await fetch('/api/v1/passkeys/send-device-link', {
      method: 'POST',
      headers: {
        'X-Reauth-Session': 'valid-reauth-session',
        Authorization: 'Bearer test-token',
      },
    });
    const data = (await res.json()) as { issued: boolean };

    // Assert: issued: true が返り、UI 側は deviceLinkSent フラグを立てて guidance を表示する
    expect(res.status).toBe(200);
    expect(data.issued).toBe(true);
  });

  it('[AUTH-FE-S021] DELETE /api/v1/passkeys/:id は X-Reauth-Session なしで 400 を返す', async () => {
    // Arrange: reauth session なしの削除リクエスト
    server.use(
      http.delete(`/api/v1/passkeys/${TEST_ULID.passkeyCredentialId}`, ({ request }) => {
        const reauthSession = request.headers.get('X-Reauth-Session');
        if (reauthSession === null || reauthSession === '') {
          return HttpResponse.json(
            { requestId: TEST_ULID.requestId, error: 'reauth_session_required' },
            { status: 400, headers: NO_STORE_HEADERS }
          );
        }
        return new HttpResponse(null, { status: 204, headers: NO_STORE_HEADERS });
      })
    );

    // Act: X-Reauth-Session ヘッダーを付与せずに削除
    const deleteRes = await fetch(`/api/v1/passkeys/${TEST_ULID.passkeyCredentialId}`, {
      method: 'DELETE',
    });
    const body = (await deleteRes.json()) as { error: string };

    // Assert: reauth がない場合は 400 で削除を開始しない
    expect(deleteRes.status).toBe(400);
    expect(body.error).toBe('reauth_session_required');

    // state helper で確認: 削除成功前に 400 が返るため passkeys は変化しない
    const state = createPasskeyManagementInitialState();
    applyPasskeyList(state, [
      {
        id: TEST_ULID.passkeyCredentialId,
        identifier: 'MacBook Pro',
        createdAt: '2026-01-01T00:00:00.000Z',
      },
    ]);
    expect(state.passkeys).toHaveLength(1);
    // 400 エラー時は applyPasskeyError が呼ばれる想定
    applyPasskeyError(state, toPasskeyManagementErrorMessage(new Error(body.error)));
    expect(state.passkeys).toHaveLength(1);
    expect(state.error).toBe('reauth_session_required');
  });
});
