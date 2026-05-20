<script lang="ts">
  import type { Snippet } from 'svelte';

  import { goto } from '$app/navigation';

  import { useAccount } from '@www-template/domain';
  import { useSessionGuard } from '@www-template/domain/auth/guard';
  import { useAuthSession } from '@www-template/domain/auth/session';
  import { useAccountLocaleSync } from '@www-template/domain/account';

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
</script>

{#if guardData.state.phase === 'authenticated' && guardData.state.session !== null}
  <section class="app-layout">
    <div class="app-layout__banner">
      <div>
        <div class="app-layout__banner-eyebrow">{i18n.t('common.eyebrow')}</div>
        <h1 class="app-layout__banner-title">{i18n.t('common.appTitle')}</h1>
      </div>
      <div class="app-layout__links">
        <a href="/">{i18n.t('common.overview')}</a>
        <a href="/passkeys/">{i18n.t('common.passkeys')}</a>
        <a href="/sessions">{i18n.t('common.sessions')}</a>
        <a href="/settings">{i18n.t('common.settings')}</a>
        <!--
          別アカウントを追加するための導線。
          SvelteKit の client-side ナビゲーションによりページ遷移し、
          メモリ上の既存セッションは保持される。
          ログイン後は複数セッションが並存する状態になる。
        -->
        <a href="/login">{i18n.t('common.addAccount')}</a>
        <a href="/logout">{i18n.t('common.logout')}</a>
      </div>
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
    </div>

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
