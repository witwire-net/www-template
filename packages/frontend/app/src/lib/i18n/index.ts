import {
  createTranslator,
  defineI18nConfig,
  loadJsonCatalog,
  normalizeLocale,
  resolveLocale,
  SUPPORTED_LOCALES,
  type CatalogKeyPath,
  type LocaleCatalogMap,
  type TranslationValues,
  type Locale,
} from '@www-template/i18n';

import commonEn from './messages/en/common.json';
import deviceManagerEn from './messages/en/device-manager.json';
import loginEn from './messages/en/login.json';
import settingsEn from './messages/en/settings.json';
import commonJa from './messages/ja/common.json';
import deviceManagerJa from './messages/ja/device-manager.json';
import loginJa from './messages/ja/login.json';
import settingsJa from './messages/ja/settings.json';

interface AppNamespaceMap {
  common: typeof commonJa;
  login: typeof loginJa;
  settings: typeof settingsJa;
  'device-manager': typeof deviceManagerJa;
}

type AppTranslationKey = {
  [Namespace in keyof AppNamespaceMap]: `${Namespace}.${CatalogKeyPath<AppNamespaceMap[Namespace]>}`;
}[keyof AppNamespaceMap];

const catalogs: LocaleCatalogMap<AppNamespaceMap> = {
  ja: {
    common: loadJsonCatalog(commonJa),
    login: loadJsonCatalog(loginJa),
    settings: loadJsonCatalog(settingsJa),
    'device-manager': loadJsonCatalog(deviceManagerJa),
  },
  en: {
    common: loadJsonCatalog(commonEn),
    login: loadJsonCatalog(loginEn),
    settings: loadJsonCatalog(settingsEn),
    'device-manager': loadJsonCatalog(deviceManagerEn),
  },
};

const LOCAL_STORAGE_KEY = 'www-template:locale';

/**
 * localStorage から保存済み locale を読み込む。
 */
function readStoredLocale(): string | null {
  try {
    return localStorage.getItem(LOCAL_STORAGE_KEY);
  } catch {
    return null;
  }
}

/**
 * localStorage に locale を保存する。
 */
function writeStoredLocale(locale: Locale): void {
  try {
    localStorage.setItem(LOCAL_STORAGE_KEY, locale);
  } catch {
    // storage 不可の環境では無視する
  }
}

/**
 * browser / OS の言語設定から locale 候補を取得する。
 */
function getBrowserLocaleCandidates(): readonly string[] {
  if (typeof navigator === 'undefined') {
    return [];
  }
  return navigator.languages;
}

/**
 * 認証前の fallback locale を決定する。
 * localStorage 優先、次に browser/OS 言語、最後に既定値 ja。
 */
export function resolveUnauthenticatedLocale(): Locale {
  const stored = readStoredLocale();
  const candidates: string[] = [];
  if (stored !== null) {
    candidates.push(stored);
  }
  candidates.push(...getBrowserLocaleCandidates());

  return resolveLocale(candidates);
}

/**
 * app 用の i18n entrypoint を生成する。
 *
 * @param locale - 現在の locale
 */
export function useI18n(locale: Locale) {
  const translator = createTranslator(defineI18nConfig({ locale }), catalogs);

  return {
    locale: translator.locale,
    formatters: translator.formatters,
    t(key: AppTranslationKey, values?: TranslationValues): string {
      const separatorIndex = key.indexOf('.');
      if (separatorIndex <= 0 || separatorIndex === key.length - 1) {
        throw new Error(`i18n key が不正です: ${key}`);
      }

      const namespace = key.slice(0, separatorIndex) as keyof AppNamespaceMap;
      const messageKey = key.slice(separatorIndex + 1);

      switch (namespace) {
        case 'common':
          return translator.t(
            namespace,
            messageKey as CatalogKeyPath<AppNamespaceMap['common']>,
            values
          );
        case 'login':
          return translator.t(
            namespace,
            messageKey as CatalogKeyPath<AppNamespaceMap['login']>,
            values
          );
        case 'settings':
          return translator.t(
            namespace,
            messageKey as CatalogKeyPath<AppNamespaceMap['settings']>,
            values
          );
        case 'device-manager':
          return translator.t(
            namespace,
            messageKey as CatalogKeyPath<AppNamespaceMap['device-manager']>,
            values
          );
      }

      throw new Error('i18n namespace が不正です。');
    },
  };
}

/**
 * 現在の locale を localStorage に保存する。
 */
export function persistAppLocale(locale: Locale): void {
  writeStoredLocale(locale);
}

export { normalizeLocale, resolveLocale, SUPPORTED_LOCALES };
export type { Locale };
export { formatAuthError } from './errorCopy';
