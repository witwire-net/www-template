<script lang="ts">
  import StatusCard from '../lib/status/StatusCard.svelte';

  const highlights = [
    'public route は SSR のまま `/app/*` の外に維持',
    '認証 UI は `/app/*` の CSR アプリとして分離',
    'passkey / recovery を含む認証 API を TypeSpec と Go backend で共有',
    '画面から API へ直接触れず domain hook を経由',
  ];
</script>

<section class="hero-grid">
  <div class="hero-copy">
    <div class="eyebrow">SvelteKit + Cloudflare Workers</div>
    <h1>公開面と認証面を再利用しやすい構成でまとめています。</h1>
    <p>
      `packages/frontend/web` は公開ルートの SSR を担い、`packages/frontend/app` は `/app/*` の
      SvelteKit SPA を担います。
    </p>
    <div class="cta-row">
      <a class="primary" href="/app/login">ログインを試す</a>
      <a class="secondary" href="/app">認証アプリを開く</a>
    </div>

    <ul>
      {#each highlights as item (item)}
        <li>{item}</li>
      {/each}
    </ul>
  </div>

  <StatusCard />
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

  .cta-row {
    display: flex;
    flex-wrap: wrap;
    gap: 0.8rem;
  }

  .primary,
  .secondary {
    display: inline-flex;
    align-items: center;
    padding: 0.85rem 1.15rem;
    border-radius: var(--radius-full);
    font-weight: var(--font-weight-bold);
  }

  .primary {
    background: var(--gradient-primary);
    color: var(--color-text-on-brand);
  }

  .secondary {
    background: color-mix(in srgb, var(--color-surface) 80%, transparent);
    color: var(--color-text);
    border: 1px solid var(--color-border);
  }

  ul {
    margin: 0;
    padding-left: 1.2rem;
    display: grid;
    gap: 0.55rem;
  }
</style>
