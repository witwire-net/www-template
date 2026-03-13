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
      <h1>プロフィール画面プレースホルダー</h1>
    </div>
  </header>

  <div class="placeholder-card">
    <p class="lead">
      `/app/profiles` は認証 shell の配置先として残しつつ、backend では専用の `/api/v1/app/*`
      endpoint が有効になりました。
    </p>

    <div class="boundary-list">
      {#each boundaries as item (item)}
        <article>
          <h2>{item}</h2>
        </article>
      {/each}
    </div>

    <p class="muted">
      次の差分では、この token 境界を frontend の domain facade と接続して CSR から実データを描画できます。
    </p>
  </div>
</section>

<style>
  .console-shell {
    display: grid;
    gap: 1.5rem;
  }

  .console-header {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 1rem;
  }

  .eyebrow {
    margin-bottom: 0.35rem;
    font-size: 0.72rem;
    font-weight: 700;
    letter-spacing: 0.2em;
    color: #c2410c;
  }

  h1 {
    margin: 0;
    font-size: clamp(1.8rem, 4vw, 2.6rem);
    color: #111827;
  }

  .placeholder-card {
    display: grid;
    gap: 1rem;
    padding: 1.25rem;
    border: 1px solid rgba(15, 23, 42, 0.08);
    border-radius: 1.25rem;
    background: rgba(255, 255, 255, 0.78);
  }

  .lead {
    margin: 0;
    color: #334155;
    line-height: 1.7;
  }

  .boundary-list {
    display: grid;
    gap: 0.85rem;
  }

  article {
    padding: 1rem 1.1rem;
    border-radius: 1rem;
    border: 1px solid rgba(15, 23, 42, 0.08);
    background: linear-gradient(135deg, rgba(255, 247, 237, 0.94), rgba(255, 255, 255, 0.98));
  }

  h2 {
    margin: 0;
    font-size: 1rem;
    color: #7c2d12;
  }

  .muted {
    margin: 0;
    color: #64748b;
  }
</style>
