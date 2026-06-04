<!--
  PageHeader — 公開面と認証面の最上部に配置する sticky ヘッダー。
  ブランドガイド 06 の flat & bright 原則に従い、フィルター・透過効果・ glassmorphism は
  一切使用せず solid surface のみで階層を構成する。
  右端の trailing slot は nav や account switcher を受け持ち、
  ブランド mark と nav は中央 slot に並べる。
-->
<script lang="ts" module>
  import type { Snippet } from 'svelte';

  export type PageHeaderSticky = boolean;

  export interface PageHeaderProps {
    /** 中央 / 左側の slot。省略可。 */
    children?: Snippet;
    /** 右端の slot。 */
    trailing?: Snippet;
    /** sticky / non-sticky を選択。 */
    sticky?: PageHeaderSticky;
    /** 追加の Tailwind クラス。 */
    class?: string;
  }
</script>

<script lang="ts">
  import { cn } from '@www-template/ui/lib/utils';

  let { children, trailing, sticky = true, class: className }: PageHeaderProps = $props();

  const stickyClass = $derived(
    sticky ? 'sticky top-4 z-20 mx-4 mt-4 sm:mx-6 sm:mt-6' : 'mx-4 mt-4 sm:mx-6 sm:mt-6'
  );
</script>

<header
  data-slot="page-header"
  data-sticky={sticky}
  class={cn(
    stickyClass,
    'flex flex-wrap items-center justify-between gap-x-6 gap-y-3',
    'rounded-2xl border border-border-subtle bg-surface px-4 py-3',
    className
  )}
>
  <div class="flex items-center gap-x-4 gap-y-2 flex-wrap">
    {@render children?.()}
  </div>
  {#if trailing}
    <div class="flex items-center gap-2">
      {@render trailing()}
    </div>
  {/if}
</header>
