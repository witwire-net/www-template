<script lang="ts">
  import { Button, Card, Divider, Link, StatusIcon, Typography } from '@www-template-frontend/ui/components';
</script>

<div class="session-expired-shell">
  <header class="session-expired-header">
    <Link variant="ghost" href="/" aria-label="www-template トップページ">
      <Typography variant="body" weight="bold" className="session-expired-logo">www-template</Typography>
    </Link>
  </header>

  <Divider />

  <main class="session-expired-main">
    <Card padding="xl" className="session-expired-card">
      <div class="session-expired-content">
        <StatusIcon variant="warning" size="md" style="outlined" />
        <Typography variant="h1" weight="bold" align="center">
          セッションが切れました
        </Typography>
        <Typography variant="body-sm" color="secondary" align="center">
          セキュリティのため、セッションが終了しました。再度ログインしてください。
        </Typography>
        <Button variant="primary" fullWidth onclick={() => { window.location.href = '/app/login'; }}>
          ログインへ
        </Button>
        <Divider />
        <Link variant="muted" href="/">公開サイトへ</Link>
      </div>
    </Card>
  </main>
</div>

<style>
  .session-expired-shell {
    display: flex;
    flex-direction: column;
    align-items: center;
    min-height: 100vh;
    padding: var(--spacing-xl) var(--spacing-md);
    font-family: var(--font-family-sans);
    background: var(--color-background);
    color: var(--color-text);
  }

  .session-expired-header {
    display: flex;
    justify-content: center;
    padding: var(--spacing-md) 0;
  }

  :global(.session-expired-logo) {
    letter-spacing: 0.08em;
  }

  .session-expired-main {
    display: flex;
    flex: 1;
    align-items: center;
    justify-content: center;
    width: 100%;
    max-width: 480px;
    padding: var(--spacing-xl) 0;
  }

  :global(.session-expired-card) {
    width: 100%;
  }

  .session-expired-content {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: var(--spacing-md);
    text-align: center;
  }
</style>
