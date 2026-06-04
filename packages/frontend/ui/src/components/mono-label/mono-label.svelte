<!--
  MonoLabel — 等幅フォントで技術的メタデータを表示するための最小プリミティブ。
  shared.css の .ww-eyebrow を踏襲し、Tailwind 任意値記法を避ける。
-->
<script lang="ts" module>
  import type { Snippet } from 'svelte';

  export type MonoLabelTone = 'default' | 'muted' | 'accent';

  export interface MonoLabelProps {
    /** ラベルとして表示するテキスト。 */
    children: Snippet;
    /** カラートーン。 */
    tone?: MonoLabelTone;
    /** 追加の Tailwind クラス。 */
    class?: string;
  }
</script>

<script lang="ts">
  import { cn } from '@www-template/ui/lib/utils';

  let { children, tone = 'default', class: className }: MonoLabelProps = $props();

  const toneClass = $derived(
    tone === 'muted'
      ? 'text-muted-foreground'
      : tone === 'accent'
        ? 'text-primary'
        : 'text-foreground'
  );
</script>

<span
  data-slot="mono-label"
  data-tone={tone}
  class={cn('ww-eyebrow', toneClass, className)}
>
  {@render children()}
</span>
