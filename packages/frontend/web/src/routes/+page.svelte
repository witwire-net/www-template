<script lang="ts">
  const highlights = [
    'public route は SSR の SvelteKit として運用',
    '認証 UI は別ドメインの CSR アプリとして分離',
  ];
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
</style>
