<script lang="ts">
  /**
   * 設定ローカルレイアウト。
   * カテゴリ（一般 / セキュリティ）とページリンクを横スクロールナビとして提供する。
   * Desktop: サイドバー + 設定ローカルナビ + コンテンツ
   * Mobile: ローカルナビは上部横スクロール
   *
   * ナビリンクには UI 層の Button コンポーネント（variant="ghost"）を使用し、
   * app 層では presentation primitive を直接実装しない。
   */
  import { SvelteMap } from 'svelte/reactivity';

  import type { Snippet } from 'svelte';

  import { page } from '$app/state';

  import { useAccount } from '@www-template/domain';
  import { Button } from '@www-template/ui/components';
  import { resolveUnauthenticatedLocale, useI18n } from '$lib/i18n';

  let { children }: { children: Snippet } = $props();

  const { data: accountData } = useAccount();
  const locale = $derived(accountData.state.account?.setting.locale ?? resolveUnauthenticatedLocale());
  const i18n = $derived(useI18n(locale));

  /** 設定ナビゲーション項目定義。 */
  interface SettingsNavItem {
    href: string;
    label: string;
    category: string;
  }

  /** 設定ナビゲーション項目一覧。 */
  const navItems: SettingsNavItem[] = [
    { href: '/settings/general/language', label: 'settings.language', category: 'settings.general' },
    { href: '/settings/security/passkeys', label: 'settings.passkeys', category: 'settings.security' },
    { href: '/settings/security/devices', label: 'settings.devices', category: 'settings.security' },
  ];

  /** カテゴリごとにグループ化されたナビ項目。 */
  const groupedItems = $derived.by(() => {
    const groups = new SvelteMap<string, SettingsNavItem[]>();
    for (const item of navItems) {
      const existing = groups.get(item.category) ?? [];
      existing.push(item);
      groups.set(item.category, existing);
    }
    return groups;
  });

  /**
   * パスが現在のパスと一致するかどうかを判定する。
   *
   * @param href - 判定対象のパス
   * @returns アクティブ状態
   */
  function isActive(href: string): boolean {
    return page.url.pathname === href;
  }
</script>

<div class="flex flex-col gap-6 p-6">
  <header class="flex flex-col gap-2 border-b border-border pb-4">
    <h1 class="text-2xl font-bold">{i18n.t('settings.title')}</h1>
  </header>

  <!-- 設定ローカルナビ: カテゴリ + ページリンク -->
  <nav aria-label={i18n.t('settings.title')} class="flex gap-4 overflow-x-auto pb-2">
    {#each groupedItems as [categoryKey, items] (categoryKey)}
      <div class="flex flex-col gap-1">
        <span class="text-xs font-medium text-muted-foreground uppercase tracking-wider">
          {i18n.t(categoryKey as Parameters<typeof i18n.t>[0])}
        </span>
        <div class="flex gap-1">
          {#each items as item (item.href)}
            <Button
              variant={isActive(item.href) ? 'secondary' : 'ghost'}
              size="sm"
              href={item.href}
              aria-current={isActive(item.href) ? 'page' : undefined}
            >
              {i18n.t(item.label as Parameters<typeof i18n.t>[0])}
            </Button>
          {/each}
        </div>
      </div>
    {/each}
  </nav>

  <!-- ページコンテンツ -->
  <div class="max-w-2xl">
    {@render children()}
  </div>
</div>
