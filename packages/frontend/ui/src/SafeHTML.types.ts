import type { Config as DOMPurifyConfig } from 'dompurify';

/** サニタイズ済みマークアップを描画する Svelte コンポーネントのプロパティ。 */
export interface SafeHTMLProps {
  /** サニタイズするマークアップ文字列。 */
  html: string;
  /** 追加のクラス名。 */
  className?: string;
  /** DOMPurify に渡す追加設定。 */
  sanitizeOptions?: DOMPurifyConfig;
}
