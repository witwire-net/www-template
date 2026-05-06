<script lang="ts">
  import { goto } from '$app/navigation';

  import { useRecoveryFlow } from '@www-template/domain/auth/recovery';
  import { Card, CardContent, Separator } from '@www-template/ui/components';

  import { removeQueryParamFromUrl } from '../../../../lib/auth/url';

  const { data, actions } = useRecoveryFlow();

  /** URL から token を取得し consume する。 */
  async function consumeTokenFromUrl() {
    const token = removeQueryParamFromUrl('token');

    if (token === null || token === '') {
      await goto('/login/recovery');
      return;
    }

    const result = await actions.consumeToken(token);
    if (result === '/login/recovery/register') {
      /*
       * consume → register は SvelteKit client-side routing で同一 module instance の
       * domain singleton state を共有する。sessionStorage には recovery secret を保存しない。
       */
      await goto('/login/recovery/register');
    } else if (result === '/login/recovery') {
      /* 画面に retry guidance を表示するのでそのまま留まる */
    }
  }

  /* mount 時に token consume を実行 */
  void consumeTokenFromUrl();
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
        <div class="flex flex-col items-center gap-4 text-center" role="region" aria-label="復旧リンク確認">
          {#if data.state.phase === 'consuming'}
            <h1 class="m-0 text-2xl font-bold text-center">復旧リンクを確認中…</h1>
            <p class="m-0 text-sm text-muted-foreground text-center">しばらくお待ちください。</p>
          {:else if data.state.phase === 'invalid'}
            <h1 class="m-0 text-2xl font-bold text-center">復旧リンクを確認できません</h1>
            <p class="m-0 text-sm text-muted-foreground text-center">
              {data.state.error ?? '復旧リンクが無効または期限切れです。再度復旧をお試しください。'}
            </p>

            <Separator />

            <a href="/login/recovery" class="text-sm text-muted-foreground no-underline hover:underline">復旧をやり直す</a>
          {:else}
            <h1 class="m-0 text-2xl font-bold text-center">復旧リンクを確認中…</h1>
            <p class="m-0 text-sm text-muted-foreground text-center">しばらくお待ちください。</p>
          {/if}
        </div>
      </CardContent>
    </Card>
  </main>

  <Separator />

  <footer class="flex justify-center py-4">
    <a href="/" class="text-sm text-muted-foreground no-underline hover:underline">公開サイトに戻る</a>
  </footer>
</div>
