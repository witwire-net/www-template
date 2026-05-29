import { finishOperatorSetup, startOperatorSetup } from '../auth';

import type { AdminOperatorSetupStartResult } from '../auth';
import type { WWWTemplateWebAuthnAttestationCredential } from '@www-template/admin-api';

interface AdminOperatorSetupState {
  setupToken: string;
  isSubmitting: boolean;
  messageKey: string | null;
  consumedTokenFromUrl: boolean;
}

interface AdminOperatorSetupData {
  state: AdminOperatorSetupState;
}

interface AdminOperatorSetupActions {
  consumeTokenFromUrl: (url: URL) => void;
  submit: (
    register: (
      options: AdminOperatorSetupStartResult['options']
    ) => Promise<WWWTemplateWebAuthnAttestationCredential>,
    navigateHome: () => void
  ) => Promise<void>;
}

interface AdminOperatorSetupOptions {
  readUrl: () => URL;
  replaceUrl: (url: string) => void;
}

function createInitialOperatorSetupState(): AdminOperatorSetupState {
  // setup token は storage へ置かず、form state と URL 取り込み済み flag だけを memory に保持する。
  return { setupToken: '', isSubmitting: false, messageKey: null, consumedTokenFromUrl: false };
}

function toSanitizedUrl(url: URL): string {
  // token query を削除した相対 URL だけを返し、origin 由来の値を history に再合成しない。
  const sanitizedUrl = new URL(url);
  sanitizedUrl.searchParams.delete('token');
  return `${sanitizedUrl.pathname}${sanitizedUrl.search}${sanitizedUrl.hash}`;
}

/**
 * Admin operator setup token の取り込みと passkey 登録を扱う domain composable です。
 *
 * route component は URL/history 操作 callback と WebAuthn callback だけを提供し、
 * token を storage に保存せず、query から即時削除する流れを domain が統制します。
 */
function useAdminOperatorSetup(options: AdminOperatorSetupOptions): {
  data: AdminOperatorSetupData;
  actions: AdminOperatorSetupActions;
} {
  const state = $state<AdminOperatorSetupState>(createInitialOperatorSetupState());

  const actions: AdminOperatorSetupActions = {
    consumeTokenFromUrl: (url) => {
      // 配送 URL の token は一度だけ form state へ移し、再描画で上書きしない。
      const tokenFromUrl = url.searchParams.get('token')?.trim() ?? '';
      if (state.consumedTokenFromUrl || tokenFromUrl === '') return;
      state.setupToken = tokenFromUrl;
      state.consumedTokenFromUrl = true;

      // 平文 token を address bar と browser history から除去する。
      options.replaceUrl(toSanitizedUrl(url));
    },
    submit: async (register, navigateHome) => {
      // one-time token の多重消費を避けるため、登録処理中は再送信を止める。
      if (state.isSubmitting) return;
      state.isSubmitting = true;
      state.messageKey = null;

      try {
        // token 検証と challenge 作成は Admin API wrapper 経由の domain function に委譲する。
        const startPayload = await startOperatorSetup(state.setupToken);
        if (startPayload === null) throw new Error('operator-setup-start-failed');

        // WebAuthn 登録応答だけを backend transaction に渡し、秘密鍵 material は browser から取り出さない。
        const attestation = await register(startPayload.options);
        const session = await finishOperatorSetup(
          state.setupToken,
          startPayload.requestId,
          attestation
        );
        if (session === null) throw new Error('operator-setup-finish-failed');
        navigateHome();
      } catch {
        // invalid / expired / consumed の状態差分を UI に出さず、同じ error message に丸める。
        state.messageKey = 'operatorSetup.error';
      } finally {
        // 失敗後も安全に再試行できるよう loading を解除する。
        state.isSubmitting = false;
      }
    },
  };

  $effect.pre(() => {
    // URL token bootstrap は route component ではなく domain composable 側の pre-effect に隔離する。
    actions.consumeTokenFromUrl(options.readUrl());
  });

  return { data: { state }, actions };
}

export { useAdminOperatorSetup };
