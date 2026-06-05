<script lang="ts">
  /**
   * 認証済みアプリのシェルレイアウト。
   * Sidebar.Provider + Sidebar + SidebarInset の構成で、
   * モバイル時のトグルとメインコンテンツ領域を提供する。
   *
   * @param children - メインコンテンツ。
   * @param sidebar - サイドバー本体（AppSidebar）。
   * @param sidebarTriggerLabel - モバイル用サイドバー開閉ボタンの aria-label。
   */
  import { Sidebar } from '@www-template/ui/components';

  import type { Snippet } from 'svelte';

  interface AppShellProps {
    /** メインコンテンツ。 */
    children: Snippet;
    /** サイドバー本体。 */
    sidebar: Snippet;
    /** モバイル用サイドバー開閉ボタンの aria-label。 */
    sidebarTriggerLabel: string;
  }

  let { children, sidebar, sidebarTriggerLabel }: AppShellProps = $props();
</script>

<Sidebar.SidebarProvider>
  {@render sidebar()}
  <Sidebar.SidebarInset>
    <header class="flex h-12 items-center gap-2 border-b border-border px-4 md:hidden">
      <Sidebar.Trigger aria-label={sidebarTriggerLabel} />
    </header>
    {@render children()}
  </Sidebar.SidebarInset>
</Sidebar.SidebarProvider>
