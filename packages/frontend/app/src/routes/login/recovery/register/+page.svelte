<script lang="ts">
  import type { RecoveryReadySnapshot } from '@www-template-frontend/domain/hooks/auth/useRecoveryFlow';

  import { useRecoveryFlow } from '@www-template-frontend/domain/hooks/auth/useRecoveryFlow';
  import { Button, Card, Divider, Link, Typography } from '@www-template-frontend/ui/components';

  const RECOVERY_SNAPSHOT_KEY = 'www-template:recovery-snapshot';

  const { data, actions } = useRecoveryFlow();

  async function handleRegisterPasskey() {
    const result = await actions.registerRecoveryPasskey();
    if (result === '/app') {
      sessionStorage.removeItem(RECOVERY_SNAPSHOT_KEY);
      window.location.href = '/app';
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
          window.location.href = '/app/login/recovery';
        }
      } catch {
        sessionStorage.removeItem(RECOVERY_SNAPSHOT_KEY);
        window.location.href = '/app/login/recovery';
      }
      sessionStorage.removeItem(RECOVERY_SNAPSHOT_KEY);
    } else {
      window.location.href = '/app/login/recovery';
    }
  }
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
        <Typography variant="h1" weight="bold" align="center">パスキー再登録</Typography>
        <Typography variant="body-sm" color="secondary" align="center">
          新しいパスキーを登録して、アカウントへのアクセスを回復してください。
        </Typography>

        {#if data.state.error}
          <Typography variant="body-sm" className="auth-error" role="alert">
            {data.state.error}
          </Typography>
        {/if}

        <Button
          variant="primary"
          fullWidth
          type="button"
          disabled={data.state.phase === 'registering'}
          isLoading={data.state.phase === 'registering'}
          onclick={handleRegisterPasskey}
        >
          {#if data.state.phase === 'registering'}
            登録中…
          {:else}
            新しいパスキーを登録
          {/if}
        </Button>

        <Divider />

        <Link variant="muted" href="/app/login/recovery">復旧をやり直す</Link>
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

  :global(.auth-error) {
    color: var(--color-error);
  }

  .auth-footer {
    display: flex;
    justify-content: center;
    padding: var(--spacing-md) 0;
  }
</style>
