<script lang="ts">
  import { goto } from '$app/navigation';

  import AuthLayout from '$lib/layouts/AuthLayout.svelte';
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

<AuthLayout>
  <Card class="w-full">
    <CardContent>
      <div class="flex flex-col items-center gap-4 text-center">
        <h1 class="m-0 text-2xl font-bold text-center">パスキー復旧</h1>
        <p class="m-0 text-sm text-muted-foreground text-center">
          登録済みのメールアドレスを入力してください。復旧用のリンクをお送りします。
        </p>

        {#if data.state.error}
          <p class="text-destructive text-sm m-0" role="alert">{data.state.error}</p>
        {/if}

        <form class="w-full flex flex-col gap-2" onsubmit={handleSubmit}>
          <div class="flex flex-col gap-1 text-left">
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

        <a href="/login" class="text-sm text-muted-foreground no-underline hover:underline">ログインに戻る</a>
      </div>
    </CardContent>
  </Card>

  {#snippet footer()}
    <a href="/" class="text-sm text-muted-foreground no-underline hover:underline">公開サイトに戻る</a>
  {/snippet}
</AuthLayout>
