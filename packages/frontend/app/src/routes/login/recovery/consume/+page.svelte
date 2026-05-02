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
        <div class="auth-card" role="region" aria-label="復旧リンク確認">
          {#if data.state.phase === 'consuming'}
            <h1 class="auth-card__title">復旧リンクを確認中…</h1>
            <p class="auth-card__desc">しばらくお待ちください。</p>
          {:else if data.state.phase === 'invalid'}
            <h1 class="auth-card__title">復旧リンクを確認できません</h1>
            <p class="auth-card__desc">
              {data.state.error ?? '復旧リンクが無効または期限切れです。再度復旧をお試しください。'}
            </p>

            <Separator />

            <a href="/login/recovery" class="link-muted">復旧をやり直す</a>
          {:else}
            <h1 class="auth-card__title">復旧リンクを確認中…</h1>
            <p class="auth-card__desc">しばらくお待ちください。</p>
          {/if}
        </div>
      </CardContent>
    </Card>
  </main>

  <Separator />

  <footer class="auth-layout__footer">
    <a href="/" class="link-muted">公開サイトに戻る</a>
  </footer>
</div>
