import { authApi } from '@www-template/api';

import type { PasskeyAddByOtpState } from 'types';

interface PasskeyAddByOtpData {
  loading: boolean;
  error: string | null;
  done: boolean;
}

interface PasskeyAddByOtpActions {
  start: (otp: string) => Promise<{ requestId: string; challenge: string; rpId: string } | null>;
  finish: (otp: string, credential: string) => Promise<void>;
}

/** 新端末でのパスキー追加（OTP handoff）フローを扱う domain composable。 */
function usePasskeyAddByOtp(): { data: PasskeyAddByOtpData; actions: PasskeyAddByOtpActions } {
  const state = $state<PasskeyAddByOtpState>({
    loading: false,
    error: null,
    done: false,
  });

  const actions: PasskeyAddByOtpActions = {
    start: async (otp: string) => {
      state.loading = true;
      state.error = null;
      state.done = false;

      try {
        const result = await authApi.startPasskeyAdditionByOtp(otp);
        return result;
      } catch (error: unknown) {
        state.error =
          error instanceof Error ? error.message : 'パスキー追加を開始できませんでした。';
        return null;
      } finally {
        state.loading = false;
      }
    },

    finish: async (otp: string, credential: string) => {
      state.loading = true;
      state.error = null;

      try {
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
