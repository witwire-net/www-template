import { accountApi } from '@www-template/api';

import {
  applyAccountSettingSnapshot,
  createAccountInitialState,
  updateAccountLocale,
  type AccountLocale,
  type AccountState,
} from './state';

interface AccountData {
  state: AccountState;
}

interface AccountActions {
  /** 認証済みセッションの Authorization ヘッダーを使って AccountSetting を読み込む。 */
  loadAccountSetting: (headers: Record<string, string>) => Promise<void>;
  /** AccountSetting.locale を更新する。 */
  updateLocale: (locale: AccountLocale, headers: Record<string, string>) => Promise<boolean>;
  /** refresh response の AccountSetting snapshot を state に反映する。 */
  applySnapshot: (accountId: string, locale: AccountLocale) => void;
  /** テスト用: state を初期状態に戻す。 */
  reset: () => void;
}

const state = $state<AccountState>(createAccountInitialState());

/**
 * Account と AccountSetting の state を管理する domain composable。
 * AccountSetting は Account の child state として扱う。
 */
function useAccount(): { data: AccountData; actions: AccountActions } {
  const actions: AccountActions = {
    loadAccountSetting: async (headers) => {
      state.loading = true;
      state.error = null;
      try {
        const result = await accountApi.getSettings({ headers });
        if ('setting' in result) {
          const locale = result.setting.locale;
          // accountId は呼び出し側から供給する前提で、ここでは setting のみ更新
          // 実際の accountId は session から取得する
          state.account = {
            id: state.account?.id ?? '',
            setting: { locale },
          };
        } else {
          // UI 層で locale に応じて翻訳できるよう、表示文ではなく安定したエラーコードだけを保持する。
          state.error = 'account-settings-load-failed';
        }
      } catch {
        // 通信例外でも同じコードへ正規化し、domain 層からユーザー向け文言を漏らさない。
        state.error = 'account-settings-load-failed';
      } finally {
        state.loading = false;
      }
    },
    updateLocale: async (locale, headers) => {
      state.loading = true;
      state.error = null;
      try {
        const result = await accountApi.updateSettings(locale, { headers });
        if ('setting' in result) {
          updateAccountLocale(state, result.setting.locale);
          return true;
        }
        // UI 層で翻訳するため、更新失敗は文言ではなくコードとして記録する。
        state.error = 'account-settings-update-failed';
        return false;
      } catch {
        // 例外内容は環境依存のため、表示に使わず更新失敗コードへ正規化する。
        state.error = 'account-settings-update-failed';
        return false;
      } finally {
        state.loading = false;
      }
    },
    applySnapshot: (accountId, locale) => {
      applyAccountSettingSnapshot(state, accountId, locale);
    },
    reset: () => {
      const initial = createAccountInitialState();
      state.account = initial.account;
      state.loading = initial.loading;
      state.error = initial.error;
    },
  };

  return {
    data: { state },
    actions,
  };
}

export type { AccountData, AccountActions };
export { useAccount };
