/**
 * Account domain で扱う locale 値。
 * API 型への依存を避け、domain 内で独立して定義する。
 */
export type AccountLocale = 'ja' | 'en';

/**
 * Account domain で扱う AccountSetting の最小表現。
 */
export interface AccountSetting {
  /** 表示言語。 */
  locale: AccountLocale;
}

/**
 * Account domain で扱う Account の最小表現。
 * AccountSetting を child state として持つ。
 */
export interface Account {
  /** Account の一意識別子。 */
  id: string;
  /** Account に属する設定。 */
  setting: AccountSetting;
}

/**
 * Account domain state。
 */
export interface AccountState {
  /** 現在の Account。未読み込み時は null。 */
  account: Account | null;
  /** 読み込み中フラグ。 */
  loading: boolean;
  /** エラーメッセージ。null の場合はエラーなし。 */
  error: string | null;
}

/**
 * Account state の初期値を生成する。
 */
export function createAccountInitialState(): AccountState {
  return {
    account: null,
    loading: false,
    error: null,
  };
}

/**
 * AccountSetting snapshot から Account state を更新する。
 */
export function applyAccountSettingSnapshot(
  state: AccountState,
  accountId: string,
  locale: AccountLocale
): void {
  state.account = {
    id: accountId,
    setting: { locale },
  };
}

/**
 * AccountSetting.locale を更新する。
 */
export function updateAccountLocale(state: AccountState, locale: AccountLocale): void {
  if (state.account !== null) {
    state.account = {
      ...state.account,
      setting: { locale },
    };
  }
}
