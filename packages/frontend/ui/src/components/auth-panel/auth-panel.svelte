<!--
  AuthPanel — 認証 surface で繰り返し現れる status card の共通パターン。
  eyebrow / title / body / actions / footer の 5 ブロックを snippet として受け付け、
  Card コンポーネントに slot する役割だけを担う。
  認証フロー全体（login / logout / account-suspended / session-expired / recovery）で
  同じ視覚比率と余白ルールを共有させるためのプリミティブ。
-->
<script lang="ts" module>
  import type { Snippet } from 'svelte';

  export type AuthPanelWidth = 'narrow' | 'default' | 'wide';

  export interface AuthPanelProps {
    /** メインコンテンツ。 */
    children: Snippet;
    /** フッターの slot。省略可。 */
    footer?: Snippet;
    /** 横幅の最大値。 */
    width?: AuthPanelWidth;
    /** 追加の Tailwind クラス。 */
    class?: string;
  }
</script>

<script lang="ts">
  import { Card, CardContent } from '@www-template/ui/components/card';
  import { cn } from '@www-template/ui/lib/utils';

  let { children, footer, width = 'default', class: className }: AuthPanelProps = $props();

  const widthClass = $derived(
    width === 'narrow' ? 'w-full max-w-sm' : width === 'wide' ? 'w-full max-w-lg' : 'w-full max-w-md'
  );
</script>

<Card data-slot="auth-panel" data-width={width} class={cn(widthClass, className)}>
  <CardContent class="flex flex-col gap-6">
    {@render children()}
  </CardContent>
  {#if footer}
    <div class="border-t border-border-subtle px-6 py-3 text-center">
      {@render footer()}
    </div>
  {/if}
</Card>
