<script lang="ts">
  import type { Snippet } from 'svelte';

  import { BrandMark } from '@www-template/ui';

  /**
   * 認証 surface の共通レイアウト。
   * brand-mark header + main 中央配置 + footer 段の 3 ブロックで構成し、
   * 内部の status card は AuthPanel を呼び出し側で利用する。
   */
  interface AuthLayoutProps {
    /** パネル本体。AuthPanel を渡して使うことを想定。 */
    children: Snippet;
    /** フッター領域の snippet。省略可。 */
    footer?: Snippet;
  }

  let { children, footer }: AuthLayoutProps = $props();
</script>

<div class="auth-shell">
  <header class="auth-shell__header">
    <BrandMark size="sm" />
  </header>

  <main class="auth-shell__main">
    {@render children()}
  </main>

  {#if footer}
    <footer class="auth-shell__footer">
      {@render footer()}
    </footer>
  {/if}
</div>
