<script lang="ts">
  import { goto } from '$app/navigation';

  import { useRecoveryFlow } from '@www-template/domain/hooks/auth/useRecoveryFlow';
  import { Button, Card, CardContent, Input, Label, Separator } from '@www-template/ui/components';

  const { data, actions } = useRecoveryFlow();

  async function handleSubmit(event: SubmitEvent) {
    event.preventDefault();
    if ((await actions.submitRecoveryRequest()) === '/login/recovery/sent') {
      await goto('/login/recovery/sent');
    }
  }

  function handleEmailInput(event: Event) {
    const target = event.target as HTMLInputElement;
    actions.setEmail(target.value);
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
          <h1 class="auth-title">パスキー復旧</h1>
          <p class="auth-desc">
            登録済みのメールアドレスを入力してください。復旧用のリンクをお送りします。
          </p>

          {#if data.state.error}
            <p class="auth-error" role="alert">{data.state.error}</p>
          {/if}

          <form class="auth-form" onsubmit={handleSubmit}>
            <div class="input-field">
              <Label for="recovery-email">メールアドレス</Label>
              <Input
                id="recovery-email"
                type="email"
                autocomplete="email"
                required
                placeholder="you@example.com"
                value={data.state.email}
                oninput={handleEmailInput}
                disabled={data.state.phase === 'submitting'}
              />
            </div>

            <Button
              class="w-full"
              type="submit"
              disabled={data.state.phase === 'submitting' || data.state.email.trim() === ''}
            >
              {#if data.state.phase === 'submitting'}
                送信中…
              {:else}
                復旧メールを送信
              {/if}
            </Button>
          </form>

          <Separator />

          <a href="/login" class="link-muted">ログインに戻る</a>
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

  .auth-form {
    width: 100%;
    display: flex;
    flex-direction: column;
    gap: var(--spacing-sm);
  }

  .input-field {
    display: flex;
    flex-direction: column;
    gap: 0.25rem;
    text-align: left;
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
