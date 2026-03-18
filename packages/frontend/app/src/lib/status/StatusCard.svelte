<script lang="ts">
  import { useStatus } from '@www-template-frontend/domain';
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
      <p class="muted">初期表示は SSR のプレースホルダーです。</p>
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
    gap: 1rem;
    padding: 1.5rem;
    border: 1px solid rgba(15, 23, 42, 0.08);
    border-radius: 1.5rem;
    background:
      linear-gradient(180deg, rgba(255, 255, 255, 0.96), rgba(248, 250, 252, 0.92)),
      linear-gradient(135deg, rgba(12, 74, 110, 0.08), rgba(190, 24, 93, 0.08));
    box-shadow: 0 24px 60px rgba(15, 23, 42, 0.08);
  }

  .eyebrow {
    font-size: 0.75rem;
    font-weight: 700;
    letter-spacing: 0.24em;
    color: #0f766e;
  }

  h2 {
    margin: 0;
    font-size: clamp(1.4rem, 3vw, 2rem);
    line-height: 1.1;
    color: #0f172a;
  }

  p {
    margin: 0;
    color: #475569;
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
    border-radius: 999px;
    background: rgba(15, 23, 42, 0.06);
    color: #0f172a;
    font-size: 0.9rem;
    font-weight: 600;
  }

  .pill.success {
    background: rgba(13, 148, 136, 0.14);
    color: #115e59;
  }

  .muted {
    color: #64748b;
  }

  .error {
    color: #b91c1c;
    font-weight: 600;
  }
</style>
