<script lang="ts">
  import { useRecoveryFlow } from '@www-template/domain/hooks/auth/useRecoveryFlow';
  import { Card, CardContent, Separator } from '@www-template/ui/components';

  const { data } = useRecoveryFlow();
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
        <div class="auth-card">
          <span class="status-icon status-icon--success" aria-hidden="true">
            <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" width="48" height="48">
              <path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"/>
              <path d="m9 11 3 3L22 4"/>
            </svg>
          </span>
          <h1 class="auth-card__title">{data.state.sentView.title}</h1>
          <p class="auth-card__desc">{data.state.sentView.description}</p>
          <p class="auth-card__desc">{data.state.sentView.helper}</p>

          <Separator />

          <a href="/login" class="link-muted">ログインに戻る</a>
        </div>
      </CardContent>
    </Card>
  </main>

  <Separator />

  <footer class="auth-layout__footer">
    <a href="/" class="link-muted">公開サイトに戻る</a>
  </footer>
</div>
