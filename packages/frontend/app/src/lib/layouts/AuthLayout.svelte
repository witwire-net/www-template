<script lang="ts">
  import type { Snippet } from 'svelte';
  import { Separator } from '@www-template/ui/components';

  interface AuthLayoutProps {
    /** メインコンテンツのスニペット。 */
    children: Snippet;
    /** フッターのスニペット。省略した場合フッターは表示されない。 */
    footer?: Snippet;
  }

  let { children, footer }: AuthLayoutProps = $props();
</script>

<!--
  認証フロー全体で共通するページレイアウト。
  header（ブランドロゴ）+ main（中央配置のカードエリア）+ footer（任意）の三段構成。
  スタイルは Tailwind ユーティリティクラスのみで記述し、<style> ブロックは一切使用しない。
-->
<div class="flex flex-col items-center min-h-screen px-4 py-8 font-sans bg-background text-foreground">
  <header class="flex justify-center py-4">
    <a href="/" class="no-underline text-inherit" aria-label="www-template トップページ">
      <span class="font-bold tracking-[0.08em]">www-template</span>
    </a>
  </header>

  <Separator />

  <main class="flex flex-1 w-full max-w-[400px] items-center justify-center py-8">
    {@render children()}
  </main>

  {#if footer}
    <Separator />
    <footer class="flex justify-center py-4">
      {@render footer()}
    </footer>
  {/if}
</div>
