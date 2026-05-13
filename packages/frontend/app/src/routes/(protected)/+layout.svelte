<script lang="ts">
  import type { Snippet } from 'svelte';

  import { goto } from '$app/navigation';

  import { useSessionGuard } from '@www-template/domain/auth/guard';
  import { useAuthSession } from '@www-template/domain/auth/session';

  import AccountSwitcher from '../../lib/components/AccountSwitcher.svelte';

  let { children }: { children: Snippet } = $props();

  const { data: guardData } = useSessionGuard({
    readPathname: () => window.location.pathname,
    redirectTo: (intent) => {
      goto(intent);
    },
  });

  const { data: sessionData, actions: sessionActions } = useAuthSession();
</script>

{#if guardData.state.phase === 'authenticated' && guardData.state.session !== null}
  <section class="app-layout">
    <div class="app-layout__banner">
      <div>
        <div class="app-layout__banner-eyebrow">SVELTEKIT SPA SHELL</div>
        <h1 class="app-layout__banner-title">認証済みアプリ</h1>
      </div>
      <div class="app-layout__links">
        <a href="/">Overview</a>
        <a href="/passkeys/">パスキー管理</a>
        <a href="/sessions">デバイス管理</a>
        <!--
          別アカウントを追加するための導線。
          SvelteKit の client-side ナビゲーションによりページ遷移し、
          メモリ上の既存セッションは保持される。
          ログイン後は複数セッションが並存する状態になる。
        -->
        <a href="/login">別アカウントを追加</a>
        <a href="/logout">ログアウト</a>
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
    <p>セッションを確認しています…</p>
  </div>
{/if}
