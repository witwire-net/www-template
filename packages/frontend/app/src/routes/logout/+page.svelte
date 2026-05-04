<script lang="ts">
  import { goto } from '$app/navigation';

  import { useAuthSession } from '@www-template/domain/auth/session';
  import { Button, Card, CardContent, Separator } from '@www-template/ui/components';

  const { actions } = useAuthSession();

  let isLoggingOut = $state(true);
  let logoutError = $state<string | null>(null);

  /** mount 時に logout を実行する。browser 環境でのみ発火。 */
  if (typeof window !== 'undefined') {
    void performLogout();
  }

  async function performLogout() {
    try {
      const intent = await actions.logoutCurrentSession();
      // 残りセッションがある場合は intent が null となり認証状態を維持する
      await goto(intent ?? '/');
    } catch {
      logoutError = 'ログアウトに失敗しました。';
      isLoggingOut = false;
      /* fail-safe: state 消去して login へ */
      actions.clearInMemorySession();
      await goto('/login');
    }
  }
</script>

<div class="auth-layout">
  <header class="auth-layout__header">
    <a href="/" class="site-link" aria-label="www-template トップページ">
      <span class="logo-text">www-template</span>
    </a>
  </header>

  <Separator />

  <main class="auth-layout__main">
    <Card class="w-full">
      <CardContent>
        <div class="auth-card" role="region" aria-label="ログアウト">
          {#if isLoggingOut}
            <h1 class="auth-card__title">ログアウト中…</h1>
            <p class="auth-card__desc">セッションを終了しています。</p>
          {:else if logoutError}
            <h1 class="auth-card__title">ログアウト</h1>
            <p class="auth-card__error" role="alert">{logoutError}</p>
            <Button variant="secondary" class="w-full" onclick={() => { void goto('/login'); }}>
              ログインへ
            </Button>
          {:else}
            <h1 class="auth-card__title">ログアウトしました</h1>
            <Button variant="secondary" class="w-full" onclick={() => { void goto('/login'); }}>
              ログインへ
            </Button>
          {/if}
        </div>
      </CardContent>
    </Card>
  </main>

  <Separator />

  <footer class="auth-layout__footer">
    <a href="/" class="link-muted">公開サイトに戻る</a>
  </footer>
</div>
