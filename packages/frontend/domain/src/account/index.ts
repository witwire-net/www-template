export { useAccount } from './hook.svelte';
export { useAccountTranslator } from './translator.svelte';
export { useAccountLocaleSync } from './localeSync.svelte';
export type { AccountData, AccountActions } from './hook.svelte';
export type { AccountTranslatorData, AccountTranslatorActions } from './translator.svelte';
export type { LocaleSyncData, LocaleSyncActions } from './localeSync.svelte';
export type { Account, AccountState } from './state';
export {
  createAccountInitialState,
  applyAccountSettingSnapshot,
  updateAccountLocale,
} from './state';
