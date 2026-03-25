<script lang="ts">
  import { useRecoveryFlow } from '@www-template-frontend/domain/hooks/auth/useRecoveryFlow';
  import { Button, Card, Divider, Input, Link, Typography } from '@www-template-frontend/ui/components';

  const { data, actions } = useRecoveryFlow();

  async function handleSubmit(event: SubmitEvent) {
    event.preventDefault();
    if ((await actions.submitRecoveryRequest()) === '/app/login/recovery/sent') {
      window.location.href = '/app/login/recovery/sent';
    }
  }

  function handleEmailInput(event: Event) {
    const target = event.target as HTMLInputElement;
    actions.setEmail(target.value);
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
        <Typography variant="h1" weight="bold" align="center">パスキー復旧</Typography>
        <Typography variant="body-sm" color="secondary" align="center">
          登録済みのメールアドレスを入力してください。復旧用のリンクをお送りします。
        </Typography>

        {#if data.state.error}
          <Typography variant="body-sm" className="auth-error" role="alert">
            {data.state.error}
          </Typography>
        {/if}

        <form class="auth-form" onsubmit={handleSubmit}>
          <Input
            id="recovery-email"
            label="メールアドレス"
            type="email"
            autocomplete="email"
            required
            placeholder="you@example.com"
            value={data.state.email}
            oninput={handleEmailInput}
            disabled={data.state.phase === 'submitting'}
            fullWidth
          />

          <Button
            variant="primary"
            fullWidth
            type="submit"
            disabled={data.state.phase === 'submitting' || data.state.email.trim() === ''}
            isLoading={data.state.phase === 'submitting'}
          >
            {#if data.state.phase === 'submitting'}
              送信中…
            {:else}
              復旧メールを送信
            {/if}
          </Button>
        </form>

        <Divider />

        <Link variant="muted" href="/app/login">ログインに戻る</Link>
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

  .auth-form {
    width: 100%;
    display: flex;
    flex-direction: column;
    gap: var(--spacing-sm);
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
