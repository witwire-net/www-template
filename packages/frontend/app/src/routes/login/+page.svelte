<script lang="ts">
  import { usePasskeyLogin } from '@www-template/domain/hooks/auth/usePasskeyLogin';
  import { Button, Card, CardContent, Separator } from '@www-template/ui/components';

  const { data, actions } = usePasskeyLogin();

  async function handlePasskeySignIn() {
    const result = await actions.signInWithPasskey();
    if (result === null && data.state.lastSession !== null) {
      window.location.href = '/';
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
        <div class="auth-card-content">
          <h1 class="auth-title">ログイン</h1>
          <p class="auth-desc">パスキーを使ってサインインしてください。</p>

          {#if data.state.error}
            <p class="auth-error" role="alert">{data.state.error}</p>
          {/if}

          <Button
            class="w-full"
            type="button"
            disabled={data.state.isSubmitting}
            onclick={handlePasskeySignIn}
          >
            {#if data.state.isSubmitting}
              認証中…
            {:else}
              パスキーでログイン
            {/if}
          </Button>

          <Separator />

          <a href="/login/recovery" class="link-muted">パスキーを紛失した場合</a>
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
