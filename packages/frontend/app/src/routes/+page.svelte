<script lang="ts">
  import StatusCard from '../lib/status/StatusCard.svelte';

  const highlights = [
    'public route は SSR のまま `/app/*` の外に維持',
    '認証 UI は `/app/profiles` へ集約して CSR shell 化',
    '画面から API へ直接触れず domain の facade/store を経由',
  ];
</script>

<section class="hero-grid">
  <div class="hero-copy">
    <div class="eyebrow">SvelteKit + Cloudflare Workers</div>
    <h1>公開面と認証面を 1 つの app に整理しました。</h1>
    <p>
      `packages/frontend/app` は SvelteKit に寄せ、公開ルートは SSR、認証ルートは `/app/*` 配下の
      CSR として分離します。
    </p>
  <div class="cta-row">
      <a class="primary" href="/timeline">公開 timeline を見る</a>
      <a class="secondary" href="/app/profiles">認証 shell を開く</a>
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
    gap: 1.5rem;
    grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
    align-items: start;
  }

  .hero-copy {
    display: grid;
    gap: 1.2rem;
    padding: clamp(1.6rem, 4vw, 2.4rem);
    border-radius: 1.75rem;
    background: linear-gradient(135deg, rgba(255, 255, 255, 0.88), rgba(255, 237, 213, 0.92));
    box-shadow: 0 30px 70px rgba(15, 23, 42, 0.08);
  }

  .eyebrow {
    font-size: 0.78rem;
    font-weight: 700;
    letter-spacing: 0.22em;
    color: #ea580c;
  }

  h1 {
    margin: 0;
    font-size: clamp(2.2rem, 6vw, 4.5rem);
    line-height: 0.95;
    letter-spacing: -0.04em;
  }

  p,
  li {
    color: #475569;
    line-height: 1.7;
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
    border-radius: 999px;
    font-weight: 700;
  }

  .primary {
    background: linear-gradient(135deg, #111827, #0f766e);
    color: white;
  }

  .secondary {
    background: rgba(255, 255, 255, 0.8);
    color: #0f172a;
    border: 1px solid rgba(15, 23, 42, 0.08);
  }

  ul {
    margin: 0;
    padding-left: 1.2rem;
    display: grid;
    gap: 0.55rem;
  }
</style>
