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

<div class="flex flex-col items-center min-h-screen px-4 py-8 font-sans bg-background text-foreground">
  <header class="flex justify-center py-4">
    <a href="/" class="no-underline text-inherit" aria-label="www-template トップページ">
      <span class="font-bold tracking-[0.08em]">www-template</span>
    </a>
  </header>

  <Separator />

  <main class="flex flex-1 w-full max-w-[400px] items-center justify-center py-8">
    <Card class="w-full">
      <CardContent>
        <div class="flex flex-col items-center gap-4 text-center">
          <h1 class="m-0 text-2xl font-bold text-center">パスキー再登録</h1>
          <p class="m-0 text-sm text-muted-foreground text-center">
            新しいパスキーを登録して、アカウントへのアクセスを回復してください。
          </p>

          {#if data.state.error}
            <p class="text-destructive text-sm m-0" role="alert">{data.state.error}</p>
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

          <a href="/login/recovery" class="text-sm text-muted-foreground no-underline hover:underline">復旧をやり直す</a>
        </div>
      </CardContent>
    </Card>
  </main>

  <Separator />

  <footer class="flex justify-center py-4">
    <a href="/" class="text-sm text-muted-foreground no-underline hover:underline">公開サイトに戻る</a>
  </footer>
</div>


