<script lang="ts">
  import { goto } from '$app/navigation';

  import { useRecoveryFlow } from '@www-template/domain/auth/recovery';
  import { Button, Card, CardContent, Separator } from '@www-template/ui/components';

  const { data, actions } = useRecoveryFlow();

  async function handleRegisterPasskey() {
    const result = await actions.registerRecoveryPasskey();
    if (result === null && data.state.phase !== 'registering' && data.state.error === null) {
      await goto('/');
    }
  }

  /*
   * recovery session は domain singleton state で共有し、sessionStorage には保存しない。
   * 直接アクセスやリロードで state が失われた場合は安全に復旧導線へ戻す。
   */
  if (typeof window !== 'undefined' && data.state.recoverySession === null) {
    void goto('/login/recovery');
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
          <h1 class="auth-card__title">パスキー再登録</h1>
          <p class="auth-card__desc">
            新しいパスキーを登録して、アカウントへのアクセスを回復してください。
          </p>

          {#if data.state.error}
            <p class="auth-card__error" role="alert">{data.state.error}</p>
          {/if}

          <Button
            class="w-full"
            type="button"
            disabled={data.state.phase === 'registering'}
            onclick={handleRegisterPasskey}
          >
            {#if data.state.phase === 'registering'}
              登録中…
            {:else}
              新しいパスキーを登録
            {/if}
          </Button>

          <Separator />

          <a href="/login/recovery" class="link-muted">復旧をやり直す</a>
        </div>
      </CardContent>
    </Card>
  </main>

  <Separator />

  <footer class="auth-layout__footer">
    <a href="/" class="link-muted">公開サイトに戻る</a>
  </footer>
</div>
