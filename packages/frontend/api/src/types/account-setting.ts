/**
 * Product Account が表示と通知に使用する保存済みロケール。
 * 運用者向け設定や画面一時状態ではなく、AccountSetting.locale の値だけを表す。
 */
export type AccountLocale = 'ja' | 'en';

/**
 * Product Account に属する現在の設定。
 * AccountSetting は Account の表示・通知設定を表し、Auth 情報や運用者向け設定を含まない。
 */
export interface AccountSetting {
  /** 現在の Product Account に保存されている表示・通知ロケール。対応値は ja または en のみ。 */
  locale: AccountLocale;
}
