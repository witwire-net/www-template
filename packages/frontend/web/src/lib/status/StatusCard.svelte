<script lang="ts">
  import { useStatus } from '@www-template-frontend/domain/hooks/status/useStatus';
  import { Button } from '@www-template-frontend/ui/components';

  const status = useStatus();
  const statusState = status.data.state;

  const formatTimestamp = (value: Date | null): string => {
    if (value === null) {
      return '';
    }

    return new Intl.DateTimeFormat('ja-JP', {
      dateStyle: 'medium',
      timeStyle: 'short',
    }).format(value);
  };
</script>

<section class="status-card">
  <div class="eyebrow">PUBLIC API STATUS</div>
  <h2>公開面は SSR、データ更新は domain 経由です。</h2>
  <p>
    まずは静的 HTML を返し、hydrate 後に `packages/frontend/domain` から公開 API を読み込みます。
  </p>

  <div class="status-body">
    {#if statusState.isLoading && statusState.timestamp === null}
      <p class="muted">公開ステータスを読み込み中です。</p>
    {:else if statusState.error != null}
      <p class="error">{statusState.error}</p>
    {:else if statusState.timestamp !== null}
      <div class="pill-row">
        <span class="pill success">{statusState.message}</span>
        <span class="pill">{formatTimestamp(statusState.timestamp)}</span>
      </div>
    {:else}
      <p class="muted">初期表示後に公開 API の状態を取得できます。</p>
    {/if}
  </div>

  <Button
    type="button"
    onclick={() => {
      void status.actions.refresh();
    }}
  >
    公開 API を再取得
  </Button>
</section>

<style>
  .status-card {
    display: grid;
    gap: var(--spacing-md);
    padding: var(--spacing-lg);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-lg);
    background:
      linear-gradient(180deg, var(--color-surface), color-mix(in srgb, var(--color-background) 92%, transparent)),
      linear-gradient(135deg, color-mix(in srgb, var(--color-primary) 8%, transparent), color-mix(in srgb, var(--color-primary-active) 8%, transparent));
  }

  .eyebrow {
    font-family: var(--font-family-display);
    font-size: var(--font-size-xs);
    font-weight: var(--font-weight-bold);
    letter-spacing: 0.24em;
    color: var(--color-primary-active);
  }

  h2 {
    margin: 0;
    font-family: var(--font-family-display);
    font-size: clamp(1.4rem, 3vw, 2rem);
    line-height: var(--line-height-tight);
    color: var(--color-text);
  }

  p {
    margin: 0;
    color: var(--color-text-secondary);
    line-height: 1.6;
  }

  .status-body {
    min-height: 2.5rem;
  }

  .pill-row {
    display: flex;
    flex-wrap: wrap;
    gap: 0.75rem;
  }

  .pill {
    display: inline-flex;
    align-items: center;
    padding: 0.55rem 0.9rem;
    border-radius: var(--radius-full);
    background: var(--color-surface-hover);
    color: var(--color-text);
    font-size: var(--font-size-sm);
    font-weight: 600;
  }

  .pill.success {
    background: color-mix(in srgb, var(--color-success) 14%, transparent);
    color: var(--color-success);
  }

  .muted {
    color: var(--color-text-muted);
  }

  .error {
    color: var(--color-error);
    font-weight: 600;
  }
</style>
