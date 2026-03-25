<script lang="ts">
  import { useRecoveryFlow } from '@www-template-frontend/domain/hooks/auth/useRecoveryFlow';
  import { Card, Divider, Link, StatusIcon, Typography } from '@www-template-frontend/ui/components';

  const { data } = useRecoveryFlow();
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
      <div class="auth-card-content">
        <StatusIcon variant="success" size="md" style="outlined" />
        <Typography variant="h1" weight="bold" align="center">
          {data.state.sentView.title}
        </Typography>
        <Typography variant="body-sm" color="secondary" align="center">
          {data.state.sentView.description}
        </Typography>
        <Typography variant="caption" color="muted" align="center">
          {data.state.sentView.helper}
        </Typography>

        <Divider />

        <Link variant="muted" href="/app/login">ログインに戻る</Link>
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
