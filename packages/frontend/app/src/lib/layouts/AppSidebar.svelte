<script lang="ts">
  /**
   * アプリ共通サイドバー。
   * ブランドヘッダー、中央ナビゲーション（ホーム・はじめる）、
   * フッターのユーザーメニューで構成する。
   *
   * @param currentPath - 現在のパス。active 状態の判定に使う。
   * @param labels - i18n 経由の翻訳済みラベル群。
   * @param userMenu - ユーザーメニュー（AppUserMenu）を配置する snippet。
   */
  import { BrandMark } from '@www-template/ui';
  import { Sidebar } from '@www-template/ui/components';

  import type { Snippet } from 'svelte';

  interface AppSidebarProps {
    /** 現在のパス。active 状態の判定に使う。 */
    currentPath: string;
    /** i18n 経由の翻訳済みラベル群。 */
    labels: {
      sidebarAriaLabel: string;
      sidebarClose: string;
      home: string;
      gettingStarted: string;
    };
    /** フッターに配置するユーザーメニュー。 */
    userMenu: Snippet;
  }

  let { currentPath, labels, userMenu }: AppSidebarProps = $props();

  /** ナビゲーション項目定義。 */
  type NavItem = {
    href: string;
    label: string;
  };

  /** サイドバー中央のナビゲーション項目。 */
  let navItems: NavItem[] = $derived([
    { href: '/', label: labels.home },
    { href: '/getting-started', label: labels.gettingStarted },
  ]);

  /**
   * パスが現在のパスと一致するかどうかを判定する。
   *
   * @param href - 判定対象のパス
   * @returns アクティブ状態
   */
  function isActive(href: string): boolean {
    if (href === '/') {
      return currentPath === '/';
    }
    return currentPath.startsWith(href);
  }
</script>

<Sidebar.Sidebar variant="floating" ariaLabel={labels.sidebarAriaLabel} closeLabel={labels.sidebarClose}>
  <Sidebar.SidebarHeader>
    <Sidebar.SidebarMenu>
      <Sidebar.SidebarMenuItem>
        <BrandMark size="sm" />
      </Sidebar.SidebarMenuItem>
    </Sidebar.SidebarMenu>
  </Sidebar.SidebarHeader>
  <Sidebar.SidebarContent>
    <Sidebar.SidebarGroup>
      <Sidebar.SidebarGroupContent>
        <Sidebar.SidebarMenu>
          {#each navItems as item (item.href)}
            <Sidebar.SidebarMenuItem>
              <Sidebar.SidebarMenuLink
                isActive={isActive(item.href)}
                href={item.href}
              >
                {item.label}
              </Sidebar.SidebarMenuLink>
            </Sidebar.SidebarMenuItem>
          {/each}
        </Sidebar.SidebarMenu>
      </Sidebar.SidebarGroupContent>
    </Sidebar.SidebarGroup>
  </Sidebar.SidebarContent>
  <Sidebar.SidebarFooter>
    <Sidebar.SidebarMenu>
      <Sidebar.SidebarMenuItem>
        {@render userMenu()}
      </Sidebar.SidebarMenuItem>
    </Sidebar.SidebarMenu>
  </Sidebar.SidebarFooter>
</Sidebar.Sidebar>
