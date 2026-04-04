<script lang="ts">
  import { useRecoveryFlow } from '@www-template/domain/hooks/auth/useRecoveryFlow';
  import { Card, CardContent, Separator } from '@www-template/ui/components';

  const { data } = useRecoveryFlow();
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
        <div class="auth-card-content">
          <span class="status-icon status-icon--success" aria-hidden="true">
            <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" width="48" height="48">
              <path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"/>
              <path d="m9 11 3 3L22 4"/>
            </svg>
          </span>
          <h1 class="auth-title">{data.state.sentView.title}</h1>
          <p class="auth-desc">{data.state.sentView.description}</p>
          <p class="auth-caption">{data.state.sentView.helper}</p>

          <Separator />

          <a href="/login" class="link-muted">ログインに戻る</a>
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

  .status-icon--success {
    color: var(--color-success, #22c55e);
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

  .auth-caption {
    margin: 0;
    font-size: 0.75rem;
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
