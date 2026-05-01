import { authApi } from '@www-template/api';

import { createWebAuthnAttestation, normalizeWebAuthnError } from '../../webauthn';

import type { PasskeyAddByOtpState } from '../../types';

interface PasskeyAddByOtpData {
  loading: boolean;
  error: string | null;
  done: boolean;
}

interface PasskeyAddByOtpActions {
  addPasskeyByOtp: (otp: string) => Promise<void>;
}

/** 新端末でのパスキー追加（OTP handoff）フローを扱う domain composable。 */
function usePasskeyAddByOtp(): { data: PasskeyAddByOtpData; actions: PasskeyAddByOtpActions } {
  const state = $state<PasskeyAddByOtpState>({
    loading: false,
    error: null,
    done: false,
  });

  const actions: PasskeyAddByOtpActions = {
    addPasskeyByOtp: async (otp: string) => {
      state.loading = true;
      state.error = null;
      state.done = false;

      try {
        // Step 1: Start — get WebAuthn creation options from server
        const startOptions = await authApi.startPasskeyAdditionByOtp(otp);

        // Step 2: Call browser WebAuthn API — normalize browser/device errors only
        let credential;
        try {
          credential = await createWebAuthnAttestation(startOptions);
        } catch (webAuthnError: unknown) {
          state.error = normalizeWebAuthnError(webAuthnError);
          return;
        }

        // Step 3: Finish — send attestation to server
        await authApi.finishPasskeyAdditionByOtp(otp, credential);
        state.done = true;
      } catch (error: unknown) {
        state.error =
          error instanceof Error ? error.message : 'パスキー追加を完了できませんでした。';
      } finally {
        state.loading = false;
      }
    },
  };

  return {
    data: {
      get loading() {
        return state.loading;
      },
      get error() {
        return state.error;
      },
      get done() {
        return state.done;
      },
    },
    actions,
  };
}

export type { PasskeyAddByOtpActions, PasskeyAddByOtpData };
export { usePasskeyAddByOtp };
