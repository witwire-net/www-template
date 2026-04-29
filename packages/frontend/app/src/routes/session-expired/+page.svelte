<script lang="ts">
  import { goto } from '$app/navigation';

  import { Button, Card, CardContent, Separator } from '@www-template/ui/components';
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
      <CardContent class="session-expired-content">
        <span class="status-icon status-icon--warning" aria-hidden="true">
          <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" width="48" height="48">
            <path d="m21.73 18-8-14a2 2 0 0 0-3.48 0l-8 14A2 2 0 0 0 4 21h16a2 2 0 0 0 1.73-3Z"/>
            <path d="M12 9v4"/>
            <path d="M12 17h.01"/>
          </svg>
        </span>
        <h1 class="auth-card__title">セッションが切れました</h1>
        <p class="auth-card__desc">
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
