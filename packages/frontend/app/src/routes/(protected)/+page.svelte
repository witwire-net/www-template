<section class="overview">
  <div class="eyebrow">AUTH ENTRY</div>
  <h2>認証済みアプリのエントリー</h2>
  <p>
    `www-template` では bearer session 付きの CSR 領域として、認証済みフローの
    入口として運用します。
  </p>
  <a href="/logout">/logout を開く</a>
</section>

<style>
  .overview {
    display: grid;
    gap: 0.85rem;
  }

  .eyebrow {
    font-family: var(--font-family-display);
    font-size: var(--font-size-xs);
    font-weight: var(--font-weight-bold);
    letter-spacing: 0.2em;
    color: var(--color-info);
  }

  h2,
  p {
    margin: 0;
  }

  p {
    color: var(--color-text-secondary);
    line-height: 1.6;
  }

  a {
    display: inline-flex;
    align-items: center;
    width: fit-content;
    padding: 0.75rem var(--spacing-md);
    border-radius: var(--radius-full);
    background: color-mix(in srgb, var(--color-info) 10%, transparent);
    color: var(--color-info);
    font-weight: var(--font-weight-bold);
  }
</style>
