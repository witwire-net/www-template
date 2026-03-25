<script lang="ts">
  const boundaries = [
    '認証面の profiles API は `/api/v1/app/profiles` と `/api/v1/app/profiles/{id}` に分離されています。',
    'app surface には `Authorization: Bearer <token>` が必須です。',
    '開発既定 token は `dev-app-auth` で、`APP_BEARER_TOKEN` で上書きできます。',
  ];
</script>

<section class="console-shell">
  <header class="console-header">
    <div>
      <div class="eyebrow">AUTH CSR SHELL</div>
      <h1>プロフィール画面サンプル</h1>
    </div>
  </header>

  <div class="placeholder-card">
    <p class="lead">
      `/app/profiles` は認証済み画面の最小サンプルです。backend では専用の `/api/v1/app/*`
      endpoint を使い、公開面と app 面の境界を分けています。
    </p>

    <div class="boundary-list">
      {#each boundaries as item (item)}
        <article>
          <h2>{item}</h2>
        </article>
      {/each}
    </div>

    <p class="muted">
      bearer token の境界、公開 API との差分、認証済み CSR の配置を確認するためのサンプルとして使えます。
    </p>
  </div>
</section>

<style>
  .console-shell {
    display: grid;
    gap: var(--spacing-lg);
  }

  .console-header {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: var(--spacing-md);
  }

  .eyebrow {
    margin-bottom: 0.35rem;
    font-family: var(--font-family-display);
    font-size: var(--font-size-xs);
    font-weight: var(--font-weight-bold);
    letter-spacing: 0.2em;
    color: var(--color-primary-active);
  }

  h1 {
    margin: 0;
    font-family: var(--font-family-display);
    font-size: clamp(1.8rem, 4vw, 2.6rem);
    color: var(--color-text);
  }

  .placeholder-card {
    display: grid;
    gap: var(--spacing-md);
    padding: 1.25rem;
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md);
    background: color-mix(in srgb, var(--color-surface) 78%, transparent);
  }

  .lead {
    margin: 0;
    color: var(--color-text-secondary);
    line-height: var(--line-height-relaxed);
  }

  .boundary-list {
    display: grid;
    gap: 0.85rem;
  }

  article {
    padding: var(--spacing-md) 1.1rem;
    border-radius: var(--radius-md);
    border: 1px solid var(--color-border);
    background: linear-gradient(135deg, var(--color-background), var(--color-surface));
  }

  h2 {
    margin: 0;
    font-size: var(--font-size-base);
    color: var(--color-text);
  }

  .muted {
    margin: 0;
    color: var(--color-text-muted);
  }
</style>
