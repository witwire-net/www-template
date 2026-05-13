<script lang="ts">
  import { goto } from '$app/navigation';

  import AuthLayout from '$lib/layouts/AuthLayout.svelte';
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

<AuthLayout>
  <Card class="w-full">
    <CardContent>
      <div class="flex flex-col items-center gap-4 text-center">
        {#if data.state.kind === 'device-link'}
          <h1 class="m-0 text-2xl font-bold text-center">新しい端末でパスキーを登録</h1>
          <p class="m-0 text-sm text-muted-foreground text-center">
            新しい端末でパスキーを登録して、ログインできるようにしてください。
          </p>
        {:else}
          <h1 class="m-0 text-2xl font-bold text-center">パスキー再登録</h1>
          <p class="m-0 text-sm text-muted-foreground text-center">
            新しいパスキーを登録して、アカウントへのアクセスを回復してください。
          </p>
        {/if}

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

  {#snippet footer()}
    <a href="/" class="text-sm text-muted-foreground no-underline hover:underline">公開サイトに戻る</a>
  {/snippet}
</AuthLayout>
