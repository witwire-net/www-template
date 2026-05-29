import { finishInitialAdminSetup, startInitialAdminSetup } from '../auth';

import type { AdminInitialSetupStartResult } from '../auth';
import type { WWWTemplateWebAuthnAttestationCredential } from '@www-template/admin-api';

type AdminInitialSetupAvailability = 'available' | 'operator-exists' | 'bootstrap-disabled';
type AdminInitialSetupStartedResult = Extract<AdminInitialSetupStartResult, { status: 'started' }>;

interface AdminInitialSetupState {
  email: string;
  displayName: string;
  bootstrapSecret: string;
  isSubmitting: boolean;
  messageKey: string | null;
  setupAvailability: AdminInitialSetupAvailability;
}

interface AdminInitialSetupData {
  state: AdminInitialSetupState;
}

interface AdminInitialSetupActions {
  submit: (
    register: (
      options: AdminInitialSetupStartedResult['options']
    ) => Promise<WWWTemplateWebAuthnAttestationCredential>,
    navigateHome: () => void
  ) => Promise<void>;
}

interface AdminInitialSetupOptions {
  readInitialAvailability?: () => AdminInitialSetupAvailability | undefined;
}

function createInitialSetupState(
  initialAvailability: AdminInitialSetupAvailability
): AdminInitialSetupState {
  // 初回 setup の secret は form state にだけ保持し、session や storage には移さない。
  return {
    email: '',
    displayName: '',
    bootstrapSecret: '',
    isSubmitting: false,
    messageKey: null,
    setupAvailability: initialAvailability,
  };
}

function applyUnavailableState(
  state: AdminInitialSetupState,
  result: Exclude<AdminInitialSetupStartResult, { status: 'started' }>
): void {
  // operator 既存や bootstrap gate 無効は form 非表示 state へ反映し、secret 入力欄を残さない。
  if (result.status === 'operator-exists') state.setupAvailability = 'operator-exists';
  if (result.status === 'bootstrap-disabled') state.setupAvailability = 'bootstrap-disabled';
}

/**
 * 初回 Admin operator setup の UI orchestration を扱う domain composable です。
 *
 * bootstrap secret の保持範囲を form state へ限定し、WebAuthn registration と navigation は
 * app 層 callback として受け取ります。
 */
function useAdminInitialSetup(options: AdminInitialSetupOptions = {}): {
  data: AdminInitialSetupData;
  actions: AdminInitialSetupActions;
} {
  const state = $state<AdminInitialSetupState>(createInitialSetupState('available'));

  const actions: AdminInitialSetupActions = {
    submit: async (register, navigateHome) => {
      // 初回管理者作成は二重送信を防ぎ、backend transaction の競合検知に過度に頼らない。
      if (state.isSubmitting) return;
      state.isSubmitting = true;
      state.messageKey = null;

      try {
        // bootstrap secret 検証と challenge 発行は Admin API wrapper 経由の domain function に委譲する。
        const startPayload = await startInitialAdminSetup({
          email: state.email,
          displayName: state.displayName,
          bootstrapSecret: state.bootstrapSecret,
        });
        if (startPayload.status !== 'started') {
          applyUnavailableState(state, startPayload);
          throw new Error('initial-setup-start-failed');
        }

        // browser authenticator で最初の admin passkey を作成し、登録応答だけを backend へ送る。
        const attestation = await register(startPayload.options);
        const session = await finishInitialAdminSetup(
          {
            email: state.email,
            displayName: state.displayName,
            bootstrapSecret: state.bootstrapSecret,
          },
          startPayload.requestId,
          attestation
        );
        if (session === null) throw new Error('initial-setup-finish-failed');
        navigateHome();
      } catch {
        // bootstrap secret や operator 件数の詳細を出さず、初回 setup の状態推測を防ぐ。
        state.messageKey = 'setup.error';
      } finally {
        // 成功・失敗にかかわらず loading を戻し、画面操作を復帰させる。
        state.isSubmitting = false;
      }
    },
  };

  $effect.pre(() => {
    // test fixture や将来の runtime state が availability を渡す場合だけ、form 表示可否へ反映する。
    const initialAvailability = options.readInitialAvailability?.();
    if (initialAvailability !== undefined) state.setupAvailability = initialAvailability;
  });

  return { data: { state }, actions };
}

export { useAdminInitialSetup };
