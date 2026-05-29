import { ADMIN_FALLBACK_LOCALE, createAdminI18n, normalizeAdminLocale } from './runtime';

import type { AdminI18n, AdminLocale } from './runtime';

const adminLocaleStorageKey = 'www-template:admin:locale';

let currentLocale = $state<AdminLocale>(initialAdminLocale());

/**
 * 現在の Admin 表示 locale を取得します。
 *
 * @returns Admin Console が現在表示に使う locale。
 */
export function getCurrentAdminLocale(): AdminLocale {
  // Svelte rune state を関数越しに読み、route 側の `$derived` が locale 変更を追跡できるようにする。
  return currentLocale;
}

/**
 * 現在の Admin 表示 locale に束縛された i18n runtime を作成します。
 *
 * @returns 現在 locale を反映した翻訳・formatter API。
 */
export function createCurrentAdminI18n(): AdminI18n {
  // 毎回 currentLocale を読むことで、settings 画面の変更が layout と各 page の `$derived` に伝播する。
  return createAdminI18n(currentLocale);
}

/**
 * Admin 表示 locale を更新し、ブラウザー内の次回表示にも反映します。
 *
 * @param locale 保存したい Admin 表示 locale。
 * @returns 保存できた場合は `true`、対応外値の場合は `false`。
 */
export function setCurrentAdminLocale(locale: string): boolean {
  // UI 入力を必ず対応 locale へ正規化し、未知値を storage や state に入れない。
  const normalized = normalizeAdminLocale(locale);
  if (normalized === null) return false;

  // locale は秘密情報ではないため browser storage に保存し、再読み込み後も同じ表示言語にする。
  currentLocale = normalized;
  persistAdminLocale(normalized);
  return true;
}

function initialAdminLocale(): AdminLocale {
  // 明示保存された locale を最優先し、settings で選んだ表示を page reload 後も維持する。
  const storedLocale = readStoredAdminLocale();
  if (storedLocale !== null) return storedLocale;

  // 保存値がない初回表示では browser 言語を候補にし、対応外なら deterministic fallback にする。
  const browserLocale = readBrowserAdminLocale();
  return browserLocale ?? ADMIN_FALLBACK_LOCALE;
}

function readStoredAdminLocale(): AdminLocale | null {
  // SSR 無効 SPA でも build/test 時の module 評価に備え、localStorage の存在を毎回検査する。
  if (!hasBrowserStorage()) return null;

  // 破損した storage 値は採用せず、次の候補へ安全に落とす。
  return normalizeAdminLocale(globalThis.localStorage.getItem(adminLocaleStorageKey));
}

function readBrowserAdminLocale(): AdminLocale | null {
  // browser API が存在しない環境では fallback を使い、server-like test 実行で例外を出さない。
  const navigatorLike = globalThis.navigator;
  const candidates = [
    ...(Array.isArray(navigatorLike?.languages) ? navigatorLike.languages : []),
    navigatorLike?.language,
  ];

  // browser が返す優先順を維持し、最初に対応した locale を採用する。
  for (const candidate of candidates) {
    const normalized = normalizeAdminLocale(candidate);
    if (normalized !== null) return normalized;
  }
  return null;
}

function persistAdminLocale(locale: AdminLocale): void {
  // storage 非対応環境では memory state だけを更新し、表示変更自体は失敗させない。
  if (!hasBrowserStorage()) return;

  // locale は公開表示設定なので保存してよいが、例外時は UI 操作を壊さないため memory state を正にする。
  try {
    globalThis.localStorage.setItem(adminLocaleStorageKey, locale);
  } catch {
    // storage quota や privacy mode の失敗は secret ではないため握りつぶし、現在 session の state を維持する。
  }
}

function hasBrowserStorage(): boolean {
  // `globalThis.localStorage` 参照前に property 存在を確認し、非 browser runtime で ReferenceError を避ける。
  return typeof globalThis.localStorage !== 'undefined';
}
