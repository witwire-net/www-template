<script lang="ts">
  import { useRecoveryFlow } from '@www-template-frontend/domain/hooks/auth/useRecoveryFlow';
  import { Card, Divider, Link, Typography } from '@www-template-frontend/ui/components';

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
      window.location.href = '/app/login/recovery';
      return;
    }

    const result = await actions.consumeToken(token);
    if (result === '/app/login/recovery/register') {
      /* フルリロード遷移で module state が消えるため sessionStorage に snapshot を保存 */
      const snapshot = actions.getReadySnapshot();
      if (snapshot !== null) {
        sessionStorage.setItem(RECOVERY_SNAPSHOT_KEY, JSON.stringify(snapshot));
      }
      window.location.href = '/app/login/recovery/register';
    } else if (result === '/app/login/recovery') {
      /* 画面に retry guidance を表示するのでそのまま留まる */
    }
  }

  /* mount 時に token consume を実行 */
  void consumeTokenFromUrl();
</script>

<div class="auth-shell">
  <header class="auth-header">
    <Link variant="ghost" href="/" aria-label="www-template トップページ">
      <Typography variant="body" weight="bold" className="auth-logo">www-template</Typography>
    </Link>
  </header>

  <Divider />

  <main class="auth-main">
    <Card padding="xl" className="auth-card">
      <div class="auth-card-content" role="region" aria-label="復旧リンク確認">
        {#if data.state.phase === 'consuming'}
          <Typography variant="h1" weight="bold" align="center">復旧リンクを確認中…</Typography>
          <Typography variant="body-sm" color="secondary" align="center">しばらくお待ちください。</Typography>
        {:else if data.state.phase === 'invalid'}
          <Typography variant="h1" weight="bold" align="center">復旧リンクを確認できません</Typography>
          <Typography variant="body-sm" color="secondary" align="center">
            {data.state.error ?? '復旧リンクが無効または期限切れです。再度復旧をお試しください。'}
          </Typography>

          <Divider />

          <Link variant="muted" href="/app/login/recovery">復旧をやり直す</Link>
        {:else}
          <Typography variant="h1" weight="bold" align="center">復旧リンクを確認中…</Typography>
          <Typography variant="body-sm" color="secondary" align="center">しばらくお待ちください。</Typography>
        {/if}
      </div>
    </Card>
  </main>

  <Divider />

  <footer class="auth-footer">
    <Link variant="muted" href="/">公開サイトに戻る</Link>
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

  :global(.auth-logo) {
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

  :global(.auth-card) {
    width: 100%;
  }

  .auth-card-content {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: var(--spacing-md);
    text-align: center;
  }

  .auth-footer {
    display: flex;
    justify-content: center;
    padding: var(--spacing-md) 0;
  }
</style>
