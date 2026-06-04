<!--
  SurfacePanel — 共通の bordered solid surface。
  topbar / app panel / settings card で繰り返される「単色背景 + 細枠 + 角丸」の
  視覚パターンを統一し、hover 時に brand color へ border が変化して状態を伝える。
  影は brand guideline に従い none を維持。
-->
<script lang="ts" module>
  import type { Snippet } from 'svelte';

  /** インタラクション挙動の variants。 */
  export type SurfacePanelInteractive = 'static' | 'brand-border' | 'subtle-border';

  export interface SurfacePanelProps {
    /** 子要素。 */
    children: Snippet;
    /** インタラクション挙動。 */
    interactive?: SurfacePanelInteractive;
    /** 追加の Tailwind クラス。 */
    class?: string;
  }
</script>

<script lang="ts">
  import { cn } from '@www-template/ui/lib/utils';

  let {
    children,
    interactive = 'static',
    class: className,
  }: SurfacePanelProps = $props();

  const interactiveClass = $derived(
    interactive === 'brand-border'
      ? 'transition-colors duration-150 hover:border-foreground/40'
      : interactive === 'subtle-border'
        ? 'transition-colors duration-150 hover:border-border'
        : ''
  );
</script>

<div
  data-slot="surface-panel"
  data-interactive={interactive}
  class={cn(
    'rounded-2xl border border-border-subtle bg-surface text-card-foreground',
    'px-6 py-5',
    interactiveClass,
    className
  )}
>
  {@render children()}
</div>
