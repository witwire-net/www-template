<script lang="ts">
  /**
   * 認証済み protected レイアウト。
   * AppShell（サイドバー + インセット）を提供し、
   * セッションガードで認証状態を検証する。
   *
   * - PageHeader を使わず、サイドバー中心の app shell に切り替える。
   * - ナビゲーションはサイドバー中央にホーム・はじめるのみ配置。
   * - アカウント切替・追加・設定・ログアウトはサイドバー下部のユーザーメニューに集約。
   */
  import type { Snippet } from 'svelte';

  import { goto } from '$app/navigation';
  import { page } from '$app/state';

  import { useAccount } from '@www-template/domain';
  import { useSessionGuard } from '@www-template/domain/auth/guard';
  import { useAuthSession } from '@www-template/domain/auth/session';
  import { useAccountLocaleSync } from '@www-template/domain/account';

  import AppShell from '$lib/layouts/AppShell.svelte';
  import AppSidebar from '$lib/layouts/AppSidebar.svelte';
  import AppUserMenu from '$lib/components/AppUserMenu.svelte';
  import { persistAppLocale, resolveUnauthenticatedLocale, useI18n } from '$lib/i18n';

  let { children }: { children: Snippet } = $props();

  const { data: accountData } = useAccount();

  const { data: guardData } = useSessionGuard({
    readPathname: () => page.url.pathname,
    redirectTo: (intent) => {
      goto(intent);
    },
  });

  const { data: sessionData, actions: sessionActions } = useAuthSession();
  const locale = $derived(accountData.state.account?.setting.locale ?? resolveUnauthenticatedLocale());
  const i18n = $derived(useI18n(locale));

  // domain 層で副作用を集約
  useAccountLocaleSync((locale: 'ja' | 'en') => {
    persistAppLocale(locale);
  });
</script>

{#if guardData.state.phase === 'authenticated' && guardData.state.session !== null}
  <AppShell
    sidebarTriggerLabel={i18n.t('common.shell.sidebarAriaLabel')}
  >
    {#snippet sidebar()}
      <AppSidebar
        currentPath={page.url.pathname}
        labels={{
          sidebarAriaLabel: i18n.t('common.shell.sidebarAriaLabel'),
          sidebarClose: i18n.t('common.shell.sidebarClose'),
          home: i18n.t('common.shell.home'),
          gettingStarted: i18n.t('common.shell.gettingStarted'),
        }}
      >
        {#snippet userMenu()}
          <AppUserMenu
            sessions={sessionData.state.sessions ?? []}
            activeSessionId={sessionData.state.activeSessionId ?? null}
            onSwitch={sessionActions.switchSession}
            labels={{
              userMenuAriaLabel: i18n.t('common.shell.userMenuAriaLabel'),
              switchAccount: i18n.t('common.shell.switchAccount'),
              addAccount: i18n.t('common.shell.addAccount'),
              accountSettings: i18n.t('common.shell.accountSettings'),
              logout: i18n.t('common.shell.logout'),
              accountLabel: i18n.t('common.shell.accountLabel'),
            }}
          />
        {/snippet}
      </AppSidebar>
    {/snippet}

    {#key sessionData.state.activeSessionId}
      {@render children()}
    {/key}
  </AppShell>
{:else}
  <div class="flex min-h-screen items-center justify-center">
    <p class="text-sm text-muted-foreground">{i18n.t('common.loading')}</p>
  </div>
{/if}
