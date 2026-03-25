<script lang="ts">
  import { useAuthSession } from '@www-template-frontend/domain/hooks/auth/useAuthSession';
  import { Button, Card, Divider, Link, Typography } from '@www-template-frontend/ui/components';

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
      window.location.href = '/app/login';
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
    <Link variant="ghost" href="/" aria-label="www-template トップページ">
      <Typography variant="body" weight="bold" className="auth-logo">www-template</Typography>
    </Link>
  </header>

  <Divider />

  <main class="auth-main">
    <Card padding="xl" className="auth-card">
      <div class="auth-card-content" role="region" aria-label="ログアウト">
        {#if isLoggingOut}
          <Typography variant="h1" weight="bold" align="center">ログアウト中…</Typography>
          <Typography variant="body-sm" color="secondary" align="center">
            セッションを終了しています。
          </Typography>
        {:else if logoutError}
          <Typography variant="h1" weight="bold" align="center">ログアウト</Typography>
          <Typography variant="body-sm" className="auth-error" role="alert">
            {logoutError}
          </Typography>
          <Button variant="secondary" fullWidth onclick={() => { window.location.href = '/app/login'; }}>
            ログインへ
          </Button>
        {:else}
          <Typography variant="h1" weight="bold" align="center">ログアウトしました</Typography>
          <Button variant="secondary" fullWidth onclick={() => { window.location.href = '/app/login'; }}>
            ログインへ
          </Button>
        {/if}
      </div>
    </Card>
  </main>

  <Divider />

  <footer class="auth-footer">
    <Link variant="muted" href="/">公開サイトに戻る</Link>
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

  :global(.auth-logo) {
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

  :global(.auth-card) {
    width: 100%;
  }

  .auth-card-content {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: var(--spacing-md);
    text-align: center;
  }

  :global(.auth-error) {
    color: var(--color-error);
  }

  .auth-footer {
    display: flex;
    justify-content: center;
    padding: var(--spacing-md) 0;
  }
</style>
