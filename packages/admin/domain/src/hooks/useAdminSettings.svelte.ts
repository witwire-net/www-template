type AdminSettingsLocale = 'ja' | 'en';

interface AdminSettingsState {
  selectedLocale: AdminSettingsLocale;
  localeUpdated: boolean;
  localeError: boolean;
}

interface AdminSettingsData {
  state: AdminSettingsState;
}

interface AdminSettingsActions {
  saveLocale: () => void;
}

interface AdminSettingsOptions {
  readLocale: () => AdminSettingsLocale;
  writeLocale: (locale: AdminSettingsLocale) => boolean;
}

function createInitialSettingsState(locale: AdminSettingsLocale): AdminSettingsState {
  // persisted locale の現在値から form state を初期化し、Admin session token とは完全に分離する。
  return { selectedLocale: locale, localeUpdated: false, localeError: false };
}

/**
 * Admin settings の locale form state と永続化 action を扱う domain composable です。
 *
 * browser storage 実装は app/i18n callback に閉じ込め、
 * domain は選択 state と保存結果だけを管理します。
 */
function useAdminSettings(options: AdminSettingsOptions): {
  data: AdminSettingsData;
  actions: AdminSettingsActions;
} {
  const state = $state<AdminSettingsState>(createInitialSettingsState(options.readLocale()));

  const actions: AdminSettingsActions = {
    saveLocale: () => {
      // 保存対象は表示 locale のみで、Admin session token は storage に置かない。
      state.localeUpdated = options.writeLocale(state.selectedLocale);
      state.localeError = !state.localeUpdated;
    },
  };

  $effect(() => {
    // 他画面や同画面の保存で locale state が変わった場合、select の表示値も同期する。
    state.selectedLocale = options.readLocale();
  });

  return { data: { state }, actions };
}

export { useAdminSettings };
