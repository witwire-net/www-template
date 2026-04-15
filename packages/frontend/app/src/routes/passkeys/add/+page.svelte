<script lang="ts">
  import { usePasskeyAddByOtp } from '@www-template/domain/hooks/auth/usePasskeyAddByOtp';
  import { Button, Card, CardContent, InputOtp, Label, Separator } from '@www-template/ui/components';

  const { data, actions } = usePasskeyAddByOtp();

  let otp = $state('');
  let localError = $state<string | null>(null);

  let displayError = $derived(data.error ?? localError);
  let isReady = $derived(otp.trim().length === 6 && !data.loading);

  async function handleSubmit(event: SubmitEvent): Promise<void> {
    event.preventDefault();
    localError = null;

    try {
      await actions.addPasskeyByOtp(otp);
    } catch {
      localError = 'パスキーの登録を完了できませんでした。';
    }
  }
</script>

<div class="flex min-h-svh flex-col items-center px-6 py-10 font-sans">
  <header class="flex justify-center py-4">
    <a href="/" aria-label="www-template トップページ" class="text-inherit no-underline">
      <span class="font-bold tracking-wide">www-template</span>
    </a>
  </header>

  <Separator />

  <main class="flex flex-1 w-full max-w-sm items-center justify-center py-10">
    <Card class="w-full">
      <CardContent>
        <div class="flex flex-col items-center gap-4 text-center">
          {#if data.done}
            <h1 class="m-0 text-2xl font-bold">パスキーを登録しました</h1>
            <p class="m-0 text-sm text-muted-foreground">このページを閉じて、既存のデバイスでログインしてください。</p>

            <Separator />

            <a href="/login" class="text-sm text-muted-foreground no-underline hover:underline">ログインページへ</a>
          {:else}
            <h1 class="m-0 text-2xl font-bold">パスキーを追加</h1>
            <p class="m-0 text-sm text-muted-foreground">認証済みデバイスで発行された 6 桁の OTP を入力してください。</p>

            {#if displayError !== null}
              <p class="m-0 text-sm text-destructive" role="alert">{displayError}</p>
            {/if}

            <form class="flex w-full flex-col gap-2" onsubmit={handleSubmit}>
              <div class="flex flex-col gap-1 text-left">
                <Label>ワンタイムパスワード</Label>
                <div class="flex justify-center">
                  <InputOtp.InputOTP
                    aria-label="ワンタイムパスワード"
                    inputId="otp-input"
                    maxlength={6}
                    bind:value={otp}
                    disabled={data.loading}
                    autocomplete="one-time-code"
                  >
                    {#snippet children({ cells })}
                      <InputOtp.InputOTPGroup>
                        {#each cells.slice(0, 3) as cell (cell)}
                          <InputOtp.InputOTPSlot {cell} />
                        {/each}
                      </InputOtp.InputOTPGroup>
                      <InputOtp.InputOTPSeparator />
                      <InputOtp.InputOTPGroup>
                        {#each cells.slice(3, 6) as cell (cell)}
                          <InputOtp.InputOTPSlot {cell} />
                        {/each}
                      </InputOtp.InputOTPGroup>
                    {/snippet}
                  </InputOtp.InputOTP>
                </div>
              </div>

              <Button class="w-full" type="submit" disabled={!isReady}>
                {#if data.loading}
                  登録中…
                {:else}
                  パスキーを登録
                {/if}
              </Button>
            </form>

            <Separator />

            <a href="/login" class="text-sm text-muted-foreground no-underline hover:underline">ログインに戻る</a>
          {/if}
        </div>
      </CardContent>
    </Card>
  </main>

  <Separator />

  <footer class="flex justify-center py-4">
    <a href="/login" class="text-sm text-muted-foreground no-underline hover:underline">ログインに戻る</a>
  </footer>
</div>
