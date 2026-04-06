<script lang="ts">
  import { goto } from '$app/navigation';

  import { Button, Card, CardContent, Separator } from '@www-template/ui/components';
</script>

<div class="session-expired-shell">
  <header class="session-expired-header">
    <a href="/" class="site-link" aria-label="www-template トップページ">
      <span class="logo-text">www-template</span>
    </a>
  </header>

  <Separator />

  <main class="session-expired-main">
    <Card class="w-full">
      <CardContent class="session-expired-content">
        <span class="status-icon status-icon--warning" aria-hidden="true">
          <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" width="48" height="48">
            <path d="m21.73 18-8-14a2 2 0 0 0-3.48 0l-8 14A2 2 0 0 0 4 21h16a2 2 0 0 0 1.73-3Z"/>
            <path d="M12 9v4"/>
            <path d="M12 17h.01"/>
          </svg>
        </span>
        <h1 class="card-title">セッションが切れました</h1>
        <p class="card-desc">
          セキュリティのため、セッションが終了しました。再度ログインしてください。
        </p>
        <Button class="w-full" onclick={() => { void goto('/login'); }}>
          ログインへ
        </Button>
        <Separator />
        <a href="/" class="link-muted">公開サイトへ</a>
      </CardContent>
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

  .site-link {
    text-decoration: none;
    color: inherit;
  }

  .logo-text {
    font-weight: bold;
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

  :global(.session-expired-content) {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: var(--spacing-md);
    text-align: center;
  }

  .status-icon--warning {
    color: var(--color-warning, #f59e0b);
  }

  .card-title {
    margin: 0;
    font-size: 1.5rem;
    font-weight: bold;
    text-align: center;
  }

  .card-desc {
    margin: 0;
    font-size: 0.875rem;
    color: var(--muted-foreground);
    text-align: center;
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
