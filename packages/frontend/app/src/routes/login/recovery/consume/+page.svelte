<script lang="ts">
  import { useRecoveryFlow } from '@www-template/domain/hooks/auth/useRecoveryFlow';
  import { Card, CardContent, Separator } from '@www-template/ui/components';

  const RECOVERY_SNAPSHOT_KEY = 'www-template:recovery-snapshot';

  const { data, actions } = useRecoveryFlow();

  /** URL から token を取得し consume する。 */
  async function consumeTokenFromUrl() {
    if (typeof window === 'undefined') {
      return;
    }

    const params = new URLSearchParams(window.location.search);
    const token = params.get('token');

    if (token === null || token === '') {
      window.location.href = '/login/recovery';
      return;
    }

    const result = await actions.consumeToken(token);
    if (result === '/login/recovery/register') {
      /* フルリロード遷移で module state が消えるため sessionStorage に snapshot を保存 */
      const snapshot = actions.getReadySnapshot();
      if (snapshot !== null) {
        sessionStorage.setItem(RECOVERY_SNAPSHOT_KEY, JSON.stringify(snapshot));
      }
      window.location.href = '/login/recovery/register';
    } else if (result === '/login/recovery') {
      /* 画面に retry guidance を表示するのでそのまま留まる */
    }
  }

  /* mount 時に token consume を実行 */
  void consumeTokenFromUrl();
</script>

<div class="auth-shell">
  <header class="auth-header">
    <a href="/" class="site-link" aria-label="www-template トップページ">
      <span class="logo-text">www-template</span>
    </a>
  </header>

  <Separator />

  <main class="auth-main">
    <Card class="w-full">
      <CardContent>
        <div class="auth-card-content" role="region" aria-label="復旧リンク確認">
          {#if data.state.phase === 'consuming'}
            <h1 class="auth-title">復旧リンクを確認中…</h1>
            <p class="auth-desc">しばらくお待ちください。</p>
          {:else if data.state.phase === 'invalid'}
            <h1 class="auth-title">復旧リンクを確認できません</h1>
            <p class="auth-desc">
              {data.state.error ?? '復旧リンクが無効または期限切れです。再度復旧をお試しください。'}
            </p>

            <Separator />

            <a href="/login/recovery" class="link-muted">復旧をやり直す</a>
          {:else}
            <h1 class="auth-title">復旧リンクを確認中…</h1>
            <p class="auth-desc">しばらくお待ちください。</p>
          {/if}
        </div>
      </CardContent>
    </Card>
  </main>

  <Separator />

  <footer class="auth-footer">
    <a href="/" class="link-muted">公開サイトに戻る</a>
  </footer>
</div>

<style>
  .auth-shell {
    display: flex;
    flex-direction: column;
    align-items: center;
    min-height: 100vh;
    padding: var(--spacing-xl) var(--spacing-md);
    font-family: var(--font-family-sans);
    background: var(--color-background);
    color: var(--color-text);
  }

  .auth-header {
    display: flex;
    justify-content: center;
    padding: var(--spacing-md) 0;
  }

  .site-link {
    text-decoration: none;
    color: inherit;
  }

  .logo-text {
    font-weight: bold;
    letter-spacing: 0.08em;
  }

  .auth-main {
    display: flex;
    flex: 1;
    align-items: center;
    justify-content: center;
    width: 100%;
    max-width: 400px;
    padding: var(--spacing-xl) 0;
  }

  .auth-card-content {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: var(--spacing-md);
    text-align: center;
  }

  .auth-title {
    margin: 0;
    font-size: 1.5rem;
    font-weight: bold;
    text-align: center;
  }

  .auth-desc {
    margin: 0;
    font-size: 0.875rem;
    color: var(--muted-foreground);
    text-align: center;
  }

  .auth-footer {
    display: flex;
    justify-content: center;
    padding: var(--spacing-md) 0;
  }

  .link-muted {
    font-size: 0.875rem;
    color: var(--muted-foreground);
    text-decoration: none;
  }

  .link-muted:hover {
    text-decoration: underline;
  }
</style>
