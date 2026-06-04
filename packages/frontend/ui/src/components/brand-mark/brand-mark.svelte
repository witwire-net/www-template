<!--
  BrandMark — 認証 surface と公開面のヘッダーで利用する wordmark。
  ブランドガイド 02 に従い Noto Sans JP Bold を wordmark 書体として使用し、
  ライト背景では N-900、ダーク背景では surface 色で表示する。
  decorative な記号を排除し、リンクと wordmark の 2 要素のみで構成される。
  表示文字は docs/brand/brand_guidelines.md 01 に従い WitWire を用いる。
-->
<script lang="ts" module>
  /** 視覚サイズの段階。 */
  export type BrandMarkSize = 'sm' | 'md' | 'lg';

  export interface BrandMarkProps {
    /** 視覚サイズの段階。 */
    size?: BrandMarkSize;
    /** 追加の Tailwind クラス。 */
    class?: string;
  }
</script>

<script lang="ts">
  import { cn } from '@www-template/ui/lib/utils';

  let { size = 'md', class: className }: BrandMarkProps = $props();

  const sizeClass = $derived(
    size === 'sm' ? 'text-sm' : size === 'lg' ? 'text-2xl' : 'text-base'
  );
</script>

<a
  href="/"
  aria-label="WitWire"
  class={cn(
    'font-sans font-bold text-foreground no-underline select-none',
    'tracking-normal transition-colors duration-150 hover:text-primary',
    sizeClass,
    className
  )}
>
  WitWire
</a>
