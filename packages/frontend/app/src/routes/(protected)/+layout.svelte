<script lang="ts">
  import type { Snippet } from 'svelte';

  import { goto } from '$app/navigation';

  import { useSessionGuard } from '@www-template/domain/auth/guard';

  let { children }: { children: Snippet } = $props();

  const { data } = useSessionGuard({
    readPathname: () => window.location.pathname,
    redirectTo: (intent) => {
      goto(intent);
    },
  });
</script>

{#if data.state.phase === 'authenticated' && data.state.session !== null}
  <section class="app-layout">
    <div class="app-layout__banner">
      <div>
        <div class="app-layout__banner-eyebrow">SVELTEKIT SPA SHELL</div>
        <h1 class="app-layout__banner-title">認証済みアプリ</h1>
      </div>
      <div class="app-layout__links">
        <a href="/">Overview</a>
        <a href="/passkeys/">パスキー管理</a>
        <a href="/logout">ログアウト</a>
      </div>
    </div>

    <div class="app-layout__panel">
      {@render children()}
    </div>
  </section>
{:else}
  <div class="app-layout__loading">
    <p>セッションを確認しています…</p>
  </div>
{/if}
