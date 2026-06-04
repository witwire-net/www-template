<script lang="ts">
  import type { Snippet } from 'svelte';

  import { goto } from '$app/navigation';

  import { useAccount } from '@www-template/domain';
  import { useSessionGuard } from '@www-template/domain/auth/guard';
  import { useAuthSession } from '@www-template/domain/auth/session';
  import { useAccountLocaleSync } from '@www-template/domain/account';
  import { BrandMark, PageHeader } from '@www-template/ui';

  import AccountSwitcher from '$lib/components/AccountSwitcher.svelte';
  import { persistAppLocale, resolveUnauthenticatedLocale, useI18n } from '$lib/i18n';

  let { children }: { children: Snippet } = $props();

  const { data: accountData } = useAccount();

  const { data: guardData } = useSessionGuard({
    readPathname: () => window.location.pathname,
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

  type NavLink = {
    href: string;
    labelKey: 'common.overview' | 'common.passkeys' | 'common.sessions' | 'common.settings';
  };

  let navLinks: NavLink[] = $derived([
    { href: '/', labelKey: 'common.overview' },
    { href: '/passkeys', labelKey: 'common.passkeys' },
    { href: '/sessions', labelKey: 'common.sessions' },
    { href: '/settings', labelKey: 'common.settings' },
  ]);
</script>

{#if guardData.state.phase === 'authenticated' && guardData.state.session !== null}
  <section class="app-layout">
    <PageHeader>
      <BrandMark size="sm" />
      <h1 class="app-layout__banner-title">{i18n.t('common.appTitle')}</h1>

      {#snippet trailing()}
        <nav class="app-layout__nav" aria-label={i18n.t('common.appNavAriaLabel')}>
          {#each navLinks as link (link.href)}
            <a class="app-layout__nav-link" href={link.href} data-active={link.href === '/'}>
              {i18n.t(link.labelKey)}
            </a>
          {/each}
          <a class="app-layout__nav-link" href="/login" data-active={false}>
            {i18n.t('common.addAccount')}
          </a>
          <a class="app-layout__nav-link" href="/logout" data-active={false}>
            {i18n.t('common.logout')}
          </a>
        </nav>
        <!--
          複数アカウントログイン時にアカウント切り替え UI を表示する。
          AccountSwitcher は sessions.length > 1 の場合のみレンダリングされる。
          切り替え操作はメモリ上の activeSessionId のみを変更し、永続化は行わない。
        -->
        <AccountSwitcher
          sessions={sessionData.state.sessions ?? []}
          activeSessionId={sessionData.state.activeSessionId ?? null}
          onSwitch={sessionActions.switchSession}
        />
      {/snippet}
    </PageHeader>

    <div class="app-layout__panel">
      <!--
        activeSessionId が変わった際に子コンポーネントを remount し、
        各ページの初期化ロジック（例: デバイス一覧取得）を再実行する。
        これによりアカウント切り替え後も正しいアカウントのデータが表示される。
      -->
      {#key sessionData.state.activeSessionId}
        {@render children()}
      {/key}
    </div>
  </section>
{:else}
  <div class="app-layout__loading">
    <p>{i18n.t('common.loading')}</p>
  </div>
{/if}
