/**
 * webauthn.ts helper のユニットテスト。
 *
 * navigator.credentials.get / create と PublicKeyCredential はブラウザ専用 API のため、
 * vi.stubGlobal で差し替える。
 *
 * テスト戦略:
 * - base64url ↔ ArrayBuffer の roundtrip を検証する
 * - getWebAuthnAssertion がサーバー options を正しく navigator.credentials.get に渡し、
 *   WebAuthnAssertionCredential を正しく構築することを検証する
 * - createWebAuthnAttestation がサーバー options の rpName / user / pubKeyCredParams を
 *   そのまま navigator.credentials.create に渡すことを検証する（fabrication なし）
 */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import type { PasskeyAddStartResponse, PasskeyStartResponse } from '@www-template/api';

import {
  base64urlToBuffer,
  bufferToBase64url,
  createWebAuthnAttestation,
  getWebAuthnAssertion,
  normalizeWebAuthnError,
} from './webauthn';

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

/** テスト用に number[] → base64url エンコードする */
function encodeBase64url(bytes: number[]): string {
  return bufferToBase64url(new Uint8Array(bytes).buffer);
}

/**
 * jsdom では PublicKeyCredential が未定義。
 * vi.stubGlobal('PublicKeyCredential', MockPublicKeyCredential) で差し替えることで
 * webauthn.ts 内の `instanceof PublicKeyCredential` チェックが通る。
 */
class MockPublicKeyCredential {
  id: string;
  rawId: ArrayBuffer;
  type: string;
  authenticatorAttachment: string;
  response: AuthenticatorAssertionResponse | AuthenticatorAttestationResponse;

  constructor(overrides: {
    id: string;
    rawId: ArrayBuffer;
    type: string;
    authenticatorAttachment: string;
    response: AuthenticatorAssertionResponse | AuthenticatorAttestationResponse;
  }) {
    this.id = overrides.id;
    this.rawId = overrides.rawId;
    this.type = overrides.type;
    this.authenticatorAttachment = overrides.authenticatorAttachment;
    this.response = overrides.response;
  }

  getClientExtensionResults(): AuthenticationExtensionsClientOutputs {
    return {};
  }
}

// ---------------------------------------------------------------------------
// base64url helpers
// ---------------------------------------------------------------------------

describe('base64urlToBuffer / bufferToBase64url', () => {
  it('roundtrip: encode → decode は元の値を返す', () => {
    const original = new Uint8Array([0, 1, 127, 128, 255]);
    const encoded = bufferToBase64url(original.buffer);
    const decoded = new Uint8Array(base64urlToBuffer(encoded));
    expect(Array.from(decoded)).toEqual(Array.from(original));
  });

  it('base64url 特殊文字（+ / =）を使わない', () => {
    const bytes = new Uint8Array([0xfb, 0xff, 0xfe]);
    const encoded = bufferToBase64url(bytes.buffer);
    expect(encoded).not.toContain('+');
    expect(encoded).not.toContain('/');
    expect(encoded).not.toContain('=');
  });

  it('空バッファは空文字列になる', () => {
    expect(bufferToBase64url(new Uint8Array(0).buffer)).toBe('');
  });
});

describe('normalizeWebAuthnError', () => {
  it('DOMException name を UI 翻訳用の安定コードへ正規化する', () => {
    // Arrange: ユーザーキャンセルやタイムアウトを表すブラウザー標準エラーを用意する
    const error = new DOMException(
      'The operation either timed out or was not allowed.',
      'NotAllowedError'
    );

    // Act & Assert: ユーザー向け文言ではなく i18n 用コードが返る
    expect(normalizeWebAuthnError(error)).toBe('passkeyOperationCancelledOrTimedOut');
  });

  it('未知の例外文は表示せず汎用コードへ fail-close する', () => {
    // Arrange: 環境依存の英語メッセージを持つ通常 Error を用意する
    const error = new Error('platform-specific browser failure');

    // Act & Assert: 生メッセージを返さず、app catalog で翻訳できる汎用コードへ丸める
    expect(normalizeWebAuthnError(error)).toBe('passkeyOperationFailed');
  });
});

// ---------------------------------------------------------------------------
// getWebAuthnAssertion
// ---------------------------------------------------------------------------

describe('getWebAuthnAssertion', () => {
  const challengeBytes = new Uint8Array([10, 20, 30, 40]);
  const userHandleBytes = new Uint8Array([50, 60]);
  const authenticatorDataBytes = new Uint8Array([70, 80]);
  const signatureBytes = new Uint8Array([90, 100]);
  const clientDataBytes = new Uint8Array([110, 120]);
  const rawIdBytes = new Uint8Array([1, 2, 3]);

  const serverOptions: PasskeyStartResponse = {
    requestId: '01ARZZZ',
    challenge: encodeBase64url([10, 20, 30, 40]),
    rpId: 'example.com',
    timeout: 30000,
    allowCredentials: [],
    userVerification: 'required',
  };

  let capturedGetOptions: CredentialRequestOptions | undefined;

  beforeEach(() => {
    capturedGetOptions = undefined;

    vi.stubGlobal('PublicKeyCredential', MockPublicKeyCredential);

    const assertionResponse = {
      clientDataJSON: clientDataBytes.buffer,
      authenticatorData: authenticatorDataBytes.buffer,
      signature: signatureBytes.buffer,
      userHandle: userHandleBytes.buffer,
    } as unknown as AuthenticatorAssertionResponse;

    const mockCredential = new MockPublicKeyCredential({
      id: 'test-cred-id',
      rawId: rawIdBytes.buffer,
      type: 'public-key',
      authenticatorAttachment: 'platform',
      response: assertionResponse,
    });

    vi.stubGlobal('navigator', {
      credentials: {
        get: vi.fn().mockImplementation((opts: CredentialRequestOptions) => {
          capturedGetOptions = opts;
          return Promise.resolve(mockCredential);
        }),
      },
    });
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it('challenge を base64url デコードして publicKey.challenge に渡す', async () => {
    await getWebAuthnAssertion(serverOptions);
    const publicKey = capturedGetOptions?.publicKey;
    expect(publicKey?.challenge).toBeDefined();
    if (publicKey?.challenge instanceof ArrayBuffer) {
      const decoded = new Uint8Array(publicKey.challenge);
      expect(Array.from(decoded)).toEqual(Array.from(challengeBytes));
    }
  });

  it('rpId をそのまま publicKey.rpId に渡す', async () => {
    await getWebAuthnAssertion(serverOptions);
    expect(capturedGetOptions?.publicKey?.rpId).toBe('example.com');
  });

  it('userVerification をそのまま publicKey.userVerification に渡す', async () => {
    await getWebAuthnAssertion(serverOptions);
    expect(capturedGetOptions?.publicKey?.userVerification).toBe('required');
  });

  it('credential.id / rawId / type / userHandle を正しく返す', async () => {
    const result = await getWebAuthnAssertion(serverOptions);
    expect(result.id).toBe('test-cred-id');
    expect(result.rawId).toBe(bufferToBase64url(rawIdBytes.buffer));
    expect(result.type).toBe('public-key');
    expect(result.response.userHandle).toBe(bufferToBase64url(userHandleBytes.buffer));
  });

  it('navigator.credentials.get が null を返す場合 TypeError を投げる', async () => {
    vi.stubGlobal('navigator', {
      credentials: {
        get: vi.fn().mockResolvedValue(null),
      },
    });
    await expect(getWebAuthnAssertion(serverOptions)).rejects.toBeInstanceOf(TypeError);
  });
});

// ---------------------------------------------------------------------------
// createWebAuthnAttestation — fabrication なし検証
// ---------------------------------------------------------------------------

describe('createWebAuthnAttestation', () => {
  const challengeBytes = new Uint8Array([11, 22, 33]);
  const userIdBytes = new Uint8Array([44, 55, 66]);
  const clientDataBytes = new Uint8Array([77, 88]);
  const attestationObjectBytes = new Uint8Array([99, 110]);
  const rawIdBytes = new Uint8Array([4, 5, 6]);

  const serverOptions: PasskeyAddStartResponse = {
    requestId: '01ARZZZ',
    challenge: encodeBase64url([11, 22, 33]),
    rpId: 'example.com',
    rpName: 'Example RP',
    user: {
      id: encodeBase64url([44, 55, 66]),
      name: 'alice@example.com',
      displayName: 'Alice',
    },
    pubKeyCredParams: [
      { type: 'public-key', alg: -7 },
      { type: 'public-key', alg: -257 },
    ],
    timeout: 60000,
    residentKey: 'required',
    requireResidentKey: true,
    userVerification: 'required',
    attestation: 'none',
  };

  let capturedCreateOptions: CredentialCreationOptions | undefined;

  beforeEach(() => {
    capturedCreateOptions = undefined;

    vi.stubGlobal('PublicKeyCredential', MockPublicKeyCredential);

    const attestationResponse = {
      clientDataJSON: clientDataBytes.buffer,
      attestationObject: attestationObjectBytes.buffer,
      getTransports: () => ['internal'],
    } as unknown as AuthenticatorAttestationResponse;

    const mockCredential = new MockPublicKeyCredential({
      id: 'created-cred-id',
      rawId: rawIdBytes.buffer,
      type: 'public-key',
      authenticatorAttachment: 'platform',
      response: attestationResponse,
    });

    vi.stubGlobal('navigator', {
      credentials: {
        create: vi.fn().mockImplementation((opts: CredentialCreationOptions) => {
          capturedCreateOptions = opts;
          return Promise.resolve(mockCredential);
        }),
      },
    });
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it('[No-fabrication] rp.name にはサーバーの rpName を使う（rpId を使わない）', async () => {
    await createWebAuthnAttestation(serverOptions);
    expect(capturedCreateOptions?.publicKey?.rp.name).toBe('Example RP');
    expect(capturedCreateOptions?.publicKey?.rp.name).not.toBe('example.com');
  });

  it('[No-fabrication] rp.id にはサーバーの rpId を使う', async () => {
    await createWebAuthnAttestation(serverOptions);
    expect(capturedCreateOptions?.publicKey?.rp.id).toBe('example.com');
  });

  it('[No-fabrication] user.id にはサーバーの user.id を base64urlToBuffer で復元する（crypto.getRandomValues を使わない）', async () => {
    await createWebAuthnAttestation(serverOptions);
    const userId = capturedCreateOptions?.publicKey?.user.id;
    expect(userId).toBeDefined();
    if (userId instanceof ArrayBuffer) {
      const decoded = new Uint8Array(userId);
      expect(Array.from(decoded)).toEqual(Array.from(userIdBytes));
    }
  });

  it('[No-fabrication] user.name にはサーバーの user.name を使う（ハードコードしない）', async () => {
    await createWebAuthnAttestation(serverOptions);
    expect(capturedCreateOptions?.publicKey?.user.name).toBe('alice@example.com');
  });

  it('[No-fabrication] user.displayName にはサーバーの user.displayName を使う（ハードコードしない）', async () => {
    await createWebAuthnAttestation(serverOptions);
    expect(capturedCreateOptions?.publicKey?.user.displayName).toBe('Alice');
  });

  it('[No-fabrication] pubKeyCredParams にはサーバーの値をそのまま使う（ハードコードしない）', async () => {
    await createWebAuthnAttestation(serverOptions);
    const params = capturedCreateOptions?.publicKey?.pubKeyCredParams;
    expect(params).toHaveLength(2);
    expect(params?.at(0)).toEqual({ type: 'public-key', alg: -7 });
    expect(params?.at(1)).toEqual({ type: 'public-key', alg: -257 });
  });

  it('challenge を base64url デコードして publicKey.challenge に渡す', async () => {
    await createWebAuthnAttestation(serverOptions);
    const challenge = capturedCreateOptions?.publicKey?.challenge;
    if (challenge instanceof ArrayBuffer) {
      expect(Array.from(new Uint8Array(challenge))).toEqual(Array.from(challengeBytes));
    }
  });

  it('attestation をサーバー値から渡す', async () => {
    await createWebAuthnAttestation(serverOptions);
    expect(capturedCreateOptions?.publicKey?.attestation).toBe('none');
  });

  it('discoverable credential 要求を authenticatorSelection に渡す', async () => {
    await createWebAuthnAttestation(serverOptions);
    expect(capturedCreateOptions?.publicKey?.authenticatorSelection).toMatchObject({
      residentKey: 'required',
      requireResidentKey: true,
      userVerification: 'required',
    });
  });

  it('navigator.credentials.create が null を返す場合 TypeError を投げる', async () => {
    vi.stubGlobal('navigator', {
      credentials: {
        create: vi.fn().mockResolvedValue(null),
      },
    });
    await expect(createWebAuthnAttestation(serverOptions)).rejects.toBeInstanceOf(TypeError);
  });

  it('完成した credential を正しく serialize する', async () => {
    const result = await createWebAuthnAttestation(serverOptions);
    expect(result.id).toBe('created-cred-id');
    expect(result.rawId).toBe(bufferToBase64url(rawIdBytes.buffer));
    expect(result.type).toBe('public-key');
    expect(result.response.clientDataJSON).toBe(bufferToBase64url(clientDataBytes.buffer));
    expect(result.response.attestationObject).toBe(
      bufferToBase64url(attestationObjectBytes.buffer)
    );
    expect(result.response.transports).toEqual(['internal']);
    expect(result.authenticatorAttachment).toBe('platform');
  });
});
