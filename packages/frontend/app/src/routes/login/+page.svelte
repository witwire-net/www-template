<script lang="ts">
  import { goto } from '$app/navigation';

  import { usePasskeyLogin } from '@www-template/domain/hooks/auth/usePasskeyLogin';
  import { Button, Card, CardContent, Separator } from '@www-template/ui/components';

  const { data, actions } = usePasskeyLogin();

  async function handlePasskeySignIn() {
    const result = await actions.signInWithPasskey();
    if (result === null && data.state.lastSession !== null) {
      await goto('/');
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
        <div class="auth-card">
          <h1 class="auth-card__title">ログイン</h1>
          <p class="auth-card__desc">パスキーを使ってサインインしてください。</p>

          {#if data.state.error}
            <p class="auth-card__error" role="alert">{data.state.error}</p>
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

  <footer class="auth-layout__footer">
    <a href="/" class="link-muted">公開サイトに戻る</a>
  </footer>
</div>
