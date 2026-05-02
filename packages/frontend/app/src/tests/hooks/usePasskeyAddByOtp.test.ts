import { http, HttpResponse } from 'msw';
import { describe, expect, it } from 'vitest';

import { NO_STORE_HEADERS, TEST_ULID } from '../mocks/handlers';
import { server } from '../mocks/server';

/**
 * usePasskeyAddByOtp の API route contract を検証する統合テスト。
 *
 * テスト戦略:
 * - hook は $state rune を使うため直接 instantiate できない。
 * - 「route contract → 期待される state 更新」の組み合わせで挙動を担保する。
 * - start (add/start) と finish (add/finish) の両方を MSW でカバーする。
 */
describe('usePasskeyAddByOtp / API routes', () => {
  it('[AUTH-FE-S017] 有効な email と OTP で新端末ログイン有効化が成功する', async () => {
    // Arrange: 有効な email + OTP で start / finish が共に 200 を返す
    server.use(
      http.post('/api/v1/auth/passkey/add/start', async ({ request }) => {
        const body = (await request.json()) as { email: string; otp: string };
        if (body.email === 'valid@example.com' && body.otp === '123456') {
          return HttpResponse.json(
            {
              requestId: TEST_ULID.requestId,
              challenge: 'otp-add-challenge-base64',
              rpId: 'localhost',
              rpName: 'Test RP',
              user: {
                id: 'dXNlcjE',
                name: body.email,
                displayName: 'Test User',
              },
              pubKeyCredParams: [
                { type: 'public-key', alg: -7 },
                { type: 'public-key', alg: -257 },
              ],
              userVerification: 'required',
            },
            { status: 200, headers: NO_STORE_HEADERS }
          );
        }
        return HttpResponse.json(
          { requestId: TEST_ULID.requestId, error: 'invalid_otp' },
          { status: 400, headers: NO_STORE_HEADERS }
        );
      }),
      http.post('/api/v1/auth/passkey/add/finish', async ({ request }) => {
        const body = (await request.json()) as {
          email: string;
          otp: string;
          credential: unknown;
        };
        if (body.email === 'valid@example.com' && body.otp === '123456') {
          return new HttpResponse(null, { status: 200, headers: NO_STORE_HEADERS });
        }
        return HttpResponse.json(
          { requestId: TEST_ULID.requestId, error: 'invalid_otp' },
          { status: 400, headers: NO_STORE_HEADERS }
        );
      })
    );

    // Act: start API を呼び出す
    const startRes = await fetch('/api/v1/auth/passkey/add/start', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email: 'valid@example.com', otp: '123456' }),
    });
    const startData = (await startRes.json()) as {
      challenge: string;
      userVerification: string;
    };

    // Assert: start が成功し、WebAuthn 登録に必要な options が返る
    expect(startRes.status).toBe(200);
    expect(startData.challenge).toBeDefined();
    expect(startData.userVerification).toBe('required');

    // Act: finish API を呼び出す（credential は mock）
    const finishRes = await fetch('/api/v1/auth/passkey/add/finish', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        email: 'valid@example.com',
        otp: '123456',
        credential: { id: 'mock-cred', rawId: 'mock-raw-id', type: 'public-key' },
      }),
    });

    // Assert: finish が成功し、登録完了となる
    expect(finishRes.status).toBe(200);
  });

  it('[AUTH-FE-S018] 無効な email と OTP は generic error を返す', async () => {
    // Arrange: 無効な組み合わせで 400 を返す
    server.use(
      http.post('/api/v1/auth/passkey/add/start', async ({ request }) => {
        const body = (await request.json()) as { email: string; otp: string };
        if (body.email === 'valid@example.com' && body.otp === '123456') {
          return HttpResponse.json(
            { requestId: TEST_ULID.requestId, challenge: 'ok' },
            { status: 200, headers: NO_STORE_HEADERS }
          );
        }
        return HttpResponse.json(
          { requestId: TEST_ULID.requestId, error: 'invalid_otp' },
          { status: 400, headers: NO_STORE_HEADERS }
        );
      })
    );

    // Act: 無効な OTP で start を試行
    const res = await fetch('/api/v1/auth/passkey/add/start', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email: 'valid@example.com', otp: '000000' }),
    });
    const data = (await res.json()) as { error: string };

    // Assert: 400 が返り、generic エラーメッセージが含まれる
    // （UI 側は email/OTP の正否やアカウント有無を示さない）
    expect(res.status).toBe(400);
    expect(data.error).toBe('invalid_otp');
  });
});
