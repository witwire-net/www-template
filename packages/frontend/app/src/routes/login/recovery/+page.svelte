<script lang="ts">
  import { goto } from '$app/navigation';

  import { useRecoveryFlow } from '@www-template/domain/auth/recovery';
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
        <div class="auth-card">
          <h1 class="auth-card__title">パスキー復旧</h1>
          <p class="auth-card__desc">
            登録済みのメールアドレスを入力してください。復旧用のリンクをお送りします。
          </p>

          {#if data.state.error}
            <p class="auth-card__error" role="alert">{data.state.error}</p>
          {/if}

          <form class="auth-card__form" onsubmit={handleSubmit}>
            <div class="auth-card__input-field">
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

  <footer class="auth-layout__footer">
    <a href="/" class="link-muted">公開サイトに戻る</a>
  </footer>
</div>
