/// <reference lib="dom" />

import type { Page } from '@playwright/test';

/**
 * Playwright 上で navigator.credentials.get / create をページロード前から注入する。
 * 実装側の `instanceof PublicKeyCredential` チェックを通過するため、
 * Object.setPrototypeOf で正しい prototype chain を設定する。
 */
export const mockWebAuthn = async (page: Page): Promise<void> => {
  await page.addInitScript(() => {
    const buildMockCredential = (type: 'get' | 'create') => {
      const credential = {
        id: 'mock-credential-id',
        rawId: new Uint8Array([1, 2, 3]).buffer,
        type: 'public-key',
        authenticatorAttachment: 'platform',
      } as unknown as PublicKeyCredential;

      const responseBase = {
        clientDataJSON: new Uint8Array(
          new TextEncoder().encode(
            JSON.stringify({
              type: type === 'get' ? 'webauthn.get' : 'webauthn.create',
              challenge: 'test',
              origin: window.location.origin,
            })
          )
        ).buffer,
      };

      if (type === 'get') {
        Object.assign(credential, {
          response: {
            ...responseBase,
            authenticatorData: new Uint8Array(new TextEncoder().encode('auth-data')).buffer,
            signature: new Uint8Array(new TextEncoder().encode('signature')).buffer,
            userHandle: undefined,
          },
        });
      } else {
        Object.assign(credential, {
          response: {
            ...responseBase,
            attestationObject: new Uint8Array(new TextEncoder().encode('attestation')).buffer,
            getTransports: () => ['internal'],
          },
        });
      }

      const publicKeyCredentialPrototype =
        typeof window.PublicKeyCredential !== 'undefined'
          ? window.PublicKeyCredential.prototype
          : undefined;
      if (publicKeyCredentialPrototype !== undefined) {
        Object.setPrototypeOf(credential, publicKeyCredentialPrototype);
      }
      return credential;
    };

    Object.defineProperty(navigator, 'credentials', {
      value: {
        get: () => Promise.resolve(buildMockCredential('get')),
        create: () => Promise.resolve(buildMockCredential('create')),
      },
      configurable: true,
    });
  });
};
