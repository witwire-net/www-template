<script lang="ts">
  import { PUBLIC_APP_URL } from '$env/static/public';
  import { useStatus } from '@www-template/domain';

  const highlights = [
    'public route は SSR の SvelteKit として運用',
    '認証 UI は別ドメインの CSR アプリとして分離',
  ];

  const { data, actions } = useStatus();
</script>

<section class="hero-grid">
  <div class="hero-copy">
    <div class="eyebrow">SvelteKit + Cloudflare Workers</div>
    <h1>公開面と認証面を再利用しやすい構成でまとめています。</h1>
    <p>
      `packages/frontend/web` は公開ルートの SSR を担い、`packages/frontend/app` は
      別ドメインの SvelteKit SPA を担います。
    </p>
    <ul>
      {#each highlights as item (item)}
        <li>{item}</li>
      {/each}
    </ul>
    <a href="{PUBLIC_APP_URL}/login" class="cta-link">ログインを試す</a>
  </div>

  <div class="status-card">
    <h2>公開面は SSR、データ更新は domain 経由です。</h2>
    <p class="status-desc">
      {#if data.state.isLoading}
        取得中…
      {:else if data.state.error !== undefined}
        エラー: {data.state.error}
      {:else if data.state.message !== ''}
        {data.state.message}
      {:else}
        未取得
      {/if}
    </p>
    <button class="fetch-btn" onclick={() => actions.refresh()} disabled={data.state.isLoading}>
      公開 API を再取得
    </button>
  </div>
</section>

<style>
  .hero-grid {
    display: grid;
    gap: var(--spacing-lg);
    grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
    align-items: start;
  }

  .hero-copy {
    display: grid;
    gap: 1.2rem;
    padding: clamp(1.6rem, 4vw, 2.4rem);
    border-radius: var(--radius-lg);
    background: linear-gradient(135deg, color-mix(in srgb, var(--color-surface) 88%, transparent), var(--color-surface));
    border: 1px solid var(--color-border-subtle);
  }

  .eyebrow {
    font-family: var(--font-family-display);
    font-size: var(--font-size-xs);
    font-weight: var(--font-weight-bold);
    letter-spacing: 0.22em;
    color: var(--color-primary-active);
  }

  h1 {
    margin: 0;
    font-family: var(--font-family-display);
    font-size: clamp(2.2rem, 6vw, 4.5rem);
    line-height: var(--line-height-tight);
    letter-spacing: -0.04em;
  }

  p,
  li {
    color: var(--color-text-secondary);
    line-height: var(--line-height-relaxed);
  }

  ul {
    margin: 0;
    padding-left: 1.2rem;
    display: grid;
    gap: 0.55rem;
  }

  .cta-link {
    display: inline-block;
    padding: 0.6rem 1.2rem;
    border-radius: var(--radius-md);
    background: var(--color-primary-active);
    color: var(--color-on-primary, #fff);
    font-weight: 600;
    text-decoration: none;
  }

  .status-card {
    display: grid;
    gap: 1rem;
    padding: clamp(1.6rem, 4vw, 2.4rem);
    border-radius: var(--radius-lg);
    background: var(--color-surface);
    border: 1px solid var(--color-border-subtle);
  }

  .status-card h2 {
    margin: 0;
    font-size: 1.25rem;
    font-weight: 700;
  }

  .status-desc {
    margin: 0;
    color: var(--color-text-secondary);
    font-size: 0.9rem;
  }

  .fetch-btn {
    align-self: start;
    padding: 0.5rem 1rem;
    border-radius: var(--radius-md);
    border: 1px solid var(--color-border-subtle);
    background: var(--color-surface-hover);
    cursor: pointer;
    font-size: 0.875rem;
  }

  .fetch-btn:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }
</style>
