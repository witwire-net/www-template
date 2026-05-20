import { useAccount } from './hook.svelte';

/**
 * useAccountTranslator の戻り値型。
 */
export interface AccountTranslatorData<T> {
  /** 現在の translator。未初期化時は null。 */
  readonly translator: T | null;
}

/**
 * useAccountTranslator の操作型。
 * translator は自動再生成されるため、外部操作は不要。
 */
export interface AccountTranslatorActions {
  /** 空の actions（translator は自動管理）。 */
  readonly _noop?: never;
}

/**
 * AccountSetting.locale の変更を監視し、translator を再生成する domain composable。
 * factory 関数を注入することで、app 固有の translator 生成を domain 外に留める。
 *
 * @param createTranslator - locale から translator を生成する factory 関数
 */
export function useAccountTranslator<T>(createTranslator: (locale: 'ja' | 'en') => Promise<T>): {
  data: AccountTranslatorData<T>;
  actions: AccountTranslatorActions;
} {
  const { data: accountData } = useAccount();

  let translator = $state<T | null>(null);

  $effect(() => {
    const locale = accountData.state.account?.setting.locale;
    if (locale !== undefined) {
      void createTranslator(locale).then((t) => {
        translator = t;
      });
    }
  });

  return {
    data: {
      get translator() {
        return translator;
      },
    },
    actions: {},
  };
}
