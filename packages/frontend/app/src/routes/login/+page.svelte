<script lang="ts">
  import { goto } from '$app/navigation';

  import { usePasskeyLogin } from '@www-template-frontend/domain/hooks/auth/usePasskeyLogin';
  import { Button, Card, Divider, Link, Typography } from '@www-template-frontend/ui/components';

  const { data, actions } = usePasskeyLogin();

  async function handlePasskeySignIn() {
    const result = await actions.signInWithPasskey();
    if (result === '/app') {
      await goto('/app');
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
      <div class="auth-card-content">
        <Typography variant="h1" weight="bold" align="center">ログイン</Typography>
        <Typography variant="body-sm" color="secondary" align="center">
          パスキーを使ってサインインしてください。
        </Typography>

        {#if data.state.error}
          <Typography variant="body-sm" color="primary" className="auth-error" role="alert">
            {data.state.error}
          </Typography>
        {/if}

        <Button
          variant="primary"
          fullWidth
          type="button"
          disabled={data.state.isSubmitting}
          isLoading={data.state.isSubmitting}
          onclick={handlePasskeySignIn}
        >
          {#if data.state.isSubmitting}
            認証中…
          {:else}
            パスキーでログイン
          {/if}
        </Button>

        <Divider />

        <Link variant="muted" href="/app/login/recovery">パスキーを紛失した場合</Link>
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
