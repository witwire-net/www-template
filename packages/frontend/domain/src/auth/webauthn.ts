/**
 * Browser WebAuthn API helpers.
 *
 * Converts between server-side base64url-encoded strings and browser
 * PublicKeyCredential objects. All serialisation lives here so that domain
 * hooks stay clean and the conversion logic is testable in isolation.
 *
 * このモジュールは pure domain module であるため、@www-template/api への依存は持ちません。
 * 引数/戻り値の型はローカルで定義し、generated 型と structural compatibility を保ちます。
 */

// ---------------------------------------------------------------------------
// Local structural types (structurally compatible with generated client types)
// ---------------------------------------------------------------------------

/** WebAuthn assertion (login) credential descriptor returned by the server. */
interface WebAuthnCredentialDescriptorLocal {
  type: string;
  id: string;
  transports?: string[];
}

/** WebAuthn user entity from the server (id is base64url-encoded bytes). */
interface WebAuthnUserEntityLocal {
  id: string;
  name: string;
  displayName: string;
}

/** WebAuthn public key credential parameter from the server. */
interface WebAuthnCredentialParameterLocal {
  type: string;
  alg: number;
}

/** Subset of PasskeyStartResponse needed by getWebAuthnAssertion. */
interface PasskeyStartOptions {
  challenge: string;
  rpId: string;
  timeout?: number;
  allowCredentials?: WebAuthnCredentialDescriptorLocal[];
  userVerification?: string;
}

/** Subset of PasskeyAddStartResponse needed by createWebAuthnAttestation. */
interface PasskeyAddStartOptions {
  challenge: string;
  rpId: string;
  rpName: string;
  user: WebAuthnUserEntityLocal;
  pubKeyCredParams: WebAuthnCredentialParameterLocal[];
  timeout?: number;
  excludeCredentials?: WebAuthnCredentialDescriptorLocal[];
  userVerification?: string;
  attestation?: string;
}

/** Serialised assertion credential ready to POST to /auth/passkey/finish. */
interface WebAuthnAssertionResult {
  id: string;
  rawId: string;
  type: string;
  response: {
    clientDataJSON: string;
    authenticatorData: string;
    signature: string;
    userHandle?: string;
  };
  authenticatorAttachment?: string;
}

/** Serialised attestation credential ready to POST to finish endpoints. */
interface WebAuthnAttestationResult {
  id: string;
  rawId: string;
  type: string;
  response: {
    clientDataJSON: string;
    attestationObject: string;
    transports?: string[];
  };
  authenticatorAttachment?: string;
}

// ---------------------------------------------------------------------------
// base64url helpers
// ---------------------------------------------------------------------------

/** Decodes a base64url string to an ArrayBuffer. */
function base64urlToBuffer(base64url: string): ArrayBuffer {
  const base64 = base64url.replace(/-/g, '+').replace(/_/g, '/');
  const padded = base64.padEnd(base64.length + ((4 - (base64.length % 4)) % 4), '=');
  const binary = atob(padded);
  return Uint8Array.from(binary, (c) => c.charCodeAt(0)).buffer;
}

/** Encodes an ArrayBuffer to a base64url string. */
function bufferToBase64url(buffer: ArrayBuffer): string {
  const bytes = new Uint8Array(buffer);
  const binary = Array.from(bytes, (b) => String.fromCharCode(b)).join('');
  return btoa(binary).replace(/\+/g, '-').replace(/\//g, '_').replace(/=/g, '');
}

// ---------------------------------------------------------------------------
// WebAuthn error normalizer
// ---------------------------------------------------------------------------

/**
 * Normalises WebAuthn browser errors into user-facing Japanese messages.
 *
 * DOMException names and their meanings:
 * - NotAllowedError: user cancelled or operation timed out
 * - InvalidStateError: credential already registered (registration), or no credential found
 * - NotSupportedError: browser/platform does not support WebAuthn or this algorithm
 * - SecurityError: origin/RP-ID mismatch
 * - AbortError: operation was aborted programmatically
 */
function normalizeWebAuthnError(error: unknown): string {
  if (error instanceof DOMException) {
    switch (error.name) {
      case 'NotAllowedError':
        return 'パスキー操作がキャンセルされたか、タイムアウトしました。もう一度お試しください。';
      case 'InvalidStateError':
        return 'このデバイスにはすでにパスキーが登録されているか、パスキーが見つかりませんでした。';
      case 'NotSupportedError':
        return 'このブラウザまたはデバイスはパスキーに対応していません。';
      case 'SecurityError':
        return 'セキュリティエラーが発生しました。ページを再読み込みして再試行してください。';
      case 'AbortError':
        return 'パスキー操作が中断されました。もう一度お試しください。';
      default:
        return `パスキー操作に失敗しました（${error.name}）。もう一度お試しください。`;
    }
  }

  if (error instanceof TypeError) {
    return 'パスキー操作を完了できませんでした。ブラウザがパスキーに対応しているか確認してください。';
  }

  if (error instanceof Error) {
    return `パスキー操作に失敗しました。時間を置いて再度お試しください。`;
  }

  return 'パスキー操作に失敗しました。時間を置いて再度お試しください。';
}

// ---------------------------------------------------------------------------
// Authentication (navigator.credentials.get)
// ---------------------------------------------------------------------------

/**
 * Calls `navigator.credentials.get` with the options from the server's
 * PasskeyStartResponse and returns a serialised assertion credential
 * ready to POST to /auth/passkey/finish.
 */
async function getWebAuthnAssertion(
  options: PasskeyStartOptions
): Promise<WebAuthnAssertionResult> {
  const publicKey: PublicKeyCredentialRequestOptions = {
    challenge: base64urlToBuffer(options.challenge),
    rpId: options.rpId,
    timeout: options.timeout ?? 60000,
    userVerification:
      (options.userVerification as UserVerificationRequirement | undefined) ?? 'preferred',
    allowCredentials:
      options.allowCredentials?.map((c) => ({
        type: c.type as PublicKeyCredentialType,
        id: base64urlToBuffer(c.id),
        transports: c.transports as AuthenticatorTransport[] | undefined,
      })) ?? [],
  };

  const credential = await navigator.credentials.get({ publicKey });

  if (!(credential instanceof PublicKeyCredential)) {
    throw new TypeError('パスキー認証を完了できませんでした。ブラウザの応答が無効でした。');
  }

  const response = credential.response as AuthenticatorAssertionResponse;

  return {
    id: credential.id,
    rawId: bufferToBase64url(credential.rawId),
    type: credential.type,
    response: {
      clientDataJSON: bufferToBase64url(response.clientDataJSON),
      authenticatorData: bufferToBase64url(response.authenticatorData),
      signature: bufferToBase64url(response.signature),
      userHandle: response.userHandle != null ? bufferToBase64url(response.userHandle) : undefined,
    },
    authenticatorAttachment: credential.authenticatorAttachment ?? undefined,
  };
}

// ---------------------------------------------------------------------------
// Registration (navigator.credentials.create)
// ---------------------------------------------------------------------------

/**
 * Calls `navigator.credentials.create` with the options from the server's
 * PasskeyAddStartResponse and returns a serialised attestation credential
 * ready to POST to finish endpoints.
 */
async function createWebAuthnAttestation(
  options: PasskeyAddStartOptions
): Promise<WebAuthnAttestationResult> {
  const publicKey: PublicKeyCredentialCreationOptions = {
    challenge: base64urlToBuffer(options.challenge),
    rp: { id: options.rpId, name: options.rpName },
    user: {
      id: base64urlToBuffer(options.user.id),
      name: options.user.name,
      displayName: options.user.displayName,
    },
    pubKeyCredParams: options.pubKeyCredParams.map((p) => ({
      type: p.type as PublicKeyCredentialType,
      alg: p.alg,
    })),
    timeout: options.timeout ?? 60000,
    excludeCredentials:
      options.excludeCredentials?.map((c) => ({
        type: c.type as PublicKeyCredentialType,
        id: base64urlToBuffer(c.id),
        transports: c.transports as AuthenticatorTransport[] | undefined,
      })) ?? [],
    authenticatorSelection: {
      userVerification:
        (options.userVerification as UserVerificationRequirement | undefined) ?? 'preferred',
    },
    attestation: (options.attestation as AttestationConveyancePreference | undefined) ?? 'none',
  };

  const credential = await navigator.credentials.create({ publicKey });

  if (!(credential instanceof PublicKeyCredential)) {
    throw new TypeError('パスキー登録を完了できませんでした。ブラウザの応答が無効でした。');
  }

  const response = credential.response as AuthenticatorAttestationResponse;

  return {
    id: credential.id,
    rawId: bufferToBase64url(credential.rawId),
    type: credential.type,
    response: {
      clientDataJSON: bufferToBase64url(response.clientDataJSON),
      attestationObject: bufferToBase64url(response.attestationObject),
      transports:
        typeof response.getTransports === 'function' ? response.getTransports() : undefined,
    },
    authenticatorAttachment: credential.authenticatorAttachment ?? undefined,
  };
}

export type {
  PasskeyAddStartOptions,
  PasskeyStartOptions,
  WebAuthnAssertionResult,
  WebAuthnAttestationResult,
};
export {
  base64urlToBuffer,
  bufferToBase64url,
  createWebAuthnAttestation,
  getWebAuthnAssertion,
  normalizeWebAuthnError,
};
