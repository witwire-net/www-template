import { useAuthSession } from '../auth/session';

import { useAccount } from './hook.svelte';

/**
 * useAccountLocaleSync の戻り値型。
 */
export interface LocaleSyncData {
  /** 常に true（副作用のみの hook のため空の data）。 */
  readonly active: boolean;
}

/**
 * useAccountLocaleSync の操作型。
 */
export interface LocaleSyncActions {
  /** 手動で AccountSetting を再読み込みする。 */
  reload: () => Promise<void>;
}

/**
 * 認証セッションの refresh 後に AccountSetting snapshot を Account state に反映する。
 * route component ではなく domain composable として副作用を集約する。
 *
 * @param onLocaleChange - locale が変更された際に呼ばれるコールバック
 */
export function useAccountLocaleSync(onLocaleChange?: (locale: 'ja' | 'en') => void): {
  data: LocaleSyncData;
  actions: LocaleSyncActions;
} {
  const { data: sessionData, actions: sessionActions } = useAuthSession();
  const { data: accountData, actions: accountActions } = useAccount();

  // 認証済みセッション確定後に AccountSetting を読み込む
  $effect(() => {
    if (sessionData.state.phase === 'authenticated' && sessionData.state.session !== null) {
      const headers = sessionActions.createAuthorizationHeaders();
      void accountActions.loadAccountSetting(headers).then(() => {
        const locale = accountData.state.account?.setting.locale;
        if (locale !== undefined) {
          onLocaleChange?.(locale);
        }
      });
    }
  });

  // refresh 後の AccountSetting snapshot を Account state に反映する
  $effect(() => {
    const snapshot = sessionData.state.lastAccountSettingSnapshot;
    if (snapshot !== null && sessionData.state.session !== null) {
      accountActions.applySnapshot(sessionData.state.session.accountId, snapshot.locale);
      onLocaleChange?.(snapshot.locale);
    }
  });

  return {
    data: { active: true },
    actions: {
      reload: async () => {
        if (sessionData.state.phase === 'authenticated' && sessionData.state.session !== null) {
          const headers = sessionActions.createAuthorizationHeaders();
          await accountActions.loadAccountSetting(headers);
        }
      },
    },
  };
}
