<script lang="ts">
  import type { RecoveryReadySnapshot } from '@www-template/domain/hooks/auth/useRecoveryFlow';

  import { goto } from '$app/navigation';

  import { useRecoveryFlow } from '@www-template/domain/hooks/auth/useRecoveryFlow';
  import { Button, Card, CardContent, Separator } from '@www-template/ui/components';

  const RECOVERY_SNAPSHOT_KEY = 'www-template:recovery-snapshot';

  const { data, actions } = useRecoveryFlow();

  async function handleRegisterPasskey() {
    const result = await actions.registerRecoveryPasskey();
    if (result === null && data.state.phase !== 'registering' && data.state.error === null) {
      sessionStorage.removeItem(RECOVERY_SNAPSHOT_KEY);
      await goto('/');
    }
  }

  /**
   * Snapshot が正しい shape かを検証する。
   * 全フィールドが存在し string であれば有効とみなす。
   */
  function isValidSnapshot(value: unknown): value is RecoveryReadySnapshot {
    if (typeof value !== 'object' || value === null) {
      return false;
    }
    const obj = value as Record<string, unknown>;
    return (
      typeof obj.requestId === 'string' &&
      typeof obj.recoveryTokenId === 'string' &&
      typeof obj.recoverySessionId === 'string' &&
      typeof obj.recoverySession === 'string' &&
      typeof obj.expiresAt === 'string'
    );
  }

  /**
   * recovery session が無い場合は sessionStorage から復元を試みる。
   * 復元に失敗した場合は recovery ページへ戻す。
   */
  if (typeof window !== 'undefined' && data.state.recoverySession === null) {
    const raw = sessionStorage.getItem(RECOVERY_SNAPSHOT_KEY);
    if (raw !== null) {
      try {
        const parsed: unknown = JSON.parse(raw);
        if (isValidSnapshot(parsed)) {
          actions.restoreReadyState(parsed);
        } else {
          sessionStorage.removeItem(RECOVERY_SNAPSHOT_KEY);
          void goto('/login/recovery');
        }
      } catch {
        sessionStorage.removeItem(RECOVERY_SNAPSHOT_KEY);
        void goto('/login/recovery');
      }
      sessionStorage.removeItem(RECOVERY_SNAPSHOT_KEY);
    } else {
      void goto('/login/recovery');
    }
  }
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
          <h1 class="auth-title">パスキー再登録</h1>
          <p class="auth-desc">
            新しいパスキーを登録して、アカウントへのアクセスを回復してください。
          </p>

          {#if data.state.error}
            <p class="auth-error" role="alert">{data.state.error}</p>
          {/if}

          <Button
            class="w-full"
            type="button"
            disabled={data.state.phase === 'registering'}
            onclick={handleRegisterPasskey}
          >
            {#if data.state.phase === 'registering'}
              登録中…
            {:else}
              新しいパスキーを登録
            {/if}
          </Button>

          <Separator />

          <a href="/login/recovery" class="link-muted">復旧をやり直す</a>
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

  .auth-error {
    color: var(--destructive);
    font-size: 0.875rem;
    margin: 0;
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
