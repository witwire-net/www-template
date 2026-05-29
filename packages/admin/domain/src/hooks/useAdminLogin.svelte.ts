import { finishAdminLogin, startAdminLogin } from '../auth';

import type { AdminLoginStartResult } from '../auth';
import type { WWWTemplateWebAuthnAssertionCredential } from '@www-template/admin-api';

interface AdminLoginState {
  email: string;
  isSubmitting: boolean;
  messageKey: string | null;
}

interface AdminLoginData {
  state: AdminLoginState;
}

interface AdminLoginActions {
  submit: (
    authenticate: (
      options: AdminLoginStartResult['options']
    ) => Promise<WWWTemplateWebAuthnAssertionCredential>,
    navigateHome: () => void
  ) => Promise<void>;
}

function createInitialLoginState(): AdminLoginState {
  // 入力値・送信中 state・表示 message を一箇所に集め、route component を描画専用に近づける。
  return { email: '', isSubmitting: false, messageKey: null };
}

/**
 * Admin passkey login の UI orchestration を扱う domain composable です。
 *
 * WebAuthn browser API と navigation は app 層 callback として受け取り、
 * Admin API start/finish と memory session 更新は domain に閉じ込めます。
 */
function useAdminLogin(): { data: AdminLoginData; actions: AdminLoginActions } {
  const state = $state<AdminLoginState>(createInitialLoginState());

  const actions: AdminLoginActions = {
    submit: async (authenticate, navigateHome) => {
      // 多重 challenge 発行を避けるため、送信中の再入を domain 側で止める。
      if (state.isSubmitting) return;
      state.isSubmitting = true;
      state.messageKey = null;

      try {
        // operator 識別子の正規化と challenge 発行は既存 auth domain function に委譲する。
        const startPayload = await startAdminLogin(state.email);
        if (startPayload === null) throw new Error('login-start-failed');

        // 秘密鍵 material は browser authenticator 内に閉じ、assertion response だけを finish API へ渡す。
        const assertion = await authenticate(startPayload.options);
        const session = await finishAdminLogin(startPayload.requestId, assertion);
        if (session === null) throw new Error('login-finish-failed');

        // 成功時だけ verified message を保持し、遷移そのものは app callback に委譲する。
        state.messageKey = 'login.verified';
        navigateHome();
      } catch {
        // unknown email / inactive / invalid passkey を同一 message に丸め、operator enumeration を防ぐ。
        state.messageKey = 'login.error';
      } finally {
        // 成功・失敗・WebAuthn cancel のいずれでも再試行できる状態へ戻す。
        state.isSubmitting = false;
      }
    },
  };

  return { data: { state }, actions };
}

export { useAdminLogin };
