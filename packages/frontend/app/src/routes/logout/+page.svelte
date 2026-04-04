<script lang="ts">
  import { useAuthSession } from '@www-template/domain/hooks/auth/useAuthSession';
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
      await actions.logoutCurrentSession();
      window.location.href = '/login';
    } catch {
      logoutError = 'ログアウトに失敗しました。';
      isLoggingOut = false;
      /* fail-safe: state 消去して login へ */
      actions.clearInMemorySession();
    }
  }
</script>

<div class="auth-shell">
  <header class="auth-header">
    <a href="/" class="site-link" aria-label="www-template トップページ">
      <span class="logo-text">www-template</span>
    </a>
  </header>

  <Separator />

  <main class="auth-main">
    <Card class="w-full">
      <CardContent>
        <div class="auth-card-content" role="region" aria-label="ログアウト">
          {#if isLoggingOut}
            <h1 class="auth-title">ログアウト中…</h1>
            <p class="auth-desc">セッションを終了しています。</p>
          {:else if logoutError}
            <h1 class="auth-title">ログアウト</h1>
            <p class="auth-error" role="alert">{logoutError}</p>
            <Button variant="secondary" class="w-full" onclick={() => { window.location.href = '/login'; }}>
              ログインへ
            </Button>
          {:else}
            <h1 class="auth-title">ログアウトしました</h1>
            <Button variant="secondary" class="w-full" onclick={() => { window.location.href = '/login'; }}>
              ログインへ
            </Button>
          {/if}
        </div>
      </CardContent>
    </Card>
  </main>

  <Separator />

  <footer class="auth-footer">
    <a href="/" class="link-muted">公開サイトに戻る</a>
  </footer>
</div>

<style>
  .auth-shell {
    display: flex;
    flex-direction: column;
    align-items: center;
    min-height: 100vh;
    padding: var(--spacing-xl) var(--spacing-md);
    font-family: var(--font-family-sans);
    background: var(--color-background);
    color: var(--color-text);
  }

  .auth-header {
    display: flex;
    justify-content: center;
    padding: var(--spacing-md) 0;
  }

  .site-link {
    text-decoration: none;
    color: inherit;
  }

  .logo-text {
    font-weight: bold;
    letter-spacing: 0.08em;
  }

  .auth-main {
    display: flex;
    flex: 1;
    align-items: center;
    justify-content: center;
    width: 100%;
    max-width: 400px;
    padding: var(--spacing-xl) 0;
  }

  .auth-card-content {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: var(--spacing-md);
    text-align: center;
  }

  .auth-title {
    margin: 0;
    font-size: 1.5rem;
    font-weight: bold;
    text-align: center;
  }

  .auth-desc {
    margin: 0;
    font-size: 0.875rem;
    color: var(--muted-foreground);
    text-align: center;
  }

  .auth-error {
    color: var(--destructive);
    font-size: 0.875rem;
    margin: 0;
  }

  .auth-footer {
    display: flex;
    justify-content: center;
    padding: var(--spacing-md) 0;
  }

  .link-muted {
    font-size: 0.875rem;
    color: var(--muted-foreground);
    text-decoration: none;
  }

  .link-muted:hover {
    text-decoration: underline;
  }
</style>
