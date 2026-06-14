import { test, expect } from '@playwright/test';

import { mockWebAuthn } from './support/webauthn';

test.describe('webauthn js mock', () => {
  test.skip(
    ({ browserName }) => browserName !== 'chromium',
    'webauthn mock tests run in Chromium only'
  );

  test('navigator.credentials.get returns a mock PublicKeyCredential', async ({ page }) => {
    await mockWebAuthn(page);
    await page.goto('http://localhost:5174/login');
    await page.evaluate(async () => {
      try {
        const c = await navigator.credentials.get({
          publicKey: {
            challenge: new Uint8Array([1]),
            rpId: location.hostname,
            userVerification: 'required',
            allowCredentials: [],
          } as PublicKeyCredentialRequestOptions,
        });
        document.body.setAttribute('data-result', (c as PublicKeyCredential).id);
      } catch (error: unknown) {
        document.body.setAttribute(
          'data-result',
          (error as Error).name + ':' + (error as Error).message
        );
      }
    });
    await expect(page.locator('body')).toHaveAttribute('data-result', /mock-credential-id/);
  });
});
