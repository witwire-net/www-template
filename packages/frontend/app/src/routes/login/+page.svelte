<script lang="ts">
  import { goto } from '$app/navigation';

  import { usePasskeyLogin } from '@www-template/domain/auth/passkey';
  import { Button, Card, CardContent, Separator } from '@www-template/ui/components';

  const { data, actions } = usePasskeyLogin();

  async function handlePasskeySignIn() {
    const result = await actions.signInWithPasskey();
    if (result === null && data.state.lastSession !== null) {
      await goto('/');
    }
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
          <h1 class="m-0 text-2xl font-bold text-center">ログイン</h1>
          <p class="m-0 text-sm text-muted-foreground text-center">パスキーを使ってサインインしてください。</p>

          {#if data.state.error}
            <p class="text-destructive text-sm m-0" role="alert">{data.state.error}</p>
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

          <a href="/login/recovery" class="text-sm text-muted-foreground no-underline hover:underline">パスキーを紛失した場合</a>
        </div>
      </CardContent>
    </Card>
  </main>

  <Separator />

  <footer class="flex justify-center py-4">
    <a href="/" class="text-sm text-muted-foreground no-underline hover:underline">公開サイトに戻る</a>
  </footer>
</div>


